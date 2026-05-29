package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"octabit/backend/internal/config"
	"octabit/backend/internal/httpapi"
	"octabit/backend/internal/storage"
	"octabit/backend/internal/workspace"
)

const shutdownTimeout = 10 * time.Second

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	cfg := config.FromEnv()
	store, err := storage.Open(context.Background(), cfg.JobRoot)
	if err != nil {
		logger.Error("failed to open workspace store", "error", err, "job_root", cfg.JobRoot)
		os.Exit(1)
	}
	defer store.Close()

	workspaceService := workspace.NewService(store, cfg, workspace.Options{})
	handler := httpapi.NewRouterWithOptions(cfg, httpapi.Options{
		WorkspaceService: workspaceService,
	})
	server := &http.Server{
		Addr:              cfg.Addr(),
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		logger.Info("starting octabit pro backend", "addr", server.Addr)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
			return
		}
		errCh <- nil
	}()

	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, os.Interrupt, syscall.SIGTERM)

	select {
	case signalValue := <-signalCh:
		logger.Info("stopping octabit pro backend", "signal", signalValue.String())
	case err := <-errCh:
		if err != nil {
			logger.Error("octabit pro backend failed", "error", err)
			os.Exit(1)
		}
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		logger.Error("octabit pro backend shutdown failed", "error", err)
		os.Exit(1)
	}
	if err := <-errCh; err != nil {
		logger.Error("octabit pro backend stopped with error", "error", err)
		os.Exit(1)
	}
}
