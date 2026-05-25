package workspace

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"octabit/backend/internal/config"
	"octabit/backend/internal/jobs"
	"octabit/backend/internal/midi"
	"octabit/backend/internal/renderer"
	"octabit/backend/internal/storage"
)

type Context struct {
	ID      int64
	Token   string
	Created bool
}

type Service struct {
	store              *storage.Store
	ttlSeconds         int
	maxQueuedFiles     int
	maxUploadBytes     int
	maxConvertedFiles  int
	defaultConfig      Config
	now                func() time.Time
	generateToken      func() (string, error)
	generateResourceID func() (string, error)
	renderExecutor     *jobs.Executor
	renderWAV          RenderWAVFunc
}

type RenderWAVFunc func(inputPath, outputPath, sourceName string, workspaceConfig Config) (string, int64, error)

type Options struct {
	TTLSeconds         int
	MaxQueuedFiles     int
	MaxUploadBytes     int
	MaxConvertedFiles  int
	DefaultConfig      Config
	Now                func() time.Time
	GenerateToken      func() (string, error)
	GenerateResourceID func() (string, error)
	RunInline          bool
	RenderWAV          RenderWAVFunc
}

type Error struct {
	Code       string
	Message    string
	StatusCode int
}

func (e Error) Error() string {
	return e.Message
}

type StatePayload struct {
	Workspace      WorkspacePayload   `json:"workspace"`
	Limits         LimitsPayload      `json:"limits"`
	Config         Config             `json:"config"`
	Uploads        []UploadPayload    `json:"uploads"`
	ConvertedFiles []ConvertedPayload `json:"converted_files"`
}

type WorkspacePayload struct {
	ExpiresAt float64 `json:"expires_at"`
}

type LimitsPayload struct {
	MaxQueuedFiles    int `json:"max_queued_files"`
	MaxUploadBytes    int `json:"max_upload_bytes"`
	MaxConvertedFiles int `json:"max_converted_files"`
}

type UploadPayload struct {
	FileID    string  `json:"file_id"`
	Name      string  `json:"name"`
	Size      int64   `json:"size"`
	CreatedAt float64 `json:"created_at"`
}

type ConvertedPayload struct {
	JobID       string  `json:"job_id"`
	Name        string  `json:"name"`
	SourceName  string  `json:"source_name"`
	Size        int64   `json:"size"`
	DownloadURL string  `json:"download_url"`
	DeleteURL   string  `json:"delete_url"`
	CreatedAt   float64 `json:"created_at"`
	UpdatedAt   float64 `json:"updated_at"`
	ExpiresAt   float64 `json:"expires_at"`
}

type JobPayload struct {
	JobID        string  `json:"job_id"`
	Status       string  `json:"status"`
	SourceName   string  `json:"source_name,omitempty"`
	CreatedAt    float64 `json:"created_at,omitempty"`
	UpdatedAt    float64 `json:"updated_at,omitempty"`
	ExpiresAt    float64 `json:"expires_at,omitempty"`
	DownloadName string  `json:"download_name,omitempty"`
	SizeBytes    int64   `json:"size_bytes,omitempty"`
	Error        string  `json:"error,omitempty"`
	DownloadURL  string  `json:"download_url,omitempty"`
	DeleteURL    string  `json:"delete_url,omitempty"`
}

func NewService(store *storage.Store, cfg config.Config, options Options) *Service {
	ttlSeconds := cfg.WorkspaceTTLSeconds
	if options.TTLSeconds > 0 {
		ttlSeconds = options.TTLSeconds
	}
	maxQueuedFiles := cfg.WorkspaceMaxQueuedFiles
	if options.MaxQueuedFiles > 0 {
		maxQueuedFiles = options.MaxQueuedFiles
	}
	maxUploadBytes := cfg.WorkspaceMaxUploadBytes
	if options.MaxUploadBytes > 0 {
		maxUploadBytes = options.MaxUploadBytes
	}
	maxConvertedFiles := cfg.WorkspaceMaxConvertedFiles
	if options.MaxConvertedFiles > 0 {
		maxConvertedFiles = options.MaxConvertedFiles
	}
	defaultConfig := options.DefaultConfig
	if defaultConfig.Schema == "" {
		defaultConfig = DefaultConfig()
	}
	now := options.Now
	if now == nil {
		now = time.Now
	}
	generateToken := options.GenerateToken
	if generateToken == nil {
		generateToken = tokenURLSafe32
	}
	generateResourceID := options.GenerateResourceID
	if generateResourceID == nil {
		generateResourceID = randomHex16
	}
	renderWAVFunc := options.RenderWAV
	if renderWAVFunc == nil {
		renderWAVFunc = renderWAV
	}

	return &Service{
		store:              store,
		ttlSeconds:         ttlSeconds,
		maxQueuedFiles:     maxQueuedFiles,
		maxUploadBytes:     maxUploadBytes,
		maxConvertedFiles:  maxConvertedFiles,
		defaultConfig:      defaultConfig,
		now:                now,
		generateToken:      generateToken,
		generateResourceID: generateResourceID,
		renderWAV:          renderWAVFunc,
		renderExecutor: jobs.NewExecutor(jobs.Config{
			MaxWorkers:   cfg.RenderWorkers,
			MaxQueueSize: cfg.RenderQueueSize,
			RunInline:    options.RunInline,
		}),
	}
}

