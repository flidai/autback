package main

import (
	"context"
	"crypto/tls"
	"encoding/json"
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

	"github.com/flidai/autback/internal/capacity"
	"github.com/flidai/autback/internal/console"
	"github.com/flidai/autback/internal/control"
	"github.com/flidai/autback/internal/control/controlapi"
	"github.com/flidai/autback/internal/control/dispatcher"
	"github.com/flidai/autback/internal/control/githuboidc"
	"github.com/flidai/autback/internal/control/mtlsproxy"
	"github.com/flidai/autback/internal/control/pki"
	"github.com/flidai/autback/internal/control/reconciler"
	"github.com/flidai/autback/internal/control/recovery"
	"github.com/flidai/autback/internal/control/secret"
	controlsqlite "github.com/flidai/autback/internal/control/sqlite"
	"github.com/flidai/autback/internal/control/swarmscheduler"
	"github.com/flidai/autback/internal/hostmetrics"
	"github.com/flidai/autback/internal/swarm"
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
		case "maintain":
			maintainWorker(os.Args[2:])
		default:
			log.Fatalf("unknown command %q", os.Args[1])
		}
		return
	}
	serve()
}

func serve() {
	startedAt := time.Now().UTC()
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	dataDir := env("AUTBACK_DATA_DIR", "/var/lib/autback")
	store := openStore(dataDir)
	defer store.Close()
	initialized, err := store.Initialized(ctx)
	if err != nil {
		log.Fatal(err)
	}
	if !initialized {
		log.Fatal("autback is not bootstrapped; run autback-server bootstrap before starting the service")
	}
	names := splitNames(env("AUTBACK_SERVER_NAMES", "localhost,127.0.0.1"))
	pkiDir := env("AUTBACK_PKI_DIR", filepath.Join(dataDir, "pki"))
	authority, err := pki.Ensure(pkiDir, names)
	if err != nil {
		log.Fatal(err)
	}
	docker := swarm.New(swarm.Config{Binary: os.Getenv("AUTBACK_DOCKER"), Host: env("AUTBACK_DOCKER_HOST", "unix:///var/run/docker.sock")})
	if err := docker.Check(ctx); err != nil {
		log.Fatal(err)
	}
	casInternal := env("AUTBACK_CAS_INTERNAL", "127.0.0.1:50051")
	casListen := env("AUTBACK_CAS_LISTEN", ":50052")
	buildKitInternal := env("AUTBACK_BUILDKIT_INTERNAL", "127.0.0.1:1234")
	buildKitListen := env("AUTBACK_BUILDKIT_LISTEN", ":1235")
	casInstance := env("AUTBACK_CAS_INSTANCE", "autback")
	serverName := names[0]
	scheduler := swarmscheduler.New(swarmscheduler.Config{
		Client: docker, CASAddress: casInternal, CASInstance: casInstance, JobsRoot: env("AUTBACK_JOBS_ROOT", "/var/lib/autback/jobs"),
		EntrypointHostPath: env("AUTBACK_JOB_ENTRYPOINT", "/usr/local/lib/autback/autback-job-entrypoint"),
		CacheRoot:          env("AUTBACK_CACHE_ROOT", "/var/lib/autback/cache"),
		HostUID:            strconv.Itoa(os.Getuid()), HostGID: strconv.Itoa(os.Getgid()),
	})
	capacityController := newCapacityController(dataDir, store, scheduler, false)
	status, capacityErr := capacityController.Maintain(ctx, capacity.TriggerManual)
	writeCapacityStatus(filepath.Join(dataDir, "capacity.json"), status)
	if capacityErr != nil {
		log.Printf("initial capacity reconciliation: %v", capacityErr)
	}
	sampler, err := hostmetrics.NewLinuxSampler(hostmetrics.LinuxSamplerConfig{DiskPath: dataDir})
	if err != nil {
		log.Fatal(err)
	}
	resourceCollector, err := hostmetrics.NewCollector(hostmetrics.CollectorConfig{
		Store: store, Sampler: sampler,
		Interval:        durationEnv("AUTBACK_METRICS_INTERVAL", 2*time.Second),
		RawRetention:    durationEnv("AUTBACK_METRICS_RAW_RETENTION", 14*24*time.Hour),
		RollupRetention: durationEnv("AUTBACK_METRICS_ROLLUP_RETENTION", 180*24*time.Hour),
		OnError:         func(err error) { log.Printf("resource metrics: %v", err) },
	})
	if err != nil {
		log.Fatal(err)
	}
	go resourceCollector.Run(ctx)
	dispatch := dispatcher.New(store, scheduler, dispatcher.WithCapacity(capacityController))
	if err := dispatch.RunOnce(ctx); err != nil {
		log.Printf("initial dispatch: %v", err)
	}
	reconcile := reconciler.New(reconciler.Config{
		Store: store, Scheduler: scheduler, Dispatcher: dispatch,
		ServiceRetention:  durationEnv("AUTBACK_SERVICE_RETENTION", time.Hour),
		AdmissionGrace:    durationEnv("AUTBACK_ADMISSION_GRACE", 15*time.Second),
		BuildLeaseTimeout: durationEnv("AUTBACK_BUILD_LEASE_TIMEOUT", 2*time.Minute),
	})
	go runReconciler(ctx, reconcile, durationEnv("AUTBACK_RECONCILE_INTERVAL", time.Second))
	go runCapacityController(ctx, capacityController, filepath.Join(dataDir, "capacity.json"), durationEnv("AUTBACK_CAPACITY_CHECK_INTERVAL", 5*time.Second), durationEnv("AUTBACK_MAINTENANCE_INTERVAL", time.Minute))
	var verifier controlapi.OIDCVerifier
	if audience := os.Getenv("AUTBACK_GITHUB_OIDC_AUDIENCE"); audience != "" {
		verifier, err = githuboidc.New(ctx, env("AUTBACK_GITHUB_OIDC_ISSUER", githuboidc.Issuer), audience)
		if err != nil {
			log.Fatal(err)
		}
	}
	controlHandler, err := controlapi.New(controlapi.Config{
		Store: store, Scheduler: scheduler, Dispatcher: dispatch, Authority: authority, OIDCVerifier: verifier,
		CASEndpoint: env("AUTBACK_CAS_ENDPOINT", endpoint(serverName, casListen)), CASInstance: casInstance,
		BuildKitEndpoint:    env("AUTBACK_BUILDKIT_ENDPOINT", endpoint(serverName, buildKitListen)),
		CredentialTTL:       durationEnv("AUTBACK_CREDENTIAL_TTL", 15*time.Minute),
		AllowUnpinnedImages: os.Getenv("AUTBACK_ALLOW_UNPINNED_IMAGES") == "1",
		Capacity:            capacityController,
	})
	if err != nil {
		log.Fatal(err)
	}
	consoleSource, err := console.NewSQLiteSource(console.SQLiteSourceConfig{
		Store: store, Scheduler: scheduler, Version: controlapi.Version, StartedAt: startedAt,
	})
	if err != nil {
		log.Fatal(err)
	}
	consoleHandler, err := console.New(console.Config{Source: consoleSource})
	if err != nil {
		log.Fatal(err)
	}
	handler := serviceHandler(controlHandler, consoleHandler)
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
		Addr: env("AUTBACK_LISTEN", ":8443"), Handler: handler, ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout: 2 * time.Minute, TLSConfig: &tls.Config{MinVersion: tls.VersionTLS13},
	}
	go func() {
		errorsChannel <- server.ListenAndServeTLS(filepath.Join(pkiDir, "server.pem"), filepath.Join(pkiDir, "server-key.pem"))
	}()
	log.Printf("autback control plane listening on %s; CAS mTLS on %s; BuildKit mTLS on %s", server.Addr, casListen, buildKitListen)
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

