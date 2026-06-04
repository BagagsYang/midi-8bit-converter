package httpapi

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"octabit/backend/internal/config"
	"octabit/backend/internal/storage"
	"octabit/backend/internal/workspace"
)

func TestHealthMatchesOpenAPIContract(t *testing.T) {
	router := NewRouter(config.Config{})
	request := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	if contentType := response.Header().Get("Content-Type"); contentType != "application/json" {
		t.Fatalf("Content-Type = %q", contentType)
	}

	var payload map[string]string
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if payload["status"] != "ok" || payload["service"] != "octabit-web" {
		t.Fatalf("payload = %#v", payload)
	}
}

func TestPreviewAssetsRouteServesConfiguredDirectory(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "pulse_50.wav"), []byte("RIFF fixture"), 0o644); err != nil {
		t.Fatal(err)
	}
	router := NewRouter(config.Config{PreviewAssetsDir: dir})
	request := httptest.NewRequest(http.MethodGet, "/static/previews/pulse_50.wav", nil)
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	if body := response.Body.String(); body != "RIFF fixture" {
		t.Fatalf("body = %q", body)
	}
}

func TestGetWorkspaceCreatesCookieAndReusesWorkspace(t *testing.T) {
	ctx := t.Context()
	store, err := storage.Open(ctx, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	service := workspace.NewService(store, config.Config{
		WorkspaceTTLSeconds:        60,
		WorkspaceMaxQueuedFiles:    20,
		WorkspaceMaxUploadBytes:    104857600,
		WorkspaceMaxConvertedFiles: 20,
	}, workspace.Options{
		Now:           func() time.Time { return time.Unix(100, 0) },
		GenerateToken: sequenceGenerator("workspace-token-a"),
	})
	router := NewRouterWithOptions(config.Config{}, Options{WorkspaceService: service})

	firstRequest := httptest.NewRequest(http.MethodGet, "/api/workspace", nil)
	firstResponse := httptest.NewRecorder()
	router.ServeHTTP(firstResponse, firstRequest)

	if firstResponse.Code != http.StatusOK {
		t.Fatalf("first status = %d", firstResponse.Code)
	}
	cookies := firstResponse.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("cookies = %#v", cookies)
	}
	if cookies[0].Name != "octabit_workspace" || cookies[0].Value != "workspace-token-a" || !cookies[0].HttpOnly {
		t.Fatalf("workspace cookie = %#v", cookies[0])
	}
	var firstPayload map[string]any
	if err := json.Unmarshal(firstResponse.Body.Bytes(), &firstPayload); err != nil {
		t.Fatal(err)
	}
	if _, ok := firstPayload["workspace"].(map[string]any)["id"]; ok {
		t.Fatalf("workspace id leaked in payload: %#v", firstPayload["workspace"])
	}
	if got := firstPayload["uploads"].([]any); len(got) != 0 {
		t.Fatalf("uploads = %#v", got)
	}
	if got := firstPayload["converted_files"].([]any); len(got) != 0 {
		t.Fatalf("converted_files = %#v", got)
	}

	secondRequest := httptest.NewRequest(http.MethodGet, "/api/workspace", nil)
	secondRequest.AddCookie(cookies[0])
	secondResponse := httptest.NewRecorder()
	router.ServeHTTP(secondResponse, secondRequest)

	if secondResponse.Code != http.StatusOK {
		t.Fatalf("second status = %d", secondResponse.Code)
	}
	if setCookie := secondResponse.Header().Get("Set-Cookie"); setCookie != "" {
		t.Fatalf("unexpected replacement cookie = %q", setCookie)
	}
}