func DefaultConfig() Config {
	return Config{
		Schema:     ConfigSchema,
		SampleRate: 48000,
		Layers: []LayerConfig{{
			Type:         "pulse",
			Duty:         0.5,
			Volume:       1.0,
			CurveEnabled: false,
			FrequencyCurve: []renderer.FrequencyCurvePoint{
				{FrequencyHz: renderer.MinCurveFrequencyHz, GainDB: 0.0},
				{FrequencyHz: renderer.MaxCurveFrequencyHz, GainDB: 0.0},
			},
		}},
	}
}

func (s *Service) GetOrCreateWorkspace(ctx context.Context, token string) (Context, error) {
	if _, err := s.store.DeleteExpiredWorkspaces(ctx, s.now()); err != nil {
		return Context{}, err
	}
	workspace, ok, err := s.GetActiveWorkspace(ctx, token)
	if err != nil || ok {
		return workspace, err
	}
	return s.CreateWorkspace(ctx)
}

func (s *Service) GetActiveWorkspace(ctx context.Context, token string) (Context, bool, error) {
	if !storage.IsPlausibleToken(token) {
		return Context{}, false, nil
	}

	row, ok, err := s.store.GetWorkspaceByTokenHash(ctx, storage.TokenHash(token))
	if err != nil || !ok {
		return Context{}, ok, err
	}
	if row.ExpiresAt <= unixSeconds(s.now()) {
		if err := s.deleteWorkspace(ctx, row.ID); err != nil {
			return Context{}, false, err
		}
		return Context{}, false, nil
	}
	if err := s.store.TouchWorkspace(ctx, row.ID, s.now(), s.ttlSeconds); err != nil {
		return Context{}, false, err
	}
	return Context{ID: row.ID, Token: token}, true, nil
}

func (s *Service) CreateWorkspace(ctx context.Context) (Context, error) {
	token, err := s.generateToken()
	if err != nil {
		return Context{}, err
	}
	configJSON := canonicalConfigJSON(s.defaultConfig)
	workspace, err := s.store.CreateWorkspace(ctx, storage.TokenHash(token), configJSON, s.now(), s.ttlSeconds)
	if err != nil {
		return Context{}, err
	}
	return Context{ID: workspace.ID, Token: token, Created: true}, nil
}

func (s *Service) StatePayload(ctx context.Context, workspace Context) (StatePayload, error) {
	if err := s.cleanupExpiredJobs(ctx, workspace); err != nil {
		return StatePayload{}, err
	}
	row, ok, err := s.workspaceByID(ctx, workspace.ID)
	if err != nil {
		return StatePayload{}, err
	}
	if !ok {
		return StatePayload{}, Error{Code: "workspace_expired", Message: "The temporary workspace has expired. Refresh to start a new workspace.", StatusCode: 410}
	}
	uploads, err := s.listUploads(ctx, workspace.ID)
	if err != nil {
		return StatePayload{}, err
	}
	jobs, err := s.listReadyJobs(ctx, workspace.ID)
	if err != nil {
		return StatePayload{}, err
	}
	var savedConfig Config
	if err := json.Unmarshal([]byte(row.ConfigJSON), &savedConfig); err != nil {
		return StatePayload{}, err
	}
	return StatePayload{
		Workspace:      WorkspacePayload{ExpiresAt: row.ExpiresAt},
		Limits:         s.LimitsPayload(),
		Config:         savedConfig,
		Uploads:        uploadPayloads(uploads),
		ConvertedFiles: convertedPayloads(jobs),
	}, nil
}

