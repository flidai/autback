package cli

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"

	"connectrpc.com/connect"
	"github.com/flidai/autback/internal/authclient"
	"github.com/flidai/autback/internal/buildkit"
	"github.com/flidai/autback/internal/cas"
	"github.com/flidai/autback/internal/config"
	"github.com/flidai/autback/internal/control/controlclient"
	"github.com/flidai/autback/internal/credentialfiles"
	autbackv1 "github.com/flidai/autback/internal/gen/rtest/v1"
	"github.com/flidai/autback/internal/gen/rtest/v1/autbackv1connect"
	"github.com/flidai/autback/internal/projectlink"
	"github.com/flidai/autback/internal/protocol"
	"github.com/flidai/autback/internal/workspace"
	"golang.org/x/term"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func runService(ctx context.Context, settings config.Config, explicitToken string, args []string, streams IO) int {
	if len(args) == 0 {
		usage(streams.Stderr)
		return 2
	}
	switch args[0] {
	case "login":
		return serviceLogin(ctx, settings, explicitToken, args[1:], streams)
	case "logout":
		return serviceLogout(settings, streams)
	case "console":
		return serviceConsole(ctx, settings, explicitToken, args[1:], streams)
	}
	// Commands such as doctor do not otherwise need a project, but an Actions
	// workload identity must select one before it can exchange its OIDC token.
	selectedProject := strings.TrimSpace(os.Getenv("AUTBACK_PROJECT"))
	if args[0] == "exec" || args[0] == "build" || args[0] == "image" || args[0] == "list" {
		explicitProject, err := explicitProject(args[1:])
		if err != nil {
			return failUsage(streams.Stderr, err.Error())
		}
		selectedProject, err = projectlink.Resolve(ctx, streams.Dir, explicitProject, os.Getenv("AUTBACK_PROJECT"))
		if err != nil {
			return failUsage(streams.Stderr, err.Error())
		}
	}
	api, source, err := authenticatedServiceClient(ctx, settings, explicitToken, selectedProject, streams.Keyring)
	if err != nil {
		return fail(streams.Stderr, err)
	}
	_ = source
	switch args[0] {
	case "init":
		return serviceInit(ctx, api, args[1:], streams)
	case "exec":
		return serviceExec(ctx, api, settings, selectedProject, args[1:], streams)
	case "build":
		return serviceBuild(ctx, api, settings, selectedProject, args[1:], streams)
	case "image":
		return serviceImage(ctx, api, settings, selectedProject, args[1:], streams)
	case "status":
		return serviceStatus(ctx, api, args[1:], streams)
	case "logs":
		return serviceLogs(ctx, api, args[1:], streams)
	case "cancel":
		return serviceCancel(ctx, api, args[1:], streams)
	case "list":
		return serviceList(ctx, api, selectedProject, args[1:], streams)
	case "doctor":
		return serviceDoctor(ctx, api, streams)
	case "token":
		return serviceToken(ctx, api, args[1:], streams)
	case "trust":
		return serviceTrust(ctx, api, args[1:], streams)
	case "admin":
		return serviceAdmin(ctx, api, args[1:], streams)
	default:
		return failUsage(streams.Stderr, "unknown service command "+args[0])
	}
}

func serviceAdmin(ctx context.Context, api autbackv1connect.ControlServiceClient, args []string, streams IO) int {
	if len(args) < 2 {
		return failUsage(streams.Stderr, "admin requires user create, project create, identity github, identity revoke, member add, or enrollment create")
	}
	resource, command, args := args[0], args[1], args[2:]
	values := map[string]string{}
	admin := false
	for len(args) > 0 {
		if args[0] == "--admin" {
			admin, args = true, args[1:]
			continue
		}
		if len(args) < 2 {
			return failUsage(streams.Stderr, args[0]+" requires a value")
		}
		values[args[0]], args = args[1], args[2:]
	}
	switch resource + " " + command {
	case "user create":
		if values["--name"] == "" {
			return failUsage(streams.Stderr, "admin user create requires --name")
		}
		response, err := api.CreateUser(ctx, connect.NewRequest(&autbackv1.CreateUserRequest{Name: values["--name"], Admin: admin}))
		if err != nil {
			return fail(streams.Stderr, err)
		}
		return encode(streams, response.Msg.User)
	case "project create":
		if values["--slug"] == "" || values["--name"] == "" {
			return failUsage(streams.Stderr, "admin project create requires --slug and --name")
		}
		response, err := api.CreateProject(ctx, connect.NewRequest(&autbackv1.CreateProjectRequest{Slug: values["--slug"], Name: values["--name"]}))
		if err != nil {
			return fail(streams.Stderr, err)
		}
		return encode(streams, response.Msg.Project)
	case "identity github":
		if values["--user"] == "" || values["--login"] == "" {
			return failUsage(streams.Stderr, "admin identity github requires --user and --login")
		}
		response, err := api.BindGitHubIdentity(ctx, connect.NewRequest(&autbackv1.BindGitHubIdentityRequest{UserId: values["--user"], Login: values["--login"]}))
		if err != nil {
			return fail(streams.Stderr, err)
		}
		return encode(streams, response.Msg.Identity)
	case "identity revoke":
		if values["--user"] == "" {
			return failUsage(streams.Stderr, "admin identity revoke requires --user")
		}
		if _, err := api.RevokeGitHubIdentity(ctx, connect.NewRequest(&autbackv1.RevokeGitHubIdentityRequest{UserId: values["--user"]})); err != nil {
			return fail(streams.Stderr, err)
		}
		fmt.Fprintln(streams.Stdout, "Revoked GitHub identity and active human credentials for "+values["--user"])
		return 0
	case "member add":
		if values["--project"] == "" || values["--user"] == "" {
			return failUsage(streams.Stderr, "admin member add requires --project and --user")
		}
		if _, err := api.AddProjectMember(ctx, connect.NewRequest(&autbackv1.AddProjectMemberRequest{Project: values["--project"], UserId: values["--user"]})); err != nil {
			return fail(streams.Stderr, err)
		}
		fmt.Fprintf(streams.Stdout, "Added user %s to project %s\n", values["--user"], values["--project"])
		return 0
	case "enrollment create":
		if values["--user"] == "" || values["--device"] == "" {
			return failUsage(streams.Stderr, "admin enrollment create requires --user and --device")
		}
		expires := 10 * time.Minute
		if values["--expires"] != "" {
			parsed, err := time.ParseDuration(values["--expires"])
			if err != nil || parsed < time.Minute || parsed > 30*time.Minute {
				return failUsage(streams.Stderr, "enrollment expiry must be between 1m and 30m")
			}
			expires = parsed
		}
		response, err := api.CreateEnrollmentCode(ctx, connect.NewRequest(&autbackv1.CreateEnrollmentCodeRequest{
			UserId: values["--user"], DeviceName: values["--device"], ExpiresAt: timestamppb.New(time.Now().Add(expires)),
		}))
		if err != nil {
			return fail(streams.Stderr, err)
		}
		fmt.Fprintln(streams.Stdout, response.Msg.Code)
		fmt.Fprintf(streams.Stderr, "Enrollment for %s expires at %s and can be used once.\n", response.Msg.Enrollment.DeviceName, response.Msg.Enrollment.ExpiresAt.AsTime().Format(time.RFC3339))
		return 0
	default:
		return failUsage(streams.Stderr, "admin requires user create, project create, identity github, identity revoke, member add, or enrollment create")
	}
}

