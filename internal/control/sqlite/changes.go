package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/flidai/autback/internal/control"
	modernsqlite "modernc.org/sqlite"
)

var driverSequence atomic.Uint64

type changeNotifier struct {
	mu          sync.Mutex
	next        uint64
	subscribers map[uint64]chan struct{}
}

func newChangeNotifier() *changeNotifier {
	return &changeNotifier{subscribers: map[uint64]chan struct{}{}}
}

func (n *changeNotifier) subscribe() (<-chan struct{}, func()) {
	n.mu.Lock()
	n.next++
	id := n.next
	updates := make(chan struct{}, 1)
	n.subscribers[id] = updates
	n.mu.Unlock()
	return updates, func() {
		n.mu.Lock()
		if current, ok := n.subscribers[id]; ok {
			delete(n.subscribers, id)
			close(current)
		}
		n.mu.Unlock()
	}
}

func (n *changeNotifier) notify() {
	n.mu.Lock()
	defer n.mu.Unlock()
	for _, updates := range n.subscribers {
		select {
		case updates <- struct{}{}:
		default:
		}
	}
}

func openDatabase(dsn string, notifier *changeNotifier) (*sql.DB, error) {
	name := fmt.Sprintf("autback-sqlite-%d", driverSequence.Add(1))
	var driver modernsqlite.Driver
	driver.RegisterConnectionHook(func(connection modernsqlite.ExecQuerierContext, _ string) error {
		hooks, ok := connection.(modernsqlite.HookRegisterer)
		if !ok {
			return fmt.Errorf("sqlite connection does not support commit hooks")
		}
		var changed atomic.Bool
		hooks.RegisterPreUpdateHook(func(data modernsqlite.SQLitePreUpdateData) {
			if data.TableName == "control_changes" {
				changed.Store(true)
			}
		})
		hooks.RegisterCommitHook(func() int32 {
			if changed.Swap(false) {
				notifier.notify()
			}
			return 0
		})
		hooks.RegisterRollbackHook(func() { changed.Store(false) })
		return nil
	})
	sql.Register(name, &driver)
	return sql.Open(name, dsn)
}

func (s *Store) migrateControlChanges(ctx context.Context) error {
	if _, err := s.db.ExecContext(ctx, `
CREATE TABLE IF NOT EXISTS control_changes (
  sequence INTEGER PRIMARY KEY AUTOINCREMENT,
  project_id TEXT,
  entity_kind TEXT NOT NULL,
  entity_id TEXT NOT NULL,
  created_at INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS control_changes_project_sequence_idx ON control_changes(project_id, sequence);
CREATE TRIGGER IF NOT EXISTS autback_changes_retention AFTER INSERT ON control_changes BEGIN
  DELETE FROM control_changes WHERE sequence < NEW.sequence - 10000;
END;
`); err != nil {
		return err
	}
	type trigger struct {
		table       string
		kind        string
		projectExpr string
		idExpr      string
	}
	triggers := []trigger{
		{table: "users", kind: "user", projectExpr: "NULL", idExpr: "ROW.id"},
		{table: "projects", kind: "project", projectExpr: "ROW.id", idExpr: "ROW.id"},
		{table: "project_members", kind: "project-member", projectExpr: "ROW.project_id", idExpr: "ROW.user_id"},
		{table: "access_tokens", kind: "access-token", projectExpr: "ROW.project_id", idExpr: "ROW.id"},
		{table: "enrollment_codes", kind: "enrollment", projectExpr: "NULL", idExpr: "ROW.id"},
		{table: "github_trusts", kind: "github-trust", projectExpr: "ROW.project_id", idExpr: "ROW.id"},
		{table: "audit_events", kind: "audit", projectExpr: "ROW.project_id", idExpr: "CAST(ROW.id AS TEXT)"},
		{table: "project_image_events", kind: "project-image", projectExpr: "ROW.project_id", idExpr: "ROW.id"},
		{table: "control_jobs", kind: "job", projectExpr: "ROW.project_id", idExpr: "ROW.id"},
		{table: "control_builds", kind: "build", projectExpr: "ROW.project_id", idExpr: "ROW.id"},
		{
			table: "control_queue", kind: "queue",
			projectExpr: "COALESCE((SELECT project_id FROM control_jobs WHERE ROW.kind='job' AND id=ROW.operation_id),(SELECT project_id FROM control_builds WHERE ROW.kind='build' AND id=ROW.operation_id))",
			idExpr:      "ROW.kind || ':' || ROW.operation_id",
		},
	}
	for _, item := range triggers {
		for _, event := range []struct{ name, row string }{{"insert", "NEW"}, {"update", "NEW"}, {"delete", "OLD"}} {
			projectExpression := replaceRow(item.projectExpr, event.row)
			idExpression := replaceRow(item.idExpr, event.row)
			statement := fmt.Sprintf(`CREATE TRIGGER IF NOT EXISTS autback_change_%s_%s AFTER %s ON %s BEGIN
INSERT INTO control_changes(project_id,entity_kind,entity_id,created_at)
VALUES(%s,%q,%s,unixepoch()*1000000000);
END;`, item.table, event.name, event.name, item.table, projectExpression, item.kind, idExpression)
			if _, err := s.db.ExecContext(ctx, statement); err != nil {
				return fmt.Errorf("create %s %s change trigger: %w", item.table, event.name, err)
			}
		}
	}
	return nil
}

func replaceRow(expression, row string) string {
	return strings.ReplaceAll(expression, "ROW.", row+".")
}

// SubscribeChanges wakes the caller after a transaction that appended at
// least one control change commits. Delivery is deliberately coalesced.
func (s *Store) SubscribeChanges() (<-chan struct{}, func()) {
	return s.changes.subscribe()
}

func (s *Store) CurrentRevision(ctx context.Context) (int64, error) {
	var revision int64
	err := s.db.QueryRowContext(ctx, `SELECT COALESCE(MAX(sequence),0) FROM control_changes`).Scan(&revision)
	return revision, err
}

func (s *Store) ChangesAfter(ctx context.Context, revision int64) ([]control.ControlChange, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT sequence,COALESCE(project_id,''),entity_kind,entity_id,created_at FROM control_changes WHERE sequence>? ORDER BY sequence`, revision)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	changes := make([]control.ControlChange, 0)
	for rows.Next() {
		var change control.ControlChange
		var created int64
		if err := rows.Scan(&change.Sequence, &change.ProjectID, &change.EntityKind, &change.EntityID, &created); err != nil {
			return nil, err
		}
		change.CreatedAt = fromUnix(created)
		changes = append(changes, change)
	}
	return changes, rows.Err()
}