func (s *Service) LimitsPayload() LimitsPayload {
	return LimitsPayload{
		MaxQueuedFiles:    s.maxQueuedFiles,
		MaxUploadBytes:    s.maxUploadBytes,
		MaxConvertedFiles: s.maxConvertedFiles,
	}
}

func (s *Service) TTLSeconds() int {
	return s.ttlSeconds
}

func (s *Service) CreateUploadFromTemp(ctx context.Context, workspace Context, sourcePath, originalName string) (UploadPayload, error) {
	info, err := os.Stat(sourcePath)
	if err != nil {
		return UploadPayload{}, err
	}

	stats, err := s.uploadStats(ctx, workspace.ID)
	if err != nil {
		return UploadPayload{}, err
	}
	if stats.count >= s.maxQueuedFiles {
		return UploadPayload{}, Error{
			Code:       "workspace_queue_limit",
			Message:    "This temporary workspace has reached the queued file limit.",
			StatusCode: 409,
		}
	}
	if stats.bytes+info.Size() > int64(s.maxUploadBytes) {
		return UploadPayload{}, Error{
			Code:       "workspace_upload_bytes_limit",
			Message:    "This temporary workspace has reached the upload storage limit.",
			StatusCode: 413,
		}
	}

	fileID, err := s.generateResourceID()
	if err != nil {
		return UploadPayload{}, err
	}
	position, err := s.nextUploadQueuePosition(ctx, workspace.ID)
	if err != nil {
		return UploadPayload{}, err
	}
	now := unixSeconds(s.now())
	name := filepath.Base(originalName)
	if name == "." || name == string(filepath.Separator) || name == "" {
		name = "upload.mid"
	}

	if _, err := s.store.InsertUpload(ctx, storage.Upload{
		WorkspaceID:   workspace.ID,
		FileID:        fileID,
		OriginalName:  name,
		SizeBytes:     info.Size(),
		QueuePosition: position,
		CreatedAt:     now,
	}); err != nil {
		return UploadPayload{}, err
	}

	destination := s.store.UploadPath(workspace.ID, fileID)
	if err := moveFile(sourcePath, destination); err != nil {
		_, _ = s.deleteUploadRow(ctx, workspace.ID, fileID)
		return UploadPayload{}, err
	}
	row, ok, err := s.GetUpload(ctx, workspace, fileID)
	if err != nil {
		return UploadPayload{}, err
	}
	if !ok {
		return UploadPayload{}, fmt.Errorf("created upload %s was not found", fileID)
	}
	return uploadPayload(row), nil
}

func (s *Service) GetUpload(ctx context.Context, workspace Context, fileID string) (storage.Upload, bool, error) {
	return s.getUpload(ctx, workspace.ID, fileID)
}

func (s *Service) DeleteUpload(ctx context.Context, workspace Context, fileID string) (bool, error) {
	deleted, err := s.deleteUploadRow(ctx, workspace.ID, fileID)
	if err != nil || !deleted {
		return deleted, err
	}
	if err := os.Remove(s.store.UploadPath(workspace.ID, fileID)); err != nil && !os.IsNotExist(err) {
		return true, err
	}
	return true, nil
}

func (s *Service) ReplaceQueue(ctx context.Context, workspace Context, fileIDs []string) ([]UploadPayload, error) {
	rows, err := s.listUploads(ctx, workspace.ID)
	if err != nil {
		return nil, err
	}
	currentIDs := make([]string, 0, len(rows))
	for _, row := range rows {
		currentIDs = append(currentIDs, row.FileID)
	}
	if !sameIDSet(currentIDs, fileIDs) {
		return nil, Error{
			Code:       "invalid_queue",
			Message:    "Queue order must contain each active file id exactly once.",
			StatusCode: 422,
		}
	}
	for position, fileID := range fileIDs {
		if _, err := s.store.DB().ExecContext(ctx, `UPDATE uploads SET queue_position = ? WHERE workspace_id = ? AND file_id = ?`, position, workspace.ID, fileID); err != nil {
			return nil, err
		}
	}
	updated, err := s.listUploads(ctx, workspace.ID)
	if err != nil {
		return nil, err
	}
	return uploadPayloads(updated), nil
}

func (s *Service) SaveConfig(ctx context.Context, workspace Context, workspaceConfig Config) (Config, error) {
	_, err := s.store.DB().ExecContext(
		ctx,
		`UPDATE workspaces SET config_json = ?, updated_at = ?, expires_at = ? WHERE id = ?`,
		canonicalConfigJSON(workspaceConfig),
		unixSeconds(s.now()),
		unixSeconds(s.now().Add(time.Duration(s.ttlSeconds)*time.Second)),
		workspace.ID,
	)
	return workspaceConfig, err
}