func TestWorkspaceUploadQueueConfigAndDeleteRoutes(t *testing.T) {
	ctx := t.Context()
	store, err := storage.Open(ctx, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	service := workspace.NewService(store, config.Config{
		WorkspaceTTLSeconds:        60,
		WorkspaceMaxQueuedFiles:    20,
		WorkspaceMaxUploadBytes:    104857600,
		WorkspaceMaxConvertedFiles: 20,
		MaxUploadBytes:             1024,
	}, workspace.Options{
		Now: func() time.Time { return time.Unix(100, 0) },
		GenerateToken: sequenceGenerator(
			"workspace-token-a",
		),
		GenerateResourceID: sequenceGenerator(
			"00000000000000000000000000000001",
			"00000000000000000000000000000002",
		),
	})
	router := NewRouterWithOptions(config.Config{MaxUploadBytes: 1024}, Options{WorkspaceService: service})
	workspaceCookie := createWorkspaceCookie(t, router)

	firstUpload := postMIDIUpload(t, router, workspaceCookie, "first.mid", []byte("MThd first"))
	secondUpload := postMIDIUpload(t, router, workspaceCookie, "second.mid", []byte("MThd second"))

	queueRequest := jsonRequest(t, http.MethodPatch, "/api/workspace/queue", map[string]any{
		"file_ids": []string{secondUpload.FileID, firstUpload.FileID},
	})
	queueRequest.AddCookie(workspaceCookie)
	queueResponse := httptest.NewRecorder()
	router.ServeHTTP(queueResponse, queueRequest)
	if queueResponse.Code != http.StatusOK {
		t.Fatalf("queue status = %d body=%s", queueResponse.Code, queueResponse.Body.String())
	}
	var queuePayload struct {
		Uploads []workspace.UploadPayload `json:"uploads"`
	}
	if err := json.Unmarshal(queueResponse.Body.Bytes(), &queuePayload); err != nil {
		t.Fatal(err)
	}
	if got := []string{queuePayload.Uploads[0].FileID, queuePayload.Uploads[1].FileID}; got[0] != secondUpload.FileID || got[1] != firstUpload.FileID {
		t.Fatalf("queue ids = %#v", got)
	}

	configRequest := jsonRequest(t, http.MethodPut, "/api/workspace/config", map[string]any{
		"schema":      workspace.ConfigSchema,
		"sample_rate": "44100",
		"layers": []any{
			map[string]any{
				"type":          "triangle",
				"duty":          0.5,
				"volume":        0.4,
				"curve_enabled": false,
				"frequency_curve": []any{
					map[string]any{"frequency_hz": 8.175798915643707, "gain_db": 0.0},
					map[string]any{"frequency_hz": 12543.853951415975, "gain_db": 0.0},
				},
			},
		},
	})
	configRequest.AddCookie(workspaceCookie)
	configResponse := httptest.NewRecorder()
	router.ServeHTTP(configResponse, configRequest)
	if configResponse.Code != http.StatusOK {
		t.Fatalf("config status = %d body=%s", configResponse.Code, configResponse.Body.String())
	}
	var configPayload struct {
		Config workspace.Config `json:"config"`
	}
	if err := json.Unmarshal(configResponse.Body.Bytes(), &configPayload); err != nil {
		t.Fatal(err)
	}
	if configPayload.Config.SampleRate != 44100 || configPayload.Config.Layers[0].Type != "triangle" {
		t.Fatalf("config payload = %#v", configPayload.Config)
	}

	deleteRequest := httptest.NewRequest(http.MethodDelete, "/api/workspace/uploads/"+firstUpload.FileID, nil)
	deleteRequest.AddCookie(workspaceCookie)
	deleteResponse := httptest.NewRecorder()
	router.ServeHTTP(deleteResponse, deleteRequest)
	if deleteResponse.Code != http.StatusNoContent {
		t.Fatalf("delete status = %d body=%s", deleteResponse.Code, deleteResponse.Body.String())
	}
	if _, ok, err := service.GetUpload(ctx, workspace.Context{ID: 1}, firstUpload.FileID); err != nil || ok {
		t.Fatalf("deleted upload lookup ok=%v err=%v", ok, err)
	}
}

func TestWorkspaceRoutesRequireActiveWorkspace(t *testing.T) {
	ctx := t.Context()
	store, err := storage.Open(ctx, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	service := workspace.NewService(store, config.Config{
		WorkspaceTTLSeconds:        60,
		WorkspaceMaxQueuedFiles:    20,
		WorkspaceMaxUploadBytes:    104857600,
		WorkspaceMaxConvertedFiles: 20,
		MaxUploadBytes:             1024,
	}, workspace.Options{Now: func() time.Time { return time.Unix(100, 0) }})
	router := NewRouterWithOptions(config.Config{MaxUploadBytes: 1024}, Options{WorkspaceService: service})

	tests := []struct {
		method string
		path   string
		body   []byte
	}{
		{method: http.MethodDelete, path: "/api/workspace/uploads/00000000000000000000000000000000"},
		{method: http.MethodPatch, path: "/api/workspace/queue", body: []byte(`{"file_ids":[]}`)},
		{method: http.MethodPut, path: "/api/workspace/config", body: []byte(`{"schema":"wrong"}`)},
	}
	for _, tt := range tests {
		t.Run(tt.method+" "+tt.path, func(t *testing.T) {
			request := httptest.NewRequest(tt.method, tt.path, bytes.NewReader(tt.body))
			if tt.body != nil {
				request.Header.Set("Content-Type", "application/json")
			}
			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)
			if response.Code != http.StatusGone {
				t.Fatalf("status = %d body=%s", response.Code, response.Body.String())
			}
			var payload map[string]map[string]string
			if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
				t.Fatal(err)
			}
			if payload["error"]["code"] != "workspace_expired" {
				t.Fatalf("payload = %#v", payload)
			}
		})
	}
}

