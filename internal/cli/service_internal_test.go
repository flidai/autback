package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/flidai/autback/internal/config"
	autbackv1 "github.com/flidai/autback/internal/gen/rtest/v1"
	"github.com/flidai/autback/internal/gen/rtest/v1/autbackv1connect"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestParseExecUsesGenericProjectImageAndArbitraryArgv(t *testing.T) {
	settings := config.Config{Service: &config.Service{Image: "image@sha256:digest"}}
	got, err := parseExec(settings, "example", []string{"--timeout", "5m", "--workdir", "service", "--env", "CI=true", "--cache", "go-build=/root/.cache/go-build", "--cache", "modules=/go/pkg/mod", "--secret-env", "registry-token=REGISTRY_TOKEN", "--secret-file", "signing-key=/run/secrets/signing-key", "--", "task", "test", "--race"})
	if err != nil {
		t.Fatal(err)
	}
	if got.project != "example" || got.image != "image@sha256:digest" || got.timeout != 5*time.Minute || got.workdir != "service" || got.environment["CI"] != "true" {
		t.Fatalf("options = %#v", got)
	}
	if !reflect.DeepEqual(got.command, []string{"task", "test", "--race"}) {
		t.Fatalf("command = %#v", got.command)
	}
	if len(got.caches) != 2 || got.caches[0].Name != "go-build" || got.caches[0].Target != "/root/.cache/go-build" || got.caches[1].Name != "modules" || got.caches[1].Target != "/go/pkg/mod" {
		t.Fatalf("caches = %#v", got.caches)
	}
	if len(got.secrets) != 2 || got.secrets[0].Name != "registry-token" || got.secrets[0].GetEnvironment() != "REGISTRY_TOKEN" || got.secrets[1].GetFile() != "/run/secrets/signing-key" {
		t.Fatalf("secrets = %#v", got.secrets)
	}
}

func TestParseExecRequiresExplicitCommandBoundary(t *testing.T) {
	settings := config.Config{Service: &config.Service{Image: "image"}}
	if _, err := parseExec(settings, "example", []string{"go", "test", "./..."}); err == nil {
		t.Fatal("exec accepted a command without -- boundary")
	}
}

func TestParseExecAllowsServerOwnedDefaultImage(t *testing.T) {
	settings := config.Config{Service: &config.Service{}}
	got, err := parseExec(settings, "example", []string{"--", "go", "version"})
	if err != nil {
		t.Fatal(err)
	}
	if got.project != "example" || got.image != "" || !reflect.DeepEqual(got.command, []string{"go", "version"}) {
		t.Fatalf("options = %#v", got)
	}
}

func TestUsageContainsOnlySharedServiceCommands(t *testing.T) {
	var output bytes.Buffer
	usage(&output)
	text := output.String()
	for _, removed := range []string{
		"autback run ",
		"<suite>",
		"legacy",
		"backend",
	} {
		if strings.Contains(text, removed) {
			t.Fatalf("usage retains removed client path %q:\n%s", removed, text)
		}
	}
	for _, required := range []string{" exec [", " build [", "autback doctor"} {
		if !strings.Contains(text, required) {
			t.Fatalf("usage missing %q:\n%s", required, text)
		}
	}
}

func TestDefaultExecWorkingDirectoryFollowsInvocationDirectory(t *testing.T) {
	root := t.TempDir()
	nested := filepath.Join(root, "services", "api")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}
	got, err := defaultExecWorkingDirectory(root, nested)
	if err != nil {
		t.Fatal(err)
	}
	if got != filepath.Join("services", "api") {
		t.Fatalf("working directory = %q", got)
	}
	if got, err := defaultExecWorkingDirectory(root, root); err != nil || got != "." {
		t.Fatalf("root working directory = %q, err = %v", got, err)
	}
}

