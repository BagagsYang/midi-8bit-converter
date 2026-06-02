package jobs

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const (
	StatusQueued    = "queued"
	StatusRendering = "rendering"
	StatusReady     = "ready"
	StatusFailed    = "failed"
	StatusExpired   = "expired"

	defaultDownloadTTLSeconds = 30 * 60
	defaultRenderWorkers      = 2
	defaultRenderQueueSize    = 8
)

var ErrRenderQueueFull = errors.New("The render queue is full. Try again after current jobs finish.")

type Config struct {
	DownloadTTLSeconds int
	MaxWorkers         int
	MaxQueueSize       int
	RunInline          bool
	Now                func() time.Time
	NewID              func() (string, error)
}

type Manager struct {
	mu                 sync.Mutex
	jobs               map[string]*Job
	downloadTTLSeconds int
	runInline          bool
	now                func() time.Time
	newID              func() (string, error)
	executor           *boundedExecutor
}

type Executor struct {
	runInline bool
	executor  *boundedExecutor
}

type Job struct {
	ID           string
	Status       string
	SourceName   string
	CreatedAt    time.Time
	UpdatedAt    time.Time
	ExpiresAt    time.Time
	DownloadName string
	SizeBytes    int64
	Error        string
}

type RenderResult struct {
	DownloadName string
	SizeBytes    int64
}

type RenderFunc func() (RenderResult, error)

type Payload struct {
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

func NewManager(cfg Config) *Manager {
	now := cfg.Now
	if now == nil {
		now = time.Now
	}
	newID := cfg.NewID
	if newID == nil {
		newID = randomID
	}
	ttlSeconds := positiveInt(cfg.DownloadTTLSeconds, defaultDownloadTTLSeconds)
	maxWorkers := positiveInt(cfg.MaxWorkers, defaultRenderWorkers)
	maxQueueSize := nonNegativeInt(cfg.MaxQueueSize, defaultRenderQueueSize)
	return &Manager{
		jobs:               map[string]*Job{},
		downloadTTLSeconds: ttlSeconds,
		runInline:          cfg.RunInline,
		now:                now,
		newID:              newID,
		executor:           newBoundedExecutor(maxWorkers, maxQueueSize),
	}
}

func NewExecutor(cfg Config) *Executor {
	maxWorkers := positiveInt(cfg.MaxWorkers, defaultRenderWorkers)
	maxQueueSize := nonNegativeInt(cfg.MaxQueueSize, defaultRenderQueueSize)
	return &Executor{
		runInline: cfg.RunInline,
		executor:  newBoundedExecutor(maxWorkers, maxQueueSize),
	}
}

func (e *Executor) Submit(fn func()) error {
	if e.runInline {
		fn()
		return nil
	}
	return e.executor.submit(fn)
}

func (m *Manager) Create(sourceName string) (*Job, error) {
	jobID, err := m.newID()
	if err != nil {
		return nil, err
	}
	now := m.now()
	job := &Job{
		ID:         jobID,
		Status:     StatusQueued,
		SourceName: filepath.Base(sourceName),
		CreatedAt:  now,
		UpdatedAt:  now,
		ExpiresAt:  now.Add(time.Duration(m.downloadTTLSeconds) * time.Second),
	}
	if job.SourceName == "." || job.SourceName == string(filepath.Separator) {
		job.SourceName = ""
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	m.jobs[jobID] = job
	return cloneJob(job), nil
}

func (m *Manager) Start(jobID string, render RenderFunc) error {
	if m.runInline {
		m.runJob(jobID, render)
		return nil
	}
	return m.executor.submit(func() {
		m.runJob(jobID, render)
	})
}

func (m *Manager) Get(jobID string) (*Job, bool, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	job, ok := m.jobs[jobID]
	if !ok {
		return nil, false, false
	}
	if !job.ExpiresAt.IsZero() && !m.now().Before(job.ExpiresAt) {
		delete(m.jobs, jobID)
		return nil, true, false
	}
	return cloneJob(job), false, true
}

func (m *Manager) Delete(jobID string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.jobs[jobID]; !ok {
		return false
	}
	delete(m.jobs, jobID)
	return true
}

func (m *Manager) Payload(job Job, routeBase string) Payload {
	status := job.Status
	if status == "" {
		status = StatusExpired
	}
	payload := Payload{
		JobID:  job.ID,
		Status: status,
	}
	if status == StatusExpired {
		return payload
	}
	payload.SourceName = job.SourceName
	payload.CreatedAt = unixSeconds(job.CreatedAt)
	payload.UpdatedAt = unixSeconds(job.UpdatedAt)
	payload.ExpiresAt = unixSeconds(job.ExpiresAt)
	payload.DownloadName = job.DownloadName
	payload.SizeBytes = job.SizeBytes
	payload.Error = job.Error
	if status == StatusReady {
		payload.DownloadURL = strings.TrimRight(routeBase, "/") + "/" + job.ID + "/download"
		payload.DeleteURL = strings.TrimRight(routeBase, "/") + "/" + job.ID
	}
	return payload
}

func (m *Manager) runJob(jobID string, render RenderFunc) {
	m.update(jobID, func(job *Job, now time.Time) {
		job.Status = StatusRendering
		job.UpdatedAt = now
	})

	result, err := render()
	if err != nil {
		m.update(jobID, func(job *Job, now time.Time) {
			job.Status = StatusFailed
			job.Error = jobErrorMessage(err)
			job.UpdatedAt = now
			job.ExpiresAt = now.Add(time.Duration(m.downloadTTLSeconds) * time.Second)
		})
		return
	}

	m.update(jobID, func(job *Job, now time.Time) {
		job.Status = StatusReady
		job.DownloadName = result.DownloadName
		job.SizeBytes = result.SizeBytes
		job.UpdatedAt = now
		job.ExpiresAt = now.Add(time.Duration(m.downloadTTLSeconds) * time.Second)
	})
}

func (m *Manager) update(jobID string, apply func(job *Job, now time.Time)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	job, ok := m.jobs[jobID]
	if !ok {
		return
	}
	if !job.ExpiresAt.IsZero() && !m.now().Before(job.ExpiresAt) {
		delete(m.jobs, jobID)
		return
	}
	apply(job, m.now())
}

type boundedExecutor struct {
	sem chan struct{}
}

func newBoundedExecutor(maxWorkers, maxQueueSize int) *boundedExecutor {
	return &boundedExecutor{
		sem: make(chan struct{}, maxWorkers+maxQueueSize),
	}
}

func (e *boundedExecutor) submit(fn func()) error {
	select {
	case e.sem <- struct{}{}:
	default:
		return ErrRenderQueueFull
	}
	go func() {
		defer func() {
			<-e.sem
		}()
		fn()
	}()
	return nil
}

func jobErrorMessage(err error) string {
	return ErrorMessage(err)
}

func ErrorMessage(err error) string {
	message := strings.TrimSpace(err.Error())
	if message != "" {
		return message
	}
	return fmt.Sprintf("%T", err)
}

func unixSeconds(value time.Time) float64 {
	if value.IsZero() {
		return 0
	}
	return float64(value.UnixNano()) / float64(time.Second)
}

func randomID() (string, error) {
	var bytes [16]byte
	if _, err := rand.Read(bytes[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes[:]), nil
}

func positiveInt(value, fallback int) int {
	if value < 1 {
		return fallback
	}
	return value
}

func nonNegativeInt(value, fallback int) int {
	if value < 0 {
		return fallback
	}
	return value
}

func cloneJob(job *Job) *Job {
	if job == nil {
		return nil
	}
	copied := *job
	return &copied
}
