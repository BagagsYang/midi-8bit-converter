package storage

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

const WorkspaceCookieMaxLength = 256

type Store struct {
	db            *sql.DB
	jobRoot       string
	workspaceRoot string
}

type Workspace struct {
	ID         int64
	TokenHash  string
	CreatedAt  float64
	UpdatedAt  float64
	ExpiresAt  float64
	ConfigJSON string
}

type Upload struct {
	ID            int64
	WorkspaceID   int64
	FileID        string
	OriginalName  string
	SizeBytes     int64
	QueuePosition int
	CreatedAt     float64
}

type Job struct {
	ID           int64
	WorkspaceID  int64
	UploadID     sql.NullInt64
	JobID        string
	Status       string
	SourceName   string
	DownloadName sql.NullString
	SizeBytes    sql.NullInt64
	Error        sql.NullString
	CreatedAt    float64
	UpdatedAt    float64
	ExpiresAt    float64
	ConfigJSON   string
}

func Open(ctx context.Context, jobRoot string) (*Store, error) {
	if err := os.MkdirAll(jobRoot, 0o755); err != nil {
		return nil, err
	}
	dbPath := filepath.Join(jobRoot, "workspaces.sqlite3")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)

	store := &Store{
		db:            db,
		jobRoot:       jobRoot,
		workspaceRoot: filepath.Join(jobRoot, "workspaces"),
	}
	if err := store.configure(ctx); err != nil {
		_ = db.Close()
		return nil, err
	}
	if err := store.EnsureSchema(ctx); err != nil {
		_ = db.Close()
		return nil, err
	}
	return store, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) DB() *sql.DB {
	return s.db
}

func (s *Store) WorkspaceDir(workspaceID int64) string {
	return filepath.Join(s.workspaceRoot, fmt.Sprint(workspaceID))
}

func (s *Store) UploadPath(workspaceID int64, fileID string) string {
	return filepath.Join(s.WorkspaceDir(workspaceID), "uploads", fileID+".mid")
}

func (s *Store) JobDir(workspaceID int64, jobID string) string {
	return filepath.Join(s.WorkspaceDir(workspaceID), "jobs", jobID)
}

func (s *Store) JobInputPath(workspaceID int64, jobID string) string {
	return filepath.Join(s.JobDir(workspaceID, jobID), "input.mid")
}

func (s *Store) JobOutputPath(workspaceID int64, jobID string) string {
	return filepath.Join(s.JobDir(workspaceID, jobID), "output.wav")
}

func (s *Store) CreateWorkspace(ctx context.Context, tokenHash string, configJSON string, now time.Time, ttlSeconds int) (Workspace, error) {
	timestamp := unixSeconds(now)
	expiresAt := unixSeconds(now.Add(time.Duration(ttlSeconds) * time.Second))
	result, err := s.db.ExecContext(
		ctx,
		`INSERT INTO workspaces (token_hash, created_at, updated_at, expires_at, config_json)
		 VALUES (?, ?, ?, ?, ?)`,
		tokenHash,
		timestamp,
		timestamp,
		expiresAt,
		configJSON,
	)
	if err != nil {
		return Workspace{}, err
	}
	workspaceID, err := result.LastInsertId()
	if err != nil {
		return Workspace{}, err
	}
	if err := os.MkdirAll(s.WorkspaceDir(workspaceID), 0o755); err != nil {
		return Workspace{}, err
	}
	return Workspace{
		ID:         workspaceID,
		TokenHash:  tokenHash,
		CreatedAt:  timestamp,
		UpdatedAt:  timestamp,
		ExpiresAt:  expiresAt,
		ConfigJSON: configJSON,
	}, nil
}

func (s *Store) GetWorkspaceByTokenHash(ctx context.Context, tokenHash string) (Workspace, bool, error) {
	row := s.db.QueryRowContext(
		ctx,
		`SELECT id, token_hash, created_at, updated_at, expires_at, config_json
		 FROM workspaces
		 WHERE token_hash = ?`,
		tokenHash,
	)
	workspace, err := scanWorkspace(row)
	if err == sql.ErrNoRows {
		return Workspace{}, false, nil
	}
	if err != nil {
		return Workspace{}, false, err
	}
	return workspace, true, nil
}

