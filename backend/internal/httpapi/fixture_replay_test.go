package httpapi

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"octabit/backend/internal/config"
	"octabit/backend/internal/storage"
	"octabit/backend/internal/workspace"
)

type apiTranscript struct {
	Records []apiRecord `json:"records"`
}

type apiRecord struct {
	Label   string         `json:"label"`
	Status  int            `json:"status"`
	Headers map[string]any `json:"headers"`
	Body    map[string]any `json:"body"`
}

func TestImplementedRoutesReplayPythonBaseline(t *testing.T) {
	transcript := loadAPITranscript(t, filepath.Join("..", "..", "testdata", "python-baseline", "api", "implemented_routes.json"))
	routes := map[string]struct {
		method string
		path   string
		binary bool
	}{
		"api_health": {
			method: http.MethodGet,
			path:   "/api/health",
		},
		"preview_pulse_50": {
			method: http.MethodGet,
			path:   "/static/previews/pulse_50.wav",
			binary: true,
		},
	}

	repoRoot := findRepoRoot(t)
	router := NewRouter(config.Config{
		PreviewAssetsDir: filepath.Join(repoRoot, "assets", "previews"),
	})

	for _, expected := range transcript.Records {
		route, ok := routes[expected.Label]
		if !ok {
			t.Fatalf("no replay route for fixture label %q", expected.Label)
		}
		t.Run(expected.Label, func(t *testing.T) {
			request := httptest.NewRequest(route.method, route.path, nil)
			response := httptest.NewRecorder()

			router.ServeHTTP(response, request)

			actual := apiRecord{
				Label:   expected.Label,
				Status:  response.Code,
				Headers: selectedHeaders(response.Result()),
				Body:    responseBody(t, response.Body.Bytes(), route.binary),
			}
			if !reflect.DeepEqual(actual, expected) {
				t.Fatalf("replay mismatch\nactual:   %#v\nexpected: %#v", actual, expected)
			}
		})
	}
}

func TestWorkspaceFlowReplaysPythonBaseline(t *testing.T) {
	ctx := t.Context()
	store, err := storage.Open(ctx, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	now := time.Unix(1000, 0)
	service := workspace.NewService(store, replayConfig(t.TempDir()), workspace.Options{
		RunInline:     true,
		Now:           func() time.Time { return now },
		GenerateToken: sequenceGenerator("workspace-token-a"),
		GenerateResourceID: sequenceGenerator(
			"00000000000000000000000000000001",
			"00000000000000000000000000000002",
			"00000000000000000000000000000003",
		),
	})
	router := NewRouterWithOptions(replayConfig(t.TempDir()), Options{WorkspaceService: service})
	transcript := loadAPITranscript(t, filepath.Join("..", "..", "testdata", "python-baseline", "api", "workspace_flow.json"))
	simpleMIDI := readFixture(t, filepath.Join("..", "..", "testdata", "python-baseline", "midi", "simple.mid"))
	melodyMIDI := readFixture(t, filepath.Join("..", "..", "testdata", "python-baseline", "midi", "melody.mid"))

	var workspaceCookie *http.Cookie
	fileIDs := map[string]string{}
	jobIDs := map[string]string{}
	for _, expected := range transcript.Records {
		t.Run(expected.Label, func(t *testing.T) {
			request := workspaceReplayRequest(t, expected.Label, fileIDs, jobIDs, simpleMIDI, melodyMIDI)
			if workspaceCookie != nil {
				request.AddCookie(workspaceCookie)
			}
			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)
			if expected.Label == "workspace_start" {
				cookies := response.Result().Cookies()
				if len(cookies) != 1 {
					t.Fatalf("cookies = %#v", cookies)
				}
				workspaceCookie = cookies[0]
			}

			actual := apiRecord{
				Label:   expected.Label,
				Status:  response.Code,
				Headers: normaliseReplayHeaders(selectedHeaders(response.Result())),
				Body:    responseBodyForReplay(t, response.Body.Bytes(), expected.Label),
			}
			normaliseWorkspaceReplayBody(actual.Body, fileIDs, jobIDs)
			expectedBody := cloneBody(expected.Body)
			normaliseBinaryExpectation(expected.Label, expectedBody)
			if !reflect.DeepEqual(actual.Body, expectedBody) || actual.Status != expected.Status || !reflect.DeepEqual(actual.Headers, expected.Headers) {
				t.Fatalf("replay mismatch\nactual:   %#v\nexpected: %#v", actual, apiRecord{
					Label:   expected.Label,
					Status:  expected.Status,
					Headers: expected.Headers,
					Body:    expectedBody,
				})
			}
		})
	}
}

