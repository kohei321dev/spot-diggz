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

	"github.com/kohei321dev/spot-diggz/internal/facility"
	"github.com/kohei321dev/spot-diggz/internal/httpapi"
)

const defaultCatalogPath = "data/facilities.json"

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	catalogPath := os.Getenv("FACILITY_CATALOG_PATH")
	if catalogPath == "" {
		catalogPath = defaultCatalogPath
	}

	catalog, err := facility.LoadCatalogFile(catalogPath)
	if err != nil {
		logger.Error("facility_catalog_load_failed", "path", catalogPath, "error", err)
		os.Exit(1)
	}
	logger.Info("facility_catalog_loaded", "count", len(catalog.List("")))

	server := &http.Server{
		Addr:              ":" + envOrDefault("PORT", "8080"),
		Handler:           httpapi.NewServer(catalog, logger),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-shutdown
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := server.Shutdown(ctx); err != nil {
			logger.Error("http_server_shutdown_failed", "error", err)
		}
	}()

	logger.Info("http_server_started", "addr", server.Addr)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		logger.Error("http_server_failed", "error", err)
		os.Exit(1)
	}
}

func envOrDefault(key string, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
