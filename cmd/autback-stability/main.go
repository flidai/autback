package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"time"
)

type invocation struct {
	Package string `json:"package"`
	Pattern string `json:"pattern"`
}

type scenario struct {
	ID          string        `json:"id"`
	Phase       string        `json:"phase"`
	Fault       string        `json:"fault"`
	Tier        string        `json:"tier"`
	Timeout     time.Duration `json:"-"`
	Invocations []invocation  `json:"invocations"`
}

type scenarioResult struct {
	ID          string       `json:"id"`
	Phase       string       `json:"phase"`
	Fault       string       `json:"fault"`
	Tier        string       `json:"tier"`
	Status      string       `json:"status"`
	Duration    string       `json:"duration"`
	Log         string       `json:"log"`
	Invocations []invocation `json:"invocations"`
}

type manifest struct {
	Seed      int64            `json:"seed"`
	StartedAt time.Time        `json:"started_at"`
	Scenarios []scenarioResult `json:"scenarios"`
}

type commandRunner interface {
	Run(context.Context, []string, []string, *os.File) error
}

type execRunner struct{}

func (execRunner) Run(ctx context.Context, environment, arguments []string, output *os.File) error {
	command := exec.CommandContext(ctx, "go", arguments...)
	command.Env = append(os.Environ(), environment...)
	command.Stdout, command.Stderr = io.MultiWriter(os.Stdout, output), io.MultiWriter(os.Stderr, output)
	return command.Run()
}

type runConfig struct {
	Artifacts string
	Seed      int64
	Now       func() time.Time
}