func authenticatedServiceClient(ctx context.Context, settings config.Config, explicitToken, project string, keyring authclient.Keyring) (autbackv1connect.ControlServiceClient, authclient.Source, error) {
	github := authclient.GitHubActions{}
	oidc := func(ctx context.Context) (string, error) {
		if !github.Available() {
			return "", errors.New("GitHub Actions OIDC identity is unavailable")
		}
		if project == "" {
			return "", errors.New("repository project selection is required for GitHub OIDC")
		}
		idToken, err := github.IDToken(ctx, settings.Service.OIDCAudience)
		if err != nil {
			return "", err
		}
		unauthenticated, err := controlclient.New(settings.URL, "", settings.Service.CACertFile)
		if err != nil {
			return "", err
		}
		response, err := unauthenticated.ExchangeGitHubOIDC(ctx, connect.NewRequest(&autbackv1.ExchangeGitHubOIDCRequest{Project: project, IdToken: idToken}))
		if err != nil {
			return "", err
		}
		return response.Msg.Token, nil
	}
	if !github.Available() {
		oidc = nil
	}
	token, source, err := authclient.Resolve(ctx, authclient.ResolveOptions{
		ExplicitToken: explicitToken, ServiceURL: settings.URL, Keyring: keyring, OIDC: oidc,
	})
	if err != nil {
		return nil, "", err
	}
	api, err := controlclient.New(settings.URL, token, settings.Service.CACertFile)
	if err != nil {
		return nil, "", err
	}
	if source == authclient.SourceOIDC {
		api = &renewableControlClient{
			ControlServiceClient: api,
			renew: func(ctx context.Context) (autbackv1connect.ControlServiceClient, error) {
				token, err := oidc(ctx)
				if err != nil {
					return nil, err
				}
				return controlclient.New(settings.URL, token, settings.Service.CACertFile)
			},
		}
	}
	return api, source, nil
}

type renewableControlClient struct {
	autbackv1connect.ControlServiceClient
	renew func(context.Context) (autbackv1connect.ControlServiceClient, error)
}

func renewServiceClient(ctx context.Context, api autbackv1connect.ControlServiceClient) (autbackv1connect.ControlServiceClient, error) {
	renewable, ok := api.(*renewableControlClient)
	if !ok {
		return api, nil
	}
	refreshed, err := renewable.renew(ctx)
	if err != nil {
		return nil, err
	}
	// Keep the renewable wrapper so a long-running command can refresh more
	// than once while it waits in FIFO or executes remotely.
	renewable.ControlServiceClient = refreshed
	return renewable, nil
}

func serviceLogin(ctx context.Context, settings config.Config, explicitToken string, args []string, streams IO) int {
	streams = defaults(streams)
	token := explicitToken
	if token == "" && len(args) == 2 && args[0] == "--token" {
		token = args[1]
		args = nil
	}
	recovery, noOpen, deviceName := false, false, ""
	for len(args) > 0 {
		switch args[0] {
		case "--recovery-code":
			recovery, args = true, args[1:]
		case "--no-open":
			noOpen, args = true, args[1:]
		case "--device":
			if len(args) < 2 || strings.TrimSpace(args[1]) == "" {
				return failUsage(streams.Stderr, "--device requires a name")
			}
			deviceName, args = strings.TrimSpace(args[1]), args[2:]
		default:
			return failUsage(streams.Stderr, "login accepts --device <name>, --no-open, or --recovery-code")
		}
	}
	if recovery && (token != "" || noOpen || deviceName != "") {
		return failUsage(streams.Stderr, "--recovery-code cannot be combined with another login method")
	}
	if token == "" && recovery {
		code, err := readEnrollmentCode(streams)
		if err != nil {
			return fail(streams.Stderr, err)
		}
		api, err := controlclient.New(settings.URL, "", settings.Service.CACertFile)
		if err != nil {
			return fail(streams.Stderr, err)
		}
		exchanged, err := api.ExchangeEnrollmentCode(ctx, connect.NewRequest(&autbackv1.ExchangeEnrollmentCodeRequest{Code: code}))
		if err != nil {
			return fail(streams.Stderr, fmt.Errorf("exchange enrollment code: %w", err))
		}
		token = exchanged.Msg.Token
	}
	if token == "" {
		if deviceName == "" {
			hostname, err := os.Hostname()
			if err != nil || strings.TrimSpace(hostname) == "" {
				deviceName = "developer-device"
			} else {
				deviceName = hostname
			}
		}
		httpClient, target, err := controlclient.NewHTTPClient(settings.URL, "", settings.Service.CACertFile)
		if err != nil {
			return fail(streams.Stderr, err)
		}
		loginClient, err := authclient.NewBrowserLoginClient(target.String(), httpClient)
		if err != nil {
			return fail(streams.Stderr, err)
		}
		login, err := loginClient.Start(ctx, deviceName)
		if err != nil {
			return fail(streams.Stderr, fmt.Errorf("start browser login: %w", err))
		}
		fmt.Fprintf(streams.Stdout, "Open %s\nCode: %s\n", login.VerificationURIComplete, login.UserCode)
		if !noOpen {
			if err := streams.OpenURL(login.VerificationURIComplete); err != nil {
				fmt.Fprintln(streams.Stderr, "Open the login URL in a browser; automatic opening failed:", err)
			}
		}
		for {
			if !time.Now().Before(login.ExpiresAt) {
				return fail(streams.Stderr, errors.New("browser login expired"))
			}
			if err := streams.Wait(ctx, login.Interval); err != nil {
				return fail(streams.Stderr, err)
			}
			issued, pending, err := loginClient.Poll(ctx, login.DeviceCode)
			if err != nil {
				return fail(streams.Stderr, err)
			}
			if pending {
				continue
			}
			token = issued.Token
			break
		}
	}
	api, err := controlclient.New(settings.URL, token, settings.Service.CACertFile)
	if err != nil {
		return fail(streams.Stderr, err)
	}
	if _, err := api.ListDeviceTokens(ctx, connect.NewRequest(&autbackv1.ListDeviceTokensRequest{})); err != nil {
		return fail(streams.Stderr, fmt.Errorf("validate device token: %w", err))
	}
	if err := authclient.StoreToken(streams.Keyring, settings.URL, token); err != nil {
		return fail(streams.Stderr, fmt.Errorf("store device token: %w", err))
	}
	fmt.Fprintln(streams.Stdout, "Authenticated to "+settings.URL)
	return 0
}

