package httpapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"octabit/backend/internal/config"
	"octabit/backend/internal/jobs"
	"octabit/backend/internal/workspace"
)

const workspaceCookieName = "octabit_workspace"

type healthResponse struct {
	Status  string `json:"status"`
	Service string `json:"service"`
}

type Options struct {
	WorkspaceService *workspace.Service
	LegacyJobs       *jobs.LegacyService
}

func NewRouter(cfg config.Config) http.Handler {
	return NewRouterWithOptions(cfg, Options{})
}

func NewRouterWithOptions(cfg config.Config, options Options) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/health", health)
	legacyJobs := options.LegacyJobs
	if legacyJobs == nil && cfg.JobRoot != "" {
		legacyJobs = jobs.NewLegacyService(cfg.JobRoot, jobs.Config{
			DownloadTTLSeconds: cfg.DownloadTTLSeconds,
			MaxWorkers:         cfg.RenderWorkers,
			MaxQueueSize:       cfg.RenderQueueSize,
		})
	}
	if options.WorkspaceService != nil {
		mux.HandleFunc("GET /api/workspace", getWorkspace(options.WorkspaceService))
		mux.HandleFunc("POST /api/workspace/uploads", createWorkspaceUpload(cfg, options.WorkspaceService))
		mux.HandleFunc("DELETE /api/workspace/uploads/{file_id}", deleteWorkspaceUpload(options.WorkspaceService))
		mux.HandleFunc("PATCH /api/workspace/queue", updateWorkspaceQueue(options.WorkspaceService))
		mux.HandleFunc("PUT /api/workspace/config", saveWorkspaceConfig(options.WorkspaceService))
		mux.HandleFunc("POST /api/synthesis-jobs", createSynthesisJob(cfg, options.WorkspaceService))
		mux.HandleFunc("GET /api/synthesis-jobs/{job_id}", getSynthesisJob(options.WorkspaceService))
		mux.HandleFunc("DELETE /api/synthesis-jobs/{job_id}", deleteSynthesisJob(options.WorkspaceService))
		mux.HandleFunc("GET /api/synthesis-jobs/{job_id}/download", downloadSynthesisJob(options.WorkspaceService))
	}
	mux.HandleFunc("POST /synthesise", synthesise(cfg))
	if legacyJobs != nil {
		mux.HandleFunc("POST /synthesise/jobs", createLegacySynthesisJob(cfg, legacyJobs))
		mux.HandleFunc("GET /synthesise/jobs/{job_id}", getLegacySynthesisJob(legacyJobs))
		mux.HandleFunc("DELETE /synthesise/jobs/{job_id}", deleteLegacySynthesisJob(legacyJobs))
		mux.HandleFunc("GET /synthesise/jobs/{job_id}/download", downloadLegacySynthesisJob(legacyJobs))
	}
	mux.HandleFunc("GET /static/previews/", previewAsset(cfg.PreviewAssetsDir))
	return mux
}

func health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, healthResponse{
		Status:  "ok",
		Service: "octabit-web",
	})
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeAPIError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, map[string]any{
		"error": map[string]string{
			"code":    code,
			"message": message,
		},
	})
}

func writeLegacyError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

func getWorkspace(service *workspace.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var token string
		if cookie, err := r.Cookie(workspaceCookieName); err == nil {
			token = cookie.Value
		}
		workspaceContext, err := service.GetOrCreateWorkspace(r.Context(), token)
		if err != nil {
			writeAPIError(w, http.StatusInternalServerError, "internal_error", "Unable to create workspace.")
			return
		}
		payload, err := service.StatePayload(r.Context(), workspaceContext)
		if err != nil {
			writeAPIError(w, http.StatusInternalServerError, "internal_error", "Unable to load workspace.")
			return
		}
		if workspaceContext.Created {
			setWorkspaceCookie(w, service, workspaceContext)
		}
		writeJSON(w, http.StatusOK, payload)
	}
}

