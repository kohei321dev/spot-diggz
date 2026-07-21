package observability

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestRegistryExposesRequiredMetrics(t *testing.T) {
	registry := NewRegistry()
	registry.ObserveHTTP("/api/facilities/{facilityId}", "get", http.StatusCreated, 75*time.Millisecond)
	if err := registry.ObserveRecommendation(RecommendationResultNoResults, 750*time.Millisecond); err != nil {
		t.Fatalf("ObserveRecommendation() error = %v", err)
	}
	if err := registry.ObserveProductEvent(ProductEventResultDisplayed); err != nil {
		t.Fatalf("ObserveProductEvent() error = %v", err)
	}
	if err := registry.ObserveExternalCall(ExternalProviderGoogleRoutes, ExternalResultSuccess, 450*time.Millisecond); err != nil {
		t.Fatalf("ObserveExternalCall() error = %v", err)
	}
	if err := registry.SetCatalog(3, 2, 1); err != nil {
		t.Fatalf("SetCatalog() error = %v", err)
	}

	body := scrapeRegistry(t, registry).Body.String()
	wantedLines := []string{
		`spot_diggz_http_requests_total{route="/api/facilities/{facilityId}",method="GET",status_class="2xx"} 1`,
		`spot_diggz_http_request_duration_seconds_bucket{route="/api/facilities/{facilityId}",method="GET",status_class="2xx",le="0.05"} 0`,
		`spot_diggz_http_request_duration_seconds_bucket{route="/api/facilities/{facilityId}",method="GET",status_class="2xx",le="0.1"} 1`,
		`spot_diggz_http_request_duration_seconds_count{route="/api/facilities/{facilityId}",method="GET",status_class="2xx"} 1`,
		`spot_diggz_recommendations_total{result="no_results"} 1`,
		`spot_diggz_recommendation_duration_seconds_bucket{result="no_results",le="0.5"} 0`,
		`spot_diggz_recommendation_duration_seconds_bucket{result="no_results",le="1"} 1`,
		`spot_diggz_recommendation_duration_seconds_count{result="no_results"} 1`,
		`spot_diggz_external_requests_total{provider="google_routes",result="success"} 1`,
		`spot_diggz_external_request_duration_seconds_bucket{provider="google_routes",result="success",le="0.25"} 0`,
		`spot_diggz_external_request_duration_seconds_bucket{provider="google_routes",result="success",le="0.5"} 1`,
		`spot_diggz_external_request_duration_seconds_count{provider="google_routes",result="success"} 1`,
		`spot_diggz_product_events_total{event="result_displayed"} 1`,
		`spot_diggz_catalog_facilities 3`,
		`spot_diggz_catalog_freshness{state="fresh"} 2`,
		`spot_diggz_catalog_freshness{state="stale"} 1`,
	}
	for _, wantedLine := range wantedLines {
		assertMetricLine(t, body, wantedLine)
	}

	for _, forbiddenLabel := range []string{"path=", "status_code=", "request_id=", "ip="} {
		if strings.Contains(body, forbiddenLabel) {
			t.Errorf("metrics output contains forbidden HTTP label %q", forbiddenLabel)
		}
	}
}

func TestObserveExternalCallRejectsUnknownLabels(t *testing.T) {
	registry := NewRegistry()

	if err := registry.ObserveExternalCall("custom_provider", ExternalResultSuccess, time.Second); !errors.Is(err, ErrUnknownExternalProvider) {
		t.Fatalf("unknown provider error = %v, want ErrUnknownExternalProvider", err)
	}
	if err := registry.ObserveExternalCall(ExternalProviderGoogleRoutes, "timeout_with_query", time.Second); !errors.Is(err, ErrUnknownExternalResult) {
		t.Fatalf("unknown result error = %v, want ErrUnknownExternalResult", err)
	}

	body := scrapeRegistry(t, registry).Body.String()
	if strings.Contains(body, "custom_provider") || strings.Contains(body, "timeout_with_query") {
		t.Fatal("unknown external label was exposed in metrics")
	}
}

