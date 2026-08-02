package console

import (
	"context"
	"errors"
	"io"
	"sort"
	"strings"
	"time"

	"github.com/flidai/autback/internal/control"
	controlsqlite "github.com/flidai/autback/internal/control/sqlite"
)

const maxLogTailBytes = 64 * 1024

type SQLiteSourceConfig struct {
	Store     *controlsqlite.Store
	Scheduler control.Scheduler
	Version   string
	StartedAt time.Time
}

type SQLiteSource struct {
	store     *controlsqlite.Store
	scheduler control.Scheduler
	version   string
	startedAt time.Time
}

func NewSQLiteSource(config SQLiteSourceConfig) (*SQLiteSource, error) {
	if config.Store == nil || config.Scheduler == nil {
		return nil, errors.New("console store and scheduler are required")
	}
	if config.Version == "" {
		return nil, errors.New("console service version is required")
	}
	if config.StartedAt.IsZero() {
		config.StartedAt = time.Now().UTC()
	}
	return &SQLiteSource{store: config.Store, scheduler: config.Scheduler, version: config.Version, startedAt: config.StartedAt.UTC()}, nil
}

func (s *SQLiteSource) Authenticate(ctx context.Context, token string) (control.Principal, error) {
	return s.store.Authenticate(ctx, token)
}

func (s *SQLiteSource) SubscribeChanges() (<-chan struct{}, func()) {
	return s.store.SubscribeChanges()
}

func (s *SQLiteSource) Authorize(ctx context.Context, principal control.Principal, route Route) error {
	switch route.Kind {
	case RouteOverview, RouteAudit:
		_, err := s.store.ListProjects(ctx, principal)
		return err
	case RouteProject:
		_, err := s.store.AuthorizeProject(ctx, principal, route.Project)
		return err
	case RouteOperation:
		projectID := ""
		switch route.OperationKind {
		case "job":
			job, err := s.store.Job(ctx, route.OperationID)
			if err != nil {
				return err
			}
			projectID = job.ProjectID
		case "build":
			build, err := s.store.Build(ctx, route.OperationID)
			if err != nil {
				return err
			}
			projectID = build.ProjectID
		default:
			return control.ErrNotFound
		}
		_, err := s.store.AuthorizeProject(ctx, principal, projectID)
		return err
	default:
		return control.ErrNotFound
	}
}

func (s *SQLiteSource) Snapshot(ctx context.Context, principal control.Principal, route Route) (Snapshot, error) {
	for attempt := 0; attempt < 5; attempt++ {
		before, err := s.store.CurrentRevision(ctx)
		if err != nil {
			return Snapshot{}, err
		}
		snapshot, err := s.readSnapshot(ctx, principal, route)
		if err != nil {
			return Snapshot{}, err
		}
		after, err := s.store.CurrentRevision(ctx)
		if err != nil {
			return Snapshot{}, err
		}
		if before == after {
			snapshot.Revision = after
			return snapshot, nil
		}
	}
	return Snapshot{}, errors.New("control state changed too quickly to produce a consistent console snapshot")
}

