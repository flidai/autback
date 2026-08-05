package controlapi_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/flidai/autback/internal/capacity"
	"github.com/flidai/autback/internal/control"
	"github.com/flidai/autback/internal/control/controlapi"
	"github.com/flidai/autback/internal/control/dispatcher"
	"github.com/flidai/autback/internal/control/pki"
	controlsqlite "github.com/flidai/autback/internal/control/sqlite"
	autbackv1 "github.com/flidai/autback/internal/gen/rtest/v1"
	"github.com/flidai/autback/internal/gen/rtest/v1/autbackv1connect"
	"github.com/flidai/autback/internal/protocol"
	"github.com/flidai/autback/internal/version"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestAuthenticatedGenericJobLifecycle(t *testing.T) {
	fixture := newFixture(t)
	client := fixture.client(fixture.bootstrap.Token)
	ctx := context.Background()
	prepared, err := client.PrepareJob(ctx, connect.NewRequest(&autbackv1.PrepareJobRequest{
		IdempotencyKey: "job-lifecycle-1",
		Project:        fixture.bootstrap.Project.Slug,
		Image:          "ghcr.io/example/ci@sha256:" + strings.Repeat("1", 64),
		Command:        []string{"task", "test"}, WorkingDirectory: "services/api",
		Environment: map[string]string{"CI": "true"}, Timeout: durationpb.New(10 * time.Minute),
		Cpus: "2", Memory: "4g",
	}))
	if err != nil {
		t.Fatal(err)
	}
	if prepared.Msg.Job.Status != autbackv1.JobStatus_JOB_STATUS_PREPARING || prepared.Msg.Cas == nil || len(prepared.Msg.Cas.CertificatePem) == 0 {
		t.Fatalf("prepared = %#v", prepared.Msg)
	}
	if fixture.scheduler.createdCount() != 0 {
		t.Fatal("job was scheduled before CAS upload completed")
	}

	started, err := client.StartJob(ctx, connect.NewRequest(&autbackv1.StartJobRequest{
		Id: prepared.Msg.Job.Id, RootDigest: strings.Repeat("a", 64) + "/123",
	}))
	if err != nil {
		t.Fatal(err)
	}
	if fixture.scheduler.createdCount() != 1 || started.Msg.Job.Status != autbackv1.JobStatus_JOB_STATUS_QUEUED {
		t.Fatalf("created=%d started=%#v", fixture.scheduler.createdCount(), started.Msg.Job)
	}
	created := fixture.scheduler.created[0]
	if created.Image != prepared.Msg.Job.Image || created.WorkingDirectory != "services/api" || created.Environment["CI"] != "true" {
		t.Fatalf("scheduled job = %#v", created)
	}

	fixture.scheduler.complete(prepared.Msg.Job.Id, protocol.StatusSucceeded, 0)
	got, err := client.GetJob(ctx, connect.NewRequest(&autbackv1.GetJobRequest{Id: prepared.Msg.Job.Id}))
	if err != nil {
		t.Fatal(err)
	}
	if got.Msg.Job.Status != autbackv1.JobStatus_JOB_STATUS_SUCCEEDED || got.Msg.Job.ExitCode == nil || *got.Msg.Job.ExitCode != 0 {
		t.Fatalf("job = %#v", got.Msg.Job)
	}
	stream, err := client.StreamJobLogs(ctx, connect.NewRequest(&autbackv1.StreamJobLogsRequest{Id: prepared.Msg.Job.Id}))
	if err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	var terminal *autbackv1.Job
	for stream.Receive() {
		output.Write(stream.Msg().Data)
		if stream.Msg().TerminalJob != nil {
			terminal = stream.Msg().TerminalJob
		}
	}
	if err := stream.Err(); err != nil {
		t.Fatal(err)
	}
	if output.String() != "remote output\n" || terminal == nil || terminal.Status != autbackv1.JobStatus_JOB_STATUS_SUCCEEDED {
		t.Fatalf("output=%q terminal=%#v", output.String(), terminal)
	}
}

func TestCancelAdmittedPreparationReleasesFIFOLease(t *testing.T) {
	fixture := newFixture(t)
	client := fixture.client(fixture.bootstrap.Token)
	prepared, err := client.PrepareJob(context.Background(), connect.NewRequest(&autbackv1.PrepareJobRequest{
		IdempotencyKey: "cancel-admitted-preparation", Project: fixture.bootstrap.Project.Slug,
		Image: "ghcr.io/example/ci@sha256:" + strings.Repeat("1", 64), Command: []string{"true"}, Timeout: durationpb.New(time.Minute),
	}))
	if err != nil || prepared.Msg.Cas == nil {
		t.Fatalf("prepared = %#v, %v", prepared, err)
	}
	cancelled, err := client.CancelJob(context.Background(), connect.NewRequest(&autbackv1.CancelJobRequest{Id: prepared.Msg.Job.Id}))
	if err != nil || cancelled.Msg.Job.Status != autbackv1.JobStatus_JOB_STATUS_CANCELLED {
		t.Fatalf("cancelled = %#v, %v", cancelled, err)
	}
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		state, stateErr := fixture.store.OperationState(context.Background(), control.OperationJob, prepared.Msg.Job.Id)
		if stateErr == nil && state == control.OperationReleased {
			return
		}
		time.Sleep(time.Millisecond)
	}
	state, stateErr := fixture.store.OperationState(context.Background(), control.OperationJob, prepared.Msg.Job.Id)
	t.Fatalf("operation state = %s, %v; want released", state, stateErr)
}

