package observability

import (
	"context"
	"time"

	"github.com/kohei321dev/spot-diggz/internal/geocoding"
	"github.com/kohei321dev/spot-diggz/internal/travel"
)

type observedTravelProvider struct {
	registry *Registry
	next     travel.Provider
}

// ObserveGoogleRoutes wraps the external provider without recording request
// coordinates, transport selections, response values, or error text.
func ObserveGoogleRoutes(registry *Registry, next travel.Provider) travel.Provider {
	if registry == nil || next == nil {
		return next
	}
	return &observedTravelProvider{registry: registry, next: next}
}

func (provider *observedTravelProvider) Matrix(ctx context.Context, request travel.Request) ([]travel.Estimate, error) {
	startedAt := time.Now()
	estimates, err := provider.next.Matrix(ctx, request)
	result := ExternalResultSuccess
	if err != nil {
		result = ExternalResultError
	}
	_ = provider.registry.ObserveExternalCall(ExternalProviderGoogleRoutes, result, time.Since(startedAt))
	return estimates, err
}

type observedGeocodingProvider struct {
	registry *Registry
	next     geocoding.Provider
}

// ObserveGoogleGeocoding records fixed outcome and latency labels only.
func ObserveGoogleGeocoding(registry *Registry, next geocoding.Provider) geocoding.Provider {
	if registry == nil || next == nil {
		return next
	}
	return &observedGeocodingProvider{registry: registry, next: next}
}

func (provider *observedGeocodingProvider) Search(ctx context.Context, query string) ([]geocoding.Result, error) {
	startedAt := time.Now()
	results, err := provider.next.Search(ctx, query)
	result := ExternalResultSuccess
	if err != nil {
		result = ExternalResultError
	}
	_ = provider.registry.ObserveExternalCall(ExternalProviderGoogleGeocoding, result, time.Since(startedAt))
	return results, err
}
