package config

import (
	"strings"
	"testing"
)

func TestFromEnvUsesFlaskCompatibleDefaults(t *testing.T) {
	t.Setenv("HOST", "")
	t.Setenv("PORT", "")
	t.Setenv("WEB_SYNTHESISE_JOB_ROOT", "")
	t.Setenv("WEB_DOWNLOAD_TTL_SECONDS", "")
	t.Setenv("WEB_WORKSPACE_TTL_SECONDS", "")
	t.Setenv("WEB_WORKSPACE_MAX_QUEUED_FILES", "")
	t.Setenv("WEB_WORKSPACE_MAX_UPLOAD_BYTES", "")
	t.Setenv("WEB_WORKSPACE_MAX_CONVERTED_FILES", "")
	t.Setenv("WEB_MAX_UPLOAD_BYTES", "")
	t.Setenv("WEB_RENDER_WORKERS", "")
	t.Setenv("WEB_RENDER_QUEUE_SIZE", "")

	cfg := FromEnv()

	if cfg.Host != DefaultHost {
		t.Fatalf("Host = %q, want %q", cfg.Host, DefaultHost)
	}
	if cfg.Port != DefaultPort {
		t.Fatalf("Port = %d, want %d", cfg.Port, DefaultPort)
	}
	if cfg.JobRoot != defaultJobRoot() {
		t.Fatalf("JobRoot = %q, want %q", cfg.JobRoot, defaultJobRoot())
	}
	if !strings.HasSuffix(filepathSlash(cfg.PreviewAssetsDir), "assets/previews") {
		t.Fatalf("PreviewAssetsDir = %q", cfg.PreviewAssetsDir)
	}
	if cfg.DownloadTTLSeconds != DefaultDownloadTTLSeconds {
		t.Fatalf("DownloadTTLSeconds = %d", cfg.DownloadTTLSeconds)
	}
	if cfg.WorkspaceTTLSeconds != DefaultWorkspaceTTLSeconds {
		t.Fatalf("WorkspaceTTLSeconds = %d", cfg.WorkspaceTTLSeconds)
	}
	if cfg.WorkspaceMaxQueuedFiles != DefaultWorkspaceMaxQueuedFiles {
		t.Fatalf("WorkspaceMaxQueuedFiles = %d", cfg.WorkspaceMaxQueuedFiles)
	}
	if cfg.WorkspaceMaxUploadBytes != DefaultWorkspaceMaxUploadBytes {
		t.Fatalf("WorkspaceMaxUploadBytes = %d", cfg.WorkspaceMaxUploadBytes)
	}
	if cfg.WorkspaceMaxConvertedFiles != DefaultWorkspaceMaxConvertedFiles {
		t.Fatalf("WorkspaceMaxConvertedFiles = %d", cfg.WorkspaceMaxConvertedFiles)
	}
	if cfg.MaxUploadBytes != DefaultMaxUploadBytes {
		t.Fatalf("MaxUploadBytes = %d", cfg.MaxUploadBytes)
	}
	if cfg.RenderWorkers != DefaultRenderWorkers {
		t.Fatalf("RenderWorkers = %d", cfg.RenderWorkers)
	}
	if cfg.RenderQueueSize != DefaultRenderQueueSize {
		t.Fatalf("RenderQueueSize = %d", cfg.RenderQueueSize)
	}
}

func filepathSlash(path string) string {
	return strings.ReplaceAll(path, "\\", "/")
}

func TestFromEnvPreservesCurrentEnvNames(t *testing.T) {
	t.Setenv("HOST", "0.0.0.0")
	t.Setenv("PORT", "8000")
	t.Setenv("WEB_SYNTHESISE_JOB_ROOT", "/var/tmp/octabit")
	t.Setenv("WEB_DOWNLOAD_TTL_SECONDS", "33")
	t.Setenv("WEB_WORKSPACE_TTL_SECONDS", "44")
	t.Setenv("WEB_WORKSPACE_MAX_QUEUED_FILES", "5")
	t.Setenv("WEB_WORKSPACE_MAX_UPLOAD_BYTES", "6")
	t.Setenv("WEB_WORKSPACE_MAX_CONVERTED_FILES", "7")
	t.Setenv("WEB_MAX_UPLOAD_BYTES", "8")
	t.Setenv("WEB_RENDER_WORKERS", "9")
	t.Setenv("WEB_RENDER_QUEUE_SIZE", "0")

	cfg := FromEnv()

	if cfg.Addr() != "0.0.0.0:8000" {
		t.Fatalf("Addr() = %q", cfg.Addr())
	}
	if cfg.JobRoot != "/var/tmp/octabit" {
		t.Fatalf("JobRoot = %q", cfg.JobRoot)
	}
	if cfg.DownloadTTLSeconds != 33 ||
		cfg.WorkspaceTTLSeconds != 44 ||
		cfg.WorkspaceMaxQueuedFiles != 5 ||
		cfg.WorkspaceMaxUploadBytes != 6 ||
		cfg.WorkspaceMaxConvertedFiles != 7 ||
		cfg.MaxUploadBytes != 8 ||
		cfg.RenderWorkers != 9 ||
		cfg.RenderQueueSize != 0 {
		t.Fatalf("FromEnv() did not preserve env overrides: %+v", cfg)
	}
}

func TestFromEnvFallsBackForInvalidNumericValues(t *testing.T) {
	t.Setenv("PORT", "not-a-port")
	t.Setenv("WEB_DOWNLOAD_TTL_SECONDS", "0")
	t.Setenv("WEB_WORKSPACE_TTL_SECONDS", "-1")
	t.Setenv("WEB_WORKSPACE_MAX_QUEUED_FILES", "bad")
	t.Setenv("WEB_WORKSPACE_MAX_UPLOAD_BYTES", "0")
	t.Setenv("WEB_WORKSPACE_MAX_CONVERTED_FILES", "-7")
	t.Setenv("WEB_MAX_UPLOAD_BYTES", "bad")
	t.Setenv("WEB_RENDER_WORKERS", "0")
	t.Setenv("WEB_RENDER_QUEUE_SIZE", "-1")

	cfg := FromEnv()

	if cfg.Port != DefaultPort ||
		cfg.DownloadTTLSeconds != DefaultDownloadTTLSeconds ||
		cfg.WorkspaceTTLSeconds != DefaultWorkspaceTTLSeconds ||
		cfg.WorkspaceMaxQueuedFiles != DefaultWorkspaceMaxQueuedFiles ||
		cfg.WorkspaceMaxUploadBytes != DefaultWorkspaceMaxUploadBytes ||
		cfg.WorkspaceMaxConvertedFiles != DefaultWorkspaceMaxConvertedFiles ||
		cfg.MaxUploadBytes != DefaultMaxUploadBytes ||
		cfg.RenderWorkers != DefaultRenderWorkers ||
		cfg.RenderQueueSize != DefaultRenderQueueSize {
		t.Fatalf("invalid values did not fall back to defaults: %+v", cfg)
	}
}