func serviceLogout(settings config.Config, streams IO) int {
	if err := authclient.DeleteToken(streams.Keyring, settings.URL); err != nil {
		return fail(streams.Stderr, err)
	}
	fmt.Fprintln(streams.Stdout, "Removed the local autback credential")
	return 0
}

func readEnrollmentCode(streams IO) (string, error) {
	fmt.Fprint(streams.Stderr, "Enrollment code: ")
	if input, ok := streams.Stdin.(*os.File); ok && term.IsTerminal(int(input.Fd())) {
		secret, err := term.ReadPassword(int(input.Fd()))
		fmt.Fprintln(streams.Stderr)
		if err != nil {
			return "", err
		}
		return validateEnrollmentInput(string(secret))
	}
	reader := bufio.NewReader(io.LimitReader(streams.Stdin, 257))
	secret, err := reader.ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return "", err
	}
	return validateEnrollmentInput(secret)
}

func validateEnrollmentInput(value string) (string, error) {
	value = strings.TrimSpace(value)
	if len(value) < 32 || len(value) > 256 {
		return "", errors.New("invalid enrollment code")
	}
	return value, nil
}

func serviceInit(ctx context.Context, api autbackv1connect.ControlServiceClient, args []string, streams IO) int {
	project := os.Getenv("AUTBACK_PROJECT")
	if len(args) == 2 && args[0] == "--project" && args[1] != "" {
		project = args[1]
	} else if len(args) != 0 {
		return failUsage(streams.Stderr, "init accepts only --project <project>")
	}
	response, err := api.ListProjects(ctx, connect.NewRequest(&autbackv1.ListProjectsRequest{}))
	if err != nil {
		return fail(streams.Stderr, fmt.Errorf("list authorized projects: %w", err))
	}
	if project == "" {
		if len(response.Msg.Projects) != 1 {
			available := make([]string, 0, len(response.Msg.Projects))
			for _, item := range response.Msg.Projects {
				available = append(available, item.Slug)
			}
			return failUsage(streams.Stderr, fmt.Sprintf("init requires --project when %d projects are available: %s", len(available), strings.Join(available, ", ")))
		}
		project = response.Msg.Projects[0].Slug
	}
	selected, err := selectAuthorizedProject(response.Msg.Projects, project)
	if err != nil {
		return fail(streams.Stderr, err)
	}
	if _, err := workspace.Root(ctx, streams.Dir); err != nil {
		return fail(streams.Stderr, err)
	}
	path, err := projectlink.Write(streams.Dir, selected.Slug)
	if err != nil {
		return fail(streams.Stderr, err)
	}
	fmt.Fprintf(streams.Stdout, "Linked %s to autback project %s\n", path, selected.Slug)
	return 0
}

func explicitProject(args []string) (string, error) {
	project := ""
	for index := 0; index < len(args); index++ {
		if args[index] == "--" {
			break
		}
		if args[index] != "--project" {
			continue
		}
		if index+1 >= len(args) || args[index+1] == "" || args[index+1] == "--" {
			return "", errors.New("--project requires a value")
		}
		candidate := args[index+1]
		if project != "" && project != candidate {
			return "", errors.New("conflicting --project values")
		}
		project = candidate
		index++
	}
	return project, nil
}

func authorizedProject(ctx context.Context, api autbackv1connect.ControlServiceClient, selector string) (*autbackv1.Project, error) {
	response, err := api.ListProjects(ctx, connect.NewRequest(&autbackv1.ListProjectsRequest{}))
	if err != nil {
		return nil, fmt.Errorf("authorize project %q: %w", selector, err)
	}
	return selectAuthorizedProject(response.Msg.Projects, selector)
}

func selectAuthorizedProject(projects []*autbackv1.Project, selector string) (*autbackv1.Project, error) {
	for _, project := range projects {
		if project.Id == selector || project.Slug == selector {
			return project, nil
		}
	}
	return nil, fmt.Errorf("not authorized for autback project %q", selector)
}

type execOptions struct {
	project, image, workdir string
	timeout                 time.Duration
	detach                  bool
	environment             map[string]string
	command                 []string
	caches                  []*autbackv1.CacheMount
	secrets                 []*autbackv1.JobSecret
}

func serviceExec(ctx context.Context, api autbackv1connect.ControlServiceClient, settings config.Config, project string, args []string, streams IO) int {
	options, err := parseExec(settings, project, args)
	if err != nil {
		return failUsage(streams.Stderr, err.Error())
	}
	if _, err := authorizedProject(ctx, api, project); err != nil {
		return fail(streams.Stderr, err)
	}
	root, err := workspace.Root(ctx, streams.Dir)
	if err != nil {
		return fail(streams.Stderr, err)
	}
	if options.workdir == "" {
		options.workdir, err = defaultExecWorkingDirectory(root, streams.Dir)
		if err != nil {
			return fail(streams.Stderr, err)
		}
	}
	files, err := workspace.Files(ctx, root)
	if err != nil {
		return fail(streams.Stderr, err)
	}
	idempotencyKey, err := jobID()
	if err != nil {
		return fail(streams.Stderr, err)
	}
	prepared, err := api.PrepareJob(ctx, connect.NewRequest(&autbackv1.PrepareJobRequest{
		Project: options.project, Image: options.image, Command: options.command, WorkingDirectory: options.workdir,
		Environment: options.environment, Timeout: durationpb.New(options.timeout),
		IdempotencyKey: idempotencyKey, Caches: options.caches, Secrets: options.secrets,
	}))
	if err != nil {
		return fail(streams.Stderr, fmt.Errorf("prepare remote job: %w", err))
	}
	if prepared.Msg.Job == nil {
		return fail(streams.Stderr, errors.New("prepare remote job returned no job"))
	}
	fmt.Fprintf(streams.Stderr, "Backend: autback shared service\nJob: %s\n", prepared.Msg.Job.Id)
	if prepared.Msg.Cas == nil {
		fmt.Fprintln(streams.Stderr, "Waiting for worker capacity...")
	}
	job, connection, err := waitForServiceJobPreparation(ctx, api, prepared.Msg.Job, prepared.Msg.Cas)
	if err != nil {
		return fail(streams.Stderr, cancelServiceJobAfterError(api, prepared.Msg.Job.Id, err))
	}
	credentials, err := credentialfiles.Write(connection.CaPem, connection.CertificatePem, connection.PrivateKeyPem)
	if err != nil {
		return fail(streams.Stderr, cancelServiceJobAfterError(api, job.Id, err))
	}
	defer credentials.Cleanup()
	uploadCtx, stopUpload := context.WithCancel(ctx)
	heartbeatCtx, stopHeartbeat := context.WithCancel(uploadCtx)
	heartbeatDone := make(chan error, 1)
	go func() {
		err := heartbeatServiceJobPreparation(heartbeatCtx, api, job.Id, 30*time.Second)
		if err != nil {
			stopUpload()
		}
		heartbeatDone <- err
	}()
	upload, err := cas.UploadConnection(uploadCtx, cas.Connection{
		Service: grpcAddress(connection.Endpoint), Instance: connection.InstanceName,
		CACertFile: credentials.CA, ClientCertFile: credentials.Certificate, ClientKeyFile: credentials.Key,
		ServerName: connection.ServerName,
	}, root, files)
	stopHeartbeat()
	heartbeatErr := <-heartbeatDone
	stopUpload()
	if heartbeatErr != nil {
		return fail(streams.Stderr, cancelServiceJobAfterError(api, job.Id, fmt.Errorf("renew remote job preparation lease: %w", heartbeatErr)))
	}
	if err != nil {
		return fail(streams.Stderr, cancelServiceJobAfterError(api, job.Id, fmt.Errorf("upload inputs: %w", err)))
	}
	started, err := api.StartJob(ctx, connect.NewRequest(&autbackv1.StartJobRequest{Id: job.Id, RootDigest: upload.RootDigest}))
	if err != nil {
		return fail(streams.Stderr, cancelServiceJobAfterError(api, job.Id, fmt.Errorf("start remote job: %w", err)))
	}
	fmt.Fprintf(streams.Stderr, "Inputs: %d files, %s\nTransfer: %s uploaded\n",
		upload.InputFiles, humanBytes(upload.TotalInputBytes), humanBytes(upload.TransferredBytes))
	if options.detach {
		return 0
	}
	return waitServiceJob(ctx, api, started.Msg.Job.Id, streams)
}

