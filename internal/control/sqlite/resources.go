package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/flidai/autback/internal/control"
)

const minuteNanoseconds = int64(time.Minute)

func (s *Store) migrateResourceMetrics(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, `
CREATE TABLE IF NOT EXISTS resource_samples (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  observed_at INTEGER NOT NULL,
  project_id TEXT NOT NULL DEFAULT '',
  operation_kind TEXT NOT NULL DEFAULT '',
  operation_id TEXT NOT NULL DEFAULT '',
  cpu_utilization REAL NOT NULL,
  cpu_cores INTEGER NOT NULL,
  memory_utilization REAL NOT NULL,
  memory_usage_bytes INTEGER NOT NULL,
  memory_total_bytes INTEGER NOT NULL,
  disk_usage_bytes INTEGER NOT NULL,
  disk_total_bytes INTEGER NOT NULL,
  disk_inodes_used INTEGER NOT NULL DEFAULT 0,
  disk_inodes_total INTEGER NOT NULL DEFAULT 0,
  cpu_pressure REAL NOT NULL DEFAULT 0,
  memory_pressure REAL NOT NULL DEFAULT 0,
  memory_full_pressure REAL NOT NULL DEFAULT 0,
  io_pressure REAL NOT NULL DEFAULT 0,
  io_full_pressure REAL NOT NULL DEFAULT 0,
  memory_high_events INTEGER NOT NULL DEFAULT 0,
  oom_events INTEGER NOT NULL DEFAULT 0,
  oom_kills INTEGER NOT NULL DEFAULT 0,
  pids_current INTEGER NOT NULL DEFAULT 0,
  pids_limit INTEGER NOT NULL DEFAULT 0
);
CREATE INDEX IF NOT EXISTS resource_samples_time_idx ON resource_samples(observed_at);
CREATE INDEX IF NOT EXISTS resource_samples_operation_idx ON resource_samples(operation_kind,operation_id,observed_at);
CREATE INDEX IF NOT EXISTS resource_samples_project_idx ON resource_samples(project_id,observed_at);
CREATE TABLE IF NOT EXISTS resource_minute_rollups (
  bucket_at INTEGER NOT NULL,
  project_id TEXT NOT NULL DEFAULT '',
  operation_kind TEXT NOT NULL DEFAULT '',
  operation_id TEXT NOT NULL DEFAULT '',
  sample_count INTEGER NOT NULL,
  cpu_sum REAL NOT NULL,
  cpu_peak REAL NOT NULL,
  memory_sum REAL NOT NULL,
  memory_peak REAL NOT NULL,
  memory_bytes_peak INTEGER NOT NULL,
  disk_usage_bytes INTEGER NOT NULL,
  disk_total_bytes INTEGER NOT NULL,
  cpu_cores INTEGER NOT NULL,
  PRIMARY KEY(bucket_at,project_id,operation_kind,operation_id)
);
CREATE INDEX IF NOT EXISTS resource_rollups_operation_idx ON resource_minute_rollups(operation_kind,operation_id,bucket_at);
CREATE TABLE IF NOT EXISTS resource_operation_summaries (
  operation_kind TEXT NOT NULL,
  operation_id TEXT NOT NULL,
  project_id TEXT NOT NULL DEFAULT '',
  sample_count INTEGER NOT NULL,
  observed_started_at INTEGER NOT NULL,
  observed_finished_at INTEGER NOT NULL,
  cpu_sum REAL NOT NULL,
  cpu_peak REAL NOT NULL,
  memory_sum REAL NOT NULL,
  memory_peak REAL NOT NULL,
  memory_bytes_peak INTEGER NOT NULL,
  PRIMARY KEY(operation_kind,operation_id)
);
CREATE INDEX IF NOT EXISTS resource_summaries_project_idx ON resource_operation_summaries(project_id,observed_finished_at DESC);
`)
	if err != nil {
		return err
	}
	columns := []struct{ name, kind, fallback string }{
		{"disk_inodes_used", "INTEGER", "0"}, {"disk_inodes_total", "INTEGER", "0"},
		{"cpu_pressure", "REAL", "0"}, {"memory_pressure", "REAL", "0"}, {"memory_full_pressure", "REAL", "0"},
		{"io_pressure", "REAL", "0"}, {"io_full_pressure", "REAL", "0"},
		{"memory_high_events", "INTEGER", "0"}, {"oom_events", "INTEGER", "0"}, {"oom_kills", "INTEGER", "0"},
		{"pids_current", "INTEGER", "0"}, {"pids_limit", "INTEGER", "0"},
	}
	for _, column := range columns {
		if err := s.ensureResourceSampleColumn(ctx, column.name, column.kind, column.fallback); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) ensureResourceSampleColumn(ctx context.Context, column, kind, fallback string) error {
	allowed := map[string]string{
		"disk_inodes_used": "INTEGER", "disk_inodes_total": "INTEGER", "cpu_pressure": "REAL", "memory_pressure": "REAL",
		"memory_full_pressure": "REAL", "io_pressure": "REAL", "io_full_pressure": "REAL", "memory_high_events": "INTEGER",
		"oom_events": "INTEGER", "oom_kills": "INTEGER", "pids_current": "INTEGER", "pids_limit": "INTEGER",
	}
	if allowed[column] != kind || fallback != "0" {
		return errors.New("unsupported resource sample migration column")
	}
	rows, err := s.db.QueryContext(ctx, `PRAGMA table_info(resource_samples)`)
	if err != nil {
		return err
	}
	found := false
	for rows.Next() {
		var cid, notNull, primaryKey int
		var name, columnKind string
		var defaultValue any
		if err := rows.Scan(&cid, &name, &columnKind, &notNull, &defaultValue, &primaryKey); err != nil {
			_ = rows.Close()
			return err
		}
		found = found || name == column
	}
	if err := rows.Close(); err != nil || found {
		return err
	}
	_, err = s.db.ExecContext(ctx, `ALTER TABLE resource_samples ADD COLUMN `+column+` `+kind+` NOT NULL DEFAULT `+fallback)
	return err
}