func (s *Service) CreateJobFromUpload(ctx context.Context, workspace Context, fileID string, workspaceConfig Config) (JobPayload, error) {
	upload, ok, err := s.GetUpload(ctx, workspace, fileID)
	if err != nil || !ok {
		if err != nil {
			return JobPayload{}, err
		}
		return JobPayload{}, Error{Code: "not_found", Message: "File not found.", StatusCode: 404}
	}
	if _, err := s.SaveConfig(ctx, workspace, workspaceConfig); err != nil {
		return JobPayload{}, err
	}

	jobID, inputPath, err := s.prepareJob(ctx, workspace, s.store.UploadPath(workspace.ID, fileID), upload.OriginalName, workspaceConfig, sql.NullInt64{Int64: upload.ID, Valid: true})
	if err != nil {
		return JobPayload{}, err
	}
	if err := s.startJob(workspace, jobID, inputPath, workspaceConfig, upload.OriginalName); err != nil {
		_, _ = s.DeleteJob(ctx, workspace, jobID)
		return JobPayload{}, err
	}
	job, expired, ok, err := s.GetJob(ctx, workspace, jobID)
	if err != nil {
		return JobPayload{}, err
	}
	if expired || !ok {
		return JobPayload{}, fmt.Errorf("created job %s was not found", jobID)
	}
	return jobPayload(job), nil
}

func (s *Service) CreateJobFromTemp(ctx context.Context, workspace Context, sourcePath, sourceName string, workspaceConfig Config) (JobPayload, error) {
	jobID, inputPath, err := s.prepareJob(ctx, workspace, sourcePath, sourceName, workspaceConfig, sql.NullInt64{})
	if err != nil {
		return JobPayload{}, err
	}
	if err := s.startJob(workspace, jobID, inputPath, workspaceConfig, sourceName); err != nil {
		_, _ = s.DeleteJob(ctx, workspace, jobID)
		return JobPayload{}, err
	}
	job, expired, ok, err := s.GetJob(ctx, workspace, jobID)
	if err != nil {
		return JobPayload{}, err
	}
	if expired || !ok {
		return JobPayload{}, fmt.Errorf("created job %s was not found", jobID)
	}
	return jobPayload(job), nil
}

func (s *Service) GetJob(ctx context.Context, workspace Context, jobID string) (storage.Job, bool, bool, error) {
	job, ok, err := s.getJob(ctx, workspace.ID, jobID)
	if err != nil || !ok {
		return storage.Job{}, false, ok, err
	}
	if job.ExpiresAt <= unixSeconds(s.now()) {
		deleted, err := s.DeleteJob(ctx, workspace, jobID)
		if err != nil {
			return storage.Job{}, false, false, err
		}
		return storage.Job{}, true, deleted, nil
	}
	return job, false, true, nil
}

func (s *Service) DeleteJob(ctx context.Context, workspace Context, jobID string) (bool, error) {
	job, ok, err := s.getJob(ctx, workspace.ID, jobID)
	if err != nil || !ok {
		return ok, err
	}
	if _, err := s.store.DB().ExecContext(ctx, `DELETE FROM jobs WHERE id = ?`, job.ID); err != nil {
		return false, err
	}
	if err := os.RemoveAll(s.store.JobDir(workspace.ID, jobID)); err != nil {
		return true, err
	}
	return true, nil
}

func (s *Service) JobPayload(job storage.Job) JobPayload {
	return jobPayload(job)
}

func (s *Service) JobOutputPath(workspace Context, jobID string) string {
	return s.store.JobOutputPath(workspace.ID, jobID)
}