func waitForServiceJobPreparation(ctx context.Context, api autbackv1connect.ControlServiceClient, job *autbackv1.Job, connection *autbackv1.DataPlaneConnection) (*autbackv1.Job, *autbackv1.DataPlaneConnection, error) {
	if job == nil {
		return nil, nil, errors.New("prepare job returned no job")
	}
	for job.Status == autbackv1.JobStatus_JOB_STATUS_PREPARING && connection == nil {
		timer := time.NewTimer(250 * time.Millisecond)
		select {
		case <-ctx.Done():
			timer.Stop()
			return nil, nil, fmt.Errorf("wait for remote job preparation: %w", ctx.Err())
		case <-timer.C:
		}
		response, err := api.GetJob(ctx, connect.NewRequest(&autbackv1.GetJobRequest{Id: job.Id}))
		if connect.CodeOf(err) == connect.CodeUnauthenticated {
			renewed, renewErr := renewServiceClient(ctx, api)
			if renewErr != nil {
				return nil, nil, renewErr
			}
			api = renewed
			response, err = renewed.GetJob(ctx, connect.NewRequest(&autbackv1.GetJobRequest{Id: job.Id}))
		}
		if err != nil {
			return nil, nil, fmt.Errorf("get remote job preparation: %w", err)
		}
		job, connection = response.Msg.Job, response.Msg.Cas
		if job == nil {
			return nil, nil, errors.New("get remote job preparation returned no job")
		}
	}
	if job.Status != autbackv1.JobStatus_JOB_STATUS_PREPARING || connection == nil {
		return job, connection, fmt.Errorf("remote job became %s before source upload", job.Status)
	}
	return job, connection, nil
}

func heartbeatServiceJobPreparation(ctx context.Context, api autbackv1connect.ControlServiceClient, id string, interval time.Duration) error {
	if interval <= 0 {
		return errors.New("job preparation heartbeat interval must be positive")
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
		}
		response, err := api.GetJob(ctx, connect.NewRequest(&autbackv1.GetJobRequest{Id: id}))
		if connect.CodeOf(err) == connect.CodeUnauthenticated {
			api, err = renewServiceClient(ctx, api)
			if err == nil {
				response, err = api.GetJob(ctx, connect.NewRequest(&autbackv1.GetJobRequest{Id: id}))
			}
		}
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			return err
		}
		if response.Msg.Job == nil || response.Msg.Job.Status != autbackv1.JobStatus_JOB_STATUS_PREPARING || response.Msg.Cas == nil {
			return errors.New("remote job preparation lease is no longer admitted")
		}
	}
}

func cancelServiceJobAfterError(api autbackv1connect.ControlServiceClient, id string, cause error) error {
	cancelCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	cancelAPI, err := renewServiceClient(cancelCtx, api)
	if err == nil {
		_, err = cancelAPI.CancelJob(cancelCtx, connect.NewRequest(&autbackv1.CancelJobRequest{Id: id}))
	}
	if err != nil {
		return errors.Join(cause, fmt.Errorf("cancel abandoned job %s: %w", id, err))
	}
	return cause
}

func parseExec(settings config.Config, project string, args []string) (execOptions, error) {
	options := execOptions{
		project: project, image: settings.Service.Image,
		timeout: 30 * time.Minute, environment: map[string]string{},
	}
	for len(args) > 0 && args[0] != "--" {
		switch args[0] {
		case "--detach":
			options.detach, args = true, args[1:]
		case "--project", "--image", "--workdir", "--timeout", "--env", "--cache", "--secret-env", "--secret-file":
			if len(args) < 2 {
				return execOptions{}, errors.New(args[0] + " requires a value")
			}
			flag, value := args[0], args[1]
			args = args[2:]
			switch flag {
			case "--project":
				options.project = value
			case "--image":
				options.image = value
			case "--workdir":
				options.workdir = value
			case "--timeout":
				parsed, err := time.ParseDuration(value)
				if err != nil {
					return execOptions{}, errors.New("invalid --timeout")
				}
				options.timeout = parsed
			case "--env":
				key, item, ok := strings.Cut(value, "=")
				if !ok || key == "" {
					return execOptions{}, errors.New("--env requires KEY=VALUE")
				}
				options.environment[key] = item
			case "--cache":
				name, target, ok := strings.Cut(value, "=")
				if !ok || name == "" || target == "" {
					return execOptions{}, errors.New("--cache requires NAME=/absolute/container/path")
				}
				options.caches = append(options.caches, &autbackv1.CacheMount{Name: name, Target: target})
			case "--secret-env":
				name, target, ok := strings.Cut(value, "=")
				if !ok || name == "" || target == "" {
					return execOptions{}, errors.New("--secret-env requires NAME=ENVIRONMENT_KEY")
				}
				options.secrets = append(options.secrets, &autbackv1.JobSecret{Name: name, Target: &autbackv1.JobSecret_Environment{Environment: target}})
			case "--secret-file":
				name, target, ok := strings.Cut(value, "=")
				if !ok || name == "" || target == "" {
					return execOptions{}, errors.New("--secret-file requires NAME=/run/secrets/PATH")
				}
				options.secrets = append(options.secrets, &autbackv1.JobSecret{Name: name, Target: &autbackv1.JobSecret_File{File: target}})
			}
		default:
			return execOptions{}, errors.New("unknown exec option " + args[0])
		}
	}
	if len(args) == 0 || args[0] != "--" || len(args) == 1 {
		return execOptions{}, errors.New("exec requires -- <command> [arguments...]")
	}
	options.command = append([]string(nil), args[1:]...)
	if options.project == "" {
		return execOptions{}, errors.New("project selection is required")
	}
	return options, nil
}