func createWorkspaceUpload(cfg config.Config, service *workspace.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		workspaceContext, ok := activeWorkspaceOrError(w, r, service)
		if !ok {
			return
		}
		if cfg.MaxUploadBytes > 0 {
			r.Body = http.MaxBytesReader(w, r.Body, int64(cfg.MaxUploadBytes))
		}
		if err := r.ParseMultipartForm(int64(cfg.MaxUploadBytes)); err != nil {
			writeAPIError(w, http.StatusRequestEntityTooLarge, "upload_too_large", "Uploaded file is too large.")
			return
		}

		files := r.MultipartForm.File["midi_file"]
		if len(files) == 0 {
			writeAPIError(w, http.StatusBadRequest, "missing_midi_file", "No MIDI file uploaded")
			return
		}
		fileHeader := files[0]
		if fileHeader.Filename == "" {
			writeAPIError(w, http.StatusBadRequest, "no_selected_file", "No selected file")
			return
		}
		if !isSupportedMIDIFile(fileHeader.Filename) {
			writeAPIError(w, http.StatusUnsupportedMediaType, "unsupported_file_type", "Unsupported file type. Upload a .mid or .midi file.")
			return
		}

		tempPath, err := saveMultipartFileToTemp(fileHeader)
		if err != nil {
			writeAPIError(w, http.StatusInternalServerError, "internal_error", err.Error())
			return
		}
		defer cleanupTempPath(tempPath)

		info, err := os.Stat(tempPath)
		if err != nil {
			writeAPIError(w, http.StatusInternalServerError, "internal_error", err.Error())
			return
		}
		if info.Size() == 0 {
			writeAPIError(w, http.StatusBadRequest, "empty_midi_file", "Uploaded MIDI file is empty or incomplete.")
			return
		}

		upload, err := service.CreateUploadFromTemp(r.Context(), workspaceContext, tempPath, fileHeader.Filename)
		if err != nil {
			writeWorkspaceError(w, err, "internal_error")
			return
		}
		writeJSON(w, http.StatusCreated, map[string]workspace.UploadPayload{"upload": upload})
	}
}

func deleteWorkspaceUpload(service *workspace.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		fileID := r.PathValue("file_id")
		if !isValidResourceID(fileID) {
			writeAPIError(w, http.StatusBadRequest, "invalid_file_id", "Invalid file id.")
			return
		}
		workspaceContext, ok := activeWorkspaceOrError(w, r, service)
		if !ok {
			return
		}
		deleted, err := service.DeleteUpload(r.Context(), workspaceContext, fileID)
		if err != nil {
			writeWorkspaceError(w, err, "internal_error")
			return
		}
		if !deleted {
			writeAPIError(w, http.StatusNotFound, "not_found", "File not found.")
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusNoContent)
	}
}