func (s *Service) prepareJob(ctx context.Context, workspace Context, sourcePath, sourceName string, workspaceConfig Config, uploadID sql.NullInt64) (string, string, error) {
	if err := s.cleanupExpiredJobs(ctx, workspace); err != nil {
		return "", "", err
	}
	count, err := s.activeJobCount(ctx, workspace.ID)
	if err != nil {
		return "", "", err
	}
	if count >= s.maxConvertedFiles {
		return "", "", Error{
			Code:       "workspace_converted_limit",
			Message:    "This temporary workspace has reached the converted file limit.",
			StatusCode: 409,
		}
	}

	jobID, err := s.generateResourceID()
	if err != nil {
		return "", "", err
	}
	now := unixSeconds(s.now())
	if _, err := s.store.InsertJob(ctx, storage.Job{
		WorkspaceID: workspace.ID,
		UploadID:    uploadID,
		JobID:       jobID,
		Status:      jobs.StatusQueued,
		SourceName:  filepath.Base(sourceName),
		CreatedAt:   now,
		UpdatedAt:   now,
		ExpiresAt:   now + float64(s.ttlSeconds),
		ConfigJSON:  canonicalConfigJSON(workspaceConfig),
	}); err != nil {
		return "", "", err
	}

	inputPath := s.store.JobInputPath(workspace.ID, jobID)
	if err := copyFile(sourcePath, inputPath); err != nil {
		_, _ = s.DeleteJob(ctx, workspace, jobID)
		return "", "", err
	}
	return jobID, inputPath, nil
}

func (s *Service) startJob(workspace Context, jobID, inputPath string, workspaceConfig Config, sourceName string) error {
	return s.renderExecutor.Submit(func() {
		s.runJob(workspace, jobID, inputPath, workspaceConfig, sourceName)
	})
}

func (s *Service) runJob(workspace Context, jobID, inputPath string, workspaceConfig Config, sourceName string) {
	ctx := context.Background()
	s.updateJob(ctx, jobID, map[string]any{"status": jobs.StatusRendering})
	outputPath := s.store.JobOutputPath(workspace.ID, jobID)
	downloadName, sizeBytes, err := s.renderWAV(inputPath, outputPath, sourceName, workspaceConfig)
	if err != nil {
		_ = s.updateJob(ctx, jobID, map[string]any{
			"status":     jobs.StatusFailed,
			"error":      jobs.ErrorMessage(err),
			"expires_at": unixSeconds(s.now().Add(time.Duration(s.ttlSeconds) * time.Second)),
		})
		_ = os.Remove(inputPath)
		return
	}
	_ = s.updateJob(ctx, jobID, map[string]any{
		"status":        jobs.StatusReady,
		"download_name": downloadName,
		"size_bytes":    sizeBytes,
		"expires_at":    unixSeconds(s.now().Add(time.Duration(s.ttlSeconds) * time.Second)),
	})
	_ = os.Remove(inputPath)
}

type uploadStatsResult struct {
	count int
	bytes int64
}

func (s *Service) activeJobCount(ctx context.Context, workspaceID int64) (int, error) {
	var count int
	err := s.store.DB().QueryRowContext(
		ctx,
		`SELECT COUNT(*) FROM jobs WHERE workspace_id = ? AND status IN ('queued', 'rendering', 'ready')`,
		workspaceID,
	).Scan(&count)
	return count, err
}

func (s *Service) getJob(ctx context.Context, workspaceID int64, jobID string) (storage.Job, bool, error) {
	row := s.store.DB().QueryRowContext(
		ctx,
		`SELECT id, workspace_id, upload_id, job_id, status, source_name, download_name, size_bytes, error, created_at, updated_at, expires_at, config_json
		 FROM jobs
		 WHERE workspace_id = ? AND job_id = ?`,
		workspaceID,
		jobID,
	)
	var job storage.Job
	if err := row.Scan(
		&job.ID,
		&job.WorkspaceID,
		&job.UploadID,
		&job.JobID,
		&job.Status,
		&job.SourceName,
		&job.DownloadName,
		&job.SizeBytes,
		&job.Error,
		&job.CreatedAt,
		&job.UpdatedAt,
		&job.ExpiresAt,
		&job.ConfigJSON,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return storage.Job{}, false, nil
		}
		return storage.Job{}, false, err
	}
	return job, true, nil
}