func defaultExecWorkingDirectory(root, directory string) (string, error) {
	absoluteRoot, err := canonicalDirectory(root)
	if err != nil {
		return "", fmt.Errorf("resolve repository root: %w", err)
	}
	absoluteDirectory, err := canonicalDirectory(directory)
	if err != nil {
		return "", fmt.Errorf("resolve invocation directory: %w", err)
	}
	relative, err := filepath.Rel(absoluteRoot, absoluteDirectory)
	if err != nil {
		return "", fmt.Errorf("resolve invocation directory: %w", err)
	}
	if relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return "", errors.New("invocation directory is outside the Git worktree")
	}
	return filepath.ToSlash(relative), nil
}

func canonicalDirectory(path string) (string, error) {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	return filepath.EvalSymlinks(absolute)
}

func serviceImage(ctx context.Context, api autbackv1connect.ControlServiceClient, settings config.Config, project string, args []string, streams IO) int {
	if len(args) == 0 {
		return failUsage(streams.Stderr, "image requires show, activate, rollback, history, overrides, or build")
	}
	command, args := args[0], args[1:]
	project, args, err := consumeProject(project, args)
	if err != nil {
		return failUsage(streams.Stderr, err.Error())
	}
	if project == "" {
		return failUsage(streams.Stderr, "image command requires repository project selection or --project")
	}
	switch command {
	case "show":
		if len(args) != 0 {
			return failUsage(streams.Stderr, "image show accepts only --project")
		}
		item, err := authorizedProject(ctx, api, project)
		if err != nil {
			return fail(streams.Stderr, err)
		}
		return encode(streams, item)
	case "activate":
		if len(args) != 2 || args[0] != "--image" || args[1] == "" {
			return failUsage(streams.Stderr, "image activate requires --image <digest>")
		}
		response, err := api.ActivateProjectImage(ctx, connect.NewRequest(&autbackv1.ActivateProjectImageRequest{Project: project, Image: args[1]}))
		if err != nil {
			return fail(streams.Stderr, err)
		}
		fmt.Fprintf(streams.Stdout, "Activated %s for project %s\n", response.Msg.Project.ActiveImage, response.Msg.Project.Slug)
		return 0
	case "rollback":
		if len(args) != 0 {
			return failUsage(streams.Stderr, "image rollback accepts only --project")
		}
		response, err := api.RollbackProjectImage(ctx, connect.NewRequest(&autbackv1.RollbackProjectImageRequest{Project: project}))
		if err != nil {
			return fail(streams.Stderr, err)
		}
		fmt.Fprintf(streams.Stdout, "Rolled project %s back to %s\n", response.Msg.Project.Slug, response.Msg.Project.ActiveImage)
		return 0
	case "history":
		if len(args) != 0 {
			return failUsage(streams.Stderr, "image history accepts only --project")
		}
		response, err := api.ListProjectImageHistory(ctx, connect.NewRequest(&autbackv1.ListProjectImageHistoryRequest{Project: project}))
		if err != nil {
			return fail(streams.Stderr, err)
		}
		return encode(streams, response.Msg.Events)
	case "overrides":
		if len(args) != 1 || args[0] != "allow" && args[0] != "deny" {
			return failUsage(streams.Stderr, "image overrides requires allow or deny")
		}
		response, err := api.SetProjectImagePolicy(ctx, connect.NewRequest(&autbackv1.SetProjectImagePolicyRequest{Project: project, AllowImageOverrides: args[0] == "allow"}))
		if err != nil {
			return fail(streams.Stderr, err)
		}
		fmt.Fprintf(streams.Stdout, "Project %s image overrides: %t\n", response.Msg.Project.Slug, response.Msg.Project.AllowImageOverrides)
		return 0
	case "build":
		return serviceImageBuild(ctx, api, settings, project, args, streams)
	default:
		return failUsage(streams.Stderr, "unknown image command "+command)
	}
}

func serviceImageBuild(ctx context.Context, api autbackv1connect.ControlServiceClient, settings config.Config, project string, args []string, streams IO) int {
	tag, dockerfile := "", "Dockerfile"
	var extra []string
	for len(args) > 0 {
		if args[0] == "--" {
			extra, args = append([]string(nil), args[1:]...), nil
			break
		}
		if len(args) < 2 {
			return failUsage(streams.Stderr, args[0]+" requires a value")
		}
		switch args[0] {
		case "--tag":
			tag = args[1]
		case "--file":
			dockerfile = args[1]
		default:
			return failUsage(streams.Stderr, "unknown image build option "+args[0])
		}
		args = args[2:]
	}
	if tag == "" || strings.Contains(tag, "@") {
		return failUsage(streams.Stderr, "image build requires a mutable --tag <registry/repository:tag> destination")
	}
	metadata, err := os.CreateTemp("", "autback-build-metadata-*.json")
	if err != nil {
		return fail(streams.Stderr, err)
	}
	metadataPath := metadata.Name()
	_ = metadata.Close()
	defer os.Remove(metadataPath)
	buildArgs := []string{"--file", dockerfile, "--tag", tag, "--push", "--metadata-file", metadataPath}
	buildArgs = append(buildArgs, extra...)
	buildArgs = append(buildArgs, ".")
	if code := serviceBuild(ctx, api, settings, project, buildArgs, streams); code != 0 {
		return code
	}
	payload, err := os.ReadFile(metadataPath)
	if err != nil {
		return fail(streams.Stderr, fmt.Errorf("read Buildx metadata: %w", err))
	}
	var values map[string]any
	if err := json.Unmarshal(payload, &values); err != nil {
		return fail(streams.Stderr, fmt.Errorf("decode Buildx metadata: %w", err))
	}
	digest, _ := values["containerimage.digest"].(string)
	if !strings.HasPrefix(digest, "sha256:") {
		return fail(streams.Stderr, errors.New("Buildx did not report containerimage.digest"))
	}
	image := repositoryFromTag(tag) + "@" + digest
	activationAPI, err := renewServiceClient(ctx, api)
	if err != nil {
		return fail(streams.Stderr, fmt.Errorf("renew authorization before image activation: %w", err))
	}
	response, err := activationAPI.ActivateProjectImage(ctx, connect.NewRequest(&autbackv1.ActivateProjectImageRequest{Project: project, Image: image}))
	if err != nil {
		return fail(streams.Stderr, fmt.Errorf("activate built image: %w", err))
	}
	fmt.Fprintf(streams.Stdout, "Activated %s for project %s\n", response.Msg.Project.ActiveImage, response.Msg.Project.Slug)
	return 0
}

func consumeProject(project string, args []string) (string, []string, error) {
	result := make([]string, 0, len(args))
	for index := 0; index < len(args); index++ {
		if args[index] == "--" {
			result = append(result, args[index:]...)
			break
		}
		if args[index] != "--project" {
			result = append(result, args[index])
			continue
		}
		if index+1 >= len(args) || args[index+1] == "" {
			return "", nil, errors.New("--project requires a value")
		}
		if project != "" && project != args[index+1] {
			return "", nil, errors.New("conflicting --project values")
		}
		project = args[index+1]
		index++
	}
	return project, result, nil
}

