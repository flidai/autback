package cli

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/flidai/autback/internal/authclient"
	"github.com/flidai/autback/internal/config"
	"github.com/flidai/autback/internal/protocol"
	"github.com/flidai/autback/internal/version"
)

const currentVersion = version.Current

type IO struct {
	Stdin   io.Reader
	Stdout  io.Writer
	Stderr  io.Writer
	Dir     string
	Keyring authclient.Keyring
	OpenURL func(string) error
	Wait    func(context.Context, time.Duration) error
}

func Run(ctx context.Context, args []string, streams IO) int {
	streams = defaults(streams)
	explicitToken, args, err := globalArgs(args)
	if err != nil {
		return failUsage(streams.Stderr, err.Error())
	}
	if len(args) == 0 || args[0] == "help" || args[0] == "--help" || args[0] == "-h" {
		usage(streams.Stdout)
		return 0
	}
	if args[0] == "version" || args[0] == "--version" {
		fmt.Fprintln(streams.Stdout, currentVersion)
		return 0
	}
	settings, err := config.Load()
	if err != nil {
		return fail(streams.Stderr, err)
	}
	return runService(ctx, settings, explicitToken, args, streams)
}

func jobID() (string, error) {
	value := make([]byte, 8)
	if _, err := rand.Read(value); err != nil {
		return "", err
	}
	return fmt.Sprintf("autback-%x", value), nil
}

func printJob(output io.Writer, job protocol.Job) {
	writer := tabwriter.NewWriter(output, 0, 4, 2, ' ', 0)
	fmt.Fprintf(writer, "Job:\t%s\nStatus:\t%s\nProject:\t%s\nImage:\t%s\nWorker:\t%s\nCommand:\t%s\nCreated:\t%s\nDuration:\t%s\n",
		job.ID, job.Status, job.ProjectID, job.Image, value(job.WorkerID), strings.Join(job.Command, " "),
		job.CreatedAt.Format(time.RFC3339), duration(job))
	if job.ErrorMessage != "" {
		fmt.Fprintf(writer, "Error:\t%s\n", job.ErrorMessage)
	}
	_ = writer.Flush()
}

func printCompletion(output io.Writer, job protocol.Job) {
	fmt.Fprintf(output, "Completed: %s in %s", job.Status, duration(job))
	if job.ExitCode != nil {
		fmt.Fprintf(output, " (exit %d)", *job.ExitCode)
	}
	fmt.Fprintln(output)
}

func duration(job protocol.Job) string {
	if job.StartedAt == nil {
		return "-"
	}
	end := time.Now()
	if job.FinishedAt != nil {
		end = *job.FinishedAt
	}
	return end.Sub(*job.StartedAt).Round(time.Millisecond).String()
}

func jobArgs(args []string) (bool, string, error) {
	jsonOutput := false
	if len(args) > 0 && args[0] == "--json" {
		jsonOutput, args = true, args[1:]
	}
	if len(args) != 1 {
		return false, "", errors.New("requires exactly one job ID")
	}
	return jsonOutput, args[0], nil
}

func encode(streams IO, value any) int {
	encoder := json.NewEncoder(streams.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(value); err != nil {
		return fail(streams.Stderr, err)
	}
	return 0
}

func defaults(streams IO) IO {
	if streams.Stdin == nil {
		streams.Stdin = os.Stdin
	}
	if streams.Stdout == nil {
		streams.Stdout = os.Stdout
	}
	if streams.Stderr == nil {
		streams.Stderr = os.Stderr
	}
	if streams.Dir == "" {
		streams.Dir = "."
	}
	if streams.Keyring == nil {
		streams.Keyring = authclient.SystemKeyring{}
	}
	if streams.OpenURL == nil {
		streams.OpenURL = openBrowser
	}
	if streams.Wait == nil {
		streams.Wait = func(ctx context.Context, duration time.Duration) error {
			timer := time.NewTimer(duration)
			defer timer.Stop()
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-timer.C:
				return nil
			}
		}
	}
	return streams
}

func usage(output io.Writer) {
	fmt.Fprintln(output, `Usage:
  autback init [--project <project>]
  autback [--token <token>] exec [--project <project>] [--image <digest>] [--detach] [--timeout 15m] [--workdir <path>] [--env KEY=VALUE] [--cache NAME=/absolute/path] [--secret-env NAME=ENV_KEY] [--secret-file NAME=/run/secrets/PATH] -- <command> [arguments...]
  autback [--token <token>] build [--project <project>] [-- <buildx arguments...>]
  autback [--token <token>] image show [--project <project>]
  autback [--token <token>] image activate [--project <project>] --image <digest>
  autback [--token <token>] image rollback [--project <project>]
  autback [--token <token>] image history [--project <project>]
  autback [--token <token>] image overrides [--project <project>] <allow|deny>
  autback [--token <token>] image build [--project <project>] --tag <registry/repository:tag> [--file Dockerfile] [-- <buildx arguments...>]
  autback login [--device <name>] [--no-open]
  autback logout
  autback console
  autback token create --name <device> [--user <id>] [--expires 720h]
  autback token list
  autback token revoke <token-id>
  autback trust github create --project <project> --owner-id <id> --repository-id <id> --workflow-ref <glob> --ref <glob> --event <event> [--environment <name>]
  autback trust github list --project <project>
  autback trust github revoke <trust-id>
  autback admin user create --name <name> [--admin]
  autback admin identity github --user <user-id> --login <github-login>
  autback admin identity revoke --user <user-id>
  autback admin project create --slug <slug> --name <name>
  autback admin member add --project <project> --user <user-id>
  autback admin enrollment create --user <user-id> --device <name> [--expires 10m]
  autback status [--json] <job-id>
  autback logs <job-id>
  autback cancel <job-id>
  autback list [--project <project>] [--limit 20] [--json]
  autback doctor
  autback version`)
}

func globalArgs(args []string) (string, []string, error) {
	token := ""
	for len(args) > 0 && args[0] == "--token" {
		if len(args) < 2 || args[1] == "" {
			return "", nil, errors.New("--token requires a value")
		}
		token, args = args[1], args[2:]
	}
	return token, args, nil
}

func fail(output io.Writer, err error) int {
	fmt.Fprintln(output, "autback:", err)
	return 1
}

func failUsage(output io.Writer, message string) int {
	fmt.Fprintln(output, "autback:", message)
	return 2
}

func value(input string) string {
	if input == "" {
		return "-"
	}
	return input
}

func humanBytes(size int64) string {
	if size < 1024 {
		return fmt.Sprintf("%d B", size)
	}
	if size < 1024*1024 {
		return fmt.Sprintf("%.1f KiB", float64(size)/1024)
	}
	return fmt.Sprintf("%.1f MiB", float64(size)/(1024*1024))
}