func (s *Service) updateJob(ctx context.Context, jobID string, updates map[string]any) error {
	if len(updates) == 0 {
		return nil
	}
	updates["updated_at"] = unixSeconds(s.now())
	keys := make([]string, 0, len(updates))
	for key := range updates {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	assignments := make([]string, 0, len(keys))
	values := make([]any, 0, len(keys)+1)
	for _, key := range keys {
		assignments = append(assignments, key+" = ?")
		values = append(values, updates[key])
	}
	values = append(values, jobID)
	_, err := s.store.DB().ExecContext(ctx, `UPDATE jobs SET `+strings.Join(assignments, ", ")+` WHERE job_id = ?`, values...)
	return err
}

func jobPayload(job storage.Job) JobPayload {
	payload := JobPayload{
		JobID:      job.JobID,
		Status:     job.Status,
		SourceName: job.SourceName,
		CreatedAt:  job.CreatedAt,
		UpdatedAt:  job.UpdatedAt,
		ExpiresAt:  job.ExpiresAt,
	}
	if job.DownloadName.Valid {
		payload.DownloadName = job.DownloadName.String
	}
	if job.SizeBytes.Valid {
		payload.SizeBytes = job.SizeBytes.Int64
	}
	if job.Error.Valid {
		payload.Error = job.Error.String
	}
	if job.Status == jobs.StatusReady {
		payload.DownloadURL = "/api/synthesis-jobs/" + job.JobID + "/download"
		payload.DeleteURL = "/api/synthesis-jobs/" + job.JobID
	}
	return payload
}

func renderWAV(inputPath, outputPath, sourceName string, workspaceConfig Config) (string, int64, error) {
	notes, err := midi.ReadNotes(inputPath)
	if err != nil {
		return "", 0, err
	}
	runtimeLayers := RuntimeLayers(workspaceConfig)
	wavData, err := renderer.RenderNotesWAV(notes, workspaceConfig.SampleRate, runtimeLayers)
	if err != nil {
		return "", 0, err
	}
	if err := os.MkdirAll(filepath.Dir(outputPath), 0o755); err != nil {
		return "", 0, err
	}
	if err := os.WriteFile(outputPath, wavData, 0o644); err != nil {
		return "", 0, err
	}
	downloadName, err := renderer.BuildOutputFilename(originalFilename(sourceName), runtimeLayers)
	if err != nil {
		return "", 0, err
	}
	return downloadName, int64(len(wavData)), nil
}

func RuntimeLayers(workspaceConfig Config) []renderer.Layer {
	runtimeLayers := make([]renderer.Layer, 0, len(workspaceConfig.Layers))
	for _, layer := range workspaceConfig.Layers {
		frequencyCurve := []renderer.FrequencyCurvePoint{}
		if layer.CurveEnabled {
			frequencyCurve = layer.FrequencyCurve
		}
		runtimeLayers = append(runtimeLayers, renderer.Layer{
			Type:           layer.Type,
			Duty:           layer.Duty,
			Volume:         layer.Volume,
			FrequencyCurve: frequencyCurve,
		})
	}
	return runtimeLayers
}

func originalFilename(sourceName string) string {
	base := filepath.Base(sourceName)
	if base == "." || base == string(filepath.Separator) || base == "" {
		return "output"
	}
	extension := filepath.Ext(base)
	name := strings.TrimSuffix(base, extension)
	if name == "" {
		return "output"
	}
	return name
}

func (s *Service) uploadStats(ctx context.Context, workspaceID int64) (uploadStatsResult, error) {
	var stats uploadStatsResult
	err := s.store.DB().QueryRowContext(
		ctx,
		`SELECT COUNT(*), COALESCE(SUM(size_bytes), 0) FROM uploads WHERE workspace_id = ?`,
		workspaceID,
	).Scan(&stats.count, &stats.bytes)
	return stats, err
}

func (s *Service) nextUploadQueuePosition(ctx context.Context, workspaceID int64) (int, error) {
	var position int
	err := s.store.DB().QueryRowContext(
		ctx,
		`SELECT COALESCE(MAX(queue_position), -1) + 1 FROM uploads WHERE workspace_id = ?`,
		workspaceID,
	).Scan(&position)
	return position, err
}

func (s *Service) getUpload(ctx context.Context, workspaceID int64, fileID string) (storage.Upload, bool, error) {
	row := s.store.DB().QueryRowContext(
		ctx,
		`SELECT id, workspace_id, file_id, original_name, size_bytes, queue_position, created_at
		 FROM uploads
		 WHERE workspace_id = ? AND file_id = ?`,
		workspaceID,
		fileID,
	)
	var upload storage.Upload
	err := row.Scan(&upload.ID, &upload.WorkspaceID, &upload.FileID, &upload.OriginalName, &upload.SizeBytes, &upload.QueuePosition, &upload.CreatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return storage.Upload{}, false, nil
		}
		return storage.Upload{}, false, err
	}
	return upload, true, nil
}