func TestGetJobRepairsStatusAfterAdmissionLeaseReleased(t *testing.T) {
	fixture := newFixture(t)
	client := fixture.client(fixture.bootstrap.Token)
	ctx := context.Background()
	prepared, err := client.PrepareJob(ctx, connect.NewRequest(&autbackv1.PrepareJobRequest{
		IdempotencyKey: "repair-released-job", Project: fixture.bootstrap.Project.Slug,
		Image: "ghcr.io/example/ci@sha256:" + strings.Repeat("1", 64), Command: []string{"task", "ci"}, Timeout: durationpb.New(time.Minute),
	}))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := client.StartJob(ctx, connect.NewRequest(&autbackv1.StartJobRequest{Id: prepared.Msg.Job.Id, RootDigest: strings.Repeat("a", 64) + "/1"})); err != nil {
		t.Fatal(err)
	}
	fixture.scheduler.complete(prepared.Msg.Job.Id, protocol.StatusSucceeded, 0)
	if err := fixture.store.BeginOperationCleanup(ctx, control.OperationJob, prepared.Msg.Job.Id); err != nil {
		t.Fatal(err)
	}
	operation, err := fixture.store.ClaimOperationCleanup(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if operation == nil || operation.ID != prepared.Msg.Job.Id {
		t.Fatalf("cleanup operation = %#v", operation)
	}
	if err := fixture.store.CompleteOperationCleanup(ctx, control.OperationJob, prepared.Msg.Job.Id); err != nil {
		t.Fatal(err)
	}

	got, err := client.GetJob(ctx, connect.NewRequest(&autbackv1.GetJobRequest{Id: prepared.Msg.Job.Id}))
	if err != nil {
		t.Fatal(err)
	}
	if got.Msg.Job.Status != autbackv1.JobStatus_JOB_STATUS_SUCCEEDED {
		t.Fatalf("job status = %s, want succeeded", got.Msg.Job.Status)
	}
}

func TestBuildsAndJobsShareStrictFIFOAdmission(t *testing.T) {
	fixture := newFixture(t)
	client := fixture.client(fixture.bootstrap.Token)
	ctx := context.Background()
	prepareJob := func(key string) *autbackv1.Job {
		t.Helper()
		prepared, err := client.PrepareJob(ctx, connect.NewRequest(&autbackv1.PrepareJobRequest{
			IdempotencyKey: key, Project: fixture.bootstrap.Project.Slug,
			Image: "ghcr.io/example/ci@sha256:" + strings.Repeat("1", 64), Command: []string{"task", "ci"}, Timeout: durationpb.New(time.Minute),
		}))
		if err != nil {
			t.Fatal(err)
		}
		started, err := client.StartJob(ctx, connect.NewRequest(&autbackv1.StartJobRequest{Id: prepared.Msg.Job.Id, RootDigest: strings.Repeat("a", 64) + "/1"}))
		if err != nil {
			t.Fatal(err)
		}
		return started.Msg.Job
	}
	first := prepareJob("fifo-first-job")
	build, err := client.PrepareBuild(ctx, connect.NewRequest(&autbackv1.PrepareBuildRequest{Project: fixture.bootstrap.Project.Slug, IdempotencyKey: "fifo-build-0001"}))
	if err != nil {
		t.Fatal(err)
	}
	if build.Msg.Build.Status != autbackv1.BuildStatus_BUILD_STATUS_QUEUED || build.Msg.Buildkit != nil {
		t.Fatalf("queued build = %#v", build.Msg)
	}
	secondPreparation, err := client.PrepareJob(ctx, connect.NewRequest(&autbackv1.PrepareJobRequest{
		IdempotencyKey: "fifo-second-job", Project: fixture.bootstrap.Project.Slug,
		Image: "ghcr.io/example/ci@sha256:" + strings.Repeat("1", 64), Command: []string{"task", "ci"}, Timeout: durationpb.New(time.Minute),
	}))
	if err != nil {
		t.Fatal(err)
	}
	second := secondPreparation.Msg.Job
	if secondPreparation.Msg.Cas != nil {
		t.Fatal("second job received CAS credentials before the earlier build")
	}
	if fixture.scheduler.createdCount() != 1 {
		t.Fatalf("created jobs = %d, want only first", fixture.scheduler.createdCount())
	}

	fixture.scheduler.complete(first.Id, protocol.StatusSucceeded, 0)
	if _, err := client.GetJob(ctx, connect.NewRequest(&autbackv1.GetJobRequest{Id: first.Id})); err != nil {
		t.Fatal(err)
	}
	var runningBuild *connect.Response[autbackv1.GetBuildResponse]
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		runningBuild, err = client.GetBuild(ctx, connect.NewRequest(&autbackv1.GetBuildRequest{Id: build.Msg.Build.Id}))
		if err == nil && runningBuild.Msg.Build.Status == autbackv1.BuildStatus_BUILD_STATUS_RUNNING {
			break
		}
		time.Sleep(time.Millisecond)
	}
	if err != nil || runningBuild == nil || runningBuild.Msg.Build.Status != autbackv1.BuildStatus_BUILD_STATUS_RUNNING || runningBuild.Msg.Buildkit == nil {
		t.Fatalf("admitted build = %#v, %v", runningBuild, err)
	}
	if _, err := client.FinishBuild(ctx, connect.NewRequest(&autbackv1.FinishBuildRequest{Id: build.Msg.Build.Id})); err != nil {
		t.Fatal(err)
	}
	var admitted *connect.Response[autbackv1.GetJobResponse]
	deadline = time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		admitted, err = client.GetJob(ctx, connect.NewRequest(&autbackv1.GetJobRequest{Id: second.Id}))
		if err == nil && admitted.Msg.Cas != nil {
			break
		}
		time.Sleep(time.Millisecond)
	}
	if err != nil || admitted == nil || admitted.Msg.Cas == nil {
		t.Fatalf("admitted second preparation = %#v, %v", admitted, err)
	}
	if _, err := client.StartJob(ctx, connect.NewRequest(&autbackv1.StartJobRequest{Id: second.Id, RootDigest: strings.Repeat("b", 64) + "/1"})); err != nil {
		t.Fatal(err)
	}
	deadline = time.Now().Add(time.Second)
	for fixture.scheduler.createdCount() != 2 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if fixture.scheduler.createdCount() != 2 || fixture.scheduler.createdAt(1).ID != second.Id {
		t.Fatalf("created jobs = %#v, want second after build", fixture.scheduler.createdSnapshot())
	}
}

func TestProjectImageActivationResolvesDefaultAndRollbackIsAudited(t *testing.T) {
	fixture := newFixture(t)
	client := fixture.client(fixture.bootstrap.Token)
	ctx := context.Background()
	first := "ghcr.io/example/runner@sha256:" + strings.Repeat("a", 64)
	second := "ghcr.io/example/runner@sha256:" + strings.Repeat("b", 64)

	activated, err := client.ActivateProjectImage(ctx, connect.NewRequest(&autbackv1.ActivateProjectImageRequest{
		Project: fixture.bootstrap.Project.Slug, Image: first,
	}))
	if err != nil {
		t.Fatal(err)
	}
	if activated.Msg.Project.ActiveImage != first || !activated.Msg.Project.AllowImageOverrides {
		t.Fatalf("activated project = %#v", activated.Msg.Project)
	}
	prepared, err := client.PrepareJob(ctx, connect.NewRequest(&autbackv1.PrepareJobRequest{
		IdempotencyKey: "default-image-job", Project: fixture.bootstrap.Project.Slug,
		Command: []string{"go", "version"}, Timeout: durationpb.New(time.Minute), Cpus: "1", Memory: "1g",
	}))
	if err != nil {
		t.Fatal(err)
	}
	if prepared.Msg.Job.Image != first {
		t.Fatalf("resolved image = %q, want %q", prepared.Msg.Job.Image, first)
	}

	if _, err := client.ActivateProjectImage(ctx, connect.NewRequest(&autbackv1.ActivateProjectImageRequest{
		Project: fixture.bootstrap.Project.Slug, Image: second,
	})); err != nil {
		t.Fatal(err)
	}
	rolledBack, err := client.RollbackProjectImage(ctx, connect.NewRequest(&autbackv1.RollbackProjectImageRequest{Project: fixture.bootstrap.Project.Slug}))
	if err != nil {
		t.Fatal(err)
	}
	if rolledBack.Msg.Project.ActiveImage != first || rolledBack.Msg.Project.PreviousImage != second {
		t.Fatalf("rolled back project = %#v", rolledBack.Msg.Project)
	}
	history, err := client.ListProjectImageHistory(ctx, connect.NewRequest(&autbackv1.ListProjectImageHistoryRequest{Project: fixture.bootstrap.Project.Slug}))
	if err != nil {
		t.Fatal(err)
	}
	if len(history.Msg.Events) != 3 || history.Msg.Events[0].Action != "rollback" || history.Msg.Events[0].Image != first {
		t.Fatalf("history = %#v", history.Msg.Events)
	}
}

