package buildkit

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"time"
)

type Commands struct {
	Create []string
	Build  []string
	Remove []string
}

type TLS struct {
	CA          string
	Certificate string
	Key         string
	ServerName  string
}

func Plan(address, name string, arguments []string) Commands {
	return PlanWithTLS(address, name, arguments, TLS{})
}

func PlanWithTLS(address, name string, arguments []string, credentials TLS) Commands {
	create := []string{"buildx", "create", "--name", name, "--driver", "remote"}
	if credentials.CA != "" || credentials.Certificate != "" || credentials.Key != "" || credentials.ServerName != "" {
		create = append(create, "--driver-opt", "cacert="+credentials.CA+",cert="+credentials.Certificate+",key="+credentials.Key+",servername="+credentials.ServerName)
	}
	create = append(create, address)
	return Commands{
		Create: create,
		Build:  append([]string{"buildx", "build", "--builder", name}, arguments...),
		Remove: []string{"buildx", "rm", "--force", name},
	}
}

func Run(ctx context.Context, docker, address, name, directory string, arguments []string, stdout, stderr io.Writer) (int, error) {
	return RunWithTLS(ctx, docker, address, name, directory, arguments, TLS{}, stdout, stderr)
}

func RunWithTLS(ctx context.Context, docker, address, name, directory string, arguments []string, credentials TLS, stdout, stderr io.Writer) (int, error) {
	return runWithRunner(ctx, execRunner{}, docker, address, name, directory, arguments, credentials, stdout, stderr, removePolicy{})
}

type commandRunner interface {
	Run(context.Context, string, []string, string, io.Writer, io.Writer) error
}

type execRunner struct{}

func (execRunner) Run(ctx context.Context, binary string, arguments []string, directory string, stdout, stderr io.Writer) error {
	command := exec.CommandContext(ctx, binary, arguments...)
	command.Dir, command.Stdout, command.Stderr = directory, stdout, stderr
	return command.Run()
}

type removePolicy struct {
	Timeout    time.Duration
	Attempts   int
	RetryDelay time.Duration
	Wait       func(context.Context, time.Duration) error
}

func runWithRunner(ctx context.Context, runner commandRunner, docker, address, name, directory string, arguments []string, credentials TLS, stdout, stderr io.Writer, policy removePolicy) (int, error) {
	if docker == "" {
		docker = "docker"
	}
	commands := PlanWithTLS(address, name, arguments, credentials)
	if err := runner.Run(ctx, docker, commands.Create, directory, io.Discard, stderr); err != nil {
		cleanupErr := removeBuilder(runner, docker, commands.Remove, directory, policy)
		return 1, errors.Join(err, cleanupErr)
	}
	code := 0
	var buildErr error
	if err := runner.Run(ctx, docker, commands.Build, directory, stdout, stderr); err != nil {
		var exit *exec.ExitError
		if errors.As(err, &exit) {
			code = exit.ExitCode()
		} else {
			code, buildErr = 1, err
		}
	}
	cleanupErr := removeBuilder(runner, docker, commands.Remove, directory, policy)
	return code, errors.Join(buildErr, cleanupErr)
}

func removeBuilder(runner commandRunner, docker string, command []string, directory string, policy removePolicy) error {
	if policy.Timeout <= 0 {
		policy.Timeout = 15 * time.Second
	}
	if policy.Attempts <= 0 {
		policy.Attempts = 3
	}
	if policy.RetryDelay <= 0 {
		policy.RetryDelay = 250 * time.Millisecond
	}
	if policy.Wait == nil {
		policy.Wait = waitForRetry
	}
	ctx, cancel := context.WithTimeout(context.Background(), policy.Timeout)
	defer cancel()
	var lastErr error
	for attempt := 1; attempt <= policy.Attempts; attempt++ {
		var output bytes.Buffer
		err := runner.Run(ctx, docker, command, directory, io.Discard, &output)
		if err == nil || builderNotFound(output.String()) {
			return nil
		}
		lastErr = fmt.Errorf("remove ephemeral Buildx builder: %w: %s", err, strings.TrimSpace(output.String()))
		if attempt == policy.Attempts {
			break
		}
		if err := policy.Wait(ctx, policy.RetryDelay); err != nil {
			return errors.Join(lastErr, err)
		}
	}
	return lastErr
}

func builderNotFound(output string) bool {
	message := strings.ToLower(output)
	return strings.Contains(message, "no builder") && strings.Contains(message, "found") || strings.Contains(message, "no such builder")
}

func waitForRetry(ctx context.Context, delay time.Duration) error {
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
