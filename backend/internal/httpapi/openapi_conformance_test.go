package httpapi

import (
	"bufio"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"octabit/backend/internal/jobs"
	"octabit/backend/internal/storage"
	"octabit/backend/internal/workspace"
)

func TestOpenAPIDocumentedRoutesAreRegistered(t *testing.T) {
	ctx := t.Context()
	jobRoot := t.TempDir()
	store, err := storage.Open(ctx, jobRoot)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	cfg := replayConfig(jobRoot)
	cfg.PreviewAssetsDir = filepath.Join(findRepoRoot(t), "assets", "previews")
	service := workspace.NewService(store, cfg, workspace.Options{
		RunInline:     true,
		Now:           func() time.Time { return time.Unix(1000, 0) },
		GenerateToken: sequenceGenerator("workspace-token-a"),
		GenerateResourceID: sequenceGenerator(
			"00000000000000000000000000000001",
			"00000000000000000000000000000002",
			"00000000000000000000000000000003",
		),
	})
	legacyJobs := jobs.NewLegacyService(jobRoot, jobs.Config{
		RunInline: true,
		Now:       func() time.Time { return time.Unix(1000, 0) },
		NewID:     sequenceGenerator("00000000000000000000000000000004"),
	})
	router := NewRouterWithOptions(cfg, Options{WorkspaceService: service, LegacyJobs: legacyJobs})

	for _, route := range openAPIRoutes(t) {
		t.Run(route.method+" "+route.path, func(t *testing.T) {
			request := httptest.NewRequest(strings.ToUpper(route.method), concreteOpenAPIPath(route.path), nil)
			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)
			if response.Code == http.StatusNotFound || response.Code == http.StatusMethodNotAllowed {
				t.Fatalf("%s %s was not registered; status=%d body=%s", route.method, route.path, response.Code, response.Body.String())
			}
		})
	}
}

type openAPIRoute struct {
	method string
	path   string
}

func openAPIRoutes(t *testing.T) []openAPIRoute {
	t.Helper()
	file, err := os.Open(filepath.Join(findRepoRoot(t), "docs", "openapi.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()

	var routes []openAPIRoute
	var currentPath string
	pathPattern := regexp.MustCompile(`^  (/[^:]+):$`)
	methodPattern := regexp.MustCompile(`^    (get|post|put|patch|delete):$`)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if match := pathPattern.FindStringSubmatch(line); match != nil {
			currentPath = match[1]
			continue
		}
		if currentPath == "" {
			continue
		}
		if match := methodPattern.FindStringSubmatch(line); match != nil {
			routes = append(routes, openAPIRoute{method: strings.ToUpper(match[1]), path: currentPath})
		}
	}
	if err := scanner.Err(); err != nil {
		t.Fatal(err)
	}
	if len(routes) == 0 {
		t.Fatal("no OpenAPI routes parsed")
	}
	return routes
}

func concreteOpenAPIPath(path string) string {
	replacements := map[string]string{
		"{file_id}":  "00000000000000000000000000000001",
		"{job_id}":   "00000000000000000000000000000002",
		"{filename}": "pulse_50.wav",
	}
	for placeholder, value := range replacements {
		path = strings.ReplaceAll(path, placeholder, value)
	}
	return path
}
