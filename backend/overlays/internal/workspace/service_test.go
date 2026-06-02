package workspace

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"octabit/backend/internal/config"
	"octabit/backend/internal/storage"
)

func TestServiceCreatesReusesAndReplacesExpiredWorkspace(t *testing.T) {
	ctx := context.Background()
	service, store, clock := newTestService(t, ctx, Options{
		GenerateToken: sequenceGenerator("workspace-token-a", "workspace-token-b"),
	})
	defer store.Close()

	workspace, err := service.GetOrCreateWorkspace(ctx, "")
	if err != nil {
		t.Fatal(err)
	}
	if !workspace.Created || workspace.Token != "workspace-token-a" {
		t.Fatalf("workspace = %#v", workspace)
	}

	state, err := service.StatePayload(ctx, workspace)
	if err != nil {
		t.Fatal(err)
	}
	if state.Workspace.ExpiresAt != 100+config.DefaultWorkspaceTTLSeconds {
		t.Fatalf("expires_at = %.0f", state.Workspace.ExpiresAt)
	}
	if state.Config.Schema != ConfigSchema || len(state.Uploads) != 0 || len(state.ConvertedFiles) != 0 {
		t.Fatalf("unexpected state = %#v", state)
	}
	if state.Limits.MaxQueuedFiles != config.DefaultWorkspaceMaxQueuedFiles {
		t.Fatalf("limits = %#v", state.Limits)
	}

	clock.set(time.Unix(110, 0))
	reused, err := service.GetOrCreateWorkspace(ctx, workspace.Token)
	if err != nil {
		t.Fatal(err)
	}
	if reused.Created || reused.ID != workspace.ID {
		t.Fatalf("reused workspace = %#v", reused)
	}

	clock.set(time.Unix(110+config.DefaultWorkspaceTTLSeconds+1, 0))
	replacement, err := service.GetOrCreateWorkspace(ctx, workspace.Token)
	if err != nil {
		t.Fatal(err)
	}
	if !replacement.Created || replacement.ID == workspace.ID || replacement.Token != "workspace-token-b" {
		t.Fatalf("replacement = %#v", replacement)
	}
	if _, err := os.Stat(store.WorkspaceDir(workspace.ID)); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expired workspace dir still exists or unexpected error: %v", err)
	}
}

func TestServiceUploadRestoreReorderDeleteAndQuota(t *testing.T) {
	ctx := context.Background()
	service, store, _ := newTestService(t, ctx, Options{
		GenerateToken:      sequenceGenerator("workspace-token"),
		GenerateResourceID: sequenceGenerator("file-one", "file-two", "file-three"),
		MaxQueuedFiles:     2,
	})
	defer store.Close()
	workspace, err := service.CreateWorkspace(ctx)
	if err != nil {
		t.Fatal(err)
	}

	first := createTempUpload(t, []byte("first"))
	firstUpload, err := service.CreateUploadFromTemp(ctx, workspace, first, "first.mid")
	if err != nil {
		t.Fatal(err)
	}
	second := createTempUpload(t, []byte("second"))
	secondUpload, err := service.CreateUploadFromTemp(ctx, workspace, second, "second.mid")
	if err != nil {
		t.Fatal(err)
	}
	if firstUpload.FileID != "file-one" || secondUpload.FileID != "file-two" {
		t.Fatalf("uploads = %#v %#v", firstUpload, secondUpload)
	}
	if _, err := os.Stat(store.UploadPath(workspace.ID, firstUpload.FileID)); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(first); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("source was not moved: %v", err)
	}

	reordered, err := service.ReplaceQueue(ctx, workspace, []string{secondUpload.FileID, firstUpload.FileID})
	if err != nil {
		t.Fatal(err)
	}
	if got := uploadIDs(reordered); !reflect.DeepEqual(got, []string{"file-two", "file-one"}) {
		t.Fatalf("reordered ids = %#v", got)
	}
	state, err := service.StatePayload(ctx, workspace)
	if err != nil {
		t.Fatal(err)
	}
	if got := uploadNames(state.Uploads); !reflect.DeepEqual(got, []string{"second.mid", "first.mid"}) {
		t.Fatalf("upload names = %#v", got)
	}

	deleted, err := service.DeleteUpload(ctx, workspace, firstUpload.FileID)
	if err != nil {
		t.Fatal(err)
	}
	if !deleted {
		t.Fatal("expected upload delete")
	}
	if _, err := os.Stat(store.UploadPath(workspace.ID, firstUpload.FileID)); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("upload file still exists or unexpected error: %v", err)
	}

	third := createTempUpload(t, []byte("third"))
	if _, err := service.CreateUploadFromTemp(ctx, workspace, third, "third.mid"); err != nil {
		t.Fatal(err)
	}
	overLimit := createTempUpload(t, []byte("overflow"))
	_, err = service.CreateUploadFromTemp(ctx, workspace, overLimit, "overflow.mid")
	var workspaceErr Error
	if !errors.As(err, &workspaceErr) {
		t.Fatalf("quota error type = %T %[1]v", err)
	}
	if workspaceErr.Code != "workspace_queue_limit" || workspaceErr.StatusCode != 409 {
		t.Fatalf("quota error = %#v", workspaceErr)
	}
}