func TestPrometheusHandlerSetsContentType(t *testing.T) {
	response := scrapeRegistry(t, NewRegistry())

	if got := response.Header().Get("Content-Type"); got != PrometheusContentType {
		t.Errorf("Content-Type = %q, want %q", got, PrometheusContentType)
	}
	if response.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", response.Code, http.StatusOK)
	}
}

func TestPrometheusHandlerEscapesLabelValues(t *testing.T) {
	registry := NewRegistry()
	routeTemplate := "/api/{facility\"id}\\segment\nnext"
	registry.ObserveHTTP(routeTemplate, http.MethodGet, http.StatusOK, time.Millisecond)

	body := scrapeRegistry(t, registry).Body.String()
	wantedLabels := `route="/api/{facility\"id}\\segment\nnext",method="GET",status_class="2xx"`
	if !strings.Contains(body, wantedLabels) {
		t.Fatalf("metrics output does not contain escaped labels %q\noutput:\n%s", wantedLabels, body)
	}
	if strings.Contains(body, "segment\nnext") {
		t.Fatal("metrics output contains an unescaped label newline")
	}

	input := "quote\" slash\\ line\nnext"
	want := `quote\" slash\\ line\nnext`
	if got := escapePrometheusLabelValue(input); got != want {
		t.Errorf("escapePrometheusLabelValue() = %q, want %q", got, want)
	}
}

func TestObserveProductEventRejectsUnknownEvent(t *testing.T) {
	registry := NewRegistry()

	err := registry.ObserveProductEvent(ProductEvent("session_identifier"))
	if !errors.Is(err, ErrUnknownProductEvent) {
		t.Fatalf("ObserveProductEvent() error = %v, want ErrUnknownProductEvent", err)
	}

	body := scrapeRegistry(t, registry).Body.String()
	if strings.Contains(body, "session_identifier") {
		t.Fatal("unknown product event was exposed as a metric label")
	}
	for _, event := range productEvents {
		assertMetricLine(
			t,
			body,
			fmt.Sprintf(`spot_diggz_product_events_total{event=%q} 0`, event),
		)
	}
}

func TestObserveClientProductEventRejectsServerOwnedEvent(t *testing.T) {
	registry := NewRegistry()

	for _, event := range []ProductEvent{
		ProductEventResultDisplayed,
		ProductEventVideoEmbedDisplayed,
		ProductEventVideoEmbedLoaded,
		ProductEventVideoExternalOpened,
		ProductEventSocialProfileOpened,
	} {
		if err := registry.ObserveClientProductEvent(event); err != nil {
			t.Fatalf("ObserveClientProductEvent(%q) error = %v", event, err)
		}
	}
	if err := registry.ObserveClientProductEvent(ProductEventCorrectionSubmitted); !errors.Is(err, ErrUnknownProductEvent) {
		t.Fatalf("ObserveClientProductEvent() error = %v, want ErrUnknownProductEvent", err)
	}

	body := scrapeRegistry(t, registry).Body.String()
	assertMetricLine(t, body, `spot_diggz_product_events_total{event="result_displayed"} 1`)
	assertMetricLine(t, body, `spot_diggz_product_events_total{event="video_embed_displayed"} 1`)
	assertMetricLine(t, body, `spot_diggz_product_events_total{event="video_embed_loaded"} 1`)
	assertMetricLine(t, body, `spot_diggz_product_events_total{event="video_external_opened"} 1`)
	assertMetricLine(t, body, `spot_diggz_product_events_total{event="social_profile_opened"} 1`)
	assertMetricLine(t, body, `spot_diggz_product_events_total{event="correction_submitted"} 0`)
}