func runCapacityController(ctx context.Context, controller *capacity.Controller, statusPath string, pressureInterval, maintenanceInterval time.Duration) {
	if pressureInterval <= 0 {
		pressureInterval = 5 * time.Second
	}
	if maintenanceInterval <= 0 {
		maintenanceInterval = time.Minute
	}
	pressure := time.NewTicker(pressureInterval)
	maintenance := time.NewTicker(maintenanceInterval)
	defer pressure.Stop()
	defer maintenance.Stop()
	for {
		var trigger capacity.Trigger
		select {
		case <-ctx.Done():
			return
		case <-pressure.C:
			trigger = capacity.TriggerPressure
		case <-maintenance.C:
			trigger = capacity.TriggerPeriodic
		}
		status, err := controller.Maintain(ctx, trigger)
		writeCapacityStatus(statusPath, status)
		if err != nil && ctx.Err() == nil {
			log.Printf("capacity %s: %v", trigger, err)
		}
	}
}

func serviceHandler(controlHandler, consoleHandler http.Handler) http.Handler {
	mux := http.NewServeMux()
	mux.Handle("/app", consoleHandler)
	mux.Handle("/app/", consoleHandler)
	mux.Handle("/", controlHandler)
	return mux
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
	flags := flag.NewFlagSet("autback-server bootstrap", flag.ExitOnError)
	dataDir := flags.String("data-dir", env("AUTBACK_DATA_DIR", "/var/lib/autback"), "persistent autback data directory")
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
			log.Fatal("autback is already bootstrapped; create another device token through the authenticated API")
		}
		log.Fatal(err)
	}
	fmt.Printf("Project: %s (%s)\nUser: %s (%s)\nToken: %s\n", result.Project.Slug, result.Project.ID, result.User.Name, result.User.ID, result.Token)
}