func TestProjectImageActivationFailsClosedBeforeMutation(t *testing.T) {
	fixture := newFixture(t)
	client := fixture.client(fixture.bootstrap.Token)
	ctx := context.Background()
	good := "ghcr.io/example/runner@sha256:" + strings.Repeat("c", 64)
	bad := "ghcr.io/example/runner@sha256:" + strings.Repeat("d", 64)
	if _, err := client.ActivateProjectImage(ctx, connect.NewRequest(&autbackv1.ActivateProjectImageRequest{Project: "example", Image: good})); err != nil {
		t.Fatal(err)
	}
	fixture.scheduler.validateErr = errors.New("registry unavailable")
	if _, err := client.ActivateProjectImage(ctx, connect.NewRequest(&autbackv1.ActivateProjectImageRequest{Project: "example", Image: bad})); connect.CodeOf(err) != connect.CodeUnavailable {
		t.Fatalf("activation error = %v", err)
	}
	projects, err := client.ListProjects(ctx, connect.NewRequest(&autbackv1.ListProjectsRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	if projects.Msg.Projects[0].ActiveImage != good || projects.Msg.Projects[0].PreviousImage != "" {
		t.Fatalf("failed activation mutated project = %#v", projects.Msg.Projects[0])
	}
}

func TestProjectImageOverridePolicyIsEnforcedAtAdmission(t *testing.T) {
	fixture := newFixture(t)
	client := fixture.client(fixture.bootstrap.Token)
	ctx := context.Background()
	active := "ghcr.io/example/runner@sha256:" + strings.Repeat("e", 64)
	override := "ghcr.io/example/runner@sha256:" + strings.Repeat("f", 64)
	if _, err := client.ActivateProjectImage(ctx, connect.NewRequest(&autbackv1.ActivateProjectImageRequest{Project: "example", Image: active})); err != nil {
		t.Fatal(err)
	}
	if _, err := client.SetProjectImagePolicy(ctx, connect.NewRequest(&autbackv1.SetProjectImagePolicyRequest{Project: "example", AllowImageOverrides: false})); err != nil {
		t.Fatal(err)
	}
	_, err := client.PrepareJob(ctx, connect.NewRequest(&autbackv1.PrepareJobRequest{
		IdempotencyKey: "denied-override-job", Project: "example", Image: override,
		Command: []string{"true"}, Timeout: durationpb.New(time.Minute), Cpus: "1", Memory: "1g",
	}))
	if connect.CodeOf(err) != connect.CodePermissionDenied {
		t.Fatalf("override error = %v", err)
	}
}

func TestJobAdmissionValidatesAndPersistsGenericProjectCaches(t *testing.T) {
	fixture := newFixture(t)
	client := fixture.client(fixture.bootstrap.Token)
	ctx := context.Background()
	request := &autbackv1.PrepareJobRequest{
		IdempotencyKey: "generic-cache-job", Project: "example",
		Image:   "ghcr.io/example/ci@sha256:" + strings.Repeat("a", 64),
		Command: []string{"go", "test", "./..."}, Timeout: durationpb.New(time.Minute), Cpus: "1", Memory: "1g",
		Caches: []*autbackv1.CacheMount{{Name: "modules", Target: "/go/pkg/mod"}, {Name: "go-build", Target: "/root/.cache/go-build"}},
	}
	prepared, err := client.PrepareJob(ctx, connect.NewRequest(request))
	if err != nil {
		t.Fatal(err)
	}
	if len(prepared.Msg.Job.Caches) != 2 || prepared.Msg.Job.Caches[0].Name != "go-build" || prepared.Msg.Job.Caches[1].Name != "modules" {
		t.Fatalf("caches = %#v", prepared.Msg.Job.Caches)
	}

	invalid := []struct {
		name   string
		caches []*autbackv1.CacheMount
	}{
		{name: "unsafe name", caches: []*autbackv1.CacheMount{{Name: "../shared", Target: "/cache"}}},
		{name: "relative target", caches: []*autbackv1.CacheMount{{Name: "cache", Target: "cache"}}},
		{name: "duplicate name", caches: []*autbackv1.CacheMount{{Name: "cache", Target: "/one"}, {Name: "cache", Target: "/two"}}},
		{name: "duplicate target", caches: []*autbackv1.CacheMount{{Name: "one", Target: "/cache"}, {Name: "two", Target: "/cache"}}},
		{name: "docker socket overlap", caches: []*autbackv1.CacheMount{{Name: "socket", Target: "/var/run"}}},
	}
	for index, test := range invalid {
		t.Run(test.name, func(t *testing.T) {
			copyRequest := proto.Clone(request).(*autbackv1.PrepareJobRequest)
			copyRequest.IdempotencyKey = fmt.Sprintf("invalid-cache-%02d", index)
			copyRequest.Caches = test.caches
			if _, err := client.PrepareJob(ctx, connect.NewRequest(copyRequest)); connect.CodeOf(err) != connect.CodeInvalidArgument {
				t.Fatalf("error = %v", err)
			}
		})
	}
}

func TestJobAdmissionPersistsOnlySecretReferences(t *testing.T) {
	fixture := newFixture(t)
	client := fixture.client(fixture.bootstrap.Token)
	request := &autbackv1.PrepareJobRequest{
		IdempotencyKey: "first-class-secrets", Project: "example",
		Image: "ghcr.io/example/ci@sha256:" + strings.Repeat("a", 64), Command: []string{"task", "ci"},
		Secrets: []*autbackv1.JobSecret{
			{Name: "registry-token", Target: &autbackv1.JobSecret_Environment{Environment: "REGISTRY_TOKEN"}},
			{Name: "signing-key", Target: &autbackv1.JobSecret_File{File: "/run/secrets/signing-key"}},
		},
	}
	prepared, err := client.PrepareJob(context.Background(), connect.NewRequest(request))
	if err != nil {
		t.Fatal(err)
	}
	if got := prepared.Msg.Job.Secrets; len(got) != 2 || got[0].Name != "registry-token" || got[0].GetEnvironment() != "REGISTRY_TOKEN" || got[1].GetFile() != "/run/secrets/signing-key" {
		t.Fatalf("prepared secret references = %#v", got)
	}
	stored, err := fixture.store.Job(context.Background(), prepared.Msg.Job.Id)
	if err != nil {
		t.Fatal(err)
	}
	if len(stored.Secrets) != 2 || stored.Secrets[0].Environment != "REGISTRY_TOKEN" || stored.Secrets[1].File != "/run/secrets/signing-key" {
		t.Fatalf("stored secret references = %#v", stored.Secrets)
	}
}

func TestJobAdmissionRejectsLegacyAndOverlappingSecretFields(t *testing.T) {
	fixture := newFixture(t)
	client := fixture.client(fixture.bootstrap.Token)
	base := &autbackv1.PrepareJobRequest{
		IdempotencyKey: "invalid-secret-00", Project: "example",
		Image: "ghcr.io/example/ci@sha256:" + strings.Repeat("a", 64), Command: []string{"task", "ci"},
	}
	tests := []func(*autbackv1.PrepareJobRequest){
		func(request *autbackv1.PrepareJobRequest) {
			request.Environment = map[string]string{"TOKEN": "secret://registry-token"}
		},
		func(request *autbackv1.PrepareJobRequest) { request.Command = []string{"echo", "${{ secrets.TOKEN }}"} },
		func(request *autbackv1.PrepareJobRequest) {
			request.Environment = map[string]string{"TOKEN": "public"}
			request.Secrets = []*autbackv1.JobSecret{{Name: "registry-token", Target: &autbackv1.JobSecret_Environment{Environment: "TOKEN"}}}
		},
		func(request *autbackv1.PrepareJobRequest) {
			request.Secrets = []*autbackv1.JobSecret{{Name: "signing-key", Target: &autbackv1.JobSecret_File{File: "/tmp/signing-key"}}}
		},
		func(request *autbackv1.PrepareJobRequest) {
			request.Caches = []*autbackv1.CacheMount{{Name: "unsafe", Target: "/run/secrets"}}
		},
		func(request *autbackv1.PrepareJobRequest) {
			request.Secrets = []*autbackv1.JobSecret{
				{Name: "one", Target: &autbackv1.JobSecret_File{File: "/run/secrets/nested"}},
				{Name: "two", Target: &autbackv1.JobSecret_File{File: "/run/secrets/nested/value"}},
			}
		},
	}
	for index, mutate := range tests {
		request := proto.Clone(base).(*autbackv1.PrepareJobRequest)
		request.IdempotencyKey = fmt.Sprintf("invalid-secret-%02d", index)
		mutate(request)
		if _, err := client.PrepareJob(context.Background(), connect.NewRequest(request)); connect.CodeOf(err) != connect.CodeInvalidArgument {
			t.Fatalf("case %d error = %v", index, err)
		}
	}
}

func TestJobAdmissionAllowsLongRepositoryCIButBoundsRunawayJobs(t *testing.T) {
	fixture := newFixture(t)
	client := fixture.client(fixture.bootstrap.Token)
	request := &autbackv1.PrepareJobRequest{
		IdempotencyKey: "long-repository-ci", Project: "example",
		Image:   "ghcr.io/example/ci@sha256:" + strings.Repeat("a", 64),
		Command: []string{"task", "ci"}, Timeout: durationpb.New(90 * time.Minute),
	}
	if _, err := client.PrepareJob(context.Background(), connect.NewRequest(request)); err != nil {
		t.Fatalf("90-minute repository CI rejected: %v", err)
	}
	request.IdempotencyKey = "runaway-repository-ci"
	request.Timeout = durationpb.New(24*time.Hour + time.Second)
	if _, err := client.PrepareJob(context.Background(), connect.NewRequest(request)); connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("runaway timeout error = %v, want invalid argument", err)
	}
}

func TestOneTimeEnrollmentExchangeNeedsNoExistingCredential(t *testing.T) {
	fixture := newFixture(t)
	admin := fixture.client(fixture.bootstrap.Token)
	ctx := context.Background()
	user, err := admin.CreateUser(ctx, connect.NewRequest(&autbackv1.CreateUserRequest{Name: "Coworker"}))
	if err != nil {
		t.Fatal(err)
	}
	enrollment, err := admin.CreateEnrollmentCode(ctx, connect.NewRequest(&autbackv1.CreateEnrollmentCodeRequest{
		UserId: user.Msg.User.Id, DeviceName: "coworker-laptop", ExpiresAt: timestamppb.New(time.Now().Add(10 * time.Minute)),
	}))
	if err != nil {
		t.Fatal(err)
	}
	if enrollment.Msg.Code == "" || enrollment.Msg.Enrollment.MaxAttempts != 5 {
		t.Fatalf("enrollment = %#v", enrollment.Msg)
	}
	unauthenticated := fixture.client("")
	exchanged, err := unauthenticated.ExchangeEnrollmentCode(ctx, connect.NewRequest(&autbackv1.ExchangeEnrollmentCodeRequest{Code: enrollment.Msg.Code}))
	if err != nil {
		t.Fatal(err)
	}
	if exchanged.Msg.Token == "" || exchanged.Msg.DeviceToken.UserId != user.Msg.User.Id || exchanged.Msg.DeviceToken.Name != "coworker-laptop" {
		t.Fatalf("exchange = %#v", exchanged.Msg)
	}
	if _, err := unauthenticated.ExchangeEnrollmentCode(ctx, connect.NewRequest(&autbackv1.ExchangeEnrollmentCodeRequest{Code: enrollment.Msg.Code})); connect.CodeOf(err) != connect.CodeUnauthenticated {
		t.Fatalf("reuse error = %v", err)
	}
	if _, err := fixture.client(exchanged.Msg.Token).ListProjects(ctx, connect.NewRequest(&autbackv1.ListProjectsRequest{})); err != nil {
		t.Fatal(err)
	}
}

func TestStreamingLogsStopsFollowerWhenJobBecomesTerminal(t *testing.T) {
	fixture := newFixture(t)
	fixture.scheduler.blockFollowingLogs = true
	fixture.scheduler.logsStarted = make(chan struct{})
	client := fixture.client(fixture.bootstrap.Token)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	prepared, err := client.PrepareJob(ctx, connect.NewRequest(&autbackv1.PrepareJobRequest{
		IdempotencyKey: "job-follow-1",
		Project:        fixture.bootstrap.Project.Slug,
		Image:          "ghcr.io/example/ci@sha256:" + strings.Repeat("3", 64),
		Command:        []string{"true"}, Timeout: durationpb.New(time.Minute), Cpus: "1", Memory: "1g",
	}))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := client.StartJob(ctx, connect.NewRequest(&autbackv1.StartJobRequest{Id: prepared.Msg.Job.Id, RootDigest: strings.Repeat("b", 64) + "/1"})); err != nil {
		t.Fatal(err)
	}
	stream, err := client.StreamJobLogs(ctx, connect.NewRequest(&autbackv1.StreamJobLogsRequest{Id: prepared.Msg.Job.Id}))
	if err != nil {
		t.Fatal(err)
	}
	terminal := make(chan *autbackv1.Job, 1)
	streamError := make(chan error, 1)
	go func() {
		var job *autbackv1.Job
		for stream.Receive() {
			if stream.Msg().TerminalJob != nil {
				job = stream.Msg().TerminalJob
			}
		}
		terminal <- job
		streamError <- stream.Err()
	}()
	select {
	case <-fixture.scheduler.logsStarted:
	case <-ctx.Done():
		t.Fatal("log follower did not start")
	}
	fixture.scheduler.complete(prepared.Msg.Job.Id, protocol.StatusSucceeded, 0)
	if err := <-streamError; err != nil {
		t.Fatal(err)
	}
	if job := <-terminal; job == nil || job.Status != autbackv1.JobStatus_JOB_STATUS_SUCCEEDED {
		t.Fatalf("terminal job = %#v", job)
	}
}

func TestProjectAuthorizationPreventsCrossProjectJobAccess(t *testing.T) {
	fixture := newFixture(t)
	ctx := context.Background()
	owner, _ := fixture.store.Authenticate(ctx, fixture.bootstrap.Token)
	other, err := fixture.store.CreateProject(ctx, owner, "other", "Other")
	if err != nil {
		t.Fatal(err)
	}
	member, err := fixture.store.CreateUser(ctx, owner, "Member", false)
	if err != nil {
		t.Fatal(err)
	}
	if err := fixture.store.AddProjectMember(ctx, owner, other.ID, member.ID); err != nil {
		t.Fatal(err)
	}
	issued, err := fixture.store.CreateDeviceToken(ctx, owner, control.CreateDeviceToken{UserID: member.ID, Name: "member"})
	if err != nil {
		t.Fatal(err)
	}

	ownerClient := fixture.client(fixture.bootstrap.Token)
	prepared, err := ownerClient.PrepareJob(ctx, connect.NewRequest(&autbackv1.PrepareJobRequest{
		IdempotencyKey: "job-authorization-1",
		Project:        fixture.bootstrap.Project.ID, Image: "ghcr.io/example/ci@sha256:" + strings.Repeat("2", 64),
		Command: []string{"true"}, Timeout: durationpb.New(time.Minute), Cpus: "1", Memory: "1g",
	}))
	if err != nil {
		t.Fatal(err)
	}
	_, err = fixture.client(issued.Secret).GetJob(ctx, connect.NewRequest(&autbackv1.GetJobRequest{Id: prepared.Msg.Job.Id}))
	if connect.CodeOf(err) != connect.CodePermissionDenied {
		t.Fatalf("cross-project error = %v", err)
	}
	_, err = fixture.client(issued.Secret).CancelJob(ctx, connect.NewRequest(&autbackv1.CancelJobRequest{Id: prepared.Msg.Job.Id}))
	if connect.CodeOf(err) != connect.CodePermissionDenied {
		t.Fatalf("cross-project cancel error = %v", err)
	}
	stream, err := fixture.client(issued.Secret).StreamJobLogs(ctx, connect.NewRequest(&autbackv1.StreamJobLogsRequest{Id: prepared.Msg.Job.Id}))
	if err == nil {
		for stream.Receive() {
		}
		err = stream.Err()
	}
	if connect.CodeOf(err) != connect.CodePermissionDenied {
		t.Fatalf("cross-project stream error = %v", err)
	}
	_, err = fixture.client("").ListJobs(ctx, connect.NewRequest(&autbackv1.ListJobsRequest{Project: other.ID}))
	if connect.CodeOf(err) != connect.CodeUnauthenticated {
		t.Fatalf("unauthenticated error = %v", err)
	}
}

func TestBuildPreparationReturnsBuildScopedCertificate(t *testing.T) {
	fixture := newFixture(t)
	response, err := fixture.client(fixture.bootstrap.Token).PrepareBuild(context.Background(), connect.NewRequest(&autbackv1.PrepareBuildRequest{
		Project: fixture.bootstrap.Project.ID, IdempotencyKey: "build-certificate-1",
	}))
	if err != nil {
		t.Fatal(err)
	}
	if response.Msg.Buildkit.Endpoint != "buildkit.example:1234" || len(response.Msg.Buildkit.PrivateKeyPem) == 0 {
		t.Fatalf("response = %#v", response.Msg)
	}
	if fixture.store.OperationActive(context.Background(), "job", response.Msg.Build.Id) {
		t.Fatal("build certificate was accepted as job credential")
	}
	if !fixture.store.OperationActive(context.Background(), "build", response.Msg.Build.Id) {
		t.Fatal("build credential is not active")
	}
	finished, err := fixture.client(fixture.bootstrap.Token).FinishBuild(context.Background(), connect.NewRequest(&autbackv1.FinishBuildRequest{Id: response.Msg.Build.Id, ExitCode: 0}))
	if err != nil || finished.Msg.Build.Status != autbackv1.BuildStatus_BUILD_STATUS_SUCCEEDED {
		t.Fatalf("finished=%#v err=%v", finished, err)
	}
	if fixture.store.OperationActive(context.Background(), "build", response.Msg.Build.Id) {
		t.Fatal("finished build credential remains active")
	}
}

func TestFinishBuildAcknowledgesBeforeSlowQueueAdvance(t *testing.T) {
	capacityGate := &blockingAdvanceCapacity{started: make(chan struct{}), release: make(chan struct{})}
	fixture := newFixtureWithCapacity(t, nil, capacityGate)
	client := fixture.client(fixture.bootstrap.Token)
	first, err := client.PrepareBuild(context.Background(), connect.NewRequest(&autbackv1.PrepareBuildRequest{
		Project: fixture.bootstrap.Project.ID, IdempotencyKey: "terminal-ack-first",
	}))
	if err != nil {
		t.Fatal(err)
	}
	second, _, err := fixture.store.CreateBuild(context.Background(), fixture.bootstrap.Project.ID, control.Idempotency{Key: "terminal-ack-second", RequestHash: "terminal-ack-second"})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	finished, err := client.FinishBuild(ctx, connect.NewRequest(&autbackv1.FinishBuildRequest{Id: first.Msg.Build.Id}))
	if err != nil {
		t.Fatalf("FinishBuild waited for queue advance: %v", err)
	}
	if finished.Msg.Build.Status != autbackv1.BuildStatus_BUILD_STATUS_SUCCEEDED {
		t.Fatalf("finished status = %s, want succeeded", finished.Msg.Build.Status)
	}
	select {
	case <-capacityGate.started:
	case <-time.After(time.Second):
		t.Fatal("asynchronous queue advance did not start")
	}
	close(capacityGate.release)
	deadline := time.Now().Add(time.Second)
	for {
		build, lookupErr := fixture.store.Build(context.Background(), second.ID)
		if lookupErr == nil && build.Status == control.BuildRunning {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("second build status = %s, %v; want running", build.Status, lookupErr)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func TestBuildPreparationRequiresLeaseHeartbeatCapability(t *testing.T) {
	fixture := newFixtureWithBuildCapability(t, "build-lease-heartbeat")
	client := fixture.client(fixture.bootstrap.Token)
	request := func(key, capabilities string) *connect.Request[autbackv1.PrepareBuildRequest] {
		result := connect.NewRequest(&autbackv1.PrepareBuildRequest{
			Project: fixture.bootstrap.Project.ID, IdempotencyKey: key,
		})
		if capabilities != "" {
			result.Header().Set("Autback-Client-Capabilities", capabilities)
		}
		return result
	}

	if _, err := client.PrepareBuild(context.Background(), request("missing-build-capability", "")); connect.CodeOf(err) != connect.CodeFailedPrecondition {
		t.Fatalf("missing capability error = %v, want failed precondition", err)
	}
	if _, err := client.PrepareBuild(context.Background(), request("wrong-build-capability", "future-capability")); connect.CodeOf(err) != connect.CodeFailedPrecondition {
		t.Fatalf("wrong capability error = %v, want failed precondition", err)
	}
	response, err := client.PrepareBuild(context.Background(), request("compatible-build-client", "future-capability, build-lease-heartbeat"))
	if err != nil {
		t.Fatal(err)
	}
	if response.Msg.Build.Status != autbackv1.BuildStatus_BUILD_STATUS_RUNNING || response.Msg.Buildkit == nil {
		t.Fatalf("compatible build response = %#v", response.Msg)
	}
}

func TestJobPreparationRequiresDurableQueueCapability(t *testing.T) {
	fixture := newFixtureWithJobCapability(t, version.CapabilityDurableJobPrepare)
	client := fixture.client(fixture.bootstrap.Token)
	request := func(key, capabilities string) *connect.Request[autbackv1.PrepareJobRequest] {
		result := connect.NewRequest(&autbackv1.PrepareJobRequest{
			Project: fixture.bootstrap.Project.ID, IdempotencyKey: key,
			Image: "ghcr.io/example/ci@sha256:" + strings.Repeat("1", 64), Command: []string{"true"}, Timeout: durationpb.New(time.Minute),
		})
		if capabilities != "" {
			result.Header().Set(version.ClientCapabilitiesHeader, capabilities)
		}
		return result
	}

	if _, err := client.PrepareJob(context.Background(), request("missing-job-capability", "")); connect.CodeOf(err) != connect.CodeFailedPrecondition {
		t.Fatalf("missing capability error = %v, want failed precondition", err)
	}
	response, err := client.PrepareJob(context.Background(), request("compatible-job-client", "future-capability, "+version.CapabilityDurableJobPrepare))
	if err != nil {
		t.Fatal(err)
	}
	if response.Msg.Job == nil || response.Msg.Cas == nil {
		t.Fatalf("compatible job response = %#v", response.Msg)
	}
}

func TestAdmissionIdempotencyReplaysResourcesAndRejectsChangedRequests(t *testing.T) {
	fixture := newFixture(t)
	client := fixture.client(fixture.bootstrap.Token)
	ctx := context.Background()
	request := &autbackv1.PrepareJobRequest{
		Project: fixture.bootstrap.Project.ID, IdempotencyKey: "job-retry-1",
		Image: "ghcr.io/example/ci@sha256:" + strings.Repeat("4", 64), Command: []string{"task", "test"},
		Timeout: durationpb.New(time.Minute), Cpus: "1", Memory: "1g",
	}
	first, err := client.PrepareJob(ctx, connect.NewRequest(request))
	if err != nil {
		t.Fatal(err)
	}
	second, err := client.PrepareJob(ctx, connect.NewRequest(request))
	if err != nil {
		t.Fatal(err)
	}
	if second.Msg.Job.Id != first.Msg.Job.Id {
		t.Fatalf("replayed job ID = %q, want %q", second.Msg.Job.Id, first.Msg.Job.Id)
	}
	changed := proto.Clone(request).(*autbackv1.PrepareJobRequest)
	changed.Command = []string{"task", "ci"}
	if _, err := client.PrepareJob(ctx, connect.NewRequest(changed)); connect.CodeOf(err) != connect.CodeAlreadyExists {
		t.Fatalf("changed idempotent request error = %v", err)
	}

	buildRequest := &autbackv1.PrepareBuildRequest{Project: fixture.bootstrap.Project.ID, IdempotencyKey: "build-retry-1"}
	firstBuild, err := client.PrepareBuild(ctx, connect.NewRequest(buildRequest))
	if err != nil {
		t.Fatal(err)
	}
	secondBuild, err := client.PrepareBuild(ctx, connect.NewRequest(buildRequest))
	if err != nil {
		t.Fatal(err)
	}
	if secondBuild.Msg.Build.Id != firstBuild.Msg.Build.Id {
		t.Fatalf("replayed build ID = %q, want %q", secondBuild.Msg.Build.Id, firstBuild.Msg.Build.Id)
	}
}

func TestOneDeviceCredentialListsEveryAuthorizedProject(t *testing.T) {
	fixture := newFixture(t)
	ctx := context.Background()
	ownerClient := fixture.client(fixture.bootstrap.Token)
	second, err := ownerClient.CreateProject(ctx, connect.NewRequest(&autbackv1.CreateProjectRequest{Slug: "second-project", Name: "Second project"}))
	if err != nil {
		t.Fatal(err)
	}
	member, err := ownerClient.CreateUser(ctx, connect.NewRequest(&autbackv1.CreateUserRequest{Name: "Project member"}))
	if err != nil {
		t.Fatal(err)
	}
	for _, project := range []string{fixture.bootstrap.Project.ID, second.Msg.Project.Id} {
		if _, err := ownerClient.AddProjectMember(ctx, connect.NewRequest(&autbackv1.AddProjectMemberRequest{Project: project, UserId: member.Msg.User.Id})); err != nil {
			t.Fatal(err)
		}
	}
	issued, err := ownerClient.CreateDeviceToken(ctx, connect.NewRequest(&autbackv1.CreateDeviceTokenRequest{Name: "member-laptop", UserId: member.Msg.User.Id}))
	if err != nil {
		t.Fatal(err)
	}
	projects, err := fixture.client(issued.Msg.Token).ListProjects(ctx, connect.NewRequest(&autbackv1.ListProjectsRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	if len(projects.Msg.Projects) != 2 || projects.Msg.Projects[0].Slug != "example" || projects.Msg.Projects[1].Slug != "second-project" {
		t.Fatalf("projects = %#v", projects.Msg.Projects)
	}
}

func TestListJobsUsesOpaqueProjectBoundKeysetPagination(t *testing.T) {
	fixture := newFixture(t)
	client := fixture.client(fixture.bootstrap.Token)
	ctx := context.Background()
	for index := 0; index < 3; index++ {
		_, err := client.PrepareJob(ctx, connect.NewRequest(&autbackv1.PrepareJobRequest{
			Project: fixture.bootstrap.Project.ID, IdempotencyKey: fmt.Sprintf("page-job-%d", index),
			Image: "ghcr.io/example/ci@sha256:" + strings.Repeat("5", 64), Command: []string{"true"},
			Timeout: durationpb.New(time.Minute), Cpus: "1", Memory: "1g",
		}))
		if err != nil {
			t.Fatal(err)
		}
	}
	first, err := client.ListJobs(ctx, connect.NewRequest(&autbackv1.ListJobsRequest{Project: fixture.bootstrap.Project.ID, PageSize: 2}))
	if err != nil {
		t.Fatal(err)
	}
	if len(first.Msg.Jobs) != 2 || first.Msg.NextPageToken == "" {
		t.Fatalf("first page = %#v", first.Msg)
	}
	second, err := client.ListJobs(ctx, connect.NewRequest(&autbackv1.ListJobsRequest{
		Project: fixture.bootstrap.Project.ID, PageSize: 2, PageToken: first.Msg.NextPageToken,
	}))
	if err != nil {
		t.Fatal(err)
	}
	if len(second.Msg.Jobs) != 1 || second.Msg.NextPageToken != "" || second.Msg.Jobs[0].Id == first.Msg.Jobs[0].Id || second.Msg.Jobs[0].Id == first.Msg.Jobs[1].Id {
		t.Fatalf("second page = %#v", second.Msg)
	}
	if _, err := client.ListJobs(ctx, connect.NewRequest(&autbackv1.ListJobsRequest{
		Project: fixture.bootstrap.Project.ID, PageSize: 2, PageToken: first.Msg.NextPageToken + "tampered",
	})); connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("tampered page token error = %v", err)
	}
	owner, err := fixture.store.Authenticate(ctx, fixture.bootstrap.Token)
	if err != nil {
		t.Fatal(err)
	}
	otherProject, err := fixture.store.CreateProject(ctx, owner, "other-page-project", "Other page project")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := client.ListJobs(ctx, connect.NewRequest(&autbackv1.ListJobsRequest{
		Project: otherProject.ID, PageSize: 2, PageToken: first.Msg.NextPageToken,
	})); connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("cross-project page token error = %v", err)
	}
}

func TestStreamJobLogsResumesAtByteOffset(t *testing.T) {
	fixture := newFixture(t)
	client := fixture.client(fixture.bootstrap.Token)
	ctx := context.Background()
	prepared, err := client.PrepareJob(ctx, connect.NewRequest(&autbackv1.PrepareJobRequest{
		Project: fixture.bootstrap.Project.ID, IdempotencyKey: "job-log-offset-1",
		Image: "ghcr.io/example/ci@sha256:" + strings.Repeat("6", 64), Command: []string{"true"},
		Timeout: durationpb.New(time.Minute), Cpus: "1", Memory: "1g",
	}))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := client.StartJob(ctx, connect.NewRequest(&autbackv1.StartJobRequest{Id: prepared.Msg.Job.Id, RootDigest: strings.Repeat("c", 64) + "/1"})); err != nil {
		t.Fatal(err)
	}
	fixture.scheduler.complete(prepared.Msg.Job.Id, protocol.StatusSucceeded, 0)
	stream, err := client.StreamJobLogs(ctx, connect.NewRequest(&autbackv1.StreamJobLogsRequest{Id: prepared.Msg.Job.Id, Offset: 7}))
	if err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	var nextOffset int64
	for stream.Receive() {
		output.Write(stream.Msg().Data)
		nextOffset = stream.Msg().NextOffset
	}
	if err := stream.Err(); err != nil {
		t.Fatal(err)
	}
	if output.String() != "output\n" || nextOffset != int64(len("remote output\n")) {
		t.Fatalf("output=%q next offset=%d", output.String(), nextOffset)
	}
	stream, err = client.StreamJobLogs(ctx, connect.NewRequest(&autbackv1.StreamJobLogsRequest{Id: prepared.Msg.Job.Id, Offset: -1}))
	if err == nil {
		for stream.Receive() {
		}
		err = stream.Err()
	}
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("negative offset error = %v", err)
	}
}

func TestGitHubOIDCExchangeReturnsProjectScopedTemporaryToken(t *testing.T) {
	claims := control.GitHubClaims{
		Subject: "repo:flidai/leapview:environment:autback", RepositoryOwnerID: "100", RepositoryID: "200",
		WorkflowRef: "flidai/leapview/.github/workflows/ci.yml@refs/heads/main", Ref: "refs/heads/main",
		Environment: "autback", EventName: "workflow_dispatch", ExpiresAt: time.Now().Add(5 * time.Minute),
	}
	fixture := newFixtureWithVerifier(t, fakeVerifier{claims: claims})
	client := fixture.client(fixture.bootstrap.Token)
	_, err := client.CreateGitHubTrust(context.Background(), connect.NewRequest(&autbackv1.CreateGitHubTrustRequest{
		Project: fixture.bootstrap.Project.ID, RepositoryOwnerId: "100", RepositoryId: "200",
		WorkflowRef: "flidai/leapview/.github/workflows/ci.yml@refs/heads/*", Ref: "refs/heads/*",
		Environment: "autback", Events: []string{"workflow_dispatch"},
	}))
	if err != nil {
		t.Fatal(err)
	}
	exchanged, err := fixture.client("").ExchangeGitHubOIDC(context.Background(), connect.NewRequest(&autbackv1.ExchangeGitHubOIDCRequest{
		Project: fixture.bootstrap.Project.ID, IdToken: "signed-github-token",
	}))
	if err != nil {
		t.Fatal(err)
	}
	principal, err := fixture.store.Authenticate(context.Background(), exchanged.Msg.Token)
	if err != nil {
		t.Fatal(err)
	}
	if principal.Kind != control.PrincipalGitHub || principal.ProjectID != fixture.bootstrap.Project.ID || principal.Subject != claims.Subject {
		t.Fatalf("principal = %#v", principal)
	}
	if exchanged.Msg.ExpiresAt.AsTime().After(claims.ExpiresAt.Add(time.Second)) {
		t.Fatalf("exchange expiry %s exceeds identity expiry %s", exchanged.Msg.ExpiresAt.AsTime(), claims.ExpiresAt)
	}
}

func TestAdminBindsGitHubLoginToAnAutbackUserByImmutableID(t *testing.T) {
	fixture := newFixtureWithDirectory(t, fakeGitHubDirectory{identity: control.ExternalIdentity{Provider: "github", Subject: "12345678", Login: "yacobolo"}})
	client := fixture.client(fixture.bootstrap.Token)
	member, err := client.CreateUser(context.Background(), connect.NewRequest(&autbackv1.CreateUserRequest{Name: "Jacob"}))
	if err != nil {
		t.Fatal(err)
	}
	bound, err := client.BindGitHubIdentity(context.Background(), connect.NewRequest(&autbackv1.BindGitHubIdentityRequest{UserId: member.Msg.User.Id, Login: "Yacobolo"}))
	if err != nil {
		t.Fatal(err)
	}
	if bound.Msg.Identity.Subject != "12345678" || bound.Msg.Identity.Login != "yacobolo" || bound.Msg.Identity.UserId != member.Msg.User.Id {
		t.Fatalf("identity = %#v", bound.Msg.Identity)
	}
	resolved, err := fixture.store.UserByExternalIdentity(context.Background(), "github", "12345678", "yacobolo")
	if err != nil || resolved.ID != member.Msg.User.Id {
		t.Fatalf("resolved=%#v err=%v", resolved, err)
	}
}

func TestAdminRevokesGitHubIdentityAndAllHumanCredentials(t *testing.T) {
	fixture := newFixtureWithDirectory(t, fakeGitHubDirectory{identity: control.ExternalIdentity{Provider: "github", Subject: "12345678", Login: "yacobolo"}})
	client := fixture.client(fixture.bootstrap.Token)
	member, err := client.CreateUser(context.Background(), connect.NewRequest(&autbackv1.CreateUserRequest{Name: "Jacob"}))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := client.BindGitHubIdentity(context.Background(), connect.NewRequest(&autbackv1.BindGitHubIdentityRequest{UserId: member.Msg.User.Id, Login: "Yacobolo"})); err != nil {
		t.Fatal(err)
	}
	session, err := fixture.store.CreateBrowserSession(context.Background(), member.Msg.User.Id, time.Now().Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	owner, err := fixture.store.Authenticate(context.Background(), fixture.bootstrap.Token)
	if err != nil {
		t.Fatal(err)
	}
	device, err := fixture.store.CreateDeviceToken(context.Background(), owner, control.CreateDeviceToken{Name: "jacob-laptop", UserID: member.Msg.User.Id})
	if err != nil {
		t.Fatal(err)
	}

	if _, err := client.RevokeGitHubIdentity(context.Background(), connect.NewRequest(&autbackv1.RevokeGitHubIdentityRequest{UserId: member.Msg.User.Id})); err != nil {
		t.Fatal(err)
	}
	for name, token := range map[string]string{"browser": session.Token, "device": device.Secret} {
		if _, err := fixture.store.Authenticate(context.Background(), token); !errors.Is(err, control.ErrUnauthenticated) {
			t.Fatalf("%s authentication error = %v, want unauthenticated", name, err)
		}
	}
	events, err := fixture.store.ListAuditEvents(context.Background(), owner, "", 20)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.ContainsFunc(events, func(event control.AuditEvent) bool {
		return event.Action == "identity.github.revoke" && event.TargetID == member.Msg.User.Id
	}) {
		t.Fatalf("audit events = %#v", events)
	}
}

func TestBrowserSessionCannotAuthorizeControlAPI(t *testing.T) {
	fixture := newFixture(t)
	session, err := fixture.store.CreateBrowserSession(context.Background(), fixture.bootstrap.User.ID, time.Now().Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}

	_, err = fixture.client(session.Token).ListProjects(context.Background(), connect.NewRequest(&autbackv1.ListProjectsRequest{}))
	if connect.CodeOf(err) != connect.CodeUnauthenticated {
		t.Fatalf("ListProjects error = %v, want unauthenticated", err)
	}
}

type fakeGitHubDirectory struct {
	identity control.ExternalIdentity
	err      error
}

func (f fakeGitHubDirectory) Resolve(context.Context, string) (control.ExternalIdentity, error) {
	return f.identity, f.err
}

type fixture struct {
	store     *controlsqlite.Store
	bootstrap control.BootstrapResult
	scheduler *fakeScheduler
	server    *httptest.Server
	draining  *atomic.Bool
}

func newFixture(t *testing.T) *fixture {
	return newFixtureWithVerifier(t, nil)
}

func newFixtureWithVerifier(t *testing.T, verifier controlapi.OIDCVerifier) *fixture {
	return newFixtureWithCapacity(t, verifier, nil)
}

func newFixtureWithCapacity(t *testing.T, verifier controlapi.OIDCVerifier, capacity controlapi.Capacity) *fixture {
	return newFixtureWithBuildCapabilityAndCapacity(t, verifier, capacity, "")
}

func newFixtureWithBuildCapability(t *testing.T, capability string) *fixture {
	return newFixtureWithBuildCapabilityAndCapacity(t, nil, nil, capability)
}

func newFixtureWithBuildCapabilityAndCapacity(t *testing.T, verifier controlapi.OIDCVerifier, capacity controlapi.Capacity, capability string) *fixture {
	return newFixtureWithOptions(t, verifier, capacity, capability, "", nil, nil, 0)
}

func newFixtureWithJobCapability(t *testing.T, capability string) *fixture {
	return newFixtureWithOptions(t, nil, nil, "", capability, nil, nil, 0)
}

func newFixtureWithDirectory(t *testing.T, directory controlapi.GitHubDirectory) *fixture {
	return newFixtureWithOptions(t, nil, nil, "", "", directory, nil, 0)
}

func newFixtureWithOptions(t *testing.T, verifier controlapi.OIDCVerifier, capacity controlapi.Capacity, buildCapability, jobCapability string, directory controlapi.GitHubDirectory, dependencies []controlapi.ReadinessDependency, probeTimeout time.Duration) *fixture {
	t.Helper()
	root := t.TempDir()
	store, err := controlsqlite.Open(filepath.Join(root, "state"), []byte("test-pepper-that-is-at-least-32-bytes"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	bootstrap, err := store.Bootstrap(context.Background(), control.Bootstrap{UserName: "Owner", ProjectSlug: "example", ProjectName: "Example", TokenName: "test"})
	if err != nil {
		t.Fatal(err)
	}
	authority, err := pki.Ensure(filepath.Join(root, "pki"), []string{"localhost", "127.0.0.1"})
	if err != nil {
		t.Fatal(err)
	}
	scheduler := &fakeScheduler{jobs: map[string]protocol.Job{}}
	advanceCtx, cancelAdvance := context.WithCancel(context.Background())
	t.Cleanup(cancelAdvance)
	dispatch := dispatcher.New(store, scheduler, dispatcher.WithCapacity(capacity), dispatcher.WithAdvanceContext(advanceCtx))
	draining := &atomic.Bool{}
	handler, err := controlapi.New(controlapi.Config{
		Store: store, Scheduler: scheduler, Dispatcher: dispatch, Authority: authority,
		CASEndpoint: "cas.example:50051", CASInstance: "autback", BuildKitEndpoint: "buildkit.example:1234",
		CredentialTTL: 15 * time.Minute, OIDCVerifier: verifier, Capacity: capacity,
		RequiredBuildClientCapability: buildCapability,
		RequiredJobClientCapability:   jobCapability,
		Ready:                         func() bool { return !draining.Load() },
		GitHubDirectory:               directory,
		ReadinessDependencies:         dependencies,
		ReadinessProbeTimeout:         probeTimeout,
	})
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	return &fixture{store: store, bootstrap: bootstrap, scheduler: scheduler, server: server, draining: draining}
}

func TestJobPreparationQueuesWithoutCredentialsWhenCapacityIsExhausted(t *testing.T) {
	fixture := newFixtureWithCapacity(t, nil, fakeCapacity{err: &capacity.ResourceExhaustedError{FreeBytes: 1, RequiredBytes: 2}})
	client := fixture.client(fixture.bootstrap.Token)
	prepared, err := client.PrepareJob(context.Background(), connect.NewRequest(&autbackv1.PrepareJobRequest{
		IdempotencyKey: "capacity-job-1", Project: fixture.bootstrap.Project.Slug,
		Image: "ghcr.io/example/ci@sha256:" + strings.Repeat("1", 64), Command: []string{"true"}, Timeout: durationpb.New(time.Minute),
	}))
	if err != nil {
		t.Fatalf("PrepareJob returned terminal capacity error: %v", err)
	}
	if prepared.Msg.Job == nil || prepared.Msg.Job.Status != autbackv1.JobStatus_JOB_STATUS_PREPARING || prepared.Msg.Cas != nil {
		t.Fatalf("prepared response = %#v, want durable preparation without CAS credentials", prepared.Msg)
	}
	page, listErr := fixture.store.ListJobs(context.Background(), fixture.bootstrap.Project.ID, 20, "")
	if listErr != nil {
		t.Fatal(listErr)
	}
	if len(page.Jobs) != 1 || page.Jobs[0].ID != prepared.Msg.Job.Id {
		t.Fatalf("jobs = %#v, want persisted preparation", page.Jobs)
	}
	operation, stateErr := fixture.store.Operation(context.Background(), control.OperationJob, prepared.Msg.Job.Id)
	if stateErr != nil || operation.State != control.OperationQueued {
		t.Fatalf("operation = %#v, %v; want queued", operation, stateErr)
	}
	got, err := client.GetJob(context.Background(), connect.NewRequest(&autbackv1.GetJobRequest{Id: prepared.Msg.Job.Id}))
	if err != nil || got.Msg.Cas != nil {
		t.Fatalf("GetJob = %#v, %v; credentials must remain withheld", got, err)
	}
}

func TestBuildPreparationQueuesWithoutCredentialsWhenCapacityIsExhausted(t *testing.T) {
	fixture := newFixtureWithCapacity(t, nil, fakeCapacity{err: &capacity.ResourceExhaustedError{FreeBytes: 1, RequiredBytes: 2}})
	client := fixture.client(fixture.bootstrap.Token)
	prepared, err := client.PrepareBuild(context.Background(), connect.NewRequest(&autbackv1.PrepareBuildRequest{
		IdempotencyKey: "capacity-build-1", Project: fixture.bootstrap.Project.Slug,
	}))
	if err != nil {
		t.Fatalf("PrepareBuild returned terminal capacity error: %v", err)
	}
	if prepared.Msg.Build == nil || prepared.Msg.Build.Status != autbackv1.BuildStatus_BUILD_STATUS_QUEUED || prepared.Msg.Buildkit != nil {
		t.Fatalf("prepared response = %#v, want durable build without BuildKit credentials", prepared.Msg)
	}
	operation, stateErr := fixture.store.Operation(context.Background(), control.OperationBuild, prepared.Msg.Build.Id)
	if stateErr != nil || operation.State != control.OperationQueued {
		t.Fatalf("operation = %#v, %v; want queued", operation, stateErr)
	}
}

type fakeCapacity struct{ err error }

func (f fakeCapacity) Ensure(context.Context) error { return f.err }
func (f fakeCapacity) Admit(_ context.Context, reserve func() error) error {
	if f.err != nil {
		return f.err
	}
	return reserve()
}
func (f fakeCapacity) Check(context.Context) error { return f.err }

type blockingAdvanceCapacity struct {
	mu      sync.Mutex
	admits  int
	started chan struct{}
	release chan struct{}
	once    sync.Once
}

func (*blockingAdvanceCapacity) Ensure(context.Context) error { return nil }
func (*blockingAdvanceCapacity) Check(context.Context) error  { return nil }
func (b *blockingAdvanceCapacity) Admit(ctx context.Context, reserve func() error) error {
	b.mu.Lock()
	b.admits++
	admit := b.admits
	b.mu.Unlock()
	if admit > 1 {
		b.once.Do(func() { close(b.started) })
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-b.release:
		}
	}
	return reserve()
}

func TestReadinessChecksStoreAndScheduler(t *testing.T) {
	fixture := newFixture(t)
	response, err := http.Get(fixture.server.URL + "/readyz")
	if err != nil {
		t.Fatal(err)
	}
	_ = response.Body.Close()
	if response.StatusCode != http.StatusNoContent {
		t.Fatalf("ready status = %d", response.StatusCode)
	}
	fixture.draining.Store(true)
	response, err = http.Get(fixture.server.URL + "/readyz")
	if err != nil {
		t.Fatal(err)
	}
	_ = response.Body.Close()
	if response.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("draining status = %d", response.StatusCode)
	}
	fixture.draining.Store(false)

	fixture.scheduler.checkErr = errors.New("swarm unavailable")
	response, err = http.Get(fixture.server.URL + "/readyz")
	if err != nil {
		t.Fatal(err)
	}
	_ = response.Body.Close()
	if response.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("unready status = %d", response.StatusCode)
	}
}

func TestReadinessChecksBoundedDataPlaneDependenciesAndRecovers(t *testing.T) {
	casErr := errors.New("dial cas with secret credential")
	buildKitErr := error(nil)
	fixture := newFixtureWithOptions(t, nil, nil, "", "", nil, []controlapi.ReadinessDependency{
		{Name: "CAS", Check: func(context.Context) error { return casErr }},
		{Name: "BuildKit", Check: func(context.Context) error { return buildKitErr }},
	}, 20*time.Millisecond)

	response, err := http.Get(fixture.server.URL + "/readyz")
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(response.Body)
	_ = response.Body.Close()
	if response.StatusCode != http.StatusServiceUnavailable || !strings.Contains(string(body), "CAS unavailable") || strings.Contains(string(body), "secret credential") {
		t.Fatalf("CAS readiness status=%d body=%q", response.StatusCode, body)
	}

	casErr = nil
	buildKitErr = errors.New("buildkit socket failed")
	response, err = http.Get(fixture.server.URL + "/readyz")
	if err != nil {
		t.Fatal(err)
	}
	body, _ = io.ReadAll(response.Body)
	_ = response.Body.Close()
	if response.StatusCode != http.StatusServiceUnavailable || !strings.Contains(string(body), "BuildKit unavailable") || strings.Contains(string(body), "socket failed") {
		t.Fatalf("BuildKit readiness status=%d body=%q", response.StatusCode, body)
	}

	buildKitErr = nil
	response, err = http.Get(fixture.server.URL + "/readyz")
	if err != nil {
		t.Fatal(err)
	}
	_ = response.Body.Close()
	if response.StatusCode != http.StatusNoContent {
		t.Fatalf("recovered readiness status=%d", response.StatusCode)
	}
}

func TestReadinessBoundsHungDependencyProbe(t *testing.T) {
	fixture := newFixtureWithOptions(t, nil, nil, "", "", nil, []controlapi.ReadinessDependency{{
		Name: "CAS", Check: func(ctx context.Context) error { <-ctx.Done(); return ctx.Err() },
	}}, 10*time.Millisecond)
	started := time.Now()
	response, err := http.Get(fixture.server.URL + "/readyz")
	if err != nil {
		t.Fatal(err)
	}
	_ = response.Body.Close()
	if response.StatusCode != http.StatusServiceUnavailable || time.Since(started) > time.Second {
		t.Fatalf("hung probe status=%d elapsed=%s", response.StatusCode, time.Since(started))
	}
}

func (f *fixture) client(token string) autbackv1connect.ControlServiceClient {
	transport := roundTripper(func(request *http.Request) (*http.Response, error) {
		if token != "" {
			request.Header.Set("Authorization", "Bearer "+token)
		}
		return http.DefaultTransport.RoundTrip(request)
	})
	return autbackv1connect.NewControlServiceClient(&http.Client{Transport: transport}, f.server.URL)
}

type roundTripper func(*http.Request) (*http.Response, error)

func (r roundTripper) RoundTrip(request *http.Request) (*http.Response, error) { return r(request) }

type fakeScheduler struct {
	mu                 sync.Mutex
	created            []control.Job
	jobs               map[string]protocol.Job
	blockFollowingLogs bool
	logsStarted        chan struct{}
	validateErr        error
	checkErr           error
}

func (f *fakeScheduler) Check(context.Context) error { return f.checkErr }

func (f *fakeScheduler) ValidateImage(context.Context, string) error { return f.validateErr }

func (f *fakeScheduler) Create(_ context.Context, job control.Job) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.created = append(f.created, job)
	f.jobs[job.ID] = protocol.Job{ID: job.ID, ProjectID: job.ProjectID, Image: job.Image, Command: job.Command, Status: protocol.StatusQueued, CreatedAt: job.CreatedAt, TimeoutSeconds: int(job.Timeout.Seconds())}
	return nil
}

func (f *fakeScheduler) Status(_ context.Context, id string) (protocol.Job, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.jobs[id], nil
}

func (f *fakeScheduler) Logs(ctx context.Context, _ string, follow bool, output io.Writer) error {
	if _, err := io.WriteString(output, "remote output\n"); err != nil {
		return err
	}
	if follow && f.blockFollowingLogs {
		close(f.logsStarted)
		<-ctx.Done()
		return ctx.Err()
	}
	return nil
}

func (f *fakeScheduler) Cancel(_ context.Context, id string) error {
	f.complete(id, protocol.StatusCancelled, 130)
	return nil
}

func (f *fakeScheduler) createdCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.created)
}

func (f *fakeScheduler) createdAt(index int) control.Job {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.created[index]
}

func (f *fakeScheduler) createdSnapshot() []control.Job {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]control.Job(nil), f.created...)
}

func (f *fakeScheduler) complete(id string, status protocol.Status, exitCode int) {
	f.mu.Lock()
	defer f.mu.Unlock()
	job := f.jobs[id]
	now := time.Now().UTC()
	job.Status, job.ExitCode, job.FinishedAt = status, &exitCode, &now
	f.jobs[id] = job
}

var _ control.Scheduler = (*fakeScheduler)(nil)

type fakeVerifier struct {
	claims control.GitHubClaims
	err    error
}

func (f fakeVerifier) Verify(context.Context, string) (control.GitHubClaims, error) {
	return f.claims, f.err
}