func (s *Store) ActiveResourceScope(ctx context.Context) (control.ResourceScope, bool, error) {
	var scope control.ResourceScope
	err := s.db.QueryRowContext(ctx, `SELECT q.kind,q.operation_id,
COALESCE(j.project_id,b.project_id,'')
FROM control_queue q
LEFT JOIN control_jobs j ON q.kind='job' AND j.id=q.operation_id
LEFT JOIN control_builds b ON q.kind='build' AND b.id=q.operation_id
WHERE q.state=? LIMIT 1`, control.OperationActive).Scan(&scope.OperationKind, &scope.OperationID, &scope.ProjectID)
	if errors.Is(err, sql.ErrNoRows) {
		return control.ResourceScope{}, false, nil
	}
	return scope, err == nil, err
}

func (s *Store) AppendResourceSample(ctx context.Context, sample control.ResourceSample) error {
	if err := validateResourceSample(sample); err != nil {
		return err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	_, err = tx.ExecContext(ctx, `INSERT INTO resource_samples(
observed_at,project_id,operation_kind,operation_id,cpu_utilization,cpu_cores,memory_utilization,memory_usage_bytes,memory_total_bytes,disk_usage_bytes,disk_total_bytes,
disk_inodes_used,disk_inodes_total,cpu_pressure,memory_pressure,memory_full_pressure,io_pressure,io_full_pressure,memory_high_events,oom_events,oom_kills,pids_current,pids_limit)
VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`, sample.ObservedAt.UnixNano(), sample.ProjectID, sample.OperationKind, sample.OperationID,
		sample.CPUUtilization, sample.CPUCores, sample.MemoryUtilization, sample.MemoryUsageBytes, sample.MemoryTotalBytes, sample.DiskUsageBytes, sample.DiskTotalBytes,
		sample.DiskInodesUsed, sample.DiskInodesTotal, sample.CPUPressure, sample.MemoryPressure, sample.MemoryFullPressure, sample.IOPressure, sample.IOFullPressure,
		sample.MemoryHighEvents, sample.OOMEvents, sample.OOMKills, sample.PIDsCurrent, sample.PIDsLimit)
	if err != nil {
		return err
	}
	if sample.OperationID != "" {
		_, err = tx.ExecContext(ctx, `INSERT INTO resource_operation_summaries(
operation_kind,operation_id,project_id,sample_count,observed_started_at,observed_finished_at,cpu_sum,cpu_peak,memory_sum,memory_peak,memory_bytes_peak)
VALUES(?,?,?,1,?,?,?,?,?,?,?)
ON CONFLICT(operation_kind,operation_id) DO UPDATE SET
project_id=excluded.project_id,
sample_count=resource_operation_summaries.sample_count+1,
observed_started_at=MIN(resource_operation_summaries.observed_started_at,excluded.observed_started_at),
observed_finished_at=MAX(resource_operation_summaries.observed_finished_at,excluded.observed_finished_at),
cpu_sum=resource_operation_summaries.cpu_sum+excluded.cpu_sum,
cpu_peak=MAX(resource_operation_summaries.cpu_peak,excluded.cpu_peak),
memory_sum=resource_operation_summaries.memory_sum+excluded.memory_sum,
memory_peak=MAX(resource_operation_summaries.memory_peak,excluded.memory_peak),
memory_bytes_peak=MAX(resource_operation_summaries.memory_bytes_peak,excluded.memory_bytes_peak)`,
			sample.OperationKind, sample.OperationID, sample.ProjectID, sample.ObservedAt.UnixNano(), sample.ObservedAt.UnixNano(),
			sample.CPUUtilization, sample.CPUUtilization, sample.MemoryUtilization, sample.MemoryUtilization, sample.MemoryUsageBytes)
		if err != nil {
			return err
		}
	}
	entityID := fmt.Sprintf("%d", sample.ObservedAt.UnixNano())
	if sample.OperationID != "" {
		entityID = string(sample.OperationKind) + ":" + sample.OperationID
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO control_changes(project_id,entity_kind,entity_id,created_at) VALUES(NULLIF(?,''),'resource-sample',?,?)`,
		sample.ProjectID, entityID, sample.ObservedAt.UnixNano()); err != nil {
		return err
	}
	return tx.Commit()
}

func validateResourceSample(sample control.ResourceSample) error {
	if sample.ObservedAt.IsZero() || sample.CPUCores < 1 || sample.MemoryTotalBytes == 0 || sample.MemoryUsageBytes > sample.MemoryTotalBytes || sample.DiskUsageBytes > sample.DiskTotalBytes || sample.DiskInodesUsed > sample.DiskInodesTotal {
		return errors.New("resource sample capacities and observation time are invalid")
	}
	if !validRatio(sample.CPUUtilization) || !validRatio(sample.MemoryUtilization) || !validRatio(sample.CPUPressure) || !validRatio(sample.MemoryPressure) || !validRatio(sample.MemoryFullPressure) || !validRatio(sample.IOPressure) || !validRatio(sample.IOFullPressure) {
		return errors.New("resource utilization must be between zero and one")
	}
	if (sample.OperationID == "") != (sample.OperationKind == "") || sample.OperationID != "" && sample.ProjectID == "" {
		return errors.New("resource operation scope is incomplete")
	}
	return nil
}

func validRatio(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0) && value >= 0 && value <= 1
}

func (s *Store) ListResourceSamples(ctx context.Context, filter control.ResourceFilter, limit int) ([]control.ResourceSample, error) {
	if limit < 1 || limit > 5000 {
		return nil, errors.New("resource sample limit must be between 1 and 5000")
	}
	where, arguments := resourceWhere(filter, "observed_at")
	arguments = append(arguments, limit)
	rows, err := s.db.QueryContext(ctx, `SELECT observed_at,project_id,operation_kind,operation_id,cpu_utilization,cpu_cores,memory_utilization,memory_usage_bytes,memory_total_bytes,disk_usage_bytes,disk_total_bytes,
disk_inodes_used,disk_inodes_total,cpu_pressure,memory_pressure,memory_full_pressure,io_pressure,io_full_pressure,memory_high_events,oom_events,oom_kills,pids_current,pids_limit
FROM resource_samples`+where+` ORDER BY observed_at DESC,id DESC LIMIT ?`, arguments...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var reversed []control.ResourceSample
	for rows.Next() {
		var sample control.ResourceSample
		var observed int64
		if err := rows.Scan(&observed, &sample.ProjectID, &sample.OperationKind, &sample.OperationID, &sample.CPUUtilization, &sample.CPUCores,
			&sample.MemoryUtilization, &sample.MemoryUsageBytes, &sample.MemoryTotalBytes, &sample.DiskUsageBytes, &sample.DiskTotalBytes,
			&sample.DiskInodesUsed, &sample.DiskInodesTotal, &sample.CPUPressure, &sample.MemoryPressure, &sample.MemoryFullPressure, &sample.IOPressure, &sample.IOFullPressure,
			&sample.MemoryHighEvents, &sample.OOMEvents, &sample.OOMKills, &sample.PIDsCurrent, &sample.PIDsLimit); err != nil {
			return nil, err
		}
		sample.ObservedAt = time.Unix(0, observed).UTC()
		reversed = append(reversed, sample)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for left, right := 0, len(reversed)-1; left < right; left, right = left+1, right-1 {
		reversed[left], reversed[right] = reversed[right], reversed[left]
	}
	return reversed, nil
}

func (s *Store) ListResourceRollups(ctx context.Context, filter control.ResourceFilter, limit int) ([]control.ResourceRollup, error) {
	if limit < 1 || limit > 5000 {
		return nil, errors.New("resource rollup limit must be between 1 and 5000")
	}
	where, arguments := resourceWhere(filter, "bucket_at")
	arguments = append(arguments, limit)
	rows, err := s.db.QueryContext(ctx, `SELECT bucket_at,project_id,operation_kind,operation_id,sample_count,cpu_sum,cpu_peak,memory_sum,memory_peak,memory_bytes_peak,disk_usage_bytes,disk_total_bytes,cpu_cores
FROM resource_minute_rollups`+where+` ORDER BY bucket_at DESC LIMIT ?`, arguments...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var rollups []control.ResourceRollup
	for rows.Next() {
		var item control.ResourceRollup
		var bucket, memoryBytes int64
		var cpuSum, memorySum float64
		if err := rows.Scan(&bucket, &item.ProjectID, &item.OperationKind, &item.OperationID, &item.SampleCount, &cpuSum, &item.CPUPeak,
			&memorySum, &item.MemoryPeak, &memoryBytes, &item.DiskUsageBytes, &item.DiskTotalBytes, &item.CPUCores); err != nil {
			return nil, err
		}
		item.BucketAt = time.Unix(0, bucket).UTC()
		item.CPUAverage = cpuSum / float64(item.SampleCount)
		item.MemoryAverage = memorySum / float64(item.SampleCount)
		item.MemoryBytesPeak = uint64(memoryBytes)
		rollups = append(rollups, item)
	}
	return rollups, rows.Err()
}

func (s *Store) ResourceSummary(ctx context.Context, filter control.ResourceFilter) (control.ResourceSummary, error) {
	if filter.OperationKind == "" || filter.OperationID == "" {
		return control.ResourceSummary{}, errors.New("resource summary requires an operation")
	}
	var summary control.ResourceSummary
	var started, finished, memoryBytes int64
	var cpuSum, memorySum float64
	err := s.db.QueryRowContext(ctx, `SELECT project_id,sample_count,observed_started_at,observed_finished_at,cpu_sum,cpu_peak,memory_sum,memory_peak,memory_bytes_peak
FROM resource_operation_summaries WHERE operation_kind=? AND operation_id=?`, filter.OperationKind, filter.OperationID).
		Scan(&summary.ProjectID, &summary.SampleCount, &started, &finished, &cpuSum, &summary.CPUPeak, &memorySum, &summary.MemoryPeak, &memoryBytes)
	if errors.Is(err, sql.ErrNoRows) {
		return control.ResourceSummary{}, control.ErrNotFound
	}
	if err != nil {
		return control.ResourceSummary{}, err
	}
	summary.OperationKind, summary.OperationID = filter.OperationKind, filter.OperationID
	summary.ObservedStartedAt, summary.ObservedFinishedAt = time.Unix(0, started).UTC(), time.Unix(0, finished).UTC()
	summary.CPUAverage, summary.MemoryAverage = cpuSum/float64(summary.SampleCount), memorySum/float64(summary.SampleCount)
	summary.MemoryBytesPeak = uint64(memoryBytes)
	return summary, nil
}

func (s *Store) ListResourceSummaries(ctx context.Context, projectID string, limit int) ([]control.ResourceSummary, error) {
	if limit < 1 || limit > 5000 {
		return nil, errors.New("resource summary limit must be between 1 and 5000")
	}
	query := `SELECT operation_kind,operation_id,project_id,sample_count,observed_started_at,observed_finished_at,cpu_sum,cpu_peak,memory_sum,memory_peak,memory_bytes_peak
FROM resource_operation_summaries`
	arguments := []any{}
	if projectID != "" {
		query += ` WHERE project_id=?`
		arguments = append(arguments, projectID)
	}
	query += ` ORDER BY observed_finished_at DESC LIMIT ?`
	arguments = append(arguments, limit)
	rows, err := s.db.QueryContext(ctx, query, arguments...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	summaries := make([]control.ResourceSummary, 0)
	for rows.Next() {
		var summary control.ResourceSummary
		var started, finished, memoryBytes int64
		var cpuSum, memorySum float64
		if err := rows.Scan(&summary.OperationKind, &summary.OperationID, &summary.ProjectID, &summary.SampleCount, &started, &finished,
			&cpuSum, &summary.CPUPeak, &memorySum, &summary.MemoryPeak, &memoryBytes); err != nil {
			return nil, err
		}
		summary.ObservedStartedAt, summary.ObservedFinishedAt = time.Unix(0, started).UTC(), time.Unix(0, finished).UTC()
		summary.CPUAverage, summary.MemoryAverage = cpuSum/float64(summary.SampleCount), memorySum/float64(summary.SampleCount)
		summary.MemoryBytesPeak = uint64(memoryBytes)
		summaries = append(summaries, summary)
	}
	return summaries, rows.Err()
}

func (s *Store) CompactResourceSamples(ctx context.Context, rawBefore, rollupBefore time.Time) error {
	if rawBefore.IsZero() || rollupBefore.IsZero() || !rollupBefore.Before(rawBefore) {
		return errors.New("resource retention boundaries are invalid")
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	_, err = tx.ExecContext(ctx, `INSERT INTO resource_minute_rollups(
bucket_at,project_id,operation_kind,operation_id,sample_count,cpu_sum,cpu_peak,memory_sum,memory_peak,memory_bytes_peak,disk_usage_bytes,disk_total_bytes,cpu_cores)
SELECT (observed_at/?)*?,project_id,operation_kind,operation_id,COUNT(*),SUM(cpu_utilization),MAX(cpu_utilization),SUM(memory_utilization),MAX(memory_utilization),MAX(memory_usage_bytes),MAX(disk_usage_bytes),MAX(disk_total_bytes),MAX(cpu_cores)
FROM resource_samples WHERE observed_at<? GROUP BY (observed_at/?),project_id,operation_kind,operation_id
ON CONFLICT(bucket_at,project_id,operation_kind,operation_id) DO UPDATE SET
sample_count=resource_minute_rollups.sample_count+excluded.sample_count,
cpu_sum=resource_minute_rollups.cpu_sum+excluded.cpu_sum,
cpu_peak=MAX(resource_minute_rollups.cpu_peak,excluded.cpu_peak),
memory_sum=resource_minute_rollups.memory_sum+excluded.memory_sum,
memory_peak=MAX(resource_minute_rollups.memory_peak,excluded.memory_peak),
memory_bytes_peak=MAX(resource_minute_rollups.memory_bytes_peak,excluded.memory_bytes_peak),
disk_usage_bytes=excluded.disk_usage_bytes,disk_total_bytes=excluded.disk_total_bytes,cpu_cores=excluded.cpu_cores`,
		minuteNanoseconds, minuteNanoseconds, rawBefore.UnixNano(), minuteNanoseconds)
	if err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM resource_samples WHERE observed_at<?`, rawBefore.UnixNano()); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM resource_minute_rollups WHERE bucket_at<?`, rollupBefore.UnixNano()); err != nil {
		return err
	}
	return tx.Commit()
}

func resourceWhere(filter control.ResourceFilter, timeColumn string) (string, []any) {
	var clauses []string
	var arguments []any
	if filter.ProjectID != "" {
		clauses, arguments = append(clauses, "project_id=?"), append(arguments, filter.ProjectID)
	}
	if filter.OperationKind != "" {
		clauses, arguments = append(clauses, "operation_kind=?"), append(arguments, filter.OperationKind)
	}
	if filter.OperationID != "" {
		clauses, arguments = append(clauses, "operation_id=?"), append(arguments, filter.OperationID)
	}
	if !filter.From.IsZero() {
		clauses, arguments = append(clauses, timeColumn+">=?"), append(arguments, filter.From.UnixNano())
	}
	if !filter.To.IsZero() {
		clauses, arguments = append(clauses, timeColumn+"<=?"), append(arguments, filter.To.UnixNano())
	}
	if len(clauses) == 0 {
		return "", arguments
	}
	return " WHERE " + strings.Join(clauses, " AND "), arguments
}
