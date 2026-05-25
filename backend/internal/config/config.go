package config

import (
	"net"
	"os"
	"path/filepath"
	"strconv"
)

const (
	DefaultHost                       = "127.0.0.1"
	DefaultPort                       = 5002
	DefaultDownloadTTLSeconds         = 30 * 60
	DefaultWorkspaceTTLSeconds        = 24 * 60 * 60
	DefaultWorkspaceMaxQueuedFiles    = 20
	DefaultWorkspaceMaxUploadBytes    = 100 * 1024 * 1024
	DefaultWorkspaceMaxConvertedFiles = 20
	DefaultMaxUploadBytes             = 20 * 1024 * 1024
	DefaultRenderWorkers              = 2
	DefaultRenderQueueSize            = 8
)

type Config struct {
	Host                       string
	Port                       int
	JobRoot                    string
	PreviewAssetsDir           string
	DownloadTTLSeconds         int
	WorkspaceTTLSeconds        int
	WorkspaceMaxQueuedFiles    int
	WorkspaceMaxUploadBytes    int
	WorkspaceMaxConvertedFiles int
	MaxUploadBytes             int
	RenderWorkers              int
	RenderQueueSize            int
}

func FromEnv() Config {
	return Config{
		Host:                       stringFromEnv("HOST", DefaultHost),
		Port:                       positiveIntFromEnv("PORT", DefaultPort),
		JobRoot:                    stringFromEnv("WEB_SYNTHESISE_JOB_ROOT", defaultJobRoot()),
		PreviewAssetsDir:           defaultPreviewAssetsDir(),
		DownloadTTLSeconds:         positiveIntFromEnv("WEB_DOWNLOAD_TTL_SECONDS", DefaultDownloadTTLSeconds),
		WorkspaceTTLSeconds:        positiveIntFromEnv("WEB_WORKSPACE_TTL_SECONDS", DefaultWorkspaceTTLSeconds),
		WorkspaceMaxQueuedFiles:    positiveIntFromEnv("WEB_WORKSPACE_MAX_QUEUED_FILES", DefaultWorkspaceMaxQueuedFiles),
		WorkspaceMaxUploadBytes:    positiveIntFromEnv("WEB_WORKSPACE_MAX_UPLOAD_BYTES", DefaultWorkspaceMaxUploadBytes),
		WorkspaceMaxConvertedFiles: positiveIntFromEnv("WEB_WORKSPACE_MAX_CONVERTED_FILES", DefaultWorkspaceMaxConvertedFiles),
		MaxUploadBytes:             positiveIntFromEnv("WEB_MAX_UPLOAD_BYTES", DefaultMaxUploadBytes),
		RenderWorkers:              positiveIntFromEnv("WEB_RENDER_WORKERS", DefaultRenderWorkers),
		RenderQueueSize:            nonNegativeIntFromEnv("WEB_RENDER_QUEUE_SIZE", DefaultRenderQueueSize),
	}
}

func (c Config) Addr() string {
	return net.JoinHostPort(c.Host, strconv.Itoa(c.Port))
}

func stringFromEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

func positiveIntFromEnv(key string, defaultValue int) int {
	value, err := strconv.Atoi(os.Getenv(key))
	if err != nil || value < 1 {
		return defaultValue
	}
	return value
}

func nonNegativeIntFromEnv(key string, defaultValue int) int {
	value, err := strconv.Atoi(os.Getenv(key))
	if err != nil || value < 0 {
		return defaultValue
	}
	return value
}

func defaultJobRoot() string {
	return filepath.Join(os.TempDir(), "octabit-jobs")
}

func defaultPreviewAssetsDir() string {
	return findUp(filepath.Join("assets", "previews"))
}

func findUp(relativePath string) string {
	workingDirectory, err := os.Getwd()
	if err != nil {
		return relativePath
	}

	directory := workingDirectory
	for range 8 {
		candidate := filepath.Join(directory, relativePath)
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			return candidate
		}
		parent := filepath.Dir(directory)
		if parent == directory {
			break
		}
		directory = parent
	}

	return relativePath
}