func (s *Service) listUploads(ctx context.Context, workspaceID int64) ([]storage.Upload, error) {
	rows, err := s.store.DB().QueryContext(
		ctx,
		`SELECT id, workspace_id, file_id, original_name, size_bytes, queue_position, created_at
		 FROM uploads
		 WHERE workspace_id = ?
		 ORDER BY queue_position ASC, created_at ASC`,
		workspaceID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var uploads []storage.Upload
	for rows.Next() {
		var upload storage.Upload
		if err := rows.Scan(&upload.ID, &upload.WorkspaceID, &upload.FileID, &upload.OriginalName, &upload.SizeBytes, &upload.QueuePosition, &upload.CreatedAt); err != nil {
			return nil, err
		}
		uploads = append(uploads, upload)
	}
	return uploads, rows.Err()
}

func (s *Service) deleteUploadRow(ctx context.Context, workspaceID int64, fileID string) (bool, error) {
	row, ok, err := s.getUpload(ctx, workspaceID, fileID)
	if err != nil || !ok {
		return ok, err
	}
	if _, err := s.store.DB().ExecContext(ctx, `DELETE FROM uploads WHERE id = ?`, row.ID); err != nil {
		return false, err
	}
	_, err = s.store.DB().ExecContext(
		ctx,
		`UPDATE uploads SET queue_position = queue_position - 1 WHERE workspace_id = ? AND queue_position > ?`,
		workspaceID,
		row.QueuePosition,
	)
	return true, err
}

func (s *Service) workspaceByID(ctx context.Context, workspaceID int64) (storage.Workspace, bool, error) {
	row := s.store.DB().QueryRowContext(
		ctx,
		`SELECT id, token_hash, created_at, updated_at, expires_at, config_json
		 FROM workspaces
		 WHERE id = ?`,
		workspaceID,
	)
	var workspace storage.Workspace
	err := row.Scan(&workspace.ID, &workspace.TokenHash, &workspace.CreatedAt, &workspace.UpdatedAt, &workspace.ExpiresAt, &workspace.ConfigJSON)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return storage.Workspace{}, false, nil
		}
		return storage.Workspace{}, false, err
	}
	return workspace, true, nil
}

func (s *Service) deleteWorkspace(ctx context.Context, workspaceID int64) error {
	if _, err := s.store.DB().ExecContext(ctx, `DELETE FROM workspaces WHERE id = ?`, workspaceID); err != nil {
		return err
	}
	return os.RemoveAll(s.store.WorkspaceDir(workspaceID))
}