func TestErrorResponsesReplayPythonBaseline(t *testing.T) {
	ctx := t.Context()
	store, err := storage.Open(ctx, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	service := workspace.NewService(store, replayConfig(t.TempDir()), workspace.Options{
		RunInline:     true,
		Now:           func() time.Time { return time.Unix(1000, 0) },
		GenerateToken: sequenceGenerator("workspace-token-a"),
		GenerateResourceID: sequenceGenerator(
			"00000000000000000000000000000001",
			"00000000000000000000000000000002",
		),
	})
	router := NewRouterWithOptions(replayConfig(t.TempDir()), Options{WorkspaceService: service})
	transcript := loadAPITranscript(t, filepath.Join("..", "..", "testdata", "python-baseline", "api", "error_responses.json"))
	workspaceCookie := createReplayWorkspace(t, router)

	for _, expected := range transcript.Records {
		if expected.Label == "render_queue_full" {
			t.Run(expected.Label, func(t *testing.T) {
				actual := replayRenderQueueFullError(t, expected)
				if !reflect.DeepEqual(actual.Body, expected.Body) || actual.Status != expected.Status || !reflect.DeepEqual(actual.Headers, expected.Headers) {
					t.Fatalf("replay mismatch\nactual:   %#v\nexpected: %#v", actual, expected)
				}
			})
			continue
		}
		t.Run(expected.Label, func(t *testing.T) {
			request := errorReplayRequest(t, expected.Label)
			if needsWorkspaceCookie(expected.Label) {
				request.AddCookie(workspaceCookie)
			}
			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)
			actual := apiRecord{
				Label:   expected.Label,
				Status:  response.Code,
				Headers: selectedHeaders(response.Result()),
				Body:    responseBodyForReplay(t, response.Body.Bytes(), expected.Label),
			}
			if !reflect.DeepEqual(actual.Body, expected.Body) || actual.Status != expected.Status || !reflect.DeepEqual(actual.Headers, expected.Headers) {
				t.Fatalf("replay mismatch\nactual:   %#v\nexpected: %#v", actual, expected)
			}
		})
	}
}

