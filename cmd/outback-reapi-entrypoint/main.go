package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"
	"time"

	"github.com/flidai/outback/internal/ocirunner"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "outback-reapi-entrypoint: command is required")
		os.Exit(2)
	}
	actionDirectory := os.Getenv("OUTBACK_ACTION_DIRECTORY")
	if actionDirectory == "" {
		fmt.Fprintln(os.Stderr, "outback-reapi-entrypoint: OUTBACK_ACTION_DIRECTORY is required")
		os.Exit(2)
	}
	for _, directory := range []string{filepath.Join(actionDirectory, "tmp"), filepath.Join(actionDirectory, "data")} {
		if err := os.MkdirAll(directory, 0o700); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}
	workingDirectory, err := os.Getwd()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	if timeout := timeoutFromEnvironment(); timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}
	name := "outback-reapi-" + strconv.Itoa(os.Getpid())
	exitCode, runErr := ocirunner.Run(ctx, os.Getenv("OUTBACK_DOCKER"), ocirunner.Spec{
		Name: name, Image: os.Getenv("OUTBACK_RUNNER_IMAGE"), CPUs: os.Getenv("OUTBACK_JOB_CPUS"), Memory: os.Getenv("OUTBACK_JOB_MEMORY"),
		ActionDirectory: actionDirectory, WorkingDirectory: workingDirectory, Command: os.Args[1:],
	}, os.Stdout, os.Stderr)
	if ctx.Err() == context.DeadlineExceeded {
		markTimeout(os.Getenv("OUTBACK_SIDE_CHANNEL_FILE"))
		fmt.Fprintln(os.Stderr, "remote action timed out")
		os.Exit(124)
	}
	if runErr != nil {
		fmt.Fprintln(os.Stderr, runErr)
	}
	os.Exit(exitCode)
}

func timeoutFromEnvironment() time.Duration {
	millis, err := strconv.ParseInt(os.Getenv("OUTBACK_TIMEOUT_MILLIS"), 10, 64)
	if err != nil || millis <= 0 {
		return 0
	}
	return time.Duration(millis) * time.Millisecond
}

func markTimeout(path string) {
	if path == "" {
		return
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
	if err != nil {
		return
	}
	defer file.Close()
	_ = json.NewEncoder(file).Encode(map[string]string{"failure": "timeout"})
}