func backupState(args []string) {
	flags := flag.NewFlagSet("autback-server backup", flag.ExitOnError)
	dataDir := flags.String("data-dir", env("AUTBACK_DATA_DIR", "/var/lib/autback"), "persistent autback data directory")
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
	flags := flag.NewFlagSet("autback-server restore", flag.ExitOnError)
	input := flags.String("input", "", "validated autback backup directory")
	dataDir := flags.String("data-dir", env("AUTBACK_DATA_DIR", "/var/lib/autback"), "new persistent autback data directory")
	_ = flags.Parse(args)
	if *input == "" {
		log.Fatal("--input is required")
	}
	if err := recovery.Restore(*input, *dataDir); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Restored: %s\n", *dataDir)
}

func maintainWorker(args []string) {
	flags := flag.NewFlagSet("autback-server maintain", flag.ExitOnError)
	dataDir := flags.String("data-dir", env("AUTBACK_DATA_DIR", "/var/lib/autback"), "persistent autback data directory")
	dryRun := flags.Bool("dry-run", false, "report reclaim candidates without deleting them")
	jsonOutput := flags.Bool("json", false, "write the capacity report as JSON")
	_ = flags.Parse(args)
	store := openStore(*dataDir)
	defer store.Close()
	docker := swarm.New(swarm.Config{Binary: os.Getenv("AUTBACK_DOCKER"), Host: env("AUTBACK_DOCKER_HOST", "unix:///var/run/docker.sock")})
	if err := docker.Check(context.Background()); err != nil {
		log.Fatal(err)
	}
	scheduler := swarmscheduler.New(swarmscheduler.Config{Client: docker})
	controller := newCapacityController(*dataDir, store, scheduler, *dryRun)
	status, err := controller.Maintain(context.Background(), capacity.TriggerManual)
	writeCapacityStatus(filepath.Join(*dataDir, "capacity.json"), status)
	if *jsonOutput {
		if encodeErr := json.NewEncoder(os.Stdout).Encode(status); encodeErr != nil {
			log.Fatal(encodeErr)
		}
	} else {
		fmt.Printf("Capacity: %s; free: %d; reclaimed: %d\n", status.State, status.After.FreeBytes, status.Reclaim.ReclaimedBytes)
	}
	if err != nil {
		log.Fatal(err)
	}
}

func newCapacityController(dataDir string, store *controlsqlite.Store, scheduler *swarmscheduler.Scheduler, dryRun bool) *capacity.Controller {
	// Destructive Docker cleanup is valid only on a worker whose daemon is an
	// Autback ownership boundary. Local development defaults to observation.
	dryRun = dryRun || os.Getenv("AUTBACK_WORKER_OWNERSHIP") != "exclusive"
	host := capacity.NewHost(capacity.HostConfig{
		CapacityPath: env("AUTBACK_CAPACITY_PATH", dataDir),
		JobsRoot:     env("AUTBACK_JOBS_ROOT", filepath.Join(dataDir, "jobs")),
		CacheRoot:    env("AUTBACK_CACHE_ROOT", filepath.Join(dataDir, "cache")),
		LockPath:     filepath.Join(dataDir, "capacity.lock"),
		Store:        store,
		Commands: capacity.DockerCommands{
			Binary: os.Getenv("AUTBACK_DOCKER"), Host: env("AUTBACK_DOCKER_HOST", "unix:///var/run/docker.sock"),
		},
		DryRun: dryRun,
		Emergency: func(ctx context.Context) error {
			operation, err := store.EmergencyStopActiveOperation(ctx, "worker capacity exhausted")
			if err != nil || operation == nil {
				return err
			}
			if operation.Kind == control.OperationJob {
				if err := scheduler.Cancel(ctx, operation.ID); err != nil {
					log.Printf("emergency cancel job %s: %v", operation.ID, err)
				}
				if err := scheduler.Remove(ctx, operation.ID); err != nil {
					log.Printf("emergency remove job %s: %v", operation.ID, err)
				}
			}
			log.Printf("capacity emergency stopped %s %s", operation.Kind, operation.ID)
			return nil
		},
	})
	return capacity.New(capacity.DefaultPolicy(), host)
}

func writeCapacityStatus(path string, status capacity.Status) {
	contents, err := json.MarshalIndent(status, "", "  ")
	if err != nil {
		log.Printf("encode capacity status: %v", err)
		return
	}
	temporary := path + ".tmp"
	if err := os.WriteFile(temporary, append(contents, '\n'), 0o600); err != nil {
		log.Printf("write capacity status: %v", err)
		return
	}
	if err := os.Rename(temporary, path); err != nil {
		log.Printf("publish capacity status: %v", err)
	}
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
		log.Fatal("AUTBACK_SERVER_NAMES must contain at least one DNS name or IP address")
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
