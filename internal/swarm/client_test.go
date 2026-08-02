package swarm

import (
	"bytes"
	"context"
	"io"
	"strings"
	"testing"

	"github.com/flidai/outback/internal/protocol"
)

type fakeCommander struct {
	outputs map[string]string
	runs    [][]string
}

func (f *fakeCommander) Output(_ context.Context, args ...string) ([]byte, error) {
	return []byte(f.outputs[strings.Join(args, " ")]), nil
}

func (f *fakeCommander) Run(_ context.Context, stdout, _ io.Writer, args ...string) error {
	f.runs = append(f.runs, append([]string(nil), args...))
	if value := f.outputs[strings.Join(args, " ")]; value != "" {
		_, _ = io.WriteString(stdout, value)
	}
	return nil
}

func TestStatusReadsServiceAndTaskState(t *testing.T) {
	commands := &fakeCommander{outputs: map[string]string{
		"service inspect outback-job-1": `[{
          "CreatedAt":"2026-08-01T10:00:00Z",
          "Spec":{"Name":"outback-job-1","Labels":{
            "outback.managed":"true","outback.project":"prj-example",
            "outback.image":"Z2hjci5pby9leGFtcGxlL3J1bm5lckBzaGEyNTY6YWJj",
            "outback.timeout_seconds":"900","outback.root_digest":"abc/123"
          },"TaskTemplate":{"ContainerSpec":{"Args":["go","test","./..."]}}}
        }]`,
		"service ps -q --no-trunc outback-job-1": "task-1\n",
		"inspect task-1": `[{
          "Status":{"State":"complete","Timestamp":"2026-08-01T10:01:00Z",
            "ContainerStatus":{"ExitCode":0}},
          "DesiredState":"complete"
        }]`,
	}}
	client := newClient(commands)
	job, err := client.Status(context.Background(), "outback-job-1")
	if err != nil {
		t.Fatal(err)
	}
	if job.Status != protocol.StatusSucceeded || job.ProjectID != "prj-example" || job.Image != "ghcr.io/example/runner@sha256:abc" {
		t.Fatalf("job = %#v", job)
	}
	if job.ExitCode == nil || *job.ExitCode != 0 || job.FinishedAt == nil {
		t.Fatalf("completion = %#v", job)
	}
}

func TestCancelMarksThenScalesJobToZero(t *testing.T) {
	commands := &fakeCommander{outputs: map[string]string{}}
	client := newClient(commands)
	if err := client.Cancel(context.Background(), "outback-job-1"); err != nil {
		t.Fatal(err)
	}
	want := [][]string{
		{"service", "update", "--detach", "--label-add", "outback.cancelled=true", "outback-job-1"},
		{"service", "scale", "--detach", "outback-job-1=0"},
	}
	if len(commands.runs) != len(want) {
		t.Fatalf("runs = %#v", commands.runs)
	}
	for i := range want {
		if strings.Join(commands.runs[i], " ") != strings.Join(want[i], " ") {
			t.Fatalf("run %d = %#v, want %#v", i, commands.runs[i], want[i])
		}
	}
}

func TestValidateImagePullsAndInspectsThePinnedReference(t *testing.T) {
	image := "ghcr.io/example/runner@sha256:" + strings.Repeat("a", 64)
	commands := &fakeCommander{outputs: map[string]string{
		"image inspect --format {{.Id}} " + image: "sha256:" + strings.Repeat("b", 64),
	}}
	if err := newClient(commands).ValidateImage(context.Background(), image); err != nil {
		t.Fatal(err)
	}
	if len(commands.runs) != 1 || strings.Join(commands.runs[0], " ") != "image pull "+image {
		t.Fatalf("runs = %#v", commands.runs)
	}
}

func TestStatusReportsScaledToZeroJobAsCancelled(t *testing.T) {
	commands := &fakeCommander{outputs: map[string]string{
		"service inspect outback-job-1": `[{
          "CreatedAt":"2026-08-01T10:00:00Z","UpdatedAt":"2026-08-01T10:01:00Z",
          "Spec":{"Name":"outback-job-1","Labels":{"outback.managed":"true","outback.cancelled":"true"},
          "TaskTemplate":{"ContainerSpec":{"Args":["sleep","60"]}}}
        }]`,
		"service ps -q --no-trunc outback-job-1": "",
	}}
	job, err := newClient(commands).Status(context.Background(), "outback-job-1")
	if err != nil {
		t.Fatal(err)
	}
	if job.Status != protocol.StatusCancelled || job.FinishedAt == nil {
		t.Fatalf("job = %#v", job)
	}
}

func TestLogsUseDockerServiceLogDriver(t *testing.T) {
	commands := &fakeCommander{outputs: map[string]string{
		"service logs --raw outback-job-1": "first\nsecond\n",
	}}
	client := newClient(commands)
	var output bytes.Buffer
	if err := client.Logs(context.Background(), "outback-job-1", false, &output); err != nil {
		t.Fatal(err)
	}
	if output.String() != "first\nsecond\n" {
		t.Fatalf("logs = %q", output.String())
	}
}

func TestCreateRejectsMissingProjectImage(t *testing.T) {
	commands := &fakeCommander{outputs: map[string]string{}}
	if _, err := newClient(commands).Create(context.Background(), Spec{ID: "outback-job-1"}); err == nil || !strings.Contains(err.Error(), "image") {
		t.Fatalf("Create() error = %v", err)
	}
	if len(commands.runs) != 0 {
		t.Fatalf("Docker invoked for invalid spec: %#v", commands.runs)
	}
}
