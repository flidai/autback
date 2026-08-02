package protocol

import "time"

type Status string

const (
	StatusPreparing Status = "preparing"
	StatusQueued    Status = "queued"
	StatusRunning   Status = "running"
	StatusSucceeded Status = "succeeded"
	StatusFailed    Status = "failed"
	StatusCancelled Status = "cancelled"
	StatusTimedOut  Status = "timed_out"
	StatusLost      Status = "lost"
)

func (s Status) Terminal() bool {
	switch s {
	case StatusSucceeded, StatusFailed, StatusCancelled, StatusTimedOut, StatusLost:
		return true
	default:
		return false
	}
}

type Job struct {
	ID              string     `json:"id"`
	ProjectID       string     `json:"project_id"`
	Image           string     `json:"image"`
	Command         []string   `json:"command"`
	RootDigest      string     `json:"root_digest"`
	Status          Status     `json:"status"`
	WorkerID        string     `json:"worker_id,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	StartedAt       *time.Time `json:"started_at,omitempty"`
	FinishedAt      *time.Time `json:"finished_at,omitempty"`
	TimeoutSeconds  int        `json:"timeout_seconds"`
	CancelRequested bool       `json:"cancel_requested"`
	ExitCode        *int       `json:"exit_code,omitempty"`
	ErrorMessage    string     `json:"error_message,omitempty"`
}