func TestDefaultExecWorkingDirectoryCanonicalizesSymlinkedInvocationPath(t *testing.T) {
	root := t.TempDir()
	nested := filepath.Join(root, "services", "api")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(t.TempDir(), "linked-worktree")
	if err := os.Symlink(root, link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	got, err := defaultExecWorkingDirectory(root, filepath.Join(link, "services", "api"))
	if err != nil {
		t.Fatal(err)
	}
	if got != filepath.Join("services", "api") {
		t.Fatalf("working directory = %q", got)
	}
}

func TestImageActivateUsesRepositoryProjectAndPinnedImage(t *testing.T) {
	service := &projectListService{projects: []*autbackv1.Project{{Id: "prj1", Slug: "example", Name: "Example"}}}
	client, closeServer := testServiceClient(t, service)
	defer closeServer()
	image := "ghcr.io/example/runner@sha256:" + strings.Repeat("a", 64)
	var stdout, stderr bytes.Buffer
	code := serviceImage(context.Background(), client, config.Config{}, "example", []string{"activate", "--image", image}, IO{Stdout: &stdout, Stderr: &stderr})
	if code != 0 {
		t.Fatalf("code=%d stderr=%q", code, stderr.String())
	}
	service.mu.Lock()
	gotProject, gotImage := service.activatedProject, service.activatedImage
	service.mu.Unlock()
	if gotProject != "example" || gotImage != image || !strings.Contains(stdout.String(), image) {
		t.Fatalf("project=%q image=%q stdout=%q", gotProject, gotImage, stdout.String())
	}
}

func TestGlobalTokenIsRemovedBeforeCommandDispatch(t *testing.T) {
	token, args, err := globalArgs([]string{"--token", "secret", "exec", "--", "true"})
	if err != nil || token != "secret" || !reflect.DeepEqual(args, []string{"exec", "--", "true"}) {
		t.Fatalf("token=%q args=%#v err=%v", token, args, err)
	}
}

func TestExplicitProjectRejectsConflictsAndStopsAtCommandBoundary(t *testing.T) {
	project, err := explicitProject([]string{"--image", "example", "--project", "one", "--", "tool", "--project", "ignored"})
	if err != nil || project != "one" {
		t.Fatalf("project=%q err=%v", project, err)
	}
	if _, err := explicitProject([]string{"--project", "one", "--project", "two", "--", "true"}); err == nil {
		t.Fatal("conflicting project flags were accepted")
	}
}

func TestImageHelpersPreserveBuildxArgumentsAndNormalizeTags(t *testing.T) {
	project, args, err := consumeProject("", []string{"--project", "example", "--", "--label", "value=--project"})
	if err != nil || project != "example" || !reflect.DeepEqual(args, []string{"--", "--label", "value=--project"}) {
		t.Fatalf("project=%q args=%#v err=%v", project, args, err)
	}
	for input, want := range map[string]string{
		"ghcr.io/acme/runner:latest": "ghcr.io/acme/runner",
		"localhost:5000/runner:dev":  "localhost:5000/runner",
		"localhost:5000/runner":      "localhost:5000/runner",
	} {
		if got := repositoryFromTag(input); got != want {
			t.Fatalf("repositoryFromTag(%q)=%q, want %q", input, got, want)
		}
	}
}

func TestServiceLoginExchangesEnrollmentFromStdinWithoutEchoingSecrets(t *testing.T) {
	code := "autback_enr_tok123_" + strings.Repeat("a", 43)
	token := "autback_dt_tok456_" + strings.Repeat("b", 43)
	service := &enrollmentService{wantCode: code, token: token}
	path, handler := autbackv1connect.NewControlServiceHandler(service)
	mux := http.NewServeMux()
	mux.Handle(path, handler)
	server := httptest.NewServer(mux)
	defer server.Close()
	keyring := &memoryKeyring{}
	var stdout, stderr bytes.Buffer
	settings := config.Config{URL: server.URL, Service: &config.Service{}}
	result := serviceLogin(context.Background(), settings, "", []string{"--recovery-code"}, IO{
		Stdin: strings.NewReader(code + "\n"), Stdout: &stdout, Stderr: &stderr, Keyring: keyring,
	})
	if result != 0 {
		t.Fatalf("result=%d stderr=%q", result, stderr.String())
	}
	if keyring.token != token || service.gotCode != code {
		t.Fatalf("stored=%q exchanged=%q", keyring.token, service.gotCode)
	}
	if strings.Contains(stdout.String()+stderr.String(), code) || strings.Contains(stdout.String()+stderr.String(), token) {
		t.Fatalf("secret was echoed: stdout=%q stderr=%q", stdout.String(), stderr.String())
	}
	if result := serviceLogout(settings, IO{Stdout: &stdout, Stderr: &stderr, Keyring: keyring}); result != 0 || keyring.token != "" {
		t.Fatalf("logout result=%d stored=%q", result, keyring.token)
	}
}

func TestServiceLoginOpensBrowserAndStoresIssuedDeviceCredential(t *testing.T) {
	token := "autback_dt_tok456_" + strings.Repeat("b", 43)
	service := &enrollmentService{token: token}
	path, connectHandler := autbackv1connect.NewControlServiceHandler(service)
	mux := http.NewServeMux()
	mux.Handle(path, connectHandler)
	mux.HandleFunc("POST /auth/cli/start", func(response http.ResponseWriter, request *http.Request) {
		var input map[string]string
		_ = json.NewDecoder(request.Body).Decode(&input)
		if input["device_name"] != "work-laptop" {
			t.Fatalf("device name = %q", input["device_name"])
		}
		response.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(response).Encode(map[string]any{
			"device_code": "one-time-device-code", "user_code": "ABCD-EFGH",
			"verification_uri":          "https://console.autback.dev/auth/device",
			"verification_uri_complete": "https://console.autback.dev/auth/device?code=ABCD-EFGH",
			"expires_in_seconds":        600, "interval_seconds": 1,
		})
	})
	mux.HandleFunc("POST /auth/cli/token", func(response http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(response).Encode(map[string]any{"status": "authorized", "token": token, "token_id": "tok456"})
	})
	server := httptest.NewServer(mux)
	defer server.Close()
	keyring := &memoryKeyring{}
	var stdout, stderr bytes.Buffer
	var opened string
	settings := config.Config{URL: server.URL, Service: &config.Service{}}
	result := serviceLogin(context.Background(), settings, "", []string{"--device", "work-laptop"}, IO{
		Stdout: &stdout, Stderr: &stderr, Keyring: keyring,
		OpenURL: func(target string) error { opened = target; return nil },
		Wait:    func(context.Context, time.Duration) error { return nil },
	})
	if result != 0 {
		t.Fatalf("result=%d stderr=%q", result, stderr.String())
	}
	if opened != "https://console.autback.dev/auth/device?code=ABCD-EFGH" || keyring.token != token {
		t.Fatalf("opened=%q stored=%q", opened, keyring.token)
	}
	if !strings.Contains(stdout.String(), "ABCD-EFGH") || strings.Contains(stdout.String()+stderr.String(), token) || strings.Contains(stdout.String()+stderr.String(), "one-time-device-code") {
		t.Fatalf("stdout=%q stderr=%q", stdout.String(), stderr.String())
	}
}

func TestServiceAdminBindsGitHubIdentity(t *testing.T) {
	service := &identityService{}
	client, closeServer := testServiceClient(t, service)
	defer closeServer()
	var stdout, stderr bytes.Buffer
	code := serviceAdmin(context.Background(), client, []string{"identity", "github", "--user", "usr1", "--login", "yacobolo"}, IO{Stdout: &stdout, Stderr: &stderr})
	if code != 0 || service.userID != "usr1" || service.login != "yacobolo" || !strings.Contains(stdout.String(), "12345678") {
		t.Fatalf("code=%d user=%q login=%q stdout=%q stderr=%q", code, service.userID, service.login, stdout.String(), stderr.String())
	}
}

func TestServiceInitWritesOnlyAnAuthorizedProjectLink(t *testing.T) {
	service := &projectListService{projects: []*autbackv1.Project{
		{Id: "prj1", Slug: "one", Name: "One"}, {Id: "prj2", Slug: "two", Name: "Two"},
	}}
	client, closeServer := testServiceClient(t, service)
	defer closeServer()
	directory := t.TempDir()
	initGitRepository(t, directory)
	var stdout, stderr bytes.Buffer
	code := serviceInit(context.Background(), client, []string{"--project", "two"}, IO{Dir: directory, Stdout: &stdout, Stderr: &stderr})
	if code != 0 {
		t.Fatalf("code=%d stderr=%q", code, stderr.String())
	}
	data, err := os.ReadFile(filepath.Join(directory, "autback.json"))
	if err != nil || string(data) != "{\n  \"project\": \"two\"\n}\n" {
		t.Fatalf("link=%q err=%v", data, err)
	}

	other := t.TempDir()
	initGitRepository(t, other)
	stdout.Reset()
	stderr.Reset()
	code = serviceInit(context.Background(), client, []string{"--project", "unauthorized"}, IO{Dir: other, Stdout: &stdout, Stderr: &stderr})
	if code == 0 || !strings.Contains(stderr.String(), "not authorized") {
		t.Fatalf("code=%d stderr=%q", code, stderr.String())
	}
	if _, err := os.Stat(filepath.Join(other, "autback.json")); !os.IsNotExist(err) {
		t.Fatalf("unauthorized init wrote a link: %v", err)
	}
}

func TestServiceExecRejectsUnauthorizedProjectBeforeWorkspaceOrAdmission(t *testing.T) {
	service := &projectListService{projects: []*autbackv1.Project{{Id: "prj1", Slug: "authorized", Name: "Authorized"}}}
	client, closeServer := testServiceClient(t, service)
	defer closeServer()
	settings := config.Config{Service: &config.Service{
		Image: "ghcr.io/example/ci@sha256:" + strings.Repeat("a", 64),
	}}
	var stdout, stderr bytes.Buffer
	code := serviceExec(context.Background(), client, settings, "unauthorized", []string{"--", "true"}, IO{
		Dir: filepath.Join(t.TempDir(), "not-a-worktree"), Stdout: &stdout, Stderr: &stderr,
	})
	if code == 0 || !strings.Contains(stderr.String(), "not authorized") {
		t.Fatalf("code=%d stderr=%q", code, stderr.String())
	}
	service.mu.Lock()
	prepareCount := service.prepareCount
	service.mu.Unlock()
	if prepareCount != 0 {
		t.Fatalf("PrepareJob called %d times", prepareCount)
	}
}

func TestServiceDoctorUsesEnvironmentProjectForGitHubOIDC(t *testing.T) {
	oidc := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if request.URL.Query().Get("audience") != "https://autback.example" || request.Header.Get("Authorization") != "Bearer request-token" {
			t.Errorf("OIDC request audience=%q authorization=%q", request.URL.Query().Get("audience"), request.Header.Get("Authorization"))
		}
		_, _ = response.Write([]byte(`{"value":"github-id-token"}`))
	}))
	defer oidc.Close()
	t.Setenv("ACTIONS_ID_TOKEN_REQUEST_URL", oidc.URL)
	t.Setenv("ACTIONS_ID_TOKEN_REQUEST_TOKEN", "request-token")
	t.Setenv("AUTBACK_PROJECT", "poc")
	t.Setenv("AUTBACK_TOKEN", "")

	service := &oidcDoctorService{}
	path, handler := autbackv1connect.NewControlServiceHandler(service)
	mux := http.NewServeMux()
	mux.Handle(path, handler)
	server := httptest.NewServer(mux)
	defer server.Close()
	var stdout, stderr bytes.Buffer
	code := runService(context.Background(), config.Config{
		URL: server.URL, Service: &config.Service{OIDCAudience: "https://autback.example"},
	}, "", []string{"doctor"}, IO{Stdout: &stdout, Stderr: &stderr, Keyring: &memoryKeyring{}})
	if code != 0 || service.project != "poc" {
		t.Fatalf("code=%d project=%q stdout=%q stderr=%q", code, service.project, stdout.String(), stderr.String())
	}
}

