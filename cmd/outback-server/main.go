package main

import (
	"context"
	"crypto/tls"
	"errors"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/flidai/outback/internal/control"
	"github.com/flidai/outback/internal/control/controlapi"
	"github.com/flidai/outback/internal/control/dispatcher"
	"github.com/flidai/outback/internal/control/githuboidc"
	"github.com/flidai/outback/internal/control/mtlsproxy"
	"github.com/flidai/outback/internal/control/pki"
	"github.com/flidai/outback/internal/control/reconciler"
	"github.com/flidai/outback/internal/control/recovery"
	"github.com/flidai/outback/internal/control/secret"
	controlsqlite "github.com/flidai/outback/internal/control/sqlite"
	"github.com/flidai/outback/internal/control/swarmscheduler"
	"github.com/flidai/outback/internal/swarm"
)

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "bootstrap":
			bootstrap(os.Args[2:])
		case "backup":
			backupState(os.Args[2:])
		case "restore":
			restoreState(os.Args[2:])
		default:
			log.Fatalf("unknown command %q", os.Args[1])
		}
		return
	}
	serve()
}

func serve() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	dataDir := env("OUTBACK_DATA_DIR", "/var/lib/outback")
	store := openStore(dataDir)
	defer store.Close()
	initialized, err := store.Initialized(ctx)
	if err != nil {
		log.Fatal(err)
	}
	if !initialized {
		log.Fatal("outback is not bootstrapped; run outback-server bootstrap before starting the service")
	}
	names := splitNames(env("OUTBACK_SERVER_NAMES", "localhost,127.0.0.1"))
	pkiDir := env("OUTBACK_PKI_DIR", filepath.Join(dataDir, "pki"))
	authority, err := pki.Ensure(pkiDir, names)
	if err != nil {
		log.Fatal(err)
	}
	docker := swarm.New(swarm.Config{Binary: os.Getenv("OUTBACK_DOCKER"), Host: env("OUTBACK_DOCKER_HOST", "unix:///var/run/docker.sock")})
	if err := docker.Check(ctx); err != nil {
		log.Fatal(err)
	}
	casInternal := env("OUTBACK_CAS_INTERNAL", "127.0.0.1:50051")
	casListen := env("OUTBACK_CAS_LISTEN", ":50052")
	buildKitInternal := env("OUTBACK_BUILDKIT_INTERNAL", "127.0.0.1:1234")
	buildKitListen := env("OUTBACK_BUILDKIT_LISTEN", ":1235")
	casInstance := env("OUTBACK_CAS_INSTANCE", "outback")
	serverName := names[0]
	scheduler := swarmscheduler.New(swarmscheduler.Config{
		Client: docker, CASAddress: casInternal, CASInstance: casInstance, JobsRoot: env("OUTBACK_JOBS_ROOT", "/var/lib/outback/jobs"),
		EntrypointHostPath: env("OUTBACK_JOB_ENTRYPOINT", "/usr/local/lib/outback/outback-job-entrypoint"),
		CacheRoot:          env("OUTBACK_CACHE_ROOT", "/var/lib/outback/cache"),
		HostUID:            strconv.Itoa(os.Getuid()), HostGID: strconv.Itoa(os.Getgid()),
	})
	dispatch := dispatcher.New(store, scheduler)
	if err := dispatch.RunOnce(ctx); err != nil {
		log.Printf("initial dispatch: %v", err)
	}
	reconcile := reconciler.New(reconciler.Config{
		Store: store, Scheduler: scheduler, Dispatcher: dispatch,
		ServiceRetention:  durationEnv("OUTBACK_SERVICE_RETENTION", time.Hour),
		AdmissionGrace:    durationEnv("OUTBACK_ADMISSION_GRACE", 15*time.Second),
		BuildLeaseTimeout: durationEnv("OUTBACK_BUILD_LEASE_TIMEOUT", 2*time.Hour),
	})
	go runReconciler(ctx, reconcile, durationEnv("OUTBACK_RECONCILE_INTERVAL", time.Second))
	var verifier controlapi.OIDCVerifier
	if audience := os.Getenv("OUTBACK_GITHUB_OIDC_AUDIENCE"); audience != "" {
		verifier, err = githuboidc.New(ctx, env("OUTBACK_GITHUB_OIDC_ISSUER", githuboidc.Issuer), audience)
		if err != nil {
			log.Fatal(err)
		}
	}
	handler, err := controlapi.New(controlapi.Config{
		Store: store, Scheduler: scheduler, Dispatcher: dispatch, Authority: authority, OIDCVerifier: verifier,
		CASEndpoint: env("OUTBACK_CAS_ENDPOINT", endpoint(serverName, casListen)), CASInstance: casInstance,
		BuildKitEndpoint:    env("OUTBACK_BUILDKIT_ENDPOINT", endpoint(serverName, buildKitListen)),
		CredentialTTL:       durationEnv("OUTBACK_CREDENTIAL_TTL", 15*time.Minute),
		AllowUnpinnedImages: os.Getenv("OUTBACK_ALLOW_UNPINNED_IMAGES") == "1",
	})
	if err != nil {
		log.Fatal(err)
	}
	errorsChannel := make(chan error, 3)
	active := func(kind pki.Operation, id string) bool {
		return store.OperationActive(context.Background(), string(kind), id)
	}
	go func() {
		errorsChannel <- mtlsproxy.ListenAndServe(ctx, casListen, casInternal, authority.ServerTLSConfig(pki.OperationJob, active))
	}()
	go func() {
		errorsChannel <- mtlsproxy.ListenAndServe(ctx, buildKitListen, buildKitInternal, authority.ServerTLSConfig(pki.OperationBuild, active))
	}()
	server := &http.Server{
		Addr: env("OUTBACK_LISTEN", ":8443"), Handler: handler, ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout: 2 * time.Minute, TLSConfig: &tls.Config{MinVersion: tls.VersionTLS13},
	}
	go func() {
		errorsChannel <- server.ListenAndServeTLS(filepath.Join(pkiDir, "server.pem"), filepath.Join(pkiDir, "server-key.pem"))
	}()
	log.Printf("outback control plane listening on %s; CAS mTLS on %s; BuildKit mTLS on %s", server.Addr, casListen, buildKitListen)
	select {
	case err := <-errorsChannel:
		if err != nil && !errors.Is(err, http.ErrServerClosed) && ctx.Err() == nil {
			log.Fatal(err)
		}
	case <-ctx.Done():
	}
	shutdown, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdown); err != nil {
		log.Printf("shutdown: %v", err)
	}
}

