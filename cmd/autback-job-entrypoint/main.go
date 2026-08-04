package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/flidai/autback/internal/cas"
	"github.com/flidai/autback/internal/operation/redact"
	jobsecrets "github.com/flidai/autback/internal/secrets"
)

type result struct {
	Status     string    `json:"status"`
	ExitCode   int       `json:"exit_code"`
	FinishedAt time.Time `json:"finished_at"`
}

const defaultJobLogMaxBytes = int64(256 * 1024 * 1024)

var logTruncationMarker = []byte("\n[autback: durable job log reached its configured limit]\n")

func main() {
	os.Exit(run())
}

func run() int {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "autback-job-entrypoint: command is required")
		return 2
	}
	workspace := os.Getenv("AUTBACK_WORKSPACE")
	if workspace == "" {
		fmt.Fprintln(os.Stderr, "autback-job-entrypoint: AUTBACK_WORKSPACE is required")
		return 2
	}
	hostUID, hostGID, err := hostIdentityFromEnvironment()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 2
	}
	jobDirectory := filepath.Dir(workspace)
	if err := prepareJobDirectory(jobDirectory, hostUID, hostGID); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	logFile, err := os.OpenFile(filepath.Join(jobDirectory, "job.log"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	if err := logFile.Chown(hostUID, hostGID); err != nil {
		fmt.Fprintln(os.Stderr, err)
		_ = logFile.Close()
		return 1
	}
	defer logFile.Close()
	boundedLog, err := newBoundedLogWriter(logFile, jobLogMaxBytes())
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	runtimeSecrets, err := jobsecrets.LoadRuntime(jobsecrets.RuntimeDirectory)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	stdout, stderr, err := newRedactedOutputs(os.Stdout, os.Stderr, boundedLog, runtimeSecrets.Values)
	if err != nil {
		fmt.Fprintln(os.Stderr, "initialize job output redaction")
		return 1
	}
	defer stderr.Close()
	defer stdout.Close()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	if timeout := timeoutFromEnvironment(); timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}
	if err := os.RemoveAll(workspace); err != nil {
		fmt.Fprintln(stderr, err)
		return finish(jobDirectory, "failed", 1)
	}
	if err := cas.Materialize(ctx, required("AUTBACK_CAS_ADDRESS"), fallback(os.Getenv("AUTBACK_CAS_INSTANCE"), "autback"), required("AUTBACK_ROOT_DIGEST"), workspace); err != nil {
		fmt.Fprintln(stderr, err)
		return finish(jobDirectory, "failed", 1)
	}
	if err := initializeGitBaseline(ctx, workspace, stderr); err != nil {
		fmt.Fprintln(stderr, err)
		return finish(jobDirectory, "failed", 1)
	}
	for _, directory := range []string{filepath.Join(workspace, ".autback", "tmp"), filepath.Join(workspace, ".autback", "data")} {
		if err := os.MkdirAll(directory, 0o700); err != nil {
			fmt.Fprintln(stderr, err)
			return finish(jobDirectory, "failed", 1)
		}
	}
	command := exec.CommandContext(ctx, os.Args[1], os.Args[2:]...)
	workingDirectory, err := resolveWorkingDirectory(workspace, os.Getenv("AUTBACK_WORKING_DIRECTORY"))
	if err != nil {
		fmt.Fprintln(stderr, err)
		return finish(jobDirectory, "failed", 2)
	}
	command.Dir, command.Stdout, command.Stderr = workingDirectory, stdout, stderr
	command.Env = mergeEnvironment(os.Environ(), runtimeSecrets.Environment)
	command.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	command.Cancel = func() error {
		if command.Process == nil {
			return os.ErrProcessDone
		}
		return syscall.Kill(-command.Process.Pid, syscall.SIGTERM)
	}
	command.WaitDelay = 5 * time.Second
	err = command.Run()
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		fmt.Fprintln(stderr, "remote job timed out")
		return finish(jobDirectory, "timed_out", 124)
	}
	if ctx.Err() != nil {
		fmt.Fprintln(stderr, "remote job cancelled")
		return finish(jobDirectory, "cancelled", 130)
	}
	if err == nil {
		return finish(jobDirectory, "succeeded", 0)
	}
	var exit *exec.ExitError
	if errors.As(err, &exit) {
		return finish(jobDirectory, "failed", exit.ExitCode())
	}
	fmt.Fprintln(stderr, err)
	return finish(jobDirectory, "failed", 1)
}

func mergeEnvironment(base, overrides []string) []string {
	replaced := make(map[string]struct{}, len(overrides))
	for _, item := range overrides {
		key, _, found := strings.Cut(item, "=")
		if found {
			replaced[key] = struct{}{}
		}
	}
	result := make([]string, 0, len(base)+len(overrides))
	for _, item := range base {
		key, _, found := strings.Cut(item, "=")
		if _, replace := replaced[key]; found && replace {
			continue
		}
		result = append(result, item)
	}
	return append(result, overrides...)
}

func newRedactedOutputs(stdout, stderr, durable io.Writer, values []string) (*redact.Writer, *redact.Writer, error) {
	redactedStdout, err := redact.NewWriter(io.MultiWriter(stdout, durable), values)
	if err != nil {
		return nil, nil, err
	}
	redactedStderr, err := redact.NewWriter(io.MultiWriter(stderr, durable), values)
	if err != nil {
		_ = redactedStdout.Close()
		return nil, nil, err
	}
	return redactedStdout, redactedStderr, nil
}