func (s *SQLiteSource) readSnapshot(ctx context.Context, principal control.Principal, route Route) (Snapshot, error) {
	projects, err := s.store.ListProjects(ctx, principal)
	if err != nil {
		return Snapshot{}, err
	}
	userName := principal.Subject
	if principal.UserID != "" {
		user, err := s.store.User(ctx, principal.UserID)
		if err != nil {
			return Snapshot{}, err
		}
		userName = user.Name
	}
	projectViews := make([]ProjectView, 0, len(projects))
	projectByID := make(map[string]control.Project, len(projects))
	for _, project := range projects {
		members, err := s.store.ProjectMemberCount(ctx, project.ID)
		if err != nil {
			return Snapshot{}, err
		}
		trusts, err := s.store.ListGitHubTrusts(ctx, principal, project.ID)
		if err != nil {
			return Snapshot{}, err
		}
		activeTrusts := 0
		for _, trust := range trusts {
			if trust.RevokedAt == nil {
				activeTrusts++
			}
		}
		projectByID[project.ID] = project
		projectViews = append(projectViews, ProjectView{
			ID: project.ID, Slug: project.Slug, Name: project.Name, ActiveImage: project.ActiveImage,
			AllowImageOverrides: project.AllowImageOverrides, Members: members, Trusts: activeTrusts,
		})
	}

	selectedProjectID := ""
	if route.Kind == RouteProject {
		project, err := s.store.AuthorizeProject(ctx, principal, route.Project)
		if err != nil {
			return Snapshot{}, err
		}
		selectedProjectID = project.ID
	}

	queueRecords, err := s.store.ListQueue(ctx, principal)
	if err != nil {
		return Snapshot{}, err
	}
	queue := make([]QueueView, 0, len(queueRecords))
	activeID := ""
	for index, item := range queueRecords {
		if selectedProjectID != "" && item.ProjectID != selectedProjectID {
			continue
		}
		project := projectByID[item.ProjectID]
		queue = append(queue, QueueView{
			Position: index + 1, Kind: string(item.Kind), ID: item.ID, Project: project.Slug,
			ProjectName: project.Name, Status: string(item.State), AcceptedAt: item.AcceptedAt, LeasedAt: item.LeasedAt,
		})
		if item.State == control.OperationActive {
			activeID = item.ID
		}
	}

	operations, err := s.operations(ctx, projects, selectedProjectID)
	if err != nil {
		return Snapshot{}, err
	}
	var detail *OperationDetailView
	log := LogView{}
	if route.Kind == RouteOperation {
		detail, log, err = s.operation(ctx, principal, projectByID, route)
		if err != nil {
			return Snapshot{}, err
		}
	}
	auditProject := selectedProjectID
	audits, err := s.store.ListAuditEvents(ctx, principal, auditProject, 80)
	if err != nil {
		return Snapshot{}, err
	}
	auditViews := make([]AuditView, 0, len(audits))
	for _, event := range audits {
		actor := event.ActorID
		if event.ActorID == principal.UserID {
			actor = userName
		}
		project := projectByID[event.ProjectID]
		metadata := event.Metadata
		if metadata == nil {
			metadata = map[string]string{}
		}
		auditViews = append(auditViews, AuditView{ID: event.ID, Actor: actor, Action: event.Action, Target: event.TargetID, Project: project.Slug, Metadata: metadata, CreatedAt: event.CreatedAt})
	}
	workerStatus := "online"
	statusMessage := "Live from SQLite"
	if err := s.scheduler.Check(ctx); err != nil {
		workerStatus, statusMessage = "degraded", "Worker unavailable"
	}
	now := time.Now().UTC()
	return Snapshot{
		Session: SessionView{User: userName, Admin: principal.Admin, Projects: projectViews},
		Service: ServiceView{Name: "Autback", Version: s.version, Control: "CLI only", Admission: "Strict FIFO", StartedAt: s.startedAt},
		Worker:  WorkerView{Status: workerStatus, Capacity: "1 operation", ActiveID: activeID, UpdatedAt: now},
		Queue:   queue, Operations: operations, Operation: detail, Log: log, Audit: auditViews,
		Status: StatusView{Ready: true, Route: string(route.Kind), Message: statusMessage, UpdatedAt: now},
	}, nil
}