type reconciliationRunner interface {
	RunOnce(context.Context) error
}

func runReconciler(ctx context.Context, runner reconciliationRunner, interval time.Duration) {
	if err := runner.RunOnce(ctx); err != nil && ctx.Err() == nil {
		log.Printf("initial reconciliation: %v", err)
	}
	if interval <= 0 {
		interval = time.Second
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := runner.RunOnce(ctx); err != nil && ctx.Err() == nil {
				log.Printf("reconciliation: %v", err)
			}
		}
	}
}

func bootstrap(args []string) {
	flags := flag.NewFlagSet("outback-server bootstrap", flag.ExitOnError)
	dataDir := flags.String("data-dir", env("OUTBACK_DATA_DIR", "/var/lib/outback"), "persistent outback data directory")
	user := flags.String("user", "owner", "initial administrator name")
	project := flags.String("project", "default", "initial project slug")
	projectName := flags.String("project-name", "", "initial project display name")
	tokenName := flags.String("token-name", "bootstrap-device", "initial device token name")
	_ = flags.Parse(args)
	store := openStore(*dataDir)
	defer store.Close()
	result, err := store.Bootstrap(context.Background(), control.Bootstrap{
		UserName: *user, ProjectSlug: *project, ProjectName: *projectName, TokenName: *tokenName,
	})
	if err != nil {
		if errors.Is(err, control.ErrAlreadyExists) {
			log.Fatal("outback is already bootstrapped; create another device token through the authenticated API")
		}
		log.Fatal(err)
	}
	fmt.Printf("Project: %s (%s)\nUser: %s (%s)\nToken: %s\n", result.Project.Slug, result.Project.ID, result.User.Name, result.User.ID, result.Token)
}

func backupState(args []string) {
	flags := flag.NewFlagSet("outback-server backup", flag.ExitOnError)
	dataDir := flags.String("data-dir", env("OUTBACK_DATA_DIR", "/var/lib/outback"), "persistent outback data directory")
	output := flags.String("output", "", "new private backup directory")
	_ = flags.Parse(args)
	if *output == "" {
		log.Fatal("--output is required")
	}
	store := openStore(*dataDir)
	defer store.Close()
	if err := recovery.Create(context.Background(), store, *dataDir, *output); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Backup: %s\n", *output)
}

func restoreState(args []string) {
	flags := flag.NewFlagSet("outback-server restore", flag.ExitOnError)
	input := flags.String("input", "", "validated outback backup directory")
	dataDir := flags.String("data-dir", env("OUTBACK_DATA_DIR", "/var/lib/outback"), "new persistent outback data directory")
	_ = flags.Parse(args)
	if *input == "" {
		log.Fatal("--input is required")
	}
	if err := recovery.Restore(*input, *dataDir); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Restored: %s\n", *dataDir)
}

func openStore(dataDir string) *controlsqlite.Store {
	pepper, err := secret.Ensure(filepath.Join(dataDir, "control", "token-pepper"), 32)
	if err != nil {
		log.Fatal(err)
	}
	store, err := controlsqlite.Open(filepath.Join(dataDir, "control"), pepper)
	if err != nil {
		log.Fatal(err)
	}
	return store
}

func splitNames(value string) []string {
	var names []string
	for _, item := range strings.Split(value, ",") {
		if name := strings.TrimSpace(item); name != "" {
			names = append(names, name)
		}
	}
	if len(names) == 0 {
		log.Fatal("OUTBACK_SERVER_NAMES must contain at least one DNS name or IP address")
	}
	return names
}

func endpoint(serverName, listen string) string {
	_, port, err := net.SplitHostPort(listen)
	if err != nil || port == "" {
		return listen
	}
	return net.JoinHostPort(serverName, port)
}

func durationEnv(name string, fallback time.Duration) time.Duration {
	value := os.Getenv(name)
	if value == "" {
		return fallback
	}
	parsed, err := time.ParseDuration(value)
	if err != nil {
		log.Fatalf("%s: %v", name, err)
	}
	return parsed
}

func env(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}