type boundedLogWriter struct {
	mu          sync.Mutex
	destination io.Writer
	remaining   int64
	truncated   bool
}

func newBoundedLogWriter(file *os.File, maximum int64) (*boundedLogWriter, error) {
	info, err := file.Stat()
	if err != nil {
		return nil, err
	}
	remaining := maximum - info.Size()
	if remaining < 0 {
		remaining = 0
	}
	return &boundedLogWriter{destination: file, remaining: remaining}, nil
}

func (w *boundedLogWriter) Write(data []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	original := len(data)
	if w.truncated || w.remaining == 0 {
		w.truncated = true
		return original, nil
	}
	if int64(len(data)) <= w.remaining {
		written, err := w.destination.Write(data)
		w.remaining -= int64(written)
		return written, err
	}
	w.truncated = true
	payloadBytes := w.remaining - int64(len(logTruncationMarker))
	if payloadBytes > 0 {
		if _, err := w.destination.Write(data[:payloadBytes]); err != nil {
			return 0, err
		}
		w.remaining -= payloadBytes
	}
	marker := logTruncationMarker
	if int64(len(marker)) > w.remaining {
		marker = marker[:w.remaining]
	}
	if len(marker) > 0 {
		if _, err := w.destination.Write(marker); err != nil {
			return 0, err
		}
		w.remaining -= int64(len(marker))
	}
	return original, nil
}

func jobLogMaxBytes() int64 {
	value := os.Getenv("AUTBACK_JOB_LOG_MAX_BYTES")
	if value == "" {
		return defaultJobLogMaxBytes
	}
	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil || parsed < 1024 {
		return defaultJobLogMaxBytes
	}
	return parsed
}

func initializeGitBaseline(ctx context.Context, workspace string, output io.Writer) error {
	if _, err := os.Stat(filepath.Join(workspace, ".git")); err == nil {
		return nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("inspect workspace git metadata: %w", err)
	}
	git, err := exec.LookPath("git")
	if errors.Is(err, exec.ErrNotFound) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("find git: %w", err)
	}
	commands := [][]string{
		{"init", "--quiet"},
		{"add", "--all", "--force"},
		{"-c", "user.name=autback", "-c", "user.email=autback@localhost", "commit", "--quiet", "--allow-empty", "--no-gpg-sign", "--no-verify", "-m", "autback source snapshot"},
	}
	for _, arguments := range commands {
		command := exec.CommandContext(ctx, git, arguments...)
		command.Dir = workspace
		command.Stdout = output
		command.Stderr = output
		if err := command.Run(); err != nil {
			return fmt.Errorf("initialize workspace git baseline with git %s: %w", arguments[0], err)
		}
	}
	return nil
}

func prepareJobDirectory(jobDirectory string, hostUID, hostGID int) error {
	if err := os.MkdirAll(jobDirectory, 0o700); err != nil {
		return err
	}
	if err := os.Chown(jobDirectory, hostUID, hostGID); err != nil {
		return err
	}
	if err := os.Chmod(jobDirectory, 0o700); err != nil {
		return err
	}
	return nil
}

func hostIdentityFromEnvironment() (int, int, error) {
	uid, err := strconv.Atoi(os.Getenv("AUTBACK_HOST_UID"))
	if err != nil || uid < 0 {
		return 0, 0, errors.New("AUTBACK_HOST_UID must be a non-negative integer")
	}
	gid, err := strconv.Atoi(os.Getenv("AUTBACK_HOST_GID"))
	if err != nil || gid < 0 {
		return 0, 0, errors.New("AUTBACK_HOST_GID must be a non-negative integer")
	}
	return uid, gid, nil
}

func resolveWorkingDirectory(workspace, relative string) (string, error) {
	if relative == "" || relative == "." {
		return workspace, nil
	}
	if filepath.IsAbs(relative) {
		return "", errors.New("AUTBACK_WORKING_DIRECTORY must be relative to the workspace")
	}
	clean := filepath.Clean(relative)
	if clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return "", errors.New("AUTBACK_WORKING_DIRECTORY escapes the workspace")
	}
	return filepath.Join(workspace, clean), nil
}

func timeoutFromEnvironment() time.Duration {
	millis, err := strconv.ParseInt(os.Getenv("AUTBACK_TIMEOUT_MILLIS"), 10, 64)
	if err != nil || millis < 1 {
		return 0
	}
	return time.Duration(millis) * time.Millisecond
}

func finish(jobDirectory, status string, exitCode int) int {
	path := filepath.Join(jobDirectory, "result.json")
	file, err := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
	if err == nil {
		_ = json.NewEncoder(file).Encode(result{Status: status, ExitCode: exitCode, FinishedAt: time.Now().UTC()})
		_ = file.Close()
	}
	return exitCode
}

func required(name string) string {
	value := os.Getenv(name)
	if value == "" {
		fmt.Fprintf(os.Stderr, "autback-job-entrypoint: %s is required\n", name)
	}
	return value
}

func fallback(value, defaultValue string) string {
	if value == "" {
		return defaultValue
	}
	return value
}