func TestServiceUploadTotalBytesLimit(t *testing.T) {
	ctx := context.Background()
	service, store, _ := newTestService(t, ctx, Options{
		GenerateToken:      sequenceGenerator("workspace-token"),
		GenerateResourceID: sequenceGenerator("file-one"),
		MaxUploadBytes:     4,
	})
	defer store.Close()
	workspace, err := service.CreateWorkspace(ctx)
	if err != nil {
		t.Fatal(err)
	}

	_, err = service.CreateUploadFromTemp(ctx, workspace, createTempUpload(t, []byte("12345")), "too-large.mid")
	var workspaceErr Error
	if !errors.As(err, &workspaceErr) {
		t.Fatalf("upload bytes error type = %T %[1]v", err)
	}
	if workspaceErr.Code != "workspace_upload_bytes_limit" || workspaceErr.StatusCode != 413 {
		t.Fatalf("upload bytes error = %#v", workspaceErr)
	}
}

func TestServiceSaveConfigRoundTripsCanonicalJSON(t *testing.T) {
	ctx := context.Background()
	service, store, _ := newTestService(t, ctx, Options{GenerateToken: sequenceGenerator("workspace-token")})
	defer store.Close()
	workspace, err := service.CreateWorkspace(ctx)
	if err != nil {
		t.Fatal(err)
	}

	raw := map[string]any{
		"schema":      ConfigSchema,
		"sample_rate": "44100",
		"layers": []any{
			map[string]any{
				"type":          "sine",
				"duty":          0.5,
				"volume":        0.75,
				"curve_enabled": true,
				"frequency_curve": []any{
					map[string]any{"frequency_hz": 8.175798915643707, "gain_db": -6.0},
					map[string]any{"frequency_hz": 12543.853951415975, "gain_db": 0.0},
				},
			},
		},
	}
	normalised, err := NormaliseConfig(raw)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.SaveConfig(ctx, workspace, normalised); err != nil {
		t.Fatal(err)
	}
	state, err := service.StatePayload(ctx, workspace)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(state.Config, normalised) {
		t.Fatalf("state config = %#v, want %#v", state.Config, normalised)
	}

	var stored string
	if err := store.DB().QueryRowContext(ctx, `SELECT config_json FROM workspaces WHERE id = ?`, workspace.ID).Scan(&stored); err != nil {
		t.Fatal(err)
	}
	expected := `{"layers":[{"curve_enabled":true,"duty":0.5,"frequency_curve":[{"frequency_hz":8.175798915643707,"gain_db":-6.0},{"frequency_hz":12543.853951415975,"gain_db":0.0}],"type":"sine","volume":0.75}],"channel_buses":[],"limiter_enabled":true,"master_gain_db":0.0,"normalise_enabled":false,"sample_rate":44100,"schema":"octabit.workspace_config.v1"}`
	if stored != expected {
		t.Fatalf("stored config json\nactual:   %s\nexpected: %s", stored, expected)
	}
}

