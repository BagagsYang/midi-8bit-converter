package jobs

import (
	"os"
	"path/filepath"
	"strings"
	"time"

	"octabit/backend/internal/midi"
	"octabit/backend/internal/renderer"
)

type LegacyService struct {
	manager *Manager
	root    string
}

func NewLegacyService(jobRoot string, cfg Config) *LegacyService {
	return &LegacyService{
		manager: NewManager(cfg),
		root:    filepath.Join(jobRoot, "legacy-jobs"),
	}
}

func (s *LegacyService) CreateFromTemp(sourcePath, sourceName string, sampleRate int, layers []renderer.Layer) (*Job, error) {
	job, err := s.manager.Create(sourceName)
	if err != nil {
		return nil, err
	}
	inputPath := s.InputPath(job.ID)
	if err := copyFile(sourcePath, inputPath); err != nil {
		s.Delete(job.ID)
		return nil, err
	}
	if err := s.manager.Start(job.ID, func() (RenderResult, error) {
		defer os.Remove(inputPath)
		downloadName, sizeBytes, err := renderWAV(inputPath, s.OutputPath(job.ID), sourceName, sampleRate, layers)
		if err != nil {
			return RenderResult{}, err
		}
		return RenderResult{DownloadName: downloadName, SizeBytes: sizeBytes}, nil
	}); err != nil {
		s.Delete(job.ID)
		return nil, err
	}
	updated, _, ok := s.Get(job.ID)
	if ok {
		return updated, nil
	}
	return job, nil
}

func (s *LegacyService) Get(jobID string) (*Job, bool, bool) {
	job, expired, ok := s.manager.Get(jobID)
	if expired {
		_ = os.RemoveAll(s.JobDir(jobID))
	}
	return job, expired, ok
}

func (s *LegacyService) Delete(jobID string) bool {
	deleted := s.manager.Delete(jobID)
	_ = os.RemoveAll(s.JobDir(jobID))
	return deleted
}

func (s *LegacyService) Payload(job Job) Payload {
	payload := s.manager.Payload(job, "/synthesise/jobs")
	payload.SourceName = ""
	return payload
}

func (s *LegacyService) JobDir(jobID string) string {
	return filepath.Join(s.root, jobID)
}

func (s *LegacyService) InputPath(jobID string) string {
	return filepath.Join(s.JobDir(jobID), "input.mid")
}

func (s *LegacyService) OutputPath(jobID string) string {
	return filepath.Join(s.JobDir(jobID), "output.wav")
}

func renderWAV(inputPath, outputPath, sourceName string, sampleRate int, layers []renderer.Layer) (string, int64, error) {
	downloadName, wavData, err := RenderWAVBytes(inputPath, sourceName, sampleRate, layers)
	if err != nil {
		return "", 0, err
	}
	if err := os.MkdirAll(filepath.Dir(outputPath), 0o755); err != nil {
		return "", 0, err
	}
	if err := os.WriteFile(outputPath, wavData, 0o644); err != nil {
		return "", 0, err
	}
	return downloadName, int64(len(wavData)), nil
}

func RenderWAVBytes(inputPath, sourceName string, sampleRate int, layers []renderer.Layer) (string, []byte, error) {
	notes, err := midi.ReadNotes(inputPath)
	if err != nil {
		return "", nil, err
	}
	wavData, err := renderer.RenderNotesWAV(notes, sampleRate, layers)
	if err != nil {
		return "", nil, err
	}
	downloadName, err := renderer.BuildOutputFilename(originalFilename(sourceName), layers)
	if err != nil {
		return "", nil, err
	}
	return downloadName, wavData, nil
}

func originalFilename(sourceName string) string {
	base := filepath.Base(sourceName)
	if base == "." || base == string(filepath.Separator) || base == "" {
		return "output"
	}
	name := strings.TrimSuffix(base, filepath.Ext(base))
	if name == "" {
		return "output"
	}
	return name
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
	if _, err := output.ReadFrom(input); err != nil {
		_ = output.Close()
		return err
	}
	if err := output.Close(); err != nil {
		return err
	}
	return os.Chtimes(destination, time.Now(), time.Now())
}