func repositoryFromTag(reference string) string {
	lastSlash := strings.LastIndex(reference, "/")
	if colon := strings.LastIndex(reference, ":"); colon > lastSlash {
		return reference[:colon]
	}
	return reference
}

func waitServiceJob(ctx context.Context, api autbackv1connect.ControlServiceClient, id string, streams IO) int {
	var terminal *autbackv1.Job
	var offset int64
	retryDelay := 250 * time.Millisecond
	for terminal == nil {
		stream, err := api.StreamJobLogs(ctx, connect.NewRequest(&autbackv1.StreamJobLogsRequest{Id: id, Offset: offset}))
		if err == nil {
			for stream.Receive() {
				message := stream.Msg()
				if len(message.Data) > 0 {
					if _, err := streams.Stdout.Write(message.Data); err != nil {
						return fail(streams.Stderr, err)
					}
				}
				if message.NextOffset >= offset {
					offset = message.NextOffset
				}
				if message.TerminalJob != nil {
					terminal = message.TerminalJob
				}
			}
			err = stream.Err()
		}
		if terminal != nil {
			break
		}
		if ctx.Err() != nil {
			cancelCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			_, _ = api.CancelJob(cancelCtx, connect.NewRequest(&autbackv1.CancelJobRequest{Id: id}))
			cancel()
			return fail(streams.Stderr, fmt.Errorf("stream job logs: %w", ctx.Err()))
		}
		if err != nil && connect.CodeOf(err) != connect.CodeUnavailable {
			return fail(streams.Stderr, fmt.Errorf("stream job logs: %w", err))
		}
		timer := time.NewTimer(retryDelay)
		select {
		case <-ctx.Done():
			timer.Stop()
			continue
		case <-timer.C:
		}
		if retryDelay < 5*time.Second {
			retryDelay *= 2
			if retryDelay > 5*time.Second {
				retryDelay = 5 * time.Second
			}
		}
	}
	job := protocolJob(terminal)
	printCompletion(streams.Stderr, job)
	return clientExitCode(job)
}

func serviceBuild(ctx context.Context, api autbackv1connect.ControlServiceClient, settings config.Config, project string, args []string, streams IO) int {
	if len(args) >= 2 && args[0] == "--project" {
		project, args = args[1], args[2:]
	}
	if len(args) > 0 && args[0] == "--" {
		args = args[1:]
	}
	if project == "" {
		return failUsage(streams.Stderr, "build requires repository project selection or --project")
	}
	if _, err := authorizedProject(ctx, api, project); err != nil {
		return fail(streams.Stderr, err)
	}
	if len(args) == 0 {
		args = []string{"."}
	}
	if _, err := workspace.Root(ctx, streams.Dir); err != nil {
		return fail(streams.Stderr, err)
	}
	random, err := jobID()
	if err != nil {
		return fail(streams.Stderr, err)
	}
	prepared, err := api.PrepareBuild(ctx, connect.NewRequest(&autbackv1.PrepareBuildRequest{Project: project, IdempotencyKey: random}))
	if err != nil {
		return fail(streams.Stderr, fmt.Errorf("prepare remote build: %w", err))
	}
	fmt.Fprintf(streams.Stderr, "Backend: native Buildx via autback mTLS\nBuild: %s\n", prepared.Msg.Build.Id)
	build, connection, err := waitForServiceBuild(ctx, api, prepared.Msg.Build, prepared.Msg.Buildkit)
	if err != nil {
		return fail(streams.Stderr, cancelServiceBuildAfterError(api, prepared.Msg.Build.Id, err))
	}
	if build.Status != autbackv1.BuildStatus_BUILD_STATUS_RUNNING || connection == nil {
		return fail(streams.Stderr, cancelServiceBuildAfterError(api, prepared.Msg.Build.Id, errors.New("remote build was admitted without a BuildKit connection")))
	}
	credentials, err := credentialfiles.Write(connection.CaPem, connection.CertificatePem, connection.PrivateKeyPem)
	if err != nil {
		return fail(streams.Stderr, cancelServiceBuildAfterError(api, prepared.Msg.Build.Id, err))
	}
	defer credentials.Cleanup()
	builderName := strings.Replace(random, "autback-", "autback-build-", 1)
	address := connection.Endpoint
	if !strings.Contains(address, "://") {
		address = "tcp://" + address
	}
	fmt.Fprintf(streams.Stderr, "Builder: %s\n", address)
	buildCtx, stopBuild := context.WithCancel(ctx)
	heartbeatCtx, stopHeartbeat := context.WithCancel(buildCtx)
	heartbeatDone := make(chan error, 1)
	go func() {
		err := heartbeatServiceBuild(heartbeatCtx, api, prepared.Msg.Build.Id, 30*time.Second)
		if err != nil {
			stopBuild()
		}
		heartbeatDone <- err
	}()
	code, runErr := buildkit.RunWithTLS(buildCtx, os.Getenv("AUTBACK_DOCKER"), address, builderName, streams.Dir, args, buildkit.TLS{
		CA: credentials.CA, Certificate: credentials.Certificate, Key: credentials.Key, ServerName: connection.ServerName,
	}, streams.Stdout, streams.Stderr)
	stopHeartbeat()
	heartbeatErr := <-heartbeatDone
	stopBuild()
	cancelled := ctx.Err() != nil || heartbeatErr != nil
	finishCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	var finishErr error
	if cancelled {
		finishErr = cancelServiceBuildRecord(finishCtx, api, prepared.Msg.Build.Id)
	} else {
		finishErr = finishServiceBuildRecord(finishCtx, api, prepared.Msg.Build.Id, int32(code), false)
	}
	cancel()
	if heartbeatErr != nil {
		return fail(streams.Stderr, errors.Join(fmt.Errorf("renew remote build lease: %w", heartbeatErr), runErr, finishErr))
	}
	if runErr != nil {
		return fail(streams.Stderr, errors.Join(fmt.Errorf("remote build: %w", runErr), finishErr))
	}
	if finishErr != nil {
		return fail(streams.Stderr, fmt.Errorf("finish build record: %w", finishErr))
	}
	return code
}

func cancelServiceBuildAfterError(api autbackv1connect.ControlServiceClient, id string, cause error) error {
	cancelCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := cancelServiceBuildRecord(cancelCtx, api, id); err != nil {
		return errors.Join(cause, fmt.Errorf("cancel abandoned build %s: %w", id, err))
	}
	return cause
}

func heartbeatServiceBuild(ctx context.Context, api autbackv1connect.ControlServiceClient, id string, interval time.Duration) error {
	if interval <= 0 {
		return errors.New("build heartbeat interval must be positive")
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
		}
		response, err := api.GetBuild(ctx, connect.NewRequest(&autbackv1.GetBuildRequest{Id: id}))
		if connect.CodeOf(err) == connect.CodeUnauthenticated {
			api, err = renewServiceClient(ctx, api)
			if err == nil {
				response, err = api.GetBuild(ctx, connect.NewRequest(&autbackv1.GetBuildRequest{Id: id}))
			}
		}
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			return err
		}
		if response.Msg.Build == nil || response.Msg.Build.Status != autbackv1.BuildStatus_BUILD_STATUS_RUNNING {
			return errors.New("remote build lease is no longer running")
		}
	}
}