func replayRenderQueueFullError(t *testing.T, expected apiRecord) apiRecord {
	t.Helper()
	ctx := t.Context()
	store, err := storage.Open(ctx, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	started := make(chan struct{})
	release := make(chan struct{})
	t.Cleanup(func() { close(release) })
	service := workspace.NewService(store, replayConfig(t.TempDir()), workspace.Options{
		RunInline:     false,
		Now:           func() time.Time { return time.Unix(1000, 0) },
		GenerateToken: sequenceGenerator("workspace-token-b"),
		GenerateResourceID: sequenceGenerator(
			"00000000000000000000000000000001",
			"00000000000000000000000000000002",
			"00000000000000000000000000000003",
		),
		RenderWAV: func(_, _, _ string, _ workspace.Config) (string, int64, error) {
			select {
			case <-started:
			default:
				close(started)
			}
			<-release
			return "blocked.wav", 44, nil
		},
	})
	cfg := replayConfig(t.TempDir())
	cfg.RenderWorkers = 1
	cfg.RenderQueueSize = 0
	router := NewRouterWithOptions(cfg, Options{WorkspaceService: service})
	cookie := createReplayWorkspace(t, router)
	simpleMIDI := readFixture(t, filepath.Join("..", "..", "testdata", "python-baseline", "midi", "simple.mid"))
	uploadRequest := multipartRequest(t, "/api/workspace/uploads", "lead.mid", simpleMIDI, nil)
	uploadRequest.AddCookie(cookie)
	uploadResponse := httptest.NewRecorder()
	router.ServeHTTP(uploadResponse, uploadRequest)
	if uploadResponse.Code != http.StatusCreated {
		t.Fatalf("upload status = %d body=%s", uploadResponse.Code, uploadResponse.Body.String())
	}
	var uploadPayload struct {
		Upload workspace.UploadPayload `json:"upload"`
	}
	if err := json.Unmarshal(uploadResponse.Body.Bytes(), &uploadPayload); err != nil {
		t.Fatal(err)
	}
	firstJob := jsonReplayRequest(t, http.MethodPost, "/api/synthesis-jobs", map[string]any{
		"file_id": uploadPayload.Upload.FileID,
		"config":  defaultReplayConfig(),
	})
	firstJob.AddCookie(cookie)
	firstResponse := httptest.NewRecorder()
	router.ServeHTTP(firstResponse, firstJob)
	if firstResponse.Code != http.StatusAccepted {
		t.Fatalf("first job status = %d body=%s", firstResponse.Code, firstResponse.Body.String())
	}
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("first render did not start")
	}

	secondJob := jsonReplayRequest(t, http.MethodPost, "/api/synthesis-jobs", map[string]any{
		"file_id": uploadPayload.Upload.FileID,
		"config":  defaultReplayConfig(),
	})
	secondJob.AddCookie(cookie)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, secondJob)
	return apiRecord{
		Label:   expected.Label,
		Status:  response.Code,
		Headers: selectedHeaders(response.Result()),
		Body:    responseBodyForReplay(t, response.Body.Bytes(), expected.Label),
	}
}

func loadAPITranscript(t *testing.T, path string) apiTranscript {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var transcript apiTranscript
	if err := json.Unmarshal(data, &transcript); err != nil {
		t.Fatal(err)
	}
	return transcript
}