func updateWorkspaceQueue(service *workspace.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		workspaceContext, ok := activeWorkspaceOrError(w, r, service)
		if !ok {
			return
		}
		var payload struct {
			FileIDs []string `json:"file_ids"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil || payload.FileIDs == nil {
			writeAPIError(w, http.StatusUnprocessableEntity, "invalid_queue", "Request body must contain file_ids.")
			return
		}
		for _, fileID := range payload.FileIDs {
			if !isValidResourceID(fileID) {
				writeAPIError(w, http.StatusBadRequest, "invalid_file_id", "Invalid file id.")
				return
			}
		}
		uploads, err := service.ReplaceQueue(r.Context(), workspaceContext, payload.FileIDs)
		if err != nil {
			writeWorkspaceError(w, err, "internal_error")
			return
		}
		writeJSON(w, http.StatusOK, map[string][]workspace.UploadPayload{"uploads": uploads})
	}
}

func saveWorkspaceConfig(service *workspace.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		workspaceContext, ok := activeWorkspaceOrError(w, r, service)
		if !ok {
			return
		}
		var payload map[string]any
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			writeAPIError(w, http.StatusUnprocessableEntity, "invalid_workspace_config", "Workspace config must be a JSON object.")
			return
		}
		normalisedConfig, err := workspace.NormaliseConfig(payload)
		if err != nil {
			writeAPIError(w, http.StatusUnprocessableEntity, "invalid_workspace_config", err.Error())
			return
		}
		savedConfig, err := service.SaveConfig(r.Context(), workspaceContext, normalisedConfig)
		if err != nil {
			writeWorkspaceError(w, err, "internal_error")
			return
		}
		writeJSON(w, http.StatusOK, map[string]workspace.Config{"config": savedConfig})
	}
}

func createSynthesisJob(cfg config.Config, service *workspace.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !isJSONRequest(r) {
			createMultipartSynthesisJob(w, r, cfg, service)
			return
		}
		workspaceContext, ok := activeWorkspaceOrError(w, r, service)
		if !ok {
			return
		}

		var payload map[string]any
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			payload = map[string]any{}
		}
		fileID, _ := payload["file_id"].(string)
		if !isValidResourceID(fileID) {
			writeAPIError(w, http.StatusBadRequest, "invalid_file_id", "Invalid file id.")
			return
		}
		configPayload, _ := payload["config"].(map[string]any)
		normalisedConfig, err := workspace.NormaliseConfig(configPayload)
		if err != nil {
			writeAPIError(w, http.StatusUnprocessableEntity, "invalid_workspace_config", err.Error())
			return
		}

		jobPayload, err := service.CreateJobFromUpload(r.Context(), workspaceContext, fileID, normalisedConfig)
		if err != nil {
			if errors.Is(err, jobs.ErrRenderQueueFull) {
				writeAPIError(w, http.StatusTooManyRequests, "render_queue_full", err.Error())
				return
			}
			writeWorkspaceError(w, err, "internal_error")
			return
		}
		writeJSON(w, http.StatusAccepted, jobPayload)
	}
}

func createMultipartSynthesisJob(w http.ResponseWriter, r *http.Request, cfg config.Config, service *workspace.Service) {
	if cfg.MaxUploadBytes > 0 {
		r.Body = http.MaxBytesReader(w, r.Body, int64(cfg.MaxUploadBytes))
	}
	if err := r.ParseMultipartForm(int64(cfg.MaxUploadBytes)); err != nil {
		writeAPIError(w, http.StatusRequestEntityTooLarge, "upload_too_large", "Uploaded file is too large.")
		return
	}
	files := r.MultipartForm.File["midi_file"]
	if len(files) == 0 {
		writeAPIError(w, http.StatusBadRequest, "missing_midi_file", "No MIDI file uploaded")
		return
	}
	fileHeader := files[0]
	if fileHeader.Filename == "" {
		writeAPIError(w, http.StatusBadRequest, "no_selected_file", "No selected file")
		return
	}
	if !isSupportedMIDIFile(fileHeader.Filename) {
		writeAPIError(w, http.StatusUnsupportedMediaType, "unsupported_file_type", "Unsupported file type. Upload a .mid or .midi file.")
		return
	}

	workspaceConfig, err := workspace.ConfigFromFormValues(r.FormValue("rate"), r.FormValue("layers_json"))
	if err != nil {
		code := "invalid_layers"
		if strings.Contains(err.Error(), "sample rate") {
			code = "invalid_sample_rate"
		}
		writeAPIError(w, http.StatusUnprocessableEntity, code, err.Error())
		return
	}

	tempPath, err := saveMultipartFileToTemp(fileHeader)
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	defer cleanupTempPath(tempPath)
	info, err := os.Stat(tempPath)
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	if info.Size() == 0 {
		writeAPIError(w, http.StatusBadRequest, "empty_midi_file", "Uploaded MIDI file is empty or incomplete.")
		return
	}

	var token string
	if cookie, err := r.Cookie(workspaceCookieName); err == nil {
		token = cookie.Value
	}
	workspaceContext, err := service.GetOrCreateWorkspace(r.Context(), token)
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "internal_error", "Unable to create workspace.")
		return
	}
	jobPayload, err := service.CreateJobFromTemp(r.Context(), workspaceContext, tempPath, fileHeader.Filename, workspaceConfig)
	if err != nil {
		if errors.Is(err, jobs.ErrRenderQueueFull) {
			writeAPIError(w, http.StatusTooManyRequests, "render_queue_full", err.Error())
			return
		}
		writeWorkspaceError(w, err, "internal_error")
		return
	}
	if workspaceContext.Created {
		setWorkspaceCookie(w, service, workspaceContext)
	}
	writeJSON(w, http.StatusAccepted, jobPayload)
}

func getSynthesisJob(service *workspace.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		jobID := r.PathValue("job_id")
		if !isValidResourceID(jobID) {
			writeAPIError(w, http.StatusBadRequest, "invalid_job_id", "Invalid job id.")
			return
		}
		workspaceContext, ok := activeWorkspaceOrError(w, r, service)
		if !ok {
			return
		}
		job, expired, found, err := service.GetJob(r.Context(), workspaceContext, jobID)
		if err != nil {
			writeWorkspaceError(w, err, "internal_error")
			return
		}
		if expired {
			writeJSON(w, http.StatusGone, map[string]string{"job_id": jobID, "status": "expired"})
			return
		}
		if !found {
			writeAPIError(w, http.StatusNotFound, "not_found", "Job not found.")
			return
		}
		writeJSON(w, http.StatusOK, service.JobPayload(job))
	}
}

func deleteSynthesisJob(service *workspace.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		jobID := r.PathValue("job_id")
		if !isValidResourceID(jobID) {
			writeAPIError(w, http.StatusBadRequest, "invalid_job_id", "Invalid job id.")
			return
		}
		workspaceContext, ok := activeWorkspaceOrError(w, r, service)
		if !ok {
			return
		}
		deleted, err := service.DeleteJob(r.Context(), workspaceContext, jobID)
		if err != nil {
			writeWorkspaceError(w, err, "internal_error")
			return
		}
		if !deleted {
			writeAPIError(w, http.StatusNotFound, "not_found", "Job not found.")
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusNoContent)
	}
}

func downloadSynthesisJob(service *workspace.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		jobID := r.PathValue("job_id")
		if !isValidResourceID(jobID) {
			writeAPIError(w, http.StatusBadRequest, "invalid_job_id", "Invalid job id.")
			return
		}
		workspaceContext, ok := activeWorkspaceOrError(w, r, service)
		if !ok {
			return
		}
		job, expired, found, err := service.GetJob(r.Context(), workspaceContext, jobID)
		if err != nil {
			writeWorkspaceError(w, err, "internal_error")
			return
		}
		if expired {
			writeJSON(w, http.StatusGone, map[string]string{"job_id": jobID, "status": "expired"})
			return
		}
		if !found {
			writeAPIError(w, http.StatusNotFound, "not_found", "Job not found.")
			return
		}
		payload := service.JobPayload(job)
		if job.Status == jobs.StatusFailed {
			writeJSON(w, http.StatusBadRequest, payload)
			return
		}
		if job.Status != jobs.StatusReady {
			writeJSON(w, http.StatusConflict, payload)
			return
		}

		outputPath := service.JobOutputPath(workspaceContext, jobID)
		if _, err := os.Stat(outputPath); err != nil {
			_, _ = service.DeleteJob(r.Context(), workspaceContext, jobID)
			writeJSON(w, http.StatusGone, map[string]string{"job_id": jobID, "status": "expired"})
			return
		}
		downloadName := payload.DownloadName
		if downloadName == "" {
			downloadName = "output.wav"
		}
		w.Header().Set("Content-Type", "audio/x-wav")
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", downloadName))
		http.ServeFile(w, r, outputPath)
	}
}

func createLegacySynthesisJob(cfg config.Config, legacyJobs *jobs.LegacyService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if cfg.MaxUploadBytes > 0 {
			r.Body = http.MaxBytesReader(w, r.Body, int64(cfg.MaxUploadBytes))
		}
		if err := r.ParseMultipartForm(int64(cfg.MaxUploadBytes)); err != nil {
			writeLegacyError(w, http.StatusRequestEntityTooLarge, "Uploaded file is too large.")
			return
		}
		files := r.MultipartForm.File["midi_file"]
		if len(files) == 0 {
			writeLegacyError(w, http.StatusBadRequest, "No MIDI file uploaded")
			return
		}
		fileHeader := files[0]
		if fileHeader.Filename == "" {
			writeLegacyError(w, http.StatusBadRequest, "No selected file")
			return
		}
		if !isSupportedMIDIFile(fileHeader.Filename) {
			writeLegacyError(w, http.StatusBadRequest, "Unsupported file type. Upload a .mid or .midi file.")
			return
		}
		workspaceConfig, err := workspace.ConfigFromFormValues(r.FormValue("rate"), r.FormValue("layers_json"))
		if err != nil {
			writeLegacyError(w, http.StatusBadRequest, err.Error())
			return
		}

		tempPath, err := saveMultipartFileToTemp(fileHeader)
		if err != nil {
			writeLegacyError(w, http.StatusInternalServerError, err.Error())
			return
		}
		defer cleanupTempPath(tempPath)
		info, err := os.Stat(tempPath)
		if err != nil {
			writeLegacyError(w, http.StatusInternalServerError, err.Error())
			return
		}
		if info.Size() == 0 {
			writeLegacyError(w, http.StatusBadRequest, "Uploaded MIDI file is empty or incomplete.")
			return
		}

		job, err := legacyJobs.CreateFromTemp(tempPath, fileHeader.Filename, workspaceConfig.SampleRate, workspace.RuntimeLayers(workspaceConfig))
		if err != nil {
			if errors.Is(err, jobs.ErrRenderQueueFull) {
				writeLegacyError(w, http.StatusTooManyRequests, err.Error())
				return
			}
			writeLegacyError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusAccepted, legacyJobs.Payload(*job))
	}
}

func synthesise(cfg config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if cfg.MaxUploadBytes > 0 {
			r.Body = http.MaxBytesReader(w, r.Body, int64(cfg.MaxUploadBytes))
		}
		if err := r.ParseMultipartForm(int64(cfg.MaxUploadBytes)); err != nil {
			writeLegacyError(w, http.StatusRequestEntityTooLarge, "Uploaded file is too large.")
			return
		}
		files := r.MultipartForm.File["midi_file"]
		if len(files) == 0 {
			writeLegacyError(w, http.StatusBadRequest, "No MIDI file uploaded")
			return
		}
		fileHeader := files[0]
		if fileHeader.Filename == "" {
			writeLegacyError(w, http.StatusBadRequest, "No selected file")
			return
		}
		if !isSupportedMIDIFile(fileHeader.Filename) {
			writeLegacyError(w, http.StatusBadRequest, "Unsupported file type. Upload a .mid or .midi file.")
			return
		}
		workspaceConfig, err := workspace.ConfigFromFormValues(r.FormValue("rate"), r.FormValue("layers_json"))
		if err != nil {
			writeLegacyError(w, http.StatusBadRequest, err.Error())
			return
		}

		tempPath, err := saveMultipartFileToTemp(fileHeader)
		if err != nil {
			writeLegacyError(w, http.StatusInternalServerError, err.Error())
			return
		}
		defer cleanupTempPath(tempPath)
		info, err := os.Stat(tempPath)
		if err != nil {
			writeLegacyError(w, http.StatusInternalServerError, err.Error())
			return
		}
		if info.Size() == 0 {
			writeLegacyError(w, http.StatusBadRequest, "Uploaded MIDI file is empty or incomplete.")
			return
		}

		downloadName, wavData, err := jobs.RenderWAVBytes(tempPath, fileHeader.Filename, workspaceConfig.SampleRate, workspace.RuntimeLayers(workspaceConfig))
		if err != nil {
			writeLegacyError(w, http.StatusBadRequest, err.Error())
			return
		}
		w.Header().Set("Content-Type", "audio/x-wav")
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", downloadName))
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(wavData)
	}
}

func getLegacySynthesisJob(legacyJobs *jobs.LegacyService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		jobID := r.PathValue("job_id")
		job, expired, ok := legacyJobs.Get(jobID)
		if expired || !ok {
			writeJSON(w, http.StatusGone, map[string]string{"job_id": jobID, "status": "expired"})
			return
		}
		payload := legacyJobs.Payload(*job)
		if payload.Status == jobs.StatusExpired {
			writeJSON(w, http.StatusGone, payload)
			return
		}
		writeJSON(w, http.StatusOK, payload)
	}
}

func deleteLegacySynthesisJob(legacyJobs *jobs.LegacyService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		legacyJobs.Delete(r.PathValue("job_id"))
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusNoContent)
	}
}

func downloadLegacySynthesisJob(legacyJobs *jobs.LegacyService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		jobID := r.PathValue("job_id")
		job, expired, ok := legacyJobs.Get(jobID)
		if expired || !ok {
			writeJSON(w, http.StatusGone, map[string]string{"job_id": jobID, "status": "expired"})
			return
		}
		payload := legacyJobs.Payload(*job)
		if job.Status == jobs.StatusFailed {
			writeJSON(w, http.StatusBadRequest, payload)
			return
		}
		if job.Status != jobs.StatusReady {
			writeJSON(w, http.StatusConflict, payload)
			return
		}
		outputPath := legacyJobs.OutputPath(jobID)
		if _, err := os.Stat(outputPath); err != nil {
			legacyJobs.Delete(jobID)
			writeJSON(w, http.StatusGone, map[string]string{"job_id": jobID, "status": "expired"})
			return
		}
		downloadName := payload.DownloadName
		if downloadName == "" {
			downloadName = "output.wav"
		}
		w.Header().Set("Content-Type", "audio/x-wav")
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", downloadName))
		http.ServeFile(w, r, outputPath)
	}
}

func activeWorkspaceOrError(w http.ResponseWriter, r *http.Request, service *workspace.Service) (workspace.Context, bool) {
	var token string
	if cookie, err := r.Cookie(workspaceCookieName); err == nil {
		token = cookie.Value
	}
	workspaceContext, ok, err := service.GetActiveWorkspace(r.Context(), token)
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "internal_error", "Unable to load workspace.")
		return workspace.Context{}, false
	}
	if !ok {
		writeAPIError(w, http.StatusGone, "workspace_expired", "The temporary workspace has expired. Refresh to start a new workspace.")
		return workspace.Context{}, false
	}
	return workspaceContext, true
}

func setWorkspaceCookie(w http.ResponseWriter, service *workspace.Service, workspaceContext workspace.Context) {
	http.SetCookie(w, &http.Cookie{
		Name:     workspaceCookieName,
		Value:    workspaceContext.Token,
		Path:     "/",
		MaxAge:   service.TTLSeconds(),
		Expires:  time.Now().Add(time.Duration(service.TTLSeconds()) * time.Second),
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
}

func isJSONRequest(r *http.Request) bool {
	contentType := r.Header.Get("Content-Type")
	return contentType == "application/json" || strings.HasPrefix(contentType, "application/json;")
}

func writeWorkspaceError(w http.ResponseWriter, err error, fallbackCode string) {
	var workspaceError workspace.Error
	if errors.As(err, &workspaceError) {
		writeAPIError(w, workspaceError.StatusCode, workspaceError.Code, workspaceError.Message)
		return
	}
	writeAPIError(w, http.StatusInternalServerError, fallbackCode, err.Error())
}

func isSupportedMIDIFile(name string) bool {
	extension := strings.ToLower(filepath.Ext(name))
	return extension == ".mid" || extension == ".midi"
}

func isValidResourceID(value string) bool {
	if len(value) != 32 {
		return false
	}
	for _, char := range value {
		if !('0' <= char && char <= '9') && !('a' <= char && char <= 'f') {
			return false
		}
	}
	return true
}

func saveMultipartFileToTemp(fileHeader *multipart.FileHeader) (string, error) {
	source, err := fileHeader.Open()
	if err != nil {
		return "", err
	}
	defer source.Close()

	tempFile, err := os.CreateTemp("", "octabit-upload-*.mid")
	if err != nil {
		return "", err
	}
	tempPath := tempFile.Name()
	if _, err := io.Copy(tempFile, source); err != nil {
		_ = tempFile.Close()
		_ = os.Remove(tempPath)
		return "", err
	}
	if err := tempFile.Close(); err != nil {
		_ = os.Remove(tempPath)
		return "", err
	}
	return tempPath, nil
}

func cleanupTempPath(path string) {
	if path != "" {
		_ = os.Remove(path)
	}
}

func previewAsset(previewAssetsDir string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		relativeName := strings.TrimPrefix(r.URL.Path, "/static/previews/")
		if relativeName == "" {
			http.NotFound(w, r)
			return
		}

		cleanName := filepath.Clean(relativeName)
		if cleanName == "." || strings.HasPrefix(cleanName, "..") {
			http.NotFound(w, r)
			return
		}

		path := filepath.Join(previewAssetsDir, cleanName)
		if !isWithin(previewAssetsDir, path) {
			http.NotFound(w, r)
			return
		}

		w.Header().Set("Content-Type", "audio/x-wav")
		w.Header().Set("Content-Disposition", fmt.Sprintf("inline; filename=%s", filepath.Base(cleanName)))
		http.ServeFile(w, r, path)
	}
}

func isWithin(root, path string) bool {
	relativePath, err := filepath.Rel(root, path)
	if err != nil {
		return false
	}
	return relativePath == "." || (!strings.HasPrefix(relativePath, "..") && !filepath.IsAbs(relativePath))
}