func waitForServiceBuild(ctx context.Context, api autbackv1connect.ControlServiceClient, build *autbackv1.Build, connection *autbackv1.DataPlaneConnection) (*autbackv1.Build, *autbackv1.DataPlaneConnection, error) {
	if build == nil {
		return nil, nil, errors.New("prepare build returned no build")
	}
	for build.Status == autbackv1.BuildStatus_BUILD_STATUS_QUEUED {
		timer := time.NewTimer(250 * time.Millisecond)
		select {
		case <-ctx.Done():
			timer.Stop()
			return nil, nil, fmt.Errorf("wait for remote build: %w", ctx.Err())
		case <-timer.C:
		}
		response, err := api.GetBuild(ctx, connect.NewRequest(&autbackv1.GetBuildRequest{Id: build.Id}))
		if connect.CodeOf(err) == connect.CodeUnauthenticated {
			renewed, renewErr := renewServiceClient(ctx, api)
			if renewErr != nil {
				return nil, nil, renewErr
			}
			api = renewed
			response, err = renewed.GetBuild(ctx, connect.NewRequest(&autbackv1.GetBuildRequest{Id: build.Id}))
		}
		if err != nil {
			return nil, nil, fmt.Errorf("get remote build: %w", err)
		}
		build, connection = response.Msg.Build, response.Msg.Buildkit
	}
	if build.Status != autbackv1.BuildStatus_BUILD_STATUS_RUNNING {
		return build, connection, fmt.Errorf("remote build became %s before admission", build.Status)
	}
	return build, connection, nil
}

func finishServiceBuildRecord(ctx context.Context, api autbackv1connect.ControlServiceClient, id string, exitCode int32, cancelled bool) error {
	finishAPI, err := renewServiceClient(ctx, api)
	if err != nil {
		return fmt.Errorf("renew authorization: %w", err)
	}
	_, err = finishAPI.FinishBuild(ctx, connect.NewRequest(&autbackv1.FinishBuildRequest{Id: id, ExitCode: exitCode, Cancelled: cancelled}))
	return err
}

func cancelServiceBuildRecord(ctx context.Context, api autbackv1connect.ControlServiceClient, id string) error {
	cancelAPI, err := renewServiceClient(ctx, api)
	if err != nil {
		return fmt.Errorf("renew authorization: %w", err)
	}
	_, err = cancelAPI.CancelBuild(ctx, connect.NewRequest(&autbackv1.CancelBuildRequest{Id: id}))
	return err
}

func serviceStatus(ctx context.Context, api autbackv1connect.ControlServiceClient, args []string, streams IO) int {
	jsonOutput, id, err := jobArgs(args)
	if err != nil {
		return failUsage(streams.Stderr, "status "+err.Error())
	}
	response, err := api.GetJob(ctx, connect.NewRequest(&autbackv1.GetJobRequest{Id: id}))
	if err != nil {
		return fail(streams.Stderr, err)
	}
	job := protocolJob(response.Msg.Job)
	if jsonOutput {
		return encode(streams, job)
	}
	printJob(streams.Stdout, job)
	return 0
}

func serviceLogs(ctx context.Context, api autbackv1connect.ControlServiceClient, args []string, streams IO) int {
	if len(args) != 1 {
		return failUsage(streams.Stderr, "logs requires exactly one job ID")
	}
	return waitServiceJob(ctx, api, args[0], streams)
}

func serviceCancel(ctx context.Context, api autbackv1connect.ControlServiceClient, args []string, streams IO) int {
	if len(args) != 1 {
		return failUsage(streams.Stderr, "cancel requires exactly one job ID")
	}
	response, err := api.CancelJob(ctx, connect.NewRequest(&autbackv1.CancelJobRequest{Id: args[0]}))
	if err != nil {
		return fail(streams.Stderr, err)
	}
	fmt.Fprintf(streams.Stdout, "Job %s: %s\n", args[0], strings.ToLower(strings.TrimPrefix(response.Msg.Job.Status.String(), "JOB_STATUS_")))
	return 0
}

func serviceList(ctx context.Context, api autbackv1connect.ControlServiceClient, project string, args []string, streams IO) int {
	limit, jsonOutput := 20, false
	for len(args) > 0 {
		switch args[0] {
		case "--project":
			if len(args) < 2 {
				return failUsage(streams.Stderr, "--project requires a value")
			}
			project, args = args[1], args[2:]
		case "--limit":
			if len(args) < 2 {
				return failUsage(streams.Stderr, "--limit requires a value")
			}
			value, err := strconv.Atoi(args[1])
			if err != nil || value < 1 || value > 100 {
				return failUsage(streams.Stderr, "--limit must be between 1 and 100")
			}
			limit, args = value, args[2:]
		case "--json":
			jsonOutput, args = true, args[1:]
		default:
			return failUsage(streams.Stderr, "unknown list option "+args[0])
		}
	}
	if project == "" {
		return failUsage(streams.Stderr, "list requires a project")
	}
	response, err := api.ListJobs(ctx, connect.NewRequest(&autbackv1.ListJobsRequest{Project: project, PageSize: int32(limit)}))
	if err != nil {
		return fail(streams.Stderr, err)
	}
	jobs := make([]protocol.Job, 0, len(response.Msg.Jobs))
	for _, item := range response.Msg.Jobs {
		jobs = append(jobs, protocolJob(item))
	}
	if jsonOutput {
		return encode(streams, jobs)
	}
	writer := tabwriter.NewWriter(streams.Stdout, 0, 4, 2, ' ', 0)
	fmt.Fprintln(writer, "JOB\tSTATUS\tPROJECT\tAGE\tDURATION")
	for _, job := range jobs {
		fmt.Fprintf(writer, "%s\t%s\t%s\t%s\t%s\n", job.ID, job.Status, job.ProjectID, time.Since(job.CreatedAt).Round(time.Second), duration(job))
	}
	_ = writer.Flush()
	return 0
}

func serviceDoctor(ctx context.Context, api autbackv1connect.ControlServiceClient, streams IO) int {
	info, err := api.GetServiceInfo(ctx, connect.NewRequest(&autbackv1.GetServiceInfoRequest{}))
	if err != nil {
		return fail(streams.Stderr, err)
	}
	fmt.Fprintf(streams.Stdout, "autback %s\nconnection: ok (Connect over HTTPS)\nserver: %s\n", currentVersion, info.Msg.Version)
	return 0
}