func TestServiceStateListsConvertedFilesAndCleansExpiredJobs(t *testing.T) {
	ctx := context.Background()
	service, store, _ := newTestService(t, ctx, Options{GenerateToken: sequenceGenerator("workspace-token")})
	defer store.Close()
	workspace, err := service.CreateWorkspace(ctx)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := store.InsertJob(ctx, storage.Job{
		WorkspaceID:  workspace.ID,
		JobID:        "ready-new",
		Status:       "ready",
		SourceName:   "lead.mid",
		DownloadName: sql.NullString{String: "lead_mix.wav", Valid: true},
		SizeBytes:    sql.NullInt64{Int64: 123, Valid: true},
		CreatedAt:    100,
		UpdatedAt:    120,
		ExpiresAt:    200,
		ConfigJSON:   `{}`,
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := store.InsertJob(ctx, storage.Job{
		WorkspaceID: workspace.ID,
		JobID:       "expired",
		Status:      "ready",
		SourceName:  "old.mid",
		CreatedAt:   90,
		UpdatedAt:   95,
		ExpiresAt:   99,
		ConfigJSON:  `{}`,
	}); err != nil {
		t.Fatal(err)
	}
	expiredDir := store.JobDir(workspace.ID, "expired")
	if err := os.MkdirAll(expiredDir, 0o755); err != nil {
		t.Fatal(err)
	}

	state, err := service.StatePayload(ctx, workspace)
	if err != nil {
		t.Fatal(err)
	}
	if len(state.ConvertedFiles) != 1 || state.ConvertedFiles[0].JobID != "ready-new" {
		t.Fatalf("converted files = %#v", state.ConvertedFiles)
	}
	if state.ConvertedFiles[0].DownloadURL != "/api/synthesis-jobs/ready-new/download" {
		t.Fatalf("converted payload = %#v", state.ConvertedFiles[0])
	}
	if _, err := os.Stat(expiredDir); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expired job dir still exists or unexpected error: %v", err)
	}
}

func newTestService(t *testing.T, ctx context.Context, options Options) (*Service, *storage.Store, *testClock) {
	t.Helper()
	store, err := storage.Open(ctx, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	clock := &testClock{value: time.Unix(100, 0)}
	options.Now = clock.now
	return NewService(store, config.Config{
		WorkspaceTTLSeconds:        config.DefaultWorkspaceTTLSeconds,
		WorkspaceMaxQueuedFiles:    config.DefaultWorkspaceMaxQueuedFiles,
		WorkspaceMaxUploadBytes:    config.DefaultWorkspaceMaxUploadBytes,
		WorkspaceMaxConvertedFiles: config.DefaultWorkspaceMaxConvertedFiles,
	}, options), store, clock
}

type testClock struct {
	value time.Time
}

func (c *testClock) now() time.Time {
	return c.value
}

func (c *testClock) set(value time.Time) {
	c.value = value
}

func sequenceGenerator(values ...string) func() (string, error) {
	index := 0
	return func() (string, error) {
		if index >= len(values) {
			return "", errors.New("sequence exhausted")
		}
		value := values[index]
		index++
		return value, nil
	}
}

func createTempUpload(t *testing.T, content []byte) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "upload.mid")
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func uploadIDs(uploads []UploadPayload) []string {
	ids := make([]string, 0, len(uploads))
	for _, upload := range uploads {
		ids = append(ids, upload.FileID)
	}
	return ids
}

func uploadNames(uploads []UploadPayload) []string {
	names := make([]string, 0, len(uploads))
	for _, upload := range uploads {
		names = append(names, upload.Name)
	}
	return names
}