func TestFinishBuildRecordRenewsGitHubOIDCSession(t *testing.T) {
	stale := &finishBuildService{reject: true}
	staleClient, closeStale := testServiceClient(t, stale)
	defer closeStale()
	fresh := &finishBuildService{}
	freshClient, closeFresh := testServiceClient(t, fresh)
	defer closeFresh()
	renewals := 0
	client := &renewableControlClient{
		ControlServiceClient: staleClient,
		renew: func(context.Context) (autbackv1connect.ControlServiceClient, error) {
			renewals++
			return freshClient, nil
		},
	}

	if err := finishServiceBuildRecord(context.Background(), client, "bld-long", 0, false); err != nil {
		t.Fatal(err)
	}
	if renewals != 1 || fresh.finishedID != "bld-long" || stale.finishedID != "" {
		t.Fatalf("renewals=%d stale=%q fresh=%q", renewals, stale.finishedID, fresh.finishedID)
	}
}

func TestRenewServiceClientRemainsRenewableAcrossSessionExpirations(t *testing.T) {
	base := &finishBuildService{}
	baseClient, closeBase := testServiceClient(t, base)
	defer closeBase()
	renewals := 0
	client := &renewableControlClient{
		ControlServiceClient: baseClient,
		renew: func(context.Context) (autbackv1connect.ControlServiceClient, error) {
			renewals++
			return baseClient, nil
		},
	}

	first, err := renewServiceClient(context.Background(), client)
	if err != nil {
		t.Fatal(err)
	}
	second, err := renewServiceClient(context.Background(), first)
	if err != nil {
		t.Fatal(err)
	}
	if second != client || renewals != 2 {
		t.Fatalf("client=%T renewals=%d, want original renewable client and 2 renewals", second, renewals)
	}
}

