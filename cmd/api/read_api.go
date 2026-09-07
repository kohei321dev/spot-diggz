package main

import (
	"errors"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/kohei321dev/spot-diggz/internal/apiauth"
	"github.com/kohei321dev/spot-diggz/internal/facility"
	"github.com/kohei321dev/spot-diggz/internal/geocoding"
	"github.com/kohei321dev/spot-diggz/internal/httpapi"
	"github.com/kohei321dev/spot-diggz/internal/observability"
)

// API mode never initializes the legacy OAuth, correction store, or chat handlers.
func buildReadAPI(catalog *facility.Catalog, logger *slog.Logger, now func() time.Time) (http.Handler, error) {
	credentials, err := apiauth.Parse(os.Getenv("API_CLIENTS_JSON"), os.Getenv("API_OWNER_ID"), now)
	if err != nil {
		return nil, err
	}
	metrics := observability.NewRegistry()
	var geocoder geocoding.Provider
	if key := os.Getenv("GOOGLE_MAPS_API_KEY"); key != "" {
		transport := http.DefaultTransport.(*http.Transport).Clone()
		transport.MaxConnsPerHost = googleMaxConnsPerHost
		provider, err := geocoding.NewGoogleProvider(key, &http.Client{Transport: transport, Timeout: googleHTTPTimeout})
		if err != nil {
			return nil, errors.New("location provider configuration is invalid")
		}
		geocoder = observability.ObserveGoogleGeocoding(metrics, provider)
	}
	return httpapi.NewReadAPI(catalog, credentials, geocoder, logger, metrics, now), nil
}
