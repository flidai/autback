package control

import (
	"errors"
	"time"

	"github.com/flidai/autback/internal/protocol"
)

var (
	ErrAlreadyExists       = errors.New("already exists")
	ErrForbidden           = errors.New("forbidden")
	ErrIdempotencyConflict = errors.New("idempotency key was already used with a different request")
	ErrInvalidPageToken    = errors.New("invalid page token")
	ErrNotFound            = errors.New("not found")
	ErrUnauthenticated     = errors.New("unauthenticated")
)

type PrincipalKind string

const (
	PrincipalDevice PrincipalKind = "device"
	PrincipalGitHub PrincipalKind = "github"
	PrincipalSystem PrincipalKind = "system"
)

type Principal struct {
	Kind      PrincipalKind
	TokenID   string
	UserID    string
	ProjectID string
	Admin     bool
	Subject   string
}

type User struct {
	ID        string
	Name      string
	Admin     bool
	CreatedAt time.Time
}

type Project struct {
	ID                  string
	Slug                string
	Name                string
	ActiveImage         string
	PreviousImage       string
	AllowImageOverrides bool
	CreatedAt           time.Time
}

type ProjectImageEvent struct {
	ID            string
	ProjectID     string
	Action        string
	Image         string
	ReplacedImage string
	Actor         string
	CreatedAt     time.Time
}

type Bootstrap struct {
	UserName    string
	ProjectSlug string
	ProjectName string
	TokenName   string
}

type BootstrapResult struct {
	User    User
	Project Project
	Token   string
}

type DeviceToken struct {
	ID         string
	Name       string
	UserID     string
	CreatedAt  time.Time
	ExpiresAt  *time.Time
	LastUsedAt *time.Time
	RevokedAt  *time.Time
}

type CreateDeviceToken struct {
	UserID    string
	Name      string
	ExpiresAt time.Time
}

type IssuedDeviceToken struct {
	Metadata DeviceToken
	Secret   string
}

type EnrollmentCode struct {
	ID             string
	UserID         string
	DeviceName     string
	CreatedAt      time.Time
	ExpiresAt      time.Time
	ConsumedAt     *time.Time
	FailedAttempts int
	MaxAttempts    int
}

type IssuedEnrollmentCode struct {
	Metadata EnrollmentCode
	Secret   string
}

type GitHubTrust struct {
	ID                string
	ProjectID         string
	RepositoryOwnerID string
	RepositoryID      string
	WorkflowRef       string
	Ref               string
	Environment       string
	Events            []string
	CreatedAt         time.Time
	RevokedAt         *time.Time
}

type GitHubClaims struct {
	Subject           string
	RepositoryOwnerID string
	RepositoryID      string
	Repository        string
	WorkflowRef       string
	Ref               string
	Environment       string
	EventName         string
	ExpiresAt         time.Time
}

type IssuedAccessToken struct {
	Token     string
	ExpiresAt time.Time
}

type Job struct {
	ID               string
	ProjectID        string
	Image            string
	Command          []string
	WorkingDirectory string
	Environment      map[string]string
	RootDigest       string
	Status           protocol.Status
	Timeout          time.Duration
	CreatedAt        time.Time
	StartedAt        *time.Time
	FinishedAt       *time.Time
	ExitCode         *int
	ErrorMessage     string
	CancelRequested  bool
	WorkerID         string
	Caches           []CacheMount
	Secrets          []SecretBinding
}

type CacheMount struct {
	Name   string
	Target string
}

type SecretBinding struct {
	Name        string
	Environment string
	File        string
}

type PrepareJob struct {
	ProjectID        string
	Image            string
	Command          []string
	WorkingDirectory string
	Environment      map[string]string
	Timeout          time.Duration
	Caches           []CacheMount
	Secrets          []SecretBinding
}

type Idempotency struct {
	Key         string
	RequestHash string
}

type JobPage struct {
	Jobs          []Job
	NextPageToken string
}

type BuildStatus string

const (
	BuildQueued    BuildStatus = "queued"
	BuildRunning   BuildStatus = "running"
	BuildSucceeded BuildStatus = "succeeded"
	BuildFailed    BuildStatus = "failed"
	BuildCancelled BuildStatus = "cancelled"
)

type OperationKind string

const (
	OperationJob   OperationKind = "job"
	OperationBuild OperationKind = "build"
)

type OperationState string

const (
	OperationQueued        OperationState = "queued"
	OperationAdmitting     OperationState = "admitting"
	OperationActive        OperationState = "active"
	OperationTerminalizing OperationState = "terminalizing"
	OperationCleaning      OperationState = "cleaning"
	OperationReleased      OperationState = "released"
)

type Operation struct {
	Kind             OperationKind
	ID               string
	State            OperationState
	AcceptedAt       time.Time
	LeasedAt         *time.Time
	CleanupAttempts  int
	CleanupError     string
	CleanupUpdatedAt *time.Time
}

type QueueOperation struct {
	Operation
	ProjectID string
}

type AuditEvent struct {
	ID        int64
	ActorKind PrincipalKind
	ActorID   string
	ProjectID string
	Action    string
	TargetID  string
	CreatedAt time.Time
	Metadata  map[string]string
}

// ControlChange is durable sequencing and scope metadata for consumers that
// project control-plane state. It deliberately contains no duplicated domain
// state; readers re-query their authorized view after observing a change.
type ControlChange struct {
	Sequence   int64
	ProjectID  string
	EntityKind string
	EntityID   string
	CreatedAt  time.Time
}

// ResourceScope identifies which admitted operation, if any, owned the host
// while a resource sample was observed. Empty operation fields represent an
// idle host sample.
type ResourceScope struct {
	ProjectID     string
	OperationKind OperationKind
	OperationID   string
}

// ResourceSample is a normalized observation of the entire Autback host. CPU
// and memory utilization are ratios in the inclusive range [0, 1].
type ResourceSample struct {
	ResourceScope
	ObservedAt        time.Time
	CPUUtilization    float64
	CPUCores          int
	MemoryUtilization float64
	MemoryUsageBytes  uint64
	MemoryTotalBytes  uint64
	DiskUsageBytes    uint64
	DiskTotalBytes    uint64
}

type ResourceFilter struct {
	ProjectID     string
	OperationKind OperationKind
	OperationID   string
	From          time.Time
	To            time.Time
}

type ResourceSummary struct {
	ResourceScope
	SampleCount        int
	ObservedStartedAt  time.Time
	ObservedFinishedAt time.Time
	CPUAverage         float64
	CPUPeak            float64
	MemoryAverage      float64
	MemoryPeak         float64
	MemoryBytesPeak    uint64
}

type ResourceRollup struct {
	ResourceScope
	BucketAt        time.Time
	SampleCount     int
	CPUAverage      float64
	CPUPeak         float64
	MemoryAverage   float64
	MemoryPeak      float64
	MemoryBytesPeak uint64
	DiskUsageBytes  uint64
	DiskTotalBytes  uint64
	CPUCores        int
}

type Build struct {
	ID         string
	ProjectID  string
	Status     BuildStatus
	CreatedAt  time.Time
	FinishedAt *time.Time
	ExitCode   *int
}