func createReplayWorkspace(t *testing.T, router http.Handler) *http.Cookie {
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

func errorReplayRequest(t *testing.T, label string) *http.Request {
	t.Helper()
	switch label {
	case "workspace_expired_without_cookie":
		return httptest.NewRequest(http.MethodDelete, "/api/workspace/uploads/00000000000000000000000000000000", nil)
	case "missing_upload_file":
		return multipartRequest(t, "/api/workspace/uploads", "", nil, nil)
	case "unsupported_upload_extension":
		return multipartRequest(t, "/api/workspace/uploads", "lead.txt", []byte("MThd"), nil)
	case "invalid_workspace_config":
		return jsonReplayRequest(t, http.MethodPut, "/api/workspace/config", map[string]any{"schema": "wrong"})
	case "invalid_json_file_id":
		return jsonReplayRequest(t, http.MethodPost, "/api/synthesis-jobs", map[string]any{
			"file_id": "not-a-file-id",
			"config":  workspaceFlowConfig(),
		})
	case "invalid_multipart_sample_rate":
		return multipartRequest(t, "/api/synthesis-jobs", "lead.mid", []byte("MThd"), map[string]string{"rate": "16000"})
	case "invalid_multipart_layers":
		return multipartRequest(t, "/api/synthesis-jobs", "lead.mid", []byte("MThd"), map[string]string{
			"rate":        "44100",
			"layers_json": `[{"type":"sine","duty":0.5,"volume":1.0,"frequency_curve":[{"frequency_hz":440.0,"gain_db":0.0},{"frequency_hz":440.0,"gain_db":-6.0}]}]`,
		})
	default:
		t.Fatalf("no error replay request for %q", label)
		return nil
	}
}

func needsWorkspaceCookie(label string) bool {
	switch label {
	case "missing_upload_file", "unsupported_upload_extension", "invalid_workspace_config", "invalid_json_file_id":
		return true
	default:
		return false
	}
}

func selectedHeaders(response *http.Response) map[string]any {
	headers := map[string]any{}
	for _, key := range []string{"Content-Type", "Content-Disposition", "Set-Cookie"} {
		if value := response.Header.Get(key); value != "" {
			headers[key] = value
		}
	}
	return headers
}

func responseBodyForReplay(t *testing.T, data []byte, label string) map[string]any {
	t.Helper()
	if strings.HasPrefix(label, "download_") {
		summary := binarySummary(data)
		delete(summary, "sha256")
		return summary
	}
	if len(data) == 0 {
		return nil
	}
	return responseBody(t, data, false)
}

func replayConfig(jobRoot string) config.Config {
	return config.Config{
		JobRoot:                    jobRoot,
		WorkspaceTTLSeconds:        86400,
		WorkspaceMaxQueuedFiles:    20,
		WorkspaceMaxUploadBytes:    104857600,
		WorkspaceMaxConvertedFiles: 20,
		MaxUploadBytes:             1024 * 1024,
		RenderWorkers:              1,
		RenderQueueSize:            0,
	}
}

func workspaceReplayRequest(t *testing.T, label string, fileIDs map[string]string, jobIDs map[string]string, simpleMIDI, melodyMIDI []byte) *http.Request {
	t.Helper()
	switch label {
	case "workspace_start":
		return httptest.NewRequest(http.MethodGet, "/api/workspace", nil)
	case "upload_first":
		return multipartRequest(t, "/api/workspace/uploads", "first.mid", simpleMIDI, nil)
	case "upload_second":
		return multipartRequest(t, "/api/workspace/uploads", "second.mid", melodyMIDI, nil)
	case "reorder_queue":
		return jsonReplayRequest(t, http.MethodPatch, "/api/workspace/queue", map[string]any{
			"file_ids": []string{fileIDs["<file_id:2>"], fileIDs["<file_id:1>"]},
		})
	case "save_config":
		return jsonReplayRequest(t, http.MethodPut, "/api/workspace/config", workspaceFlowConfig())
	case "create_synthesis_job":
		return jsonReplayRequest(t, http.MethodPost, "/api/synthesis-jobs", map[string]any{
			"file_id": fileIDs["<file_id:2>"],
			"config":  workspaceFlowConfig(),
		})
	case "poll_synthesis_job":
		return httptest.NewRequest(http.MethodGet, "/api/synthesis-jobs/"+jobIDs["<job_id:3>"], nil)
	case "download_synthesis_job":
		return httptest.NewRequest(http.MethodGet, "/api/synthesis-jobs/"+jobIDs["<job_id:3>"]+"/download", nil)
	case "delete_synthesis_job":
		return httptest.NewRequest(http.MethodDelete, "/api/synthesis-jobs/"+jobIDs["<job_id:3>"], nil)
	case "delete_upload":
		return httptest.NewRequest(http.MethodDelete, "/api/workspace/uploads/"+fileIDs["<file_id:1>"], nil)
	default:
		t.Fatalf("no workspace replay request for %q", label)
		return nil
	}
}

func jsonReplayRequest(t *testing.T, method, path string, payload any) *http.Request {
	t.Helper()
	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(method, path, bytes.NewReader(data))
	request.Header.Set("Content-Type", "application/json")
	return request
}

func workspaceFlowConfig() map[string]any {
	return map[string]any{
		"schema":      workspace.ConfigSchema,
		"sample_rate": 44100,
		"layers": []any{
			map[string]any{
				"type":          "pulse",
				"duty":          0.25,
				"volume":        0.75,
				"curve_enabled": true,
				"frequency_curve": []any{
					map[string]any{"frequency_hz": 110.0, "gain_db": -6.0},
					map[string]any{"frequency_hz": 440.0, "gain_db": 0.0},
					map[string]any{"frequency_hz": 880.0, "gain_db": 3.0},
				},
			},
			map[string]any{
				"type":            "triangle",
				"duty":            0.5,
				"volume":          0.4,
				"curve_enabled":   false,
				"frequency_curve": []any{},
			},
		},
	}
}

func defaultReplayConfig() map[string]any {
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
					map[string]any{"frequency_hz": rendererMinCurveFrequencyHz, "gain_db": 0.0},
					map[string]any{"frequency_hz": rendererMaxCurveFrequencyHz, "gain_db": 0.0},
				},
			},
		},
	}
}