func (s *Store) TouchWorkspace(ctx context.Context, workspaceID int64, now time.Time, ttlSeconds int) error {
	_, err := s.db.ExecContext(
		ctx,
		`UPDATE workspaces SET updated_at = ?, expires_at = ? WHERE id = ?`,
		unixSeconds(now),
		unixSeconds(now.Add(time.Duration(ttlSeconds)*time.Second)),
		workspaceID,
	)
	return err
}

func (s *Store) DeleteExpiredWorkspaces(ctx context.Context, now time.Time) ([]int64, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id FROM workspaces WHERE expires_at <= ?`, unixSeconds(now))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	for _, id := range ids {
		if _, err := s.db.ExecContext(ctx, `DELETE FROM workspaces WHERE id = ?`, id); err != nil {
			return nil, err
		}
		_ = os.RemoveAll(s.WorkspaceDir(id))
	}
	return ids, nil
}

func (s *Store) InsertUpload(ctx context.Context, upload Upload) (int64, error) {
	result, err := s.db.ExecContext(
		ctx,
		`INSERT INTO uploads (workspace_id, file_id, original_name, size_bytes, queue_position, created_at)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		upload.WorkspaceID,
		upload.FileID,
		upload.OriginalName,
		upload.SizeBytes,
		upload.QueuePosition,
		upload.CreatedAt,
	)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

func (s *Store) InsertJob(ctx context.Context, job Job) (int64, error) {
	result, err := s.db.ExecContext(
		ctx,
		`INSERT INTO jobs (
			workspace_id, upload_id, job_id, status, source_name, download_name,
			size_bytes, error, created_at, updated_at, expires_at, config_json
		 )
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		job.WorkspaceID,
		job.UploadID,
		job.JobID,
		job.Status,
		job.SourceName,
		job.DownloadName,
		job.SizeBytes,
		job.Error,
		job.CreatedAt,
		job.UpdatedAt,
		job.ExpiresAt,
		job.ConfigJSON,
	)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

func (s *Store) CountRows(ctx context.Context, table string) (int, error) {
	var count int
	if err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM "+table).Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}

func (s *Store) configure(ctx context.Context) error {
	for _, statement := range []string{
		"PRAGMA foreign_keys=ON",
		"PRAGMA busy_timeout=5000",
		"PRAGMA journal_mode=WAL",
	} {
		if _, err := s.db.ExecContext(ctx, statement); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) EnsureSchema(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS workspaces (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			token_hash TEXT NOT NULL UNIQUE,
			created_at REAL NOT NULL,
			updated_at REAL NOT NULL,
			expires_at REAL NOT NULL,
			config_json TEXT NOT NULL
		);

		CREATE TABLE IF NOT EXISTS uploads (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			workspace_id INTEGER NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
			file_id TEXT NOT NULL UNIQUE,
			original_name TEXT NOT NULL,
			size_bytes INTEGER NOT NULL,
			queue_position INTEGER NOT NULL,
			created_at REAL NOT NULL
		);

		CREATE INDEX IF NOT EXISTS idx_uploads_workspace_position
			ON uploads(workspace_id, queue_position);

		CREATE TABLE IF NOT EXISTS jobs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			workspace_id INTEGER NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
			upload_id INTEGER REFERENCES uploads(id) ON DELETE SET NULL,
			job_id TEXT NOT NULL UNIQUE,
			status TEXT NOT NULL,
			source_name TEXT NOT NULL,
			download_name TEXT,
			size_bytes INTEGER,
			error TEXT,
			created_at REAL NOT NULL,
			updated_at REAL NOT NULL,
			expires_at REAL NOT NULL,
			config_json TEXT NOT NULL
		);

		CREATE INDEX IF NOT EXISTS idx_jobs_workspace_created
			ON jobs(workspace_id, created_at);
	`)
	return err
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanWorkspace(row rowScanner) (Workspace, error) {
	var workspace Workspace
	err := row.Scan(
		&workspace.ID,
		&workspace.TokenHash,
		&workspace.CreatedAt,
		&workspace.UpdatedAt,
		&workspace.ExpiresAt,
		&workspace.ConfigJSON,
	)
	return workspace, err
}

func TokenHash(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func IsPlausibleToken(token string) bool {
	return len(token) >= 16 && len(token) <= WorkspaceCookieMaxLength
}

func unixSeconds(value time.Time) float64 {
	return float64(value.UnixNano()) / float64(time.Second)
}
