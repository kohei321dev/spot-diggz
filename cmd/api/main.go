package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/kohei321dev/spot-diggz/internal/correction"
	"github.com/kohei321dev/spot-diggz/internal/facility"
	"github.com/kohei321dev/spot-diggz/internal/geocoding"
	"github.com/kohei321dev/spot-diggz/internal/httpapi"
	"github.com/kohei321dev/spot-diggz/internal/observability"
	"github.com/kohei321dev/spot-diggz/internal/recommendation"
	"github.com/kohei321dev/spot-diggz/internal/travel"
)

const (
	defaultCatalogPath      = "data/facilities.json"
	defaultCorrectionPath   = "var/corrections.jsonl"
	correctionPurgeInterval = time.Hour
	googleHTTPTimeout       = 5 * time.Second
	googleMaxConnsPerHost   = 4
)

func main() {
	if len(os.Args) > 1 {
		if os.Args[1] == "correctioncheck" {
			os.Exit(correction.RunCheckCommand(os.Args[2:], os.Stdout, os.Stderr, time.Now))
		}
		_, _ = fmt.Fprintln(os.Stderr, "unknown command")
		os.Exit(2)
	}
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		ReplaceAttr: func(_ []string, attribute slog.Attr) slog.Attr {
			if attribute.Key == slog.MessageKey {
				attribute.Key = "event_name"
			}
			return attribute
		},
	})).With(
		"service", "spotdiggz-api",
		"environment", envOrDefault("APP_ENV", "development"),
		"version", envOrDefault("APP_VERSION", "unknown"),
	)
	catalogPath := os.Getenv("FACILITY_CATALOG_PATH")
	if catalogPath == "" {
		catalogPath = defaultCatalogPath
	}

	now := time.Now
	catalog, err := facility.LoadCatalogFileAt(catalogPath, now())
	if err != nil {
		logger.Error("facility_catalog_load_failed", "path", catalogPath, "error", err)
		os.Exit(1)
	}
	logger.Info("facility_catalog_loaded", "count", len(catalog.List("")))

	var correctionStore correction.RetentionStore
	var closeCorrectionStore func() error
	if databaseURL := os.Getenv("DATABASE_URL"); databaseURL != "" {
		postgresStore, postgresErr := correction.NewPostgresStore(databaseURL, now())
		if postgresErr != nil {
			logger.Error("correction_store_initialization_failed", "backend", "postgres", "error", postgresErr)
			os.Exit(1)
		}
		correctionStore = postgresStore
		closeCorrectionStore = postgresStore.Close
		logger.Info("correction_store_initialized", "backend", "postgres")
	} else {
		fileStore, fileErr := correction.NewFileStore(envOrDefault("CORRECTION_STORE_PATH", defaultCorrectionPath), now())
		if fileErr != nil {
			logger.Error("correction_store_initialization_failed", "backend", "file", "error", fileErr)
			os.Exit(1)
		}
		correctionStore = fileStore
		closeCorrectionStore = func() error { return nil }
		logger.Info("correction_store_initialized", "backend", "file")
	}
	defer func() {
		if err := closeCorrectionStore(); err != nil {
			logger.Error("correction_store_close_failed", "error", err)
		}
	}()
	correctionService, err := correction.NewService(correctionStore, now)
	if err != nil {
		logger.Error("correction_service_initialization_failed", "error", err)
		os.Exit(1)
	}
	retentionContext, stopRetentionWorker := context.WithCancel(context.Background())
	defer stopRetentionWorker()
	go purgeExpiredCorrections(retentionContext, correctionStore, logger, now)

	metrics := observability.NewRegistry()
	travelProvider := travel.Provider(travel.NewStraightLineProvider())
	var geocoder geocoding.Provider
	if mapsAPIKey := os.Getenv("GOOGLE_MAPS_API_KEY"); mapsAPIKey != "" {
		mapsTransport := http.DefaultTransport.(*http.Transport).Clone()
		mapsTransport.MaxConnsPerHost = googleMaxConnsPerHost
		mapsClient := &http.Client{Transport: mapsTransport, Timeout: googleHTTPTimeout}
		googleRoutes, routesErr := travel.NewGoogleRoutesProvider(mapsAPIKey, mapsClient)
		if routesErr != nil {
			logger.Error("google_routes_initialization_failed", "error", routesErr)
			os.Exit(1)
		}
		observedGoogleRoutes := observability.ObserveGoogleRoutes(metrics, googleRoutes)
		travelProvider, err = travel.NewFallbackProvider(observedGoogleRoutes, travelProvider)
		if err != nil {
			logger.Error("travel_provider_initialization_failed", "error", err)
			os.Exit(1)
		}
		googleGeocoder, geocoderErr := geocoding.NewGoogleProvider(mapsAPIKey, mapsClient)
		if geocoderErr != nil {
			logger.Error("geocoder_initialization_failed", "error", geocoderErr)
			os.Exit(1)
		}
		geocoder = observability.ObserveGoogleGeocoding(metrics, googleGeocoder)
		logger.Info("google_maps_integrations_enabled")
	} else {
		logger.Warn("google_maps_integrations_disabled", "travel_estimate", travel.StraightLineKind)
	}
	recommender := recommendation.NewEngineWithTravelProvider(catalog, now, travelProvider)

	server := &http.Server{
		Addr: ":" + envOrDefault("PORT", "8080"),
		Handler: httpapi.NewServerWithOptions(catalog, logger, httpapi.Options{
			Recommender:       recommender,
			Geocoder:          geocoder,
			CorrectionService: correctionService,
			Metrics:           metrics,
			Now:               now,
		}),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-shutdown
		stopRetentionWorker()
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

func purgeExpiredCorrections(ctx context.Context, store correction.RetentionStore, logger *slog.Logger, now func() time.Time) {
	ticker := time.NewTicker(correctionPurgeInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := store.PurgeExpired(now()); err != nil {
				logger.Error("correction_retention_purge_failed", "error", err)
			}
		}
	}
}

func envOrDefault(key string, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