func (s *SQLiteSource) operations(ctx context.Context, projects []control.Project, selectedProjectID string) ([]OperationView, error) {
	operations := make([]OperationView, 0)
	for _, project := range projects {
		if selectedProjectID != "" && project.ID != selectedProjectID {
			continue
		}
		jobs, err := s.store.ListJobs(ctx, project.ID, 24, "")
		if err != nil {
			return nil, err
		}
		for _, job := range jobs.Jobs {
			operations = append(operations, jobView(job, project))
		}
		builds, err := s.store.ListBuilds(ctx, project.ID, 24)
		if err != nil {
			return nil, err
		}
		for _, build := range builds {
			operations = append(operations, buildView(build, project))
		}
	}
	sort.Slice(operations, func(i, j int) bool {
		if operations[i].CreatedAt.Equal(operations[j].CreatedAt) {
			return operations[i].ID > operations[j].ID
		}
		return operations[i].CreatedAt.After(operations[j].CreatedAt)
	})
	if len(operations) > 40 {
		operations = operations[:40]
	}
	return operations, nil
}

func (s *SQLiteSource) operation(ctx context.Context, principal control.Principal, projects map[string]control.Project, route Route) (*OperationDetailView, LogView, error) {
	switch route.OperationKind {
	case "job":
		job, err := s.store.Job(ctx, route.OperationID)
		if err != nil {
			return nil, LogView{}, err
		}
		project, err := s.store.AuthorizeProject(ctx, principal, job.ProjectID)
		if err != nil {
			return nil, LogView{}, err
		}
		projects[project.ID] = project
		environment := job.Environment
		if environment == nil {
			environment = map[string]string{}
		}
		view := &OperationDetailView{
			OperationView: jobView(job, project), WorkingDirectory: job.WorkingDirectory,
			Environment: environment, Caches: make([]CacheView, 0, len(job.Caches)), RootDigest: job.RootDigest, Error: job.ErrorMessage,
		}
		for _, cache := range job.Caches {
			view.Caches = append(view.Caches, CacheView{Name: cache.Name, Target: cache.Target})
		}
		writer := &tailWriter{limit: maxLogTailBytes}
		err = s.scheduler.Logs(ctx, job.ID, false, writer)
		return view, LogView{Available: err == nil, Truncated: writer.truncated, Content: string(writer.bytes)}, nil
	case "build":
		build, err := s.store.Build(ctx, route.OperationID)
		if err != nil {
			return nil, LogView{}, err
		}
		project, err := s.store.AuthorizeProject(ctx, principal, build.ProjectID)
		if err != nil {
			return nil, LogView{}, err
		}
		view := &OperationDetailView{OperationView: buildView(build, project), Environment: map[string]string{}, Caches: []CacheView{}}
		return view, LogView{}, nil
	default:
		return nil, LogView{}, control.ErrNotFound
	}
}

func jobView(job control.Job, project control.Project) OperationView {
	return OperationView{
		Kind: "job", ID: job.ID, Project: project.Slug, ProjectName: project.Name,
		Status: string(job.Status), Command: strings.Join(job.Command, " "), Image: job.Image,
		CreatedAt: job.CreatedAt, StartedAt: job.StartedAt, FinishedAt: job.FinishedAt, ExitCode: job.ExitCode,
	}
}

func buildView(build control.Build, project control.Project) OperationView {
	return OperationView{
		Kind: "build", ID: build.ID, Project: project.Slug, ProjectName: project.Name,
		Status: string(build.Status), CreatedAt: build.CreatedAt, FinishedAt: build.FinishedAt, ExitCode: build.ExitCode,
	}
}

type tailWriter struct {
	limit     int
	bytes     []byte
	truncated bool
}

func (w *tailWriter) Write(input []byte) (int, error) {
	written := len(input)
	if w.limit <= 0 {
		w.truncated = w.truncated || len(input) > 0
		return written, nil
	}
	if len(input) >= w.limit {
		w.bytes = append(w.bytes[:0], input[len(input)-w.limit:]...)
		w.truncated = true
		return written, nil
	}
	if overflow := len(w.bytes) + len(input) - w.limit; overflow > 0 {
		copy(w.bytes, w.bytes[overflow:])
		w.bytes = w.bytes[:len(w.bytes)-overflow]
		w.truncated = true
	}
	w.bytes = append(w.bytes, input...)
	return written, nil
}

var _ io.Writer = (*tailWriter)(nil)