func serviceToken(ctx context.Context, api autbackv1connect.ControlServiceClient, args []string, streams IO) int {
	if len(args) == 0 {
		return failUsage(streams.Stderr, "token requires create, list, or revoke")
	}
	switch args[0] {
	case "create":
		name, userID := "", ""
		var expires *timestamppb.Timestamp
		for args = args[1:]; len(args) > 0; {
			if len(args) < 2 {
				return failUsage(streams.Stderr, args[0]+" requires a value")
			}
			switch args[0] {
			case "--name":
				name = args[1]
			case "--user":
				userID = args[1]
			case "--expires":
				duration, err := time.ParseDuration(args[1])
				if err != nil || duration < time.Minute {
					return failUsage(streams.Stderr, "invalid token expiry")
				}
				expires = timestamppb.New(time.Now().Add(duration))
			default:
				return failUsage(streams.Stderr, "unknown token create option "+args[0])
			}
			args = args[2:]
		}
		if name == "" {
			return failUsage(streams.Stderr, "token create requires --name")
		}
		response, err := api.CreateDeviceToken(ctx, connect.NewRequest(&autbackv1.CreateDeviceTokenRequest{Name: name, UserId: userID, ExpiresAt: expires}))
		if err != nil {
			return fail(streams.Stderr, err)
		}
		fmt.Fprintf(streams.Stdout, "%s\n", response.Msg.Token)
		fmt.Fprintf(streams.Stderr, "Created device token %s; its secret will not be shown again.\n", response.Msg.TokenMetadata.Id)
		return 0
	case "list":
		if len(args) != 1 {
			return failUsage(streams.Stderr, "token list accepts no arguments")
		}
		response, err := api.ListDeviceTokens(ctx, connect.NewRequest(&autbackv1.ListDeviceTokensRequest{}))
		if err != nil {
			return fail(streams.Stderr, err)
		}
		return encode(streams, response.Msg.Tokens)
	case "revoke":
		if len(args) != 2 {
			return failUsage(streams.Stderr, "token revoke requires one token ID")
		}
		if _, err := api.RevokeDeviceToken(ctx, connect.NewRequest(&autbackv1.RevokeDeviceTokenRequest{Id: args[1]})); err != nil {
			return fail(streams.Stderr, err)
		}
		fmt.Fprintln(streams.Stdout, "Revoked device token "+args[1])
		return 0
	default:
		return failUsage(streams.Stderr, "unknown token command "+args[0])
	}
}

func serviceTrust(ctx context.Context, api autbackv1connect.ControlServiceClient, args []string, streams IO) int {
	if len(args) < 2 || args[0] != "github" {
		return failUsage(streams.Stderr, "trust requires github create, list, or revoke")
	}
	command, args := args[1], args[2:]
	switch command {
	case "create":
		values := map[string]string{}
		var events []string
		for len(args) > 0 {
			if len(args) < 2 {
				return failUsage(streams.Stderr, args[0]+" requires a value")
			}
			if args[0] == "--event" {
				events = append(events, args[1])
			} else {
				values[args[0]] = args[1]
			}
			args = args[2:]
		}
		for _, required := range []string{"--project", "--owner-id", "--repository-id", "--workflow-ref", "--ref"} {
			if values[required] == "" {
				return failUsage(streams.Stderr, "trust github create requires "+required)
			}
		}
		if len(events) == 0 {
			return failUsage(streams.Stderr, "trust github create requires at least one --event")
		}
		response, err := api.CreateGitHubTrust(ctx, connect.NewRequest(&autbackv1.CreateGitHubTrustRequest{
			Project: values["--project"], RepositoryOwnerId: values["--owner-id"], RepositoryId: values["--repository-id"],
			WorkflowRef: values["--workflow-ref"], Ref: values["--ref"], Environment: values["--environment"], Events: events,
		}))
		if err != nil {
			return fail(streams.Stderr, err)
		}
		return encode(streams, response.Msg.Trust)
	case "list":
		project := ""
		if len(args) == 2 && args[0] == "--project" {
			project = args[1]
		} else if len(args) != 0 {
			return failUsage(streams.Stderr, "trust github list accepts only --project")
		}
		response, err := api.ListGitHubTrusts(ctx, connect.NewRequest(&autbackv1.ListGitHubTrustsRequest{Project: project}))
		if err != nil {
			return fail(streams.Stderr, err)
		}
		return encode(streams, response.Msg.Trusts)
	case "revoke":
		if len(args) != 1 {
			return failUsage(streams.Stderr, "trust github revoke requires one trust ID")
		}
		if _, err := api.RevokeGitHubTrust(ctx, connect.NewRequest(&autbackv1.RevokeGitHubTrustRequest{Id: args[0]})); err != nil {
			return fail(streams.Stderr, err)
		}
		fmt.Fprintln(streams.Stdout, "Revoked GitHub trust "+args[0])
		return 0
	default:
		return failUsage(streams.Stderr, "unknown GitHub trust command "+command)
	}
}

func grpcAddress(endpoint string) string {
	return strings.TrimPrefix(strings.TrimPrefix(endpoint, "https://"), "grpcs://")
}

func protocolJob(input *autbackv1.Job) protocol.Job {
	job := protocol.Job{
		ID: input.Id, ProjectID: input.ProjectId, Image: input.Image, Command: append([]string(nil), input.Command...),
		RootDigest: input.RootDigest, Status: protocolStatus(input.Status),
		CancelRequested: input.CancelRequested, ErrorMessage: input.ErrorMessage, WorkerID: input.WorkerId,
	}
	if input.Timeout != nil {
		job.TimeoutSeconds = int(input.Timeout.AsDuration().Seconds())
	}
	if input.CreatedAt != nil {
		job.CreatedAt = input.CreatedAt.AsTime()
	}
	job.StartedAt, job.FinishedAt = protoTime(input.StartedAt), protoTime(input.FinishedAt)
	if input.ExitCode != nil {
		value := int(*input.ExitCode)
		job.ExitCode = &value
	}
	return job
}

func protocolStatus(status autbackv1.JobStatus) protocol.Status {
	switch status {
	case autbackv1.JobStatus_JOB_STATUS_PREPARING:
		return protocol.StatusPreparing
	case autbackv1.JobStatus_JOB_STATUS_QUEUED:
		return protocol.StatusQueued
	case autbackv1.JobStatus_JOB_STATUS_RUNNING:
		return protocol.StatusRunning
	case autbackv1.JobStatus_JOB_STATUS_SUCCEEDED:
		return protocol.StatusSucceeded
	case autbackv1.JobStatus_JOB_STATUS_FAILED:
		return protocol.StatusFailed
	case autbackv1.JobStatus_JOB_STATUS_CANCELLED:
		return protocol.StatusCancelled
	case autbackv1.JobStatus_JOB_STATUS_TIMED_OUT:
		return protocol.StatusTimedOut
	case autbackv1.JobStatus_JOB_STATUS_LOST:
		return protocol.StatusLost
	default:
		return protocol.StatusLost
	}
}

func protoTime(value *timestamppb.Timestamp) *time.Time {
	if value == nil {
		return nil
	}
	result := value.AsTime()
	return &result
}

func clientExitCode(job protocol.Job) int {
	if job.ExitCode != nil {
		return *job.ExitCode
	}
	if job.Status == protocol.StatusSucceeded {
		return 0
	}
	return 1
}