const (
	rendererMinCurveFrequencyHz = 8.175798915643707
	rendererMaxCurveFrequencyHz = 12543.853951415975
)

func normaliseReplayHeaders(headers map[string]any) map[string]any {
	if _, ok := headers["Set-Cookie"]; ok {
		headers["Set-Cookie"] = "octabit_workspace=<workspace-token>; Expires=<timestamp>; Max-Age=86400; HttpOnly; Path=/; SameSite=Lax"
	}
	return headers
}

func normaliseWorkspaceReplayBody(body map[string]any, fileIDs map[string]string, jobIDs map[string]string) {
	switch {
	case body == nil:
		return
	case body["upload"] != nil:
		upload := body["upload"].(map[string]any)
		if upload["name"] == "first.mid" {
			fileIDs["<file_id:1>"] = upload["file_id"].(string)
			upload["file_id"] = "<file_id:1>"
		}
		if upload["name"] == "second.mid" {
			fileIDs["<file_id:2>"] = upload["file_id"].(string)
			upload["file_id"] = "<file_id:2>"
		}
		normaliseTimestamps(body)
	case body["uploads"] != nil:
		for _, rawUpload := range body["uploads"].([]any) {
			upload := rawUpload.(map[string]any)
			for placeholder, actual := range fileIDs {
				if upload["file_id"] == actual {
					upload["file_id"] = placeholder
				}
			}
		}
		normaliseTimestamps(body)
	case body["job_id"] != nil:
		jobID := body["job_id"].(string)
		if _, ok := jobIDs["<job_id:3>"]; !ok {
			jobIDs["<job_id:3>"] = jobID
		}
		replaceJobID(body, jobIDs["<job_id:3>"], "<job_id:3>")
		normaliseTimestamps(body)
	default:
		normaliseTimestamps(body)
	}
}

func replaceJobID(body map[string]any, actual, placeholder string) {
	for key, rawValue := range body {
		value, ok := rawValue.(string)
		if ok {
			body[key] = strings.ReplaceAll(value, actual, placeholder)
		}
	}
}

func normaliseTimestamps(value any) {
	switch typedValue := value.(type) {
	case map[string]any:
		for key, child := range typedValue {
			if key == "created_at" || key == "updated_at" || key == "expires_at" {
				typedValue[key] = "<timestamp>"
				continue
			}
			normaliseTimestamps(child)
		}
	case []any:
		for _, child := range typedValue {
			normaliseTimestamps(child)
		}
	}
}

func normaliseBinaryExpectation(label string, body map[string]any) {
	if strings.HasPrefix(label, "download_") && body != nil {
		delete(body, "sha256")
	}
}

func cloneBody(body map[string]any) map[string]any {
	if body == nil {
		return nil
	}
	data, _ := json.Marshal(body)
	var cloned map[string]any
	_ = json.Unmarshal(data, &cloned)
	return cloned
}

func readFixture(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func responseBody(t *testing.T, data []byte, binary bool) map[string]any {
	t.Helper()
	if binary {
		return binarySummary(data)
	}
	var payload map[string]any
	if err := json.Unmarshal(data, &payload); err != nil {
		t.Fatal(err)
	}
	return payload
}

func binarySummary(data []byte) map[string]any {
	sum := sha256.Sum256(data)
	prefixLength := min(12, len(data))
	return map[string]any{
		"prefix_hex": fmt.Sprintf("%x", data[:prefixLength]),
		"sha256":     fmt.Sprintf("%x", sum[:]),
		"size":       float64(len(data)),
	}
}

func findRepoRoot(t *testing.T) string {
	t.Helper()
	workingDirectory, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	directory := workingDirectory
	for range 8 {
		if _, err := os.Stat(filepath.Join(directory, "assets", "previews", "pulse_50.wav")); err == nil {
			return directory
		}
		parent := filepath.Dir(directory)
		if parent == directory {
			break
		}
		directory = parent
	}
	t.Fatal("could not locate repository root")
	return ""
}
