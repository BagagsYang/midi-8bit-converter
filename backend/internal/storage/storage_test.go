package storage

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

func TestOpenConfiguresSQLitePragmasAndSchema(t *testing.T) {
	ctx := context.Background()
	store := openTestStore(t, ctx)
	defer store.Close()

	var foreignKeys int
	if err := store.DB().QueryRowContext(ctx, "PRAGMA foreign_keys").Scan(&foreignKeys); err != nil {
		t.Fatal(err)
	}
	if foreignKeys != 1 {
		t.Fatalf("foreign_keys = %d, want 1", foreignKeys)
	}

	var busyTimeout int
	if err := store.DB().QueryRowContext(ctx, "PRAGMA busy_timeout").Scan(&busyTimeout); err != nil {
		t.Fatal(err)
	}
	if busyTimeout != 5000 {
		t.Fatalf("busy_timeout = %d, want 5000", busyTimeout)
	}

	var journalMode string
	if err := store.DB().QueryRowContext(ctx, "PRAGMA journal_mode").Scan(&journalMode); err != nil {
		t.Fatal(err)
	}
	if journalMode != "wal" {
		t.Fatalf("journal_mode = %q, want wal", journalMode)
	}

	for _, table := range []string{"workspaces", "uploads", "jobs"} {
		t.Run(table, func(t *testing.T) {
			var name string
			err := store.DB().QueryRowContext(
				ctx,
				"SELECT name FROM sqlite_master WHERE type = 'table' AND name = ?",
				table,
			).Scan(&name)
			if err != nil {
				t.Fatal(err)
			}
			if name != table {
				t.Fatalf("table = %q, want %q", name, table)
			}
		})
	}
}

func TestWorkspaceLifecycleAndTokenHashMatchPythonContract(t *testing.T) {
	ctx := context.Background()
	store := openTestStore(t, ctx)
	defer store.Close()

	if !IsPlausibleToken("1234567890abcdef") {
		t.Fatal("expected 16 byte token to be plausible")
	}
	if IsPlausibleToken("short") {
		t.Fatal("short token should not be plausible")
	}

	tokenHash := TokenHash("workspace-token")
	if tokenHash != "05345e1e4c58f23506853a610f26783ecef71ae6a4fa7788ba31304137d3c65c" {
		t.Fatalf("token hash = %s", tokenHash)
	}

	now := time.Unix(100, 0)
	workspace, err := store.CreateWorkspace(ctx, tokenHash, `{"schema":"octabit.workspace_config.v1"}`, now, 60)
	if err != nil {
		t.Fatal(err)
	}
	if workspace.ID == 0 {
		t.Fatal("workspace id was not assigned")
	}
	if _, err := os.Stat(store.WorkspaceDir(workspace.ID)); err != nil {
		t.Fatal(err)
	}

	got, ok, err := store.GetWorkspaceByTokenHash(ctx, tokenHash)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("workspace not found by token hash")
	}
	if !reflect.DeepEqual(got, workspace) {
		t.Fatalf("workspace mismatch\nactual:   %#v\nexpected: %#v", got, workspace)
	}

	if err := store.TouchWorkspace(ctx, workspace.ID, time.Unix(110, 0), 120); err != nil {
		t.Fatal(err)
	}
	touched, ok, err := store.GetWorkspaceByTokenHash(ctx, tokenHash)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("touched workspace missing")
	}
	if touched.UpdatedAt != 110 || touched.ExpiresAt != 230 {
		t.Fatalf("updated/expires = %.0f/%.0f, want 110/230", touched.UpdatedAt, touched.ExpiresAt)
	}
}

func TestForeignKeyCascadeAndSetNull(t *testing.T) {
	ctx := context.Background()
	store := openTestStore(t, ctx)
	defer store.Close()

	workspace, err := store.CreateWorkspace(ctx, "hash", `{}`, time.Unix(1, 0), 60)
	if err != nil {
		t.Fatal(err)
	}
	uploadID, err := store.InsertUpload(ctx, Upload{
		WorkspaceID:   workspace.ID,
		FileID:        "file-id",
		OriginalName:  "lead.mid",
		SizeBytes:     10,
		QueuePosition: 0,
		CreatedAt:     1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.InsertJob(ctx, Job{
		WorkspaceID: workspace.ID,
		UploadID:    sql.NullInt64{Int64: uploadID, Valid: true},
		JobID:       "job-id",
		Status:      "ready",
		SourceName:  "lead.mid",
		CreatedAt:   1,
		UpdatedAt:   2,
		ExpiresAt:   3,
		ConfigJSON:  `{}`,
	}); err != nil {
		t.Fatal(err)
	}

	if _, err := store.DB().ExecContext(ctx, "DELETE FROM uploads WHERE id = ?", uploadID); err != nil {
		t.Fatal(err)
	}
	var uploadIDValid bool
	if err := store.DB().QueryRowContext(ctx, "SELECT upload_id IS NOT NULL FROM jobs WHERE job_id = 'job-id'").Scan(&uploadIDValid); err != nil {
		t.Fatal(err)
	}
	if uploadIDValid {
		t.Fatal("job upload_id should be set NULL after upload delete")
	}

	if _, err := store.DB().ExecContext(ctx, "DELETE FROM workspaces WHERE id = ?", workspace.ID); err != nil {
		t.Fatal(err)
	}
	for _, table := range []string{"uploads", "jobs"} {
		count, err := store.CountRows(ctx, table)
		if err != nil {
			t.Fatal(err)
		}
		if count != 0 {
			t.Fatalf("%s count = %d, want 0", table, count)
		}
	}
}

func TestDeleteExpiredWorkspacesRemovesRowsAndFiles(t *testing.T) {
	ctx := context.Background()
	store := openTestStore(t, ctx)
	defer store.Close()

	expired, err := store.CreateWorkspace(ctx, "expired", `{}`, time.Unix(10, 0), 5)
	if err != nil {
		t.Fatal(err)
	}
	active, err := store.CreateWorkspace(ctx, "active", `{}`, time.Unix(10, 0), 100)
	if err != nil {
		t.Fatal(err)
	}
	expiredMarker := filepath.Join(store.WorkspaceDir(expired.ID), "marker")
	if err := os.WriteFile(expiredMarker, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	ids, err := store.DeleteExpiredWorkspaces(ctx, time.Unix(20, 0))
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(ids, []int64{expired.ID}) {
		t.Fatalf("expired ids = %#v", ids)
	}
	if _, err := os.Stat(store.WorkspaceDir(expired.ID)); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expired workspace dir still exists or unexpected stat error: %v", err)
	}
	if _, err := os.Stat(store.WorkspaceDir(active.ID)); err != nil {
		t.Fatalf("active workspace dir missing: %v", err)
	}
}

func openTestStore(t *testing.T, ctx context.Context) *Store {
	t.Helper()
	store, err := Open(ctx, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	return store
}