func TestSynthesisJobWorkspaceFileFlow(t *testing.T) {
	ctx := t.Context()
	store, err := storage.Open(ctx, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	service := workspace.NewService(store, config.Config{
		WorkspaceTTLSeconds:        60,
		WorkspaceMaxQueuedFiles:    20,
		WorkspaceMaxUploadBytes:    104857600,
		WorkspaceMaxConvertedFiles: 20,
		MaxUploadBytes:             1024 * 1024,
		RenderWorkers:              1,
		RenderQueueSize:            0,
	}, workspace.Options{
		RunInline:     true,
		Now:           func() time.Time { return time.Unix(100, 0) },
		GenerateToken: sequenceGenerator("workspace-token-a"),
		GenerateResourceID: sequenceGenerator(
			"00000000000000000000000000000001",
			"00000000000000000000000000000002",
			"00000000000000000000000000000003",
		),
	})
	router := NewRouterWithOptions(config.Config{MaxUploadBytes: 1024 * 1024}, Options{WorkspaceService: service})
	workspaceCookie := createWorkspaceCookie(t, router)
	midiBytes, err := os.ReadFile(filepath.Join("..", "..", "testdata", "python-baseline", "midi", "simple.mid"))
	if err != nil {
		t.Fatal(err)
	}
	upload := postMIDIUpload(t, router, workspaceCookie, "lead.mid", midiBytes)

	createRequest := jsonRequest(t, http.MethodPost, "/api/synthesis-jobs", map[string]any{
		"file_id": upload.FileID,
		"config":  defaultWorkspaceConfigPayload(),
	})
	createRequest.AddCookie(workspaceCookie)
	createResponse := httptest.NewRecorder()
	router.ServeHTTP(createResponse, createRequest)
	if createResponse.Code != http.StatusAccepted {
		t.Fatalf("create status = %d body=%s", createResponse.Code, createResponse.Body.String())
	}
	var createPayload workspace.JobPayload
	if err := json.Unmarshal(createResponse.Body.Bytes(), &createPayload); err != nil {
		t.Fatal(err)
	}
	if createPayload.Status != "ready" || createPayload.DownloadName != "lead_pulse.wav" {
		t.Fatalf("create payload = %#v", createPayload)
	}
	if createPayload.DownloadURL != "/api/synthesis-jobs/"+createPayload.JobID+"/download" {
		t.Fatalf("download_url = %q", createPayload.DownloadURL)
	}

	pollRequest := httptest.NewRequest(http.MethodGet, "/api/synthesis-jobs/"+createPayload.JobID, nil)
	pollRequest.AddCookie(workspaceCookie)
	pollResponse := httptest.NewRecorder()
	router.ServeHTTP(pollResponse, pollRequest)
	if pollResponse.Code != http.StatusOK {
		t.Fatalf("poll status = %d body=%s", pollResponse.Code, pollResponse.Body.String())
	}

	downloadRequest := httptest.NewRequest(http.MethodGet, createPayload.DownloadURL, nil)
	downloadRequest.AddCookie(workspaceCookie)
	downloadResponse := httptest.NewRecorder()
	router.ServeHTTP(downloadResponse, downloadRequest)
	if downloadResponse.Code != http.StatusOK {
		t.Fatalf("download status = %d body=%s", downloadResponse.Code, downloadResponse.Body.String())
	}
	if !bytes.HasPrefix(downloadResponse.Body.Bytes(), []byte("RIFF")) {
		t.Fatalf("download prefix = %x", downloadResponse.Body.Bytes()[:4])
	}
	if disposition := downloadResponse.Header().Get("Content-Disposition"); disposition != "attachment; filename=lead_pulse.wav" {
		t.Fatalf("Content-Disposition = %q", disposition)
	}

	nonDefaultRequest := jsonRequest(t, http.MethodPost, "/api/synthesis-jobs", map[string]any{
		"file_id": upload.FileID,
		"config":  sineWorkspaceConfigPayload(),
	})
	nonDefaultRequest.AddCookie(workspaceCookie)
	nonDefaultCreateResponse := httptest.NewRecorder()
	router.ServeHTTP(nonDefaultCreateResponse, nonDefaultRequest)
	if nonDefaultCreateResponse.Code != http.StatusAccepted {
		t.Fatalf("non-default create status = %d body=%s", nonDefaultCreateResponse.Code, nonDefaultCreateResponse.Body.String())
	}
	var nonDefaultPayload workspace.JobPayload
	if err := json.Unmarshal(nonDefaultCreateResponse.Body.Bytes(), &nonDefaultPayload); err != nil {
		t.Fatal(err)
	}
	if nonDefaultPayload.Status != "ready" || nonDefaultPayload.DownloadName != "lead_sine.wav" {
		t.Fatalf("non-default create payload = %#v", nonDefaultPayload)
	}
	nonDefaultDownloadRequest := httptest.NewRequest(http.MethodGet, nonDefaultPayload.DownloadURL, nil)
	nonDefaultDownloadRequest.AddCookie(workspaceCookie)
	nonDefaultDownloadResponse := httptest.NewRecorder()
	router.ServeHTTP(nonDefaultDownloadResponse, nonDefaultDownloadRequest)
	if nonDefaultDownloadResponse.Code != http.StatusOK {
		t.Fatalf("non-default download status = %d body=%s", nonDefaultDownloadResponse.Code, nonDefaultDownloadResponse.Body.String())
	}
	if bytes.Equal(downloadResponse.Body.Bytes(), nonDefaultDownloadResponse.Body.Bytes()) {
		t.Fatal("non-default synthesis config did not change downloaded WAV bytes")
	}

	deleteRequest := httptest.NewRequest(http.MethodDelete, createPayload.DeleteURL, nil)
	deleteRequest.AddCookie(workspaceCookie)
	deleteResponse := httptest.NewRecorder()
	router.ServeHTTP(deleteResponse, deleteRequest)
	if deleteResponse.Code != http.StatusNoContent {
		t.Fatalf("delete status = %d body=%s", deleteResponse.Code, deleteResponse.Body.String())
	}
	notFoundRequest := httptest.NewRequest(http.MethodGet, "/api/synthesis-jobs/"+createPayload.JobID, nil)
	notFoundRequest.AddCookie(workspaceCookie)
	notFoundResponse := httptest.NewRecorder()
	router.ServeHTTP(notFoundResponse, notFoundRequest)
	if notFoundResponse.Code != http.StatusNotFound {
		t.Fatalf("post-delete status = %d body=%s", notFoundResponse.Code, notFoundResponse.Body.String())
	}
}

func TestSynthesisJobConvertedLimitAndOwnership(t *testing.T) {
	ctx := t.Context()
	store, err := storage.Open(ctx, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	service := workspace.NewService(store, config.Config{
		WorkspaceTTLSeconds:        60,
		WorkspaceMaxQueuedFiles:    20,
		WorkspaceMaxUploadBytes:    104857600,
		WorkspaceMaxConvertedFiles: 1,
		MaxUploadBytes:             1024 * 1024,
		RenderWorkers:              1,
		RenderQueueSize:            0,
	}, workspace.Options{
		RunInline:     true,
		Now:           func() time.Time { return time.Unix(100, 0) },
		GenerateToken: sequenceGenerator("workspace-token-a", "workspace-token-b"),
		GenerateResourceID: sequenceGenerator(
			"00000000000000000000000000000001",
			"00000000000000000000000000000002",
			"00000000000000000000000000000003",
		),
	})
	router := NewRouterWithOptions(config.Config{MaxUploadBytes: 1024 * 1024}, Options{WorkspaceService: service})
	workspaceCookie := createWorkspaceCookie(t, router)
	midiBytes, err := os.ReadFile(filepath.Join("..", "..", "testdata", "python-baseline", "midi", "simple.mid"))
	if err != nil {
		t.Fatal(err)
	}
	upload := postMIDIUpload(t, router, workspaceCookie, "lead.mid", midiBytes)

	firstRequest := jsonRequest(t, http.MethodPost, "/api/synthesis-jobs", map[string]any{
		"file_id": upload.FileID,
		"config":  defaultWorkspaceConfigPayload(),
	})
	firstRequest.AddCookie(workspaceCookie)
	firstResponse := httptest.NewRecorder()
	router.ServeHTTP(firstResponse, firstRequest)
	if firstResponse.Code != http.StatusAccepted {
		t.Fatalf("first status = %d body=%s", firstResponse.Code, firstResponse.Body.String())
	}
	var firstPayload workspace.JobPayload
	if err := json.Unmarshal(firstResponse.Body.Bytes(), &firstPayload); err != nil {
		t.Fatal(err)
	}

	otherCookie := createWorkspaceCookie(t, router)
	otherRequest := httptest.NewRequest(http.MethodGet, "/api/synthesis-jobs/"+firstPayload.JobID, nil)
	otherRequest.AddCookie(otherCookie)
	otherResponse := httptest.NewRecorder()
	router.ServeHTTP(otherResponse, otherRequest)
	if otherResponse.Code != http.StatusNotFound {
		t.Fatalf("other status = %d body=%s", otherResponse.Code, otherResponse.Body.String())
	}

	quotaRequest := jsonRequest(t, http.MethodPost, "/api/synthesis-jobs", map[string]any{
		"file_id": upload.FileID,
		"config":  defaultWorkspaceConfigPayload(),
	})
	quotaRequest.AddCookie(workspaceCookie)
	quotaResponse := httptest.NewRecorder()
	router.ServeHTTP(quotaResponse, quotaRequest)
	if quotaResponse.Code != http.StatusConflict {
		t.Fatalf("quota status = %d body=%s", quotaResponse.Code, quotaResponse.Body.String())
	}
	var quotaPayload map[string]map[string]string
	if err := json.Unmarshal(quotaResponse.Body.Bytes(), &quotaPayload); err != nil {
		t.Fatal(err)
	}
	if quotaPayload["error"]["code"] != "workspace_converted_limit" {
		t.Fatalf("quota payload = %#v", quotaPayload)
	}
}

func TestMultipartSynthesisJobCreatesWorkspaceAndRenders(t *testing.T) {
	ctx := t.Context()
	store, err := storage.Open(ctx, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	service := workspace.NewService(store, config.Config{
		WorkspaceTTLSeconds:        60,
		WorkspaceMaxQueuedFiles:    20,
		WorkspaceMaxUploadBytes:    104857600,
		WorkspaceMaxConvertedFiles: 20,
		MaxUploadBytes:             1024 * 1024,
		RenderWorkers:              1,
		RenderQueueSize:            0,
	}, workspace.Options{
		RunInline:     true,
		Now:           func() time.Time { return time.Unix(100, 0) },
		GenerateToken: sequenceGenerator("workspace-token-a"),
		GenerateResourceID: sequenceGenerator(
			"00000000000000000000000000000001",
		),
	})
	router := NewRouterWithOptions(config.Config{MaxUploadBytes: 1024 * 1024}, Options{WorkspaceService: service})
	midiBytes, err := os.ReadFile(filepath.Join("..", "..", "testdata", "python-baseline", "midi", "simple.mid"))
	if err != nil {
		t.Fatal(err)
	}

	request := multipartSynthesisRequest(t, "lead.mid", midiBytes, map[string]string{
		"rate":        "48000",
		"layers_json": `[{"type":"pulse","duty":0.5,"volume":1.0,"frequency_curve":[]}]`,
	})
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusAccepted {
		t.Fatalf("status = %d body=%s", response.Code, response.Body.String())
	}
	if cookies := response.Result().Cookies(); len(cookies) != 1 || cookies[0].Name != "octabit_workspace" {
		t.Fatalf("cookies = %#v", cookies)
	}
	var payload workspace.JobPayload
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if payload.Status != "ready" || payload.DownloadName != "lead_pulse.wav" {
		t.Fatalf("payload = %#v", payload)
	}
	if !strings.HasPrefix(payload.DownloadURL, "/api/synthesis-jobs/") {
		t.Fatalf("download_url = %q", payload.DownloadURL)
	}
}

func TestMultipartSynthesisJobFormValidationErrors(t *testing.T) {
	ctx := t.Context()
	store, err := storage.Open(ctx, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	service := workspace.NewService(store, config.Config{
		WorkspaceTTLSeconds:        60,
		WorkspaceMaxQueuedFiles:    20,
		WorkspaceMaxUploadBytes:    104857600,
		WorkspaceMaxConvertedFiles: 20,
		MaxUploadBytes:             1024 * 1024,
	}, workspace.Options{Now: func() time.Time { return time.Unix(100, 0) }})
	router := NewRouterWithOptions(config.Config{MaxUploadBytes: 1024 * 1024}, Options{WorkspaceService: service})

	tests := []struct {
		name       string
		form       map[string]string
		statusCode int
		errorCode  string
	}{
		{
			name:       "invalid sample rate",
			form:       map[string]string{"rate": "12345", "layers_json": "[]"},
			statusCode: http.StatusUnprocessableEntity,
			errorCode:  "invalid_sample_rate",
		},
		{
			name:       "invalid layer json",
			form:       map[string]string{"rate": "48000", "layers_json": "{"},
			statusCode: http.StatusUnprocessableEntity,
			errorCode:  "invalid_layers",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := multipartSynthesisRequest(t, "lead.mid", []byte("MThd"), tt.form)
			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)
			if response.Code != tt.statusCode {
				t.Fatalf("status = %d body=%s", response.Code, response.Body.String())
			}
			var payload map[string]map[string]string
			if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
				t.Fatal(err)
			}
			if payload["error"]["code"] != tt.errorCode {
				t.Fatalf("payload = %#v", payload)
			}
		})
	}
}

func sequenceGenerator(values ...string) func() (string, error) {
	index := 0
	return func() (string, error) {
		if index >= len(values) {
			return "", nil
		}
		value := values[index]
		index++
		return value, nil
	}
}

func createWorkspaceCookie(t *testing.T, router http.Handler) *http.Cookie {
	t.Helper()
	request := httptest.NewRequest(http.MethodGet, "/api/workspace", nil)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("workspace status = %d body=%s", response.Code, response.Body.String())
	}
	cookies := response.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("cookies = %#v", cookies)
	}
	return cookies[0]
}

