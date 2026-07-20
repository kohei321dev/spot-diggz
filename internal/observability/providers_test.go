package observability

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/kohei321dev/spot-diggz/internal/geocoding"
	"github.com/kohei321dev/spot-diggz/internal/travel"
)

var errTestProvider = errors.New("test provider failed")

type travelProviderStub struct {
	estimates []travel.Estimate
	err       error
}

func (provider travelProviderStub) Matrix(context.Context, travel.Request) ([]travel.Estimate, error) {
	return provider.estimates, provider.err
}

type geocodingProviderStub struct {
	results []geocoding.Result
	err     error
}

func (provider geocodingProviderStub) Search(context.Context, string) ([]geocoding.Result, error) {
	return provider.results, provider.err
}

func TestObservedGoogleProvidersRecordOnlyFixedOutcomeLabels(t *testing.T) {
	registry := NewRegistry()
	wantEstimates := []travel.Estimate{{FacilityID: "OSK-F001"}}
	estimates, err := ObserveGoogleRoutes(registry, travelProviderStub{estimates: wantEstimates}).Matrix(context.Background(), travel.Request{})
	if err != nil || len(estimates) != 1 {
		t.Fatalf("observed travel provider = %#v, %v", estimates, err)
	}
	if _, err := ObserveGoogleRoutes(registry, travelProviderStub{err: errTestProvider}).Matrix(context.Background(), travel.Request{}); !errors.Is(err, errTestProvider) {
		t.Fatalf("observed travel error = %v, want test error", err)
	}

	wantResults := []geocoding.Result{{Label: "Osaka"}}
	results, err := ObserveGoogleGeocoding(registry, geocodingProviderStub{results: wantResults}).Search(context.Background(), "private query")
	if err != nil || len(results) != 1 {
		t.Fatalf("observed geocoding provider = %#v, %v", results, err)
	}
	if _, err := ObserveGoogleGeocoding(registry, geocodingProviderStub{err: errTestProvider}).Search(context.Background(), "another query"); !errors.Is(err, errTestProvider) {
		t.Fatalf("observed geocoding error = %v, want test error", err)
	}

	body := scrapeRegistry(t, registry).Body.String()
	for _, wantedLine := range []string{
		`spot_diggz_external_requests_total{provider="google_routes",result="success"} 1`,
		`spot_diggz_external_requests_total{provider="google_routes",result="error"} 1`,
		`spot_diggz_external_requests_total{provider="google_geocoding",result="success"} 1`,
		`spot_diggz_external_requests_total{provider="google_geocoding",result="error"} 1`,
	} {
		assertMetricLine(t, body, wantedLine)
	}
	if strings.Contains(body, "private query") || strings.Contains(body, "another query") || strings.Contains(body, errTestProvider.Error()) {
		t.Fatal("metrics contain provider input or error text")
	}
}
