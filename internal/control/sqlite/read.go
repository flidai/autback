package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/flidai/autback/internal/control"
)

func (s *Store) User(ctx context.Context, id string) (control.User, error) {
	var user control.User
	var admin int
	var created int64
	err := s.db.QueryRowContext(ctx, `SELECT id,name,admin,created_at FROM users WHERE id=?`, id).Scan(&user.ID, &user.Name, &admin, &created)
	if errors.Is(err, sql.ErrNoRows) {
		return control.User{}, control.ErrNotFound
	}
	user.Admin, user.CreatedAt = admin != 0, fromUnix(created)
	return user, err
}

func (s *Store) ProjectMemberCount(ctx context.Context, projectID string) (int, error) {
	var count int
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM project_members WHERE project_id=?`, projectID).Scan(&count)
	return count, err
}

func (s *Store) ListBuilds(ctx context.Context, projectID string, limit int) ([]control.Build, error) {
	if limit < 1 || limit > 100 {
		return nil, errors.New("build limit must be between 1 and 100")
	}
	rows, err := s.db.QueryContext(ctx, `SELECT id,project_id,status,created_at,finished_at,exit_code FROM control_builds WHERE project_id=? ORDER BY created_at DESC,id DESC LIMIT ?`, projectID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	builds := make([]control.Build, 0)
	for rows.Next() {
		var build control.Build
		var created int64
		var finished, exitCode sql.NullInt64
		if err := rows.Scan(&build.ID, &build.ProjectID, &build.Status, &created, &finished, &exitCode); err != nil {
			return nil, err
		}
		build.CreatedAt, build.FinishedAt = fromUnix(created), nullableTime(finished)
		if exitCode.Valid {
			value := int(exitCode.Int64)
			build.ExitCode = &value
		}
		builds = append(builds, build)
	}
	return builds, rows.Err()
}

func (s *Store) ListQueue(ctx context.Context, principal control.Principal) ([]control.QueueOperation, error) {
	projects, err := s.ListProjects(ctx, principal)
	if err != nil {
		return nil, err
	}
	authorized := make(map[string]struct{}, len(projects))
	for _, project := range projects {
		authorized[project.ID] = struct{}{}
	}
	rows, err := s.db.QueryContext(ctx, `SELECT q.kind,q.operation_id,q.state,q.accepted_at,q.leased_at,
COALESCE(j.project_id,b.project_id,'')
FROM control_queue q
LEFT JOIN control_jobs j ON q.kind='job' AND j.id=q.operation_id
LEFT JOIN control_builds b ON q.kind='build' AND b.id=q.operation_id
WHERE q.state<>?
ORDER BY q.sequence`, control.OperationReleased)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	queue := make([]control.QueueOperation, 0)
	for rows.Next() {
		var item control.QueueOperation
		var accepted int64
		var leased sql.NullInt64
		if err := rows.Scan(&item.Kind, &item.ID, &item.State, &accepted, &leased, &item.ProjectID); err != nil {
			return nil, err
		}
		if item.ProjectID == "" {
			continue
		}
		if _, ok := authorized[item.ProjectID]; !ok {
			continue
		}
		item.AcceptedAt = time.Unix(0, accepted).UTC()
		if leased.Valid {
			value := time.Unix(0, leased.Int64).UTC()
			item.LeasedAt = &value
		}
		queue = append(queue, item)
	}
	return queue, rows.Err()
}

func (s *Store) ListAuditEvents(ctx context.Context, principal control.Principal, projectID string, limit int) ([]control.AuditEvent, error) {
	if limit < 1 || limit > 200 {
		return nil, errors.New("audit limit must be between 1 and 200")
	}
	query := `SELECT id,actor_kind,actor_id,COALESCE(project_id,''),action,target_id,created_at,metadata_json FROM audit_events`
	arguments := make([]any, 0, 3)
	if projectID != "" {
		project, err := s.AuthorizeProject(ctx, principal, projectID)
		if err != nil {
			return nil, err
		}
		query += ` WHERE project_id=?`
		arguments = append(arguments, project.ID)
	} else if principal.Kind == control.PrincipalGitHub {
		query += ` WHERE project_id=?`
		arguments = append(arguments, principal.ProjectID)
	} else if !principal.Admin {
		query += ` WHERE project_id IN (SELECT project_id FROM project_members WHERE user_id=?) OR (project_id IS NULL AND actor_id=?)`
		arguments = append(arguments, principal.UserID, principal.UserID)
	}
	query += ` ORDER BY created_at DESC,id DESC LIMIT ?`
	arguments = append(arguments, limit)
	rows, err := s.db.QueryContext(ctx, query, arguments...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	events := make([]control.AuditEvent, 0)
	for rows.Next() {
		var event control.AuditEvent
		var created int64
		var metadata string
		if err := rows.Scan(&event.ID, &event.ActorKind, &event.ActorID, &event.ProjectID, &event.Action, &event.TargetID, &created, &metadata); err != nil {
			return nil, err
		}
		if err := json.Unmarshal([]byte(metadata), &event.Metadata); err != nil {
			return nil, err
		}
		event.CreatedAt = fromUnix(created)
		events = append(events, event)
	}
	return events, rows.Err()
}