func (s *Service) cleanupExpiredJobs(ctx context.Context, workspace Context) error {
	rows, err := s.store.DB().QueryContext(ctx, `SELECT job_id FROM jobs WHERE workspace_id = ? AND expires_at <= ?`, workspace.ID, unixSeconds(s.now()))
	if err != nil {
		return err
	}
	defer rows.Close()

	var jobIDs []string
	for rows.Next() {
		var jobID string
		if err := rows.Scan(&jobID); err != nil {
			return err
		}
		jobIDs = append(jobIDs, jobID)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	if _, err := s.store.DB().ExecContext(ctx, `DELETE FROM jobs WHERE workspace_id = ? AND expires_at <= ?`, workspace.ID, unixSeconds(s.now())); err != nil {
		return err
	}
	for _, jobID := range jobIDs {
		if err := os.RemoveAll(s.store.JobDir(workspace.ID, jobID)); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) listReadyJobs(ctx context.Context, workspaceID int64) ([]storage.Job, error) {
	rows, err := s.store.DB().QueryContext(
		ctx,
		`SELECT id, workspace_id, upload_id, job_id, status, source_name, download_name, size_bytes, error, created_at, updated_at, expires_at, config_json
		 FROM jobs
		 WHERE workspace_id = ? AND status = 'ready' AND expires_at > ?
		 ORDER BY updated_at DESC`,
		workspaceID,
		unixSeconds(s.now()),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var jobs []storage.Job
	for rows.Next() {
		var job storage.Job
		if err := rows.Scan(
			&job.ID,
			&job.WorkspaceID,
			&job.UploadID,
			&job.JobID,
			&job.Status,
			&job.SourceName,
			&job.DownloadName,
			&job.SizeBytes,
			&job.Error,
			&job.CreatedAt,
			&job.UpdatedAt,
			&job.ExpiresAt,
			&job.ConfigJSON,
		); err != nil {
			return nil, err
		}
		jobs = append(jobs, job)
	}
	return jobs, rows.Err()
}

func uploadPayloads(rows []storage.Upload) []UploadPayload {
	payloads := make([]UploadPayload, 0, len(rows))
	for _, row := range rows {
		payloads = append(payloads, uploadPayload(row))
	}
	return payloads
}

func uploadPayload(row storage.Upload) UploadPayload {
	return UploadPayload{
		FileID:    row.FileID,
		Name:      row.OriginalName,
		Size:      row.SizeBytes,
		CreatedAt: row.CreatedAt,
	}
}

func convertedPayloads(rows []storage.Job) []ConvertedPayload {
	payloads := make([]ConvertedPayload, 0, len(rows))
	for _, row := range rows {
		name := "output.wav"
		if row.DownloadName.Valid {
			name = row.DownloadName.String
		}
		var size int64
		if row.SizeBytes.Valid {
			size = row.SizeBytes.Int64
		}
		payloads = append(payloads, ConvertedPayload{
			JobID:       row.JobID,
			Name:        name,
			SourceName:  row.SourceName,
			Size:        size,
			DownloadURL: "/api/synthesis-jobs/" + row.JobID + "/download",
			DeleteURL:   "/api/synthesis-jobs/" + row.JobID,
			CreatedAt:   row.CreatedAt,
			UpdatedAt:   row.UpdatedAt,
			ExpiresAt:   row.ExpiresAt,
		})
	}
	return payloads
}

func sameIDSet(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	leftCopy := append([]string(nil), left...)
	rightCopy := append([]string(nil), right...)
	sort.Strings(leftCopy)
	sort.Strings(rightCopy)
	for index := range leftCopy {
		if leftCopy[index] != rightCopy[index] {
			return false
		}
		if index > 0 && rightCopy[index] == rightCopy[index-1] {
			return false
		}
	}
	return true
}

func moveFile(source, destination string) error {
	if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
		return err
	}
	if err := os.Rename(source, destination); err == nil {
		return nil
	}

	input, err := os.Open(source)
	if err != nil {
		return err
	}
	defer input.Close()

	output, err := os.OpenFile(destination, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	if _, err := io.Copy(output, input); err != nil {
		_ = output.Close()
		return err
	}
	if err := output.Close(); err != nil {
		return err
	}
	return os.Remove(source)
}

func copyFile(source, destination string) error {
	if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
		return err
	}
	input, err := os.Open(source)
	if err != nil {
		return err
	}
	defer input.Close()
	output, err := os.OpenFile(destination, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	if _, err := io.Copy(output, input); err != nil {
		_ = output.Close()
		return err
	}
	return output.Close()
}

func tokenURLSafe32() (string, error) {
	buffer := make([]byte, 32)
	if _, err := rand.Read(buffer); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buffer), nil
}

func randomHex16() (string, error) {
	buffer := make([]byte, 16)
	if _, err := rand.Read(buffer); err != nil {
		return "", err
	}
	return hex.EncodeToString(buffer), nil
}

func canonicalConfigJSON(config Config) string {
	var builder strings.Builder
	builder.WriteString(`{"layers":[`)
	for index, layer := range config.Layers {
		if index > 0 {
			builder.WriteByte(',')
		}
		builder.WriteString(`{"curve_enabled":`)
		builder.WriteString(strconv.FormatBool(layer.CurveEnabled))
		builder.WriteString(`,"duty":`)
		builder.WriteString(formatPythonFloat(layer.Duty))
		builder.WriteString(`,"frequency_curve":[`)
		for pointIndex, point := range layer.FrequencyCurve {
			if pointIndex > 0 {
				builder.WriteByte(',')
			}
			builder.WriteString(`{"frequency_hz":`)
			builder.WriteString(formatPythonFloat(point.FrequencyHz))
			builder.WriteString(`,"gain_db":`)
			builder.WriteString(formatPythonFloat(point.GainDB))
			builder.WriteByte('}')
		}
		builder.WriteString(`],"type":`)
		typeJSON, _ := json.Marshal(layer.Type)
		builder.Write(typeJSON)
		builder.WriteString(`,"volume":`)
		builder.WriteString(formatPythonFloat(layer.Volume))
		builder.WriteByte('}')
	}
	builder.WriteString(`],"sample_rate":`)
	builder.WriteString(strconv.Itoa(config.SampleRate))
	builder.WriteString(`,"schema":`)
	schemaJSON, _ := json.Marshal(config.Schema)
	builder.Write(schemaJSON)
	builder.WriteByte('}')
	return builder.String()
}

func formatPythonFloat(value float64) string {
	formatted := strconv.FormatFloat(value, 'f', -1, 64)
	if !strings.ContainsAny(formatted, ".eE") {
		formatted += ".0"
	}
	return formatted
}

func unixSeconds(value time.Time) float64 {
	return float64(value.UnixNano()) / float64(time.Second)
}
