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

	"github.com/flidai/outback/internal/authclient"
	"github.com/flidai/outback/internal/config"
	"github.com/flidai/outback/internal/protocol"
)

const version = "0.1.0"

type IO struct {
	Stdin   io.Reader
	Stdout  io.Writer
	Stderr  io.Writer
	Dir     string
	Keyring authclient.Keyring
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
		fmt.Fprintln(streams.Stdout, version)
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
	return fmt.Sprintf("outback-%x", value), nil
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
	return streams
}

func usage(output io.Writer) {
	fmt.Fprintln(output, `Usage:
  outback init [--project <project>]
  outback [--token <token>] exec [--project <project>] [--image <digest>] [--detach] [--timeout 15m] [--workdir <path>] [--env KEY=VALUE] [--cache NAME=/absolute/path] -- <command> [arguments...]
  outback [--token <token>] build [--project <project>] [-- <buildx arguments...>]
  outback [--token <token>] image show [--project <project>]
  outback [--token <token>] image activate [--project <project>] --image <digest>
  outback [--token <token>] image rollback [--project <project>]
  outback [--token <token>] image history [--project <project>]
  outback [--token <token>] image overrides [--project <project>] <allow|deny>
  outback [--token <token>] image build [--project <project>] --tag <registry/repository:tag> [--file Dockerfile] [-- <buildx arguments...>]
  outback login
  outback logout
  outback token create --name <device> [--user <id>] [--expires 720h]
  outback token list
  outback token revoke <token-id>
  outback trust github create --project <project> --owner-id <id> --repository-id <id> --workflow-ref <glob> --ref <glob> --event <event> [--environment <name>]
  outback trust github list --project <project>
  outback trust github revoke <trust-id>
  outback admin user create --name <name> [--admin]
  outback admin project create --slug <slug> --name <name>
  outback admin member add --project <project> --user <user-id>
  outback admin enrollment create --user <user-id> --device <name> [--expires 10m]
  outback status [--json] <job-id>
  outback logs <job-id>
  outback cancel <job-id>
  outback list [--project <project>] [--limit 20] [--json]
  outback doctor
  outback version`)
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
	fmt.Fprintln(output, "outback:", err)
	return 1
}

func failUsage(output io.Writer, message string) int {
	fmt.Fprintln(output, "outback:", message)
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
