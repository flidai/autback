package console

import "time"

type RouteKind string

const (
	RouteOverview  RouteKind = "overview"
	RouteProject   RouteKind = "project"
	RouteOperation RouteKind = "operation"
	RouteAudit     RouteKind = "audit"
)

type Route struct {
	Kind          RouteKind
	Project       string
	OperationKind string
	OperationID   string
}

type Snapshot struct {
	Revision   int64
	Session    SessionView
	Service    ServiceView
	Worker     WorkerView
	Clock      ClockView
	Resources  ResourceView
	Queue      []QueueView
	Operations []OperationView
	Operation  *OperationDetailView
	Log        LogView
	Audit      []AuditView
	Status     StatusView
}

type SessionView struct {
	User     string        `json:"user"`
	Admin    bool          `json:"admin"`
	Projects []ProjectView `json:"projects"`
}

type ProjectView struct {
	ID                  string `json:"id"`
	Slug                string `json:"slug"`
	Name                string `json:"name"`
	ActiveImage         string `json:"activeImage"`
	AllowImageOverrides bool   `json:"allowImageOverrides"`
	Members             int    `json:"members"`
	Trusts              int    `json:"trusts"`
}

type ServiceView struct {
	Name      string    `json:"name"`
	Version   string    `json:"version"`
	Control   string    `json:"control"`
	Admission string    `json:"admission"`
	StartedAt time.Time `json:"startedAt"`
}

type WorkerView struct {
	Status    string    `json:"status"`
	Capacity  string    `json:"capacity"`
	ActiveID  string    `json:"activeId"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type ClockView struct {
	Now time.Time `json:"now"`
}

type QueueView struct {
	Position    int        `json:"position"`
	Kind        string     `json:"kind"`
	ID          string     `json:"id"`
	Project     string     `json:"project"`
	ProjectName string     `json:"projectName"`
	Status      string     `json:"status"`
	AcceptedAt  time.Time  `json:"acceptedAt"`
	LeasedAt    *time.Time `json:"leasedAt"`
}

// ResourceView describes the shared runner capacity for the selected route.
// Samples are intentionally bounded and normalized so the browser only renders
// an authorized read model, never raw host or database state.
type ResourceView struct {
	Samples            []ResourceSampleView `json:"samples"`
	SampleCount        int                  `json:"sampleCount"`
	ActiveSampleCount  int                  `json:"activeSampleCount"`
	CPUCores           int                  `json:"cpuCores"`
	MemoryTotalBytes   uint64               `json:"memoryTotalBytes"`
	DiskUsageBytes     uint64               `json:"diskUsageBytes"`
	DiskTotalBytes     uint64               `json:"diskTotalBytes"`
	BusyRatio          float64              `json:"busyRatio"`
	CPUAverage         float64              `json:"cpuAverage"`
	CPUPeak            float64              `json:"cpuPeak"`
	MemoryAverage      float64              `json:"memoryAverage"`
	MemoryPeak         float64              `json:"memoryPeak"`
	MemoryBytesPeak    uint64               `json:"memoryBytesPeak"`
	QueueWaitP95Millis int64                `json:"queueWaitP95Millis"`
}

type ResourceSampleView struct {
	ObservedAt        time.Time `json:"observedAt"`
	CPUUtilization    float64   `json:"cpuUtilization"`
	MemoryUtilization float64   `json:"memoryUtilization"`
}

type OperationResourceView struct {
	SampleCount     int     `json:"sampleCount"`
	CPUAverage      float64 `json:"cpuAverage"`
	CPUPeak         float64 `json:"cpuPeak"`
	MemoryAverage   float64 `json:"memoryAverage"`
	MemoryPeak      float64 `json:"memoryPeak"`
	MemoryBytesPeak uint64  `json:"memoryBytesPeak"`
}

type OperationView struct {
	Kind            string                `json:"kind"`
	ID              string                `json:"id"`
	Project         string                `json:"project"`
	ProjectName     string                `json:"projectName"`
	Status          string                `json:"status"`
	Command         string                `json:"command"`
	Image           string                `json:"image"`
	CreatedAt       time.Time             `json:"createdAt"`
	StartedAt       *time.Time            `json:"startedAt"`
	FinishedAt      *time.Time            `json:"finishedAt"`
	ExitCode        *int                  `json:"exitCode"`
	QueueWaitMillis *int64                `json:"queueWaitMillis"`
	Resources       OperationResourceView `json:"resources"`
}

type OperationDetailView struct {
	OperationView
	WorkingDirectory string            `json:"workingDirectory"`
	Environment      map[string]string `json:"environment"`
	Caches           []CacheView       `json:"caches"`
	RootDigest       string            `json:"rootDigest"`
	Error            string            `json:"error"`
}

type CacheView struct {
	Name   string `json:"name"`
	Target string `json:"target"`
}

type LogView struct {
	Available bool   `json:"available"`
	Truncated bool   `json:"truncated"`
	Content   string `json:"content"`
}

type AuditView struct {
	ID        int64             `json:"id"`
	Actor     string            `json:"actor"`
	Action    string            `json:"action"`
	Target    string            `json:"target"`
	Project   string            `json:"project"`
	Metadata  map[string]string `json:"metadata"`
	CreatedAt time.Time         `json:"createdAt"`
}

type StatusView struct {
	Ready     bool      `json:"ready"`
	Route     string    `json:"route"`
	Message   string    `json:"message"`
	UpdatedAt time.Time `json:"updatedAt"`
}

func (s Snapshot) patch() map[string]any {
	return map[string]any{
		"session": s.Session, "service": s.Service, "worker": s.Worker,
		"clock":     s.Clock,
		"resources": s.Resources,
		"queue":     s.Queue, "operations": s.Operations, "operation": s.Operation,
		"log": s.Log, "audit": s.Audit, "status": s.Status,
	}
}
