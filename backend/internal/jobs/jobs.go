package jobs

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

const (
	StatusQueued    = "queued"
	StatusRendering = "rendering"
	StatusReady     = "ready"
	StatusFailed    = "failed"
	StatusExpired   = "expired"

	defaultRenderWorkers   = 2
	defaultRenderQueueSize = 8
)

var ErrRenderQueueFull = errors.New("The render queue is full. Try again after current jobs finish.")

type Config struct {
	MaxWorkers   int
	MaxQueueSize int
	RunInline    bool
	Now          func() time.Time
	NewID        func() (string, error)
}

type Executor struct {
	runInline bool
	executor  *boundedExecutor
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

func ErrorMessage(err error) string {
	message := strings.TrimSpace(err.Error())
	if message != "" {
		return message
	}
	return fmt.Sprintf("%T", err)
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