func postMIDIUpload(t *testing.T, router http.Handler, cookie *http.Cookie, name string, content []byte) workspace.UploadPayload {
	t.Helper()
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("midi_file", name)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := part.Write(content); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}

	request := httptest.NewRequest(http.MethodPost, "/api/workspace/uploads", body)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	request.AddCookie(cookie)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusCreated {
		t.Fatalf("upload status = %d body=%s", response.Code, response.Body.String())
	}

	var payload struct {
		Upload workspace.UploadPayload `json:"upload"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	return payload.Upload
}

func multipartSynthesisRequest(t *testing.T, name string, content []byte, fields map[string]string) *http.Request {
	return multipartRequest(t, "/api/synthesis-jobs", name, content, fields)
}

func multipartRequest(t *testing.T, path string, name string, content []byte, fields map[string]string) *http.Request {
	t.Helper()
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	for key, value := range fields {
		if err := writer.WriteField(key, value); err != nil {
			t.Fatal(err)
		}
	}
	part, err := writer.CreateFormFile("midi_file", name)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := part.Write(content); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodPost, path, body)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	return request
}

func jsonRequest(t *testing.T, method, path string, payload any) *http.Request {
	t.Helper()
	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(method, path, bytes.NewReader(data))
	request.Header.Set("Content-Type", "application/json")
	return request
}

func defaultWorkspaceConfigPayload() map[string]any {
	return map[string]any{
		"schema":      workspace.ConfigSchema,
		"sample_rate": 48000,
		"layers": []any{
			map[string]any{
				"type":          "pulse",
				"duty":          0.5,
				"volume":        1.0,
				"curve_enabled": false,
				"frequency_curve": []any{
					map[string]any{"frequency_hz": 8.175798915643707, "gain_db": 0.0},
					map[string]any{"frequency_hz": 12543.853951415975, "gain_db": 0.0},
				},
			},
		},
	}
}

func sineWorkspaceConfigPayload() map[string]any {
	payload := defaultWorkspaceConfigPayload()
	layers := payload["layers"].([]any)
	layer := layers[0].(map[string]any)
	layer["type"] = "sine"
	return payload
}