func TestWaitForServiceBuildPollsUntilBuildKitIsAdmitted(t *testing.T) {
	service := &queuedBuildService{}
	client, closeServer := testServiceClient(t, service)
	defer closeServer()
	build, connection, err := waitForServiceBuild(context.Background(), client, &autbackv1.Build{
		Id: "bld-queued", Status: autbackv1.BuildStatus_BUILD_STATUS_QUEUED,
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if service.polls != 1 || build.Status != autbackv1.BuildStatus_BUILD_STATUS_RUNNING || connection == nil || connection.Endpoint != "buildkit.example:1234" {
		t.Fatalf("polls=%d build=%#v connection=%#v", service.polls, build, connection)
	}
}

func TestWaitForServiceJobPreparationPollsUntilCASIsAdmitted(t *testing.T) {
	service := &queuedJobPreparationService{}
	client, closeServer := testServiceClient(t, service)
	defer closeServer()
	job, connection, err := waitForServiceJobPreparation(context.Background(), client, &autbackv1.Job{
		Id: "job-queued", Status: autbackv1.JobStatus_JOB_STATUS_PREPARING,
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if service.polls != 1 || job.Status != autbackv1.JobStatus_JOB_STATUS_PREPARING || connection == nil || connection.Endpoint != "cas.example:50052" {
		t.Fatalf("polls=%d job=%#v connection=%#v", service.polls, job, connection)
	}
}

func TestHeartbeatServiceJobPreparationKeepsUploadLeaseAlive(t *testing.T) {
	service := &heartbeatJobPreparationService{polled: make(chan struct{}, 1)}
	client, closeServer := testServiceClient(t, service)
	defer closeServer()
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		done <- heartbeatServiceJobPreparation(ctx, client, "job-uploading", time.Millisecond)
	}()
	select {
	case <-service.polled:
		cancel()
	case <-time.After(time.Second):
		t.Fatal("job preparation lease was not renewed")
	}
	if err := <-done; err != nil {
		t.Fatal(err)
	}
}

func TestHeartbeatServiceBuildKeepsRunningLeaseAlive(t *testing.T) {
	service := &heartbeatBuildService{polled: make(chan struct{}, 1)}
	client, closeServer := testServiceClient(t, service)
	defer closeServer()
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		done <- heartbeatServiceBuild(ctx, client, "bld-running", time.Millisecond)
	}()
	select {
	case <-service.polled:
		cancel()
	case <-time.After(time.Second):
		t.Fatal("build lease was not renewed")
	}
	if err := <-done; err != nil {
		t.Fatal(err)
	}
}

func TestWaitForServiceBuildRenewsGitHubOIDCSessionMoreThanOnce(t *testing.T) {
	stale := &expiringQueuedBuildService{rejectAfter: 0}
	staleClient, closeStale := testServiceClient(t, stale)
	defer closeStale()
	queued := &expiringQueuedBuildService{rejectAfter: 1}
	queuedClient, closeQueued := testServiceClient(t, queued)
	defer closeQueued()
	running := &queuedBuildService{}
	runningClient, closeRunning := testServiceClient(t, running)
	defer closeRunning()

	renewals := 0
	client := &renewableControlClient{
		ControlServiceClient: staleClient,
		renew: func(context.Context) (autbackv1connect.ControlServiceClient, error) {
			renewals++
			if renewals == 1 {
				return queuedClient, nil
			}
			return runningClient, nil
		},
	}

	build, connection, err := waitForServiceBuild(context.Background(), client, &autbackv1.Build{
		Id: "bld-long-queued", Status: autbackv1.BuildStatus_BUILD_STATUS_QUEUED,
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if renewals != 2 || queued.polls != 2 || running.polls != 1 || build.Status != autbackv1.BuildStatus_BUILD_STATUS_RUNNING || connection == nil {
		t.Fatalf("renewals=%d queued polls=%d running polls=%d build=%#v connection=%#v", renewals, queued.polls, running.polls, build, connection)
	}
}

func initGitRepository(t *testing.T, directory string) {
	t.Helper()
	command := exec.Command("git", "init", "-q")
	command.Dir = directory
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("git init: %v: %s", err, output)
	}
}

func TestWaitServiceJobReconnectsWithoutDuplicatingLogBytes(t *testing.T) {
	service := &interruptedLogService{}
	path, handler := autbackv1connect.NewControlServiceHandler(service)
	mux := http.NewServeMux()
	mux.Handle(path, handler)
	server := httptest.NewServer(mux)
	defer server.Close()
	client := autbackv1connect.NewControlServiceClient(server.Client(), server.URL)
	var stdout, stderr bytes.Buffer
	code := waitServiceJob(context.Background(), client, "job-1", IO{Stdout: &stdout, Stderr: &stderr})
	if code != 0 || stdout.String() != "abcdef" {
		t.Fatalf("code=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	service.mu.Lock()
	offsets := append([]int64(nil), service.offsets...)
	service.mu.Unlock()
	if !reflect.DeepEqual(offsets, []int64{0, 3}) {
		t.Fatalf("stream offsets = %v", offsets)
	}
}

type interruptedLogService struct {
	autbackv1connect.UnimplementedControlServiceHandler
	mu      sync.Mutex
	offsets []int64
}

type finishBuildService struct {
	autbackv1connect.UnimplementedControlServiceHandler
	reject     bool
	finishedID string
}

type queuedBuildService struct {
	autbackv1connect.UnimplementedControlServiceHandler
	polls int
}

type heartbeatBuildService struct {
	autbackv1connect.UnimplementedControlServiceHandler
	polled chan struct{}
}

type queuedJobPreparationService struct {
	autbackv1connect.UnimplementedControlServiceHandler
	polls int
}

func (s *queuedJobPreparationService) GetJob(_ context.Context, request *connect.Request[autbackv1.GetJobRequest]) (*connect.Response[autbackv1.GetJobResponse], error) {
	s.polls++
	return connect.NewResponse(&autbackv1.GetJobResponse{
		Job: &autbackv1.Job{Id: request.Msg.Id, Status: autbackv1.JobStatus_JOB_STATUS_PREPARING},
		Cas: &autbackv1.DataPlaneConnection{Endpoint: "cas.example:50052"},
	}), nil
}

type heartbeatJobPreparationService struct {
	autbackv1connect.UnimplementedControlServiceHandler
	polled chan struct{}
}

func (s *heartbeatJobPreparationService) GetJob(_ context.Context, request *connect.Request[autbackv1.GetJobRequest]) (*connect.Response[autbackv1.GetJobResponse], error) {
	select {
	case s.polled <- struct{}{}:
	default:
	}
	return connect.NewResponse(&autbackv1.GetJobResponse{
		Job: &autbackv1.Job{Id: request.Msg.Id, Status: autbackv1.JobStatus_JOB_STATUS_PREPARING},
		Cas: &autbackv1.DataPlaneConnection{Endpoint: "cas.example:50052"},
	}), nil
}

func (s *heartbeatBuildService) GetBuild(_ context.Context, request *connect.Request[autbackv1.GetBuildRequest]) (*connect.Response[autbackv1.GetBuildResponse], error) {
	select {
	case s.polled <- struct{}{}:
	default:
	}
	return connect.NewResponse(&autbackv1.GetBuildResponse{Build: &autbackv1.Build{Id: request.Msg.Id, Status: autbackv1.BuildStatus_BUILD_STATUS_RUNNING}}), nil
}

type expiringQueuedBuildService struct {
	autbackv1connect.UnimplementedControlServiceHandler
	polls       int
	rejectAfter int
}

func (s *expiringQueuedBuildService) GetBuild(_ context.Context, request *connect.Request[autbackv1.GetBuildRequest]) (*connect.Response[autbackv1.GetBuildResponse], error) {
	s.polls++
	if s.polls > s.rejectAfter {
		return nil, connect.NewError(connect.CodeUnauthenticated, context.Canceled)
	}
	return connect.NewResponse(&autbackv1.GetBuildResponse{
		Build: &autbackv1.Build{Id: request.Msg.Id, Status: autbackv1.BuildStatus_BUILD_STATUS_QUEUED},
	}), nil
}

func (s *queuedBuildService) GetBuild(_ context.Context, request *connect.Request[autbackv1.GetBuildRequest]) (*connect.Response[autbackv1.GetBuildResponse], error) {
	s.polls++
	return connect.NewResponse(&autbackv1.GetBuildResponse{
		Build:    &autbackv1.Build{Id: request.Msg.Id, Status: autbackv1.BuildStatus_BUILD_STATUS_RUNNING},
		Buildkit: &autbackv1.DataPlaneConnection{Endpoint: "buildkit.example:1234"},
	}), nil
}

func (s *finishBuildService) FinishBuild(_ context.Context, request *connect.Request[autbackv1.FinishBuildRequest]) (*connect.Response[autbackv1.FinishBuildResponse], error) {
	if s.reject {
		return nil, connect.NewError(connect.CodeUnauthenticated, context.Canceled)
	}
	s.finishedID = request.Msg.Id
	return connect.NewResponse(&autbackv1.FinishBuildResponse{Build: &autbackv1.Build{Id: request.Msg.Id}}), nil
}

type enrollmentService struct {
	autbackv1connect.UnimplementedControlServiceHandler
	wantCode string
	gotCode  string
	token    string
}

type identityService struct {
	autbackv1connect.UnimplementedControlServiceHandler
	userID string
	login  string
}

func (s *identityService) BindGitHubIdentity(_ context.Context, request *connect.Request[autbackv1.BindGitHubIdentityRequest]) (*connect.Response[autbackv1.BindGitHubIdentityResponse], error) {
	s.userID, s.login = request.Msg.UserId, request.Msg.Login
	return connect.NewResponse(&autbackv1.BindGitHubIdentityResponse{Identity: &autbackv1.ExternalIdentity{Provider: "github", Subject: "12345678", Login: request.Msg.Login, UserId: request.Msg.UserId}}), nil
}

func (s *enrollmentService) ExchangeEnrollmentCode(_ context.Context, request *connect.Request[autbackv1.ExchangeEnrollmentCodeRequest]) (*connect.Response[autbackv1.ExchangeEnrollmentCodeResponse], error) {
	s.gotCode = request.Msg.Code
	if request.Msg.Code != s.wantCode {
		return nil, connect.NewError(connect.CodeUnauthenticated, context.Canceled)
	}
	return connect.NewResponse(&autbackv1.ExchangeEnrollmentCodeResponse{
		Token: s.token, DeviceToken: &autbackv1.DeviceToken{Id: "tok456", UserId: "usr1", Name: "laptop"},
	}), nil
}

func (s *enrollmentService) ListDeviceTokens(context.Context, *connect.Request[autbackv1.ListDeviceTokensRequest]) (*connect.Response[autbackv1.ListDeviceTokensResponse], error) {
	return connect.NewResponse(&autbackv1.ListDeviceTokensResponse{}), nil
}

type memoryKeyring struct{ token string }

func (m *memoryKeyring) Get(string, string) (string, error) { return m.token, nil }
func (m *memoryKeyring) Set(_, _, token string) error       { m.token = token; return nil }
func (m *memoryKeyring) Delete(string, string) error        { m.token = ""; return nil }

type projectListService struct {
	autbackv1connect.UnimplementedControlServiceHandler
	mu               sync.Mutex
	projects         []*autbackv1.Project
	prepareCount     int
	activatedProject string
	activatedImage   string
}

type oidcDoctorService struct {
	autbackv1connect.UnimplementedControlServiceHandler
	project string
}

func (s *oidcDoctorService) ExchangeGitHubOIDC(_ context.Context, request *connect.Request[autbackv1.ExchangeGitHubOIDCRequest]) (*connect.Response[autbackv1.ExchangeGitHubOIDCResponse], error) {
	s.project = request.Msg.Project
	return connect.NewResponse(&autbackv1.ExchangeGitHubOIDCResponse{Token: "project-session"}), nil
}

func (s *oidcDoctorService) GetServiceInfo(context.Context, *connect.Request[autbackv1.GetServiceInfoRequest]) (*connect.Response[autbackv1.GetServiceInfoResponse], error) {
	return connect.NewResponse(&autbackv1.GetServiceInfoResponse{Version: "test"}), nil
}

func (s *projectListService) ActivateProjectImage(_ context.Context, request *connect.Request[autbackv1.ActivateProjectImageRequest]) (*connect.Response[autbackv1.ActivateProjectImageResponse], error) {
	s.mu.Lock()
	s.activatedProject, s.activatedImage = request.Msg.Project, request.Msg.Image
	s.mu.Unlock()
	return connect.NewResponse(&autbackv1.ActivateProjectImageResponse{Project: &autbackv1.Project{Slug: request.Msg.Project, ActiveImage: request.Msg.Image}}), nil
}

func (s *projectListService) ListProjects(context.Context, *connect.Request[autbackv1.ListProjectsRequest]) (*connect.Response[autbackv1.ListProjectsResponse], error) {
	return connect.NewResponse(&autbackv1.ListProjectsResponse{Projects: s.projects}), nil
}

func (s *projectListService) PrepareJob(context.Context, *connect.Request[autbackv1.PrepareJobRequest]) (*connect.Response[autbackv1.PrepareJobResponse], error) {
	s.mu.Lock()
	s.prepareCount++
	s.mu.Unlock()
	return nil, connect.NewError(connect.CodeInternal, context.Canceled)
}

func testServiceClient(t *testing.T, service autbackv1connect.ControlServiceHandler) (autbackv1connect.ControlServiceClient, func()) {
	t.Helper()
	path, handler := autbackv1connect.NewControlServiceHandler(service)
	mux := http.NewServeMux()
	mux.Handle(path, handler)
	server := httptest.NewServer(mux)
	return autbackv1connect.NewControlServiceClient(server.Client(), server.URL), server.Close
}

func (s *interruptedLogService) StreamJobLogs(_ context.Context, request *connect.Request[autbackv1.StreamJobLogsRequest], stream *connect.ServerStream[autbackv1.StreamJobLogsResponse]) error {
	s.mu.Lock()
	s.offsets = append(s.offsets, request.Msg.Offset)
	call := len(s.offsets)
	s.mu.Unlock()
	if call == 1 {
		if err := stream.Send(&autbackv1.StreamJobLogsResponse{Data: []byte("abc"), NextOffset: 3}); err != nil {
			return err
		}
		return connect.NewError(connect.CodeUnavailable, context.DeadlineExceeded)
	}
	exitCode := int32(0)
	return stream.Send(&autbackv1.StreamJobLogsResponse{
		Data: []byte("def"), NextOffset: 6,
		TerminalJob: &autbackv1.Job{Id: "job-1", Status: autbackv1.JobStatus_JOB_STATUS_SUCCEEDED, ExitCode: &exitCode, CreatedAt: timestamppb.Now()},
	})
}