func main() {
	mode := flag.String("mode", "fast", "fault matrix tier: fast or full")
	artifacts := flag.String("artifacts", filepath.Join(".tmp", "stability"), "diagnostic artifact directory")
	seed := flag.Int64("seed", 20260804, "reproducible fault seed")
	flag.Parse()
	if *mode != "fast" && *mode != "full" {
		fmt.Fprintln(os.Stderr, "mode must be fast or full")
		os.Exit(2)
	}
	if err := runMatrix(context.Background(), execRunner{}, scenarios(*mode), runConfig{Artifacts: *artifacts, Seed: *seed}); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func runMatrix(ctx context.Context, runner commandRunner, matrix []scenario, config runConfig) error {
	if config.Now == nil {
		config.Now = time.Now
	}
	if err := os.MkdirAll(config.Artifacts, 0o700); err != nil {
		return err
	}
	report := manifest{Seed: config.Seed, StartedAt: config.Now().UTC(), Scenarios: make([]scenarioResult, 0, len(matrix))}
	var failures []error
	for _, item := range matrix {
		started := time.Now()
		logName := item.ID + ".log"
		logFile, err := os.OpenFile(filepath.Join(config.Artifacts, logName), os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
		if err != nil {
			return err
		}
		_, _ = fmt.Fprintf(logFile, "scenario=%s phase=%s fault=%s seed=%d timeout=%s\n", item.ID, item.Phase, item.Fault, config.Seed, item.Timeout)
		scenarioCtx, cancel := context.WithTimeout(ctx, item.Timeout)
		status := "passed"
		for _, call := range item.Invocations {
			_, _ = fmt.Fprintf(logFile, "package=%s pattern=%s\n", call.Package, call.Pattern)
			environment := []string{
				"AUTBACK_FAULT_SEED=" + strconv.FormatInt(config.Seed, 10),
				"AUTBACK_FAULT_SCENARIO=" + item.ID,
				"AUTBACK_FAULT_PHASE=" + item.Phase,
			}
			arguments := []string{"test", "-count=1", "-timeout", item.Timeout.String(), "-run", "^(" + call.Pattern + ")$", "-v", call.Package}
			if err := runner.Run(scenarioCtx, environment, arguments, logFile); err != nil {
				status = "failed"
				failures = append(failures, fmt.Errorf("%s (%s): %w", item.ID, call.Package, err))
			}
		}
		cancel()
		_ = logFile.Close()
		if status == "passed" {
			if contents, readErr := os.ReadFile(filepath.Join(config.Artifacts, logName)); readErr == nil && bytes.Contains(contents, []byte("--- SKIP:")) {
				status = "skipped"
			}
		}
		report.Scenarios = append(report.Scenarios, scenarioResult{
			ID: item.ID, Phase: item.Phase, Fault: item.Fault, Tier: item.Tier, Status: status,
			Duration: time.Since(started).Round(time.Millisecond).String(), Log: logName, Invocations: item.Invocations,
		})
	}
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	if err := os.WriteFile(filepath.Join(config.Artifacts, "manifest.json"), data, 0o600); err != nil {
		return err
	}
	return errors.Join(failures...)
}

func scenarios(mode string) []scenario {
	const timeout = 30 * time.Second
	matrix := []scenario{
		{ID: "cas-restart-transfer", Phase: "source-upload/result-download", Fault: "CAS connection loss and recovery", Tier: "fast", Timeout: timeout, Invocations: []invocation{
			{Package: "./internal/cli", Pattern: "TestHeartbeatServiceJobPreparationRetriesTransientFailureAndPublishesCredential|TestWaitServiceJobReconnectsWithoutDuplicatingLogBytes"},
			{Package: "./internal/dataplane", Pattern: "TestProxyUsesRefreshedCredentialForNewUpstreamConnections"},
		}},
		{ID: "buildkit-restart", Phase: "build", Fault: "BuildKit connection loss and recovery", Tier: "fast", Timeout: timeout, Invocations: []invocation{
			{Package: "./internal/cli", Pattern: "TestHeartbeatServiceBuildRetriesTransientFailureAndPublishesCredential"},
			{Package: "./internal/buildkit", Pattern: "TestRunWithRunnerRetriesBoundedBuilderRemovalAfterCancelledBuild"},
		}},
		{ID: "docker-daemon-loss", Phase: "running/reconciliation", Fault: "Docker API unavailable then restored", Tier: "fast", Timeout: timeout, Invocations: []invocation{
			{Package: "./internal/adapter/docker", Pattern: "TestTypedSwarmCheckRecoversAfterDaemonOutage|TestTypedSwarmConvergesAfterRestartDuringConcurrentRefreshAndCancellation"},
			{Package: "./internal/control/reconciler", Pattern: "TestRunOnceDoesNotMarkJobsLostDuringDockerDaemonOutage"},
		}},
		{ID: "swarm-node-drain", Phase: "scheduled task", Fault: "shutdown/remove/orphaned task transitions", Tier: "fast", Timeout: timeout, Invocations: []invocation{
			{Package: "./internal/adapter/docker", Pattern: "TestRuntimeTaskStatusClassifiesEverySwarmState|TestTypedSwarmTerminalizesTasksWithoutContainerStatus"},
		}},
		{ID: "server-sigkill-phases", Phase: "queued/running/upload/cleanup", Fault: "control process disappears and durable state reopens", Tier: "fast", Timeout: timeout, Invocations: []invocation{
			{Package: "./internal/control/sqlite", Pattern: "TestActiveWorkerLeaseSurvivesStoreRestart|TestOperationCleanupLifecycleSurvivesRestartAndBlocksFIFO|TestPreparingJobLeaseExpiresUntilSourceUploadIsCommitted"},
			{Package: "./internal/control/reconciler", Pattern: "TestRunOnceConvergesTerminalOrphanAndMissingJobs|TestRunOnceCancelsExpiredJobPreparationAndAdvancesQueue"},
			{Package: "./internal/operation/cleanup", Pattern: "TestCoordinatorDrainWaitsWithoutReportingExpectedCancellation"},
		}},
		{ID: "disk-and-inode-pressure", Phase: "admission/running", Fault: "soft floor, hard full, and inode exhaustion", Tier: "fast", Timeout: timeout, Invocations: []invocation{
			{Package: "./internal/capacity", Pattern: "TestEnsureCollectsPastSoftFloor|TestEnsureReturnsResourceExhaustedWhenReclaimCannotReachFloor|TestEnsureUsesInodePressure|TestMaintainStopsActiveOperationBeforeHardReclaim"},
		}},
		{ID: "credential-rotation", Phase: "long upload/build", Fault: "operation certificate expires and rotates", Tier: "fast", Timeout: timeout, Invocations: []invocation{
			{Package: "./internal/dataplane", Pattern: "TestProxyUsesRefreshedCredentialForNewUpstreamConnections|TestProxyRejectsInvalidCredentialUpdateAndKeepsServing"},
			{Package: "./internal/cli", Pattern: "TestRenewServiceClientRemainsRenewableAcrossSessionExpirations|TestWaitForServiceBuildRenewsGitHubOIDCSessionMoreThanOnce"},
		}},
		{ID: "partial-cleanup-restart", Phase: "cleanup/FIFO release", Fault: "partial cleanup error and process cancellation", Tier: "fast", Timeout: timeout, Invocations: []invocation{
			{Package: "./internal/operation/cleanup", Pattern: "TestResourceManagerRetriesPartialCleanupFromPersistedBaseline|TestResourceManagerDoesNotReleaseWhileOwnedResourcesRemain|TestCoordinatorRetriesFailedCleanupAndRecordsAttempts|TestCoordinatorDrainWaitsWithoutReportingExpectedCancellation"},
		}},
		{ID: "term-resistant-process-tree", Phase: "job cancellation", Fault: "workload process group ignores TERM", Tier: "fast", Timeout: timeout, Invocations: []invocation{
			{Package: "./cmd/autback-job-entrypoint", Pattern: "TestCommandCancellationKillsTERMResistantProcessGroup"},
		}},
		{ID: "memory-and-pid-exhaustion", Phase: "running", Fault: "memory OOM and PID limit", Tier: "fast", Timeout: timeout, Invocations: []invocation{
			{Package: "./internal/adapter/docker", Pattern: "TestTypedSwarmExplainsResourceExhaustion|TestTypedSwarmCreatePreservesJobRuntimeContract"},
			{Package: "./internal/hostmetrics", Pattern: "TestLinuxSamplerReportsPressureOOMAndInodeEvidence|TestCollectorAttributesOnlyNewCgroupEvents"},
		}},
	}
	if mode == "full" {
		matrix = append(matrix, scenario{ID: "docker-owned-resource-cleanup", Phase: "privileged cleanup", Fault: "real Swarm/Testcontainers resource leakage", Tier: "full", Timeout: 5 * time.Minute, Invocations: []invocation{{
			Package: "./internal/adapter/docker", Pattern: "TestResourceManagerCleansRealOperationResources",
		}}})
	}
	return matrix
}
