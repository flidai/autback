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
	"sync/atomic"
	"syscall"
	"time"

	dockeradapter "github.com/flidai/autback/internal/adapter/docker"
	appserver "github.com/flidai/autback/internal/app/server"
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
	operationcleanup "github.com/flidai/autback/internal/operation/cleanup"
	"github.com/flidai/autback/internal/version"
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
	if err := serve(); err != nil {
		log.Printf("autback server: %v", err)
		os.Exit(1)
	}
}

func serve() error {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	return run(ctx)
}

func run(ctx context.Context) error {
	startedAt := time.Now().UTC()
	processCtx, stopProcess := context.WithCancel(ctx)
	defer stopProcess()
	dataDir := env("AUTBACK_DATA_DIR", "/var/lib/autback")
	store, err := openStoreE(dataDir)
	if err != nil {
		return err
	}
	closeStore := true
	defer func() {
		if closeStore {
			_ = store.Close()
		}
	}()
	initialized, err := store.Initialized(processCtx)
	if err != nil {
		return err
	}
	if !initialized {
		return errors.New("autback is not bootstrapped; run autback-server bootstrap before starting the service")
	}
	names, err := splitNames(env("AUTBACK_SERVER_NAMES", "localhost,127.0.0.1"))
	if err != nil {
		return err
	}
	pkiDir := env("AUTBACK_PKI_DIR", filepath.Join(dataDir, "pki"))
	authority, err := pki.Ensure(pkiDir, names)
	if err != nil {
		return err
	}
	dockerHost := env("AUTBACK_DOCKER_HOST", "unix:///var/run/docker.sock")
	docker, err := dockeradapter.New(dockeradapter.Config{Host: dockerHost})
	if err != nil {
		return err
	}
	defer docker.Close()
	if err := docker.Check(processCtx); err != nil {
		return err
	}
	resourceManager := operationcleanup.NewResourceManager(operationcleanup.ResourceManagerConfig{
		Store:       store,
		Runtime:     docker,
		GracePeriod: durationEnv("AUTBACK_RESOURCE_CLEANUP_GRACE", 10*time.Second),
		Timeout:     durationEnv("AUTBACK_RESOURCE_CLEANUP_TIMEOUT", 2*time.Minute),
	})
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
	capacityController := newCapacityController(dataDir, store, scheduler, docker, resourceManager, false)
	status, capacityErr := capacityController.Maintain(processCtx, capacity.TriggerManual)
	writeCapacityStatus(filepath.Join(dataDir, "capacity.json"), status)
	if capacityErr != nil {
		log.Printf("initial capacity reconciliation: %v", capacityErr)
	}
	sampler, err := hostmetrics.NewLinuxSampler(hostmetrics.LinuxSamplerConfig{DiskPath: dataDir})
	if err != nil {
		return err
	}
	resourceCollector, err := hostmetrics.NewCollector(hostmetrics.CollectorConfig{
		Store: store, Sampler: sampler,
		Interval:        durationEnv("AUTBACK_METRICS_INTERVAL", 2*time.Second),
		RawRetention:    durationEnv("AUTBACK_METRICS_RAW_RETENTION", 14*24*time.Hour),
		RollupRetention: durationEnv("AUTBACK_METRICS_ROLLUP_RETENTION", 180*24*time.Hour),
		OnError:         func(err error) { log.Printf("resource metrics: %v", err) },
	})
	if err != nil {
		return err
	}
	dispatch := dispatcher.New(store, scheduler,
		dispatcher.WithCapacity(capacityController),
		dispatcher.WithAdmissionPreparer(resourceManager),
		dispatcher.WithCleaner(resourceManager),
		dispatcher.WithAdvanceContext(processCtx),
		dispatcher.WithErrorHandler(func(err error) { log.Printf("advance FIFO: %v", err) }),
	)
	reconcile := reconciler.New(reconciler.Config{
		Store: store, Scheduler: scheduler, Dispatcher: dispatch,
		ServiceRetention:  durationEnv("AUTBACK_SERVICE_RETENTION", time.Hour),
		AdmissionGrace:    durationEnv("AUTBACK_ADMISSION_GRACE", 15*time.Second),
		BuildLeaseTimeout: durationEnv("AUTBACK_BUILD_LEASE_TIMEOUT", 2*time.Minute),
	})
	var verifier controlapi.OIDCVerifier
	if audience := os.Getenv("AUTBACK_GITHUB_OIDC_AUDIENCE"); audience != "" {
		verifier, err = githuboidc.New(processCtx, env("AUTBACK_GITHUB_OIDC_ISSUER", githuboidc.Issuer), audience)
		if err != nil {
			return err
		}
	}
	draining := &atomic.Bool{}
	controlHandler, err := controlapi.New(controlapi.Config{
		Store: store, Scheduler: scheduler, Dispatcher: dispatch, Authority: authority, OIDCVerifier: verifier,
		CASEndpoint: env("AUTBACK_CAS_ENDPOINT", endpoint(serverName, casListen)), CASInstance: casInstance,
		BuildKitEndpoint:              env("AUTBACK_BUILDKIT_ENDPOINT", endpoint(serverName, buildKitListen)),
		CredentialTTL:                 durationEnv("AUTBACK_CREDENTIAL_TTL", 15*time.Minute),
		AllowUnpinnedImages:           os.Getenv("AUTBACK_ALLOW_UNPINNED_IMAGES") == "1",
		Capacity:                      capacityController,
		RequiredBuildClientCapability: version.CapabilityBuildLeaseHeartbeat,
		Ready:                         func() bool { return !draining.Load() },
	})
	if err != nil {
		return err
	}
	consoleSource, err := console.NewSQLiteSource(console.SQLiteSourceConfig{
		Store: store, Scheduler: scheduler, Version: controlapi.Version, StartedAt: startedAt,
	})
	if err != nil {
		return err
	}
	consoleHandler, err := console.New(console.Config{Source: consoleSource})
	if err != nil {
		return err
	}
	handler := serviceHandler(controlHandler, consoleHandler)
	active := func(kind pki.Operation, id string) bool {
		return store.OperationActive(context.Background(), string(kind), id)
	}
	casListener, err := net.Listen("tcp", casListen)
	if err != nil {
		return fmt.Errorf("listen for CAS proxy: %w", err)
	}
	defer casListener.Close()
	buildKitListener, err := net.Listen("tcp", buildKitListen)
	if err != nil {
		return fmt.Errorf("listen for BuildKit proxy: %w", err)
	}
	defer buildKitListener.Close()
	server := &http.Server{
		Addr: env("AUTBACK_LISTEN", ":8443"), Handler: handler, ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout: 2 * time.Minute, TLSConfig: &tls.Config{MinVersion: tls.VersionTLS13},
	}
	controlListener, err := net.Listen("tcp", server.Addr)
	if err != nil {
		return fmt.Errorf("listen for control plane: %w", err)
	}
	defer controlListener.Close()

	group := appserver.New(appserver.Config{
		ShutdownTimeout: 15 * time.Second,
		OnDrain: func() {
			draining.Store(true)
			dispatch.Drain()
			stopProcess()
		},
	})
	group.Add(appserver.Component{Name: "resource metrics", Run: func(ctx context.Context) error {
		resourceCollector.Run(ctx)
		return nil
	}})
	group.Add(appserver.Component{Name: "reconciler", Run: func(ctx context.Context) error {
		runReconciler(ctx, reconcile, durationEnv("AUTBACK_RECONCILE_INTERVAL", time.Second))
		return nil
	}})
	group.Add(appserver.Component{Name: "capacity controller", Run: func(ctx context.Context) error {
		runCapacityController(ctx, capacityController, filepath.Join(dataDir, "capacity.json"), durationEnv("AUTBACK_CAPACITY_CHECK_INTERVAL", 5*time.Second), durationEnv("AUTBACK_MAINTENANCE_INTERVAL", time.Minute))
		return nil
	}})
	group.Add(appserver.Component{
		Name: "dispatcher",
		Run: func(ctx context.Context) error {
			<-ctx.Done()
			return nil
		},
		Stop: dispatch.Wait,
	})
	group.Add(appserver.Component{
		Name: "CAS mTLS proxy",
		Run: func(ctx context.Context) error {
			return mtlsproxy.Serve(ctx, casListener, casInternal, authority.ServerTLSConfig(pki.OperationJob, active))
		},
		Stop: func(context.Context) error { return closeListener(casListener) },
	})
	group.Add(appserver.Component{
		Name: "BuildKit mTLS proxy",
		Run: func(ctx context.Context) error {
			return mtlsproxy.Serve(ctx, buildKitListener, buildKitInternal, authority.ServerTLSConfig(pki.OperationBuild, active))
		},
		Stop: func(context.Context) error { return closeListener(buildKitListener) },
	})
	group.Add(appserver.Component{
		Name: "control HTTP server",
		Run: func(ctx context.Context) error {
			err := server.ServeTLS(controlListener, filepath.Join(pkiDir, "server.pem"), filepath.Join(pkiDir, "server-key.pem"))
			if errors.Is(err, http.ErrServerClosed) || ctx.Err() != nil {
				return nil
			}
			return err
		},
		Stop: server.Shutdown,
	})

	dispatch.Advance()
	log.Printf("autback control plane listening on %s; CAS mTLS on %s; BuildKit mTLS on %s", controlListener.Addr(), casListener.Addr(), buildKitListener.Addr())
	if err := group.Run(processCtx); err != nil {
		// A timed-out goroutine may still hold a Store reference. Let process
		// exit close its descriptors instead of racing sql.DB.Close against it.
		if errors.Is(err, appserver.ErrJoinTimeout) {
			closeStore = false
		}
		return err
	}
	return nil
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
	docker, err := dockeradapter.New(dockeradapter.Config{Host: env("AUTBACK_DOCKER_HOST", "unix:///var/run/docker.sock")})
	if err != nil {
		log.Fatal(err)
	}
	defer docker.Close()
	if err := docker.Check(context.Background()); err != nil {
		log.Fatal(err)
	}
	scheduler := swarmscheduler.New(swarmscheduler.Config{Client: docker})
	resourceManager := operationcleanup.NewResourceManager(operationcleanup.ResourceManagerConfig{
		Store:       store,
		Runtime:     docker,
		GracePeriod: durationEnv("AUTBACK_RESOURCE_CLEANUP_GRACE", 10*time.Second),
		Timeout:     durationEnv("AUTBACK_RESOURCE_CLEANUP_TIMEOUT", 2*time.Minute),
	})
	controller := newCapacityController(*dataDir, store, scheduler, docker, resourceManager, *dryRun)
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

func newCapacityController(dataDir string, store *controlsqlite.Store, scheduler *swarmscheduler.Scheduler, runtime capacity.Runtime, cleaner operationcleanup.Cleaner, dryRun bool) *capacity.Controller {
	// Destructive Docker cleanup is valid only on a worker whose daemon is an
	// Autback ownership boundary. Local development defaults to observation.
	dryRun = dryRun || os.Getenv("AUTBACK_WORKER_OWNERSHIP") != "exclusive"
	host := capacity.NewHost(capacity.HostConfig{
		CapacityPath: env("AUTBACK_CAPACITY_PATH", dataDir),
		JobsRoot:     env("AUTBACK_JOBS_ROOT", filepath.Join(dataDir, "jobs")),
		CacheRoot:    env("AUTBACK_CACHE_ROOT", filepath.Join(dataDir, "cache")),
		LockPath:     filepath.Join(dataDir, "capacity.lock"),
		Store:        store,
		Runtime:      runtime,
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
			claimed, err := store.ClaimOperationCleanup(ctx)
			if err != nil {
				return err
			}
			if claimed == nil || claimed.Kind != operation.Kind || claimed.ID != operation.ID {
				return fmt.Errorf("claim emergency cleanup for %s %s", operation.Kind, operation.ID)
			}
			if cleaner != nil {
				if err := cleaner.Cleanup(ctx, *claimed); err != nil {
					return fmt.Errorf("emergency resource cleanup for %s %s: %w", operation.Kind, operation.ID, err)
				}
			}
			if err := store.CompleteOperationCleanup(ctx, operation.Kind, operation.ID); err != nil {
				return err
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
	store, err := openStoreE(dataDir)
	if err != nil {
		log.Fatal(err)
	}
	return store
}

func openStoreE(dataDir string) (*controlsqlite.Store, error) {
	pepper, err := secret.Ensure(filepath.Join(dataDir, "control", "token-pepper"), 32)
	if err != nil {
		return nil, err
	}
	store, err := controlsqlite.Open(filepath.Join(dataDir, "control"), pepper)
	if err != nil {
		return nil, err
	}
	return store, nil
}

func closeListener(listener net.Listener) error {
	err := listener.Close()
	if errors.Is(err, net.ErrClosed) {
		return nil
	}
	return err
}

func splitNames(value string) ([]string, error) {
	var names []string
	for _, item := range strings.Split(value, ",") {
		if name := strings.TrimSpace(item); name != "" {
			names = append(names, name)
		}
	}
	if len(names) == 0 {
		return nil, errors.New("AUTBACK_SERVER_NAMES must contain at least one DNS name or IP address")
	}
	return names, nil
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