func TestRegistrySupportsConcurrentUpdatesAndScrapes(t *testing.T) {
	const (
		workerCount           = 24
		observationsPerWorker = 250
		scraperCount          = 6
		scrapesPerWorker      = 40
	)

	registry := NewRegistry()
	start := make(chan struct{})
	errorsChannel := make(chan error, workerCount*2)
	var waitGroup sync.WaitGroup

	for range workerCount {
		waitGroup.Add(1)
		go func() {
			defer waitGroup.Done()
			<-start
			for range observationsPerWorker {
				registry.ObserveHTTP("POST /api/recommendations", http.MethodPost, http.StatusOK, 25*time.Millisecond)
				if err := registry.ObserveRecommendation(RecommendationResultSuccess, 100*time.Millisecond); err != nil {
					errorsChannel <- err
					return
				}
				if err := registry.ObserveProductEvent(ProductEventRecommendationCompleted); err != nil {
					errorsChannel <- err
					return
				}
				if err := registry.ObserveExternalCall(ExternalProviderGoogleRoutes, ExternalResultSuccess, 50*time.Millisecond); err != nil {
					errorsChannel <- err
					return
				}
				if err := registry.SetCatalog(2, 1, 1); err != nil {
					errorsChannel <- err
					return
				}
			}
		}()
	}
	for range scraperCount {
		waitGroup.Add(1)
		go func() {
			defer waitGroup.Done()
			<-start
			for range scrapesPerWorker {
				response := httptest.NewRecorder()
				request := httptest.NewRequest(http.MethodGet, "/metrics", nil)
				registry.Handler().ServeHTTP(response, request)
				if response.Code != http.StatusOK {
					errorsChannel <- fmt.Errorf("scrape status = %d", response.Code)
					return
				}
			}
		}()
	}

	close(start)
	waitGroup.Wait()
	close(errorsChannel)
	for err := range errorsChannel {
		t.Errorf("concurrent registry operation failed: %v", err)
	}

	wantCount := workerCount * observationsPerWorker
	body := scrapeRegistry(t, registry).Body.String()
	wantedLines := []string{
		fmt.Sprintf(`spot_diggz_http_requests_total{route="/api/recommendations",method="POST",status_class="2xx"} %d`, wantCount),
		fmt.Sprintf(`spot_diggz_http_request_duration_seconds_count{route="/api/recommendations",method="POST",status_class="2xx"} %d`, wantCount),
		fmt.Sprintf(`spot_diggz_recommendations_total{result="success"} %d`, wantCount),
		fmt.Sprintf(`spot_diggz_recommendation_duration_seconds_count{result="success"} %d`, wantCount),
		fmt.Sprintf(`spot_diggz_product_events_total{event="recommendation_completed"} %d`, wantCount),
		fmt.Sprintf(`spot_diggz_external_requests_total{provider="google_routes",result="success"} %d`, wantCount),
	}
	for _, wantedLine := range wantedLines {
		assertMetricLine(t, body, wantedLine)
	}
}

func TestSetCatalogRejectsInconsistentCountsWithoutChangingGauges(t *testing.T) {
	registry := NewRegistry()
	if err := registry.SetCatalog(3, 2, 1); err != nil {
		t.Fatalf("SetCatalog() error = %v", err)
	}

	err := registry.SetCatalog(4, 1, 1)
	if !errors.Is(err, ErrInvalidCatalogCounts) {
		t.Fatalf("SetCatalog() error = %v, want ErrInvalidCatalogCounts", err)
	}

	body := scrapeRegistry(t, registry).Body.String()
	assertMetricLine(t, body, "spot_diggz_catalog_facilities 3")
	assertMetricLine(t, body, `spot_diggz_catalog_freshness{state="fresh"} 2`)
	assertMetricLine(t, body, `spot_diggz_catalog_freshness{state="stale"} 1`)
}

func scrapeRegistry(t *testing.T, registry *Registry) *httptest.ResponseRecorder {
	t.Helper()
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	registry.Handler().ServeHTTP(response, request)
	return response
}

func assertMetricLine(t *testing.T, body string, wantedLine string) {
	t.Helper()
	for _, line := range strings.Split(body, "\n") {
		if line == wantedLine {
			return
		}
	}
	t.Errorf("metrics output does not contain line %q", wantedLine)
}
