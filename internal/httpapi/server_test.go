package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/kohei321dev/spot-diggz/internal/correction"
	"github.com/kohei321dev/spot-diggz/internal/facility"
	"github.com/kohei321dev/spot-diggz/internal/geocoding"
	"github.com/kohei321dev/spot-diggz/internal/observability"
	"github.com/kohei321dev/spot-diggz/internal/ratelimit"
	"github.com/kohei321dev/spot-diggz/internal/recommendation"
)

func TestServerListsVerifiedFacilities(t *testing.T) {
	catalog, err := facility.NewCatalog([]facility.Facility{testFacility()})
	if err != nil {
		t.Fatalf("NewCatalog() error = %v", err)
	}
	handler := NewServer(catalog, slog.Default())

	req := httptest.NewRequest(http.MethodGet, "/api/facilities?activity=skateboard", nil)
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.Code, http.StatusOK)
	}
	if resp.Header().Get("X-Request-ID") == "" {
		t.Fatal("X-Request-ID is empty")
	}
	if got := resp.Header().Get("X-Content-Type-Options"); got != "nosniff" {
		t.Fatalf("X-Content-Type-Options = %q, want nosniff", got)
	}

	var body struct {
		Facilities []facility.Facility `json:"facilities"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(body.Facilities) != 1 || body.Facilities[0].ID != "facility-a" {
		t.Fatalf("facilities = %#v, want facility-a", body.Facilities)
	}
}

func TestServerGetsFacilityAndReturnsNotFound(t *testing.T) {
	catalog, err := facility.NewCatalog([]facility.Facility{testFacility()})
	if err != nil {
		t.Fatalf("NewCatalog() error = %v", err)
	}
	handler := NewServer(catalog, nil)

	found := httptest.NewRecorder()
	handler.ServeHTTP(found, httptest.NewRequest(http.MethodGet, "/api/facilities/facility-a", nil))
	if found.Code != http.StatusOK {
		t.Fatalf("found status = %d, want %d", found.Code, http.StatusOK)
	}

	notFound := httptest.NewRecorder()
	handler.ServeHTTP(notFound, httptest.NewRequest(http.MethodGet, "/api/facilities/missing", nil))
	if notFound.Code != http.StatusNotFound {
		t.Fatalf("missing status = %d, want %d", notFound.Code, http.StatusNotFound)
	}

	var body errorBody
	if err := json.NewDecoder(notFound.Body).Decode(&body); err != nil {
		t.Fatalf("decode error response: %v", err)
	}
	if body.Error.Code != "facility_not_found" {
		t.Fatalf("error code = %q, want facility_not_found", body.Error.Code)
	}
}

func TestServerReturnsCuratedExternalMetadataInFacilitiesAndRecommendations(t *testing.T) {
	item := testFacility()
	item.Media = &facility.FacilityMedia{
		YouTube: &facility.YouTubeVideo{
			Provider:        "youtube",
			VideoID:         "a1B2c3D4e5F",
			Title:           "Test Facility overview",
			SourceURL:       "https://www.youtube.com/watch?v=a1B2c3D4e5F",
			SelectedAt:      item.VerifiedAt,
			VerifiedAt:      item.VerifiedAt,
			SelectionReason: "施設のセクションを確認できるため",
		},
	}
	item.SocialLinks = []facility.SocialLink{
		{
			Platform:   facility.SocialPlatformInstagram,
			URL:        "https://www.instagram.com/spot_diggz/",
			VerifiedAt: item.VerifiedAt,
		},
		{
			Platform:   facility.SocialPlatformX,
			URL:        "https://x.com/spotdiggz",
			VerifiedAt: item.VerifiedAt,
		},
	}
	catalog, err := facility.NewCatalog([]facility.Facility{item})
	if err != nil {
		t.Fatalf("NewCatalog() error = %v", err)
	}
	handler := NewServer(catalog, nil)

	t.Run("facility list", func(t *testing.T) {
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/facilities", nil))
		if response.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
		}

		var body struct {
			Facilities []facility.Facility `json:"facilities"`
		}
		if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
			t.Fatalf("decode response: %v", err)
		}
		if len(body.Facilities) != 1 {
			t.Fatalf("facilities = %#v, want one facility", body.Facilities)
		}
		assertCuratedExternalMetadata(t, body.Facilities[0])
	})

	t.Run("recommendations", func(t *testing.T) {
		requestBody := `{
			"purpose":"basics",
			"mood":"focused",
			"level":"beginner",
			"availableMinutes":120,
			"transport":"public_transit",
			"origin":{"mode":"specified_location","latitude":34.7025,"longitude":135.4960}
		}`
		request := httptest.NewRequest(http.MethodPost, "/api/recommendations", strings.NewReader(requestBody))
		request.Header.Set("Content-Type", "application/json")
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d; body = %s", response.Code, http.StatusOK, response.Body.String())
		}

		var body recommendation.Response
		if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
			t.Fatalf("decode response: %v", err)
		}
		if len(body.Recommendations) != 1 {
			t.Fatalf("recommendations = %#v, want one facility", body.Recommendations)
		}
		assertCuratedExternalMetadata(t, body.Recommendations[0].Facility)
	})
}

func TestServerRejectsLongActivityQuery(t *testing.T) {
	catalog, err := facility.NewCatalog([]facility.Facility{testFacility()})
	if err != nil {
		t.Fatalf("NewCatalog() error = %v", err)
	}
	handler := NewServer(catalog, nil)
	requestURL := "/api/facilities?activity=" + strings.Repeat("a", 51)
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, httptest.NewRequest(http.MethodGet, requestURL, nil))

	if resp.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", resp.Code, http.StatusBadRequest)
	}
}

func TestServerHealth(t *testing.T) {
	catalog, err := facility.NewCatalog(nil)
	if err != nil {
		t.Fatalf("NewCatalog() error = %v", err)
	}
	handler := NewServer(catalog, nil)
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, httptest.NewRequest(http.MethodGet, "/healthz", nil))

	if resp.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.Code, http.StatusOK)
	}
}

func TestServerReadiness(t *testing.T) {
	catalog, err := facility.NewCatalog([]facility.Facility{testFacility()})
	if err != nil {
		t.Fatalf("NewCatalog() error = %v", err)
	}
	response := httptest.NewRecorder()
	NewServer(catalog, nil).ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/readyz", nil))

	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"status":"ready"`) {
		t.Fatalf("response = %d %s", response.Code, response.Body.String())
	}
}

func TestServerReadinessFailsWithoutFreshFacility(t *testing.T) {
	item := testFacility()
	item.DynamicVerifiedAt = fixedServerTime().Add(-facility.DynamicInformationFreshnessWindow - time.Minute)
	catalog, err := facility.NewCatalog([]facility.Facility{item})
	if err != nil {
		t.Fatalf("NewCatalog() error = %v", err)
	}
	response := httptest.NewRecorder()
	NewServerWithOptions(catalog, nil, Options{Now: fixedServerTime}).ServeHTTP(
		response,
		httptest.NewRequest(http.MethodGet, "/readyz", nil),
	)

	if response.Code != http.StatusServiceUnavailable || !strings.Contains(response.Body.String(), `"status":"not_ready"`) {
		t.Fatalf("response = %d %s", response.Code, response.Body.String())
	}
}

func TestServerReadinessIsRecalculatedAfterCatalogBecomesStale(t *testing.T) {
	currentTime := fixedServerTime()
	catalog, err := facility.NewCatalog([]facility.Facility{testFacility()})
	if err != nil {
		t.Fatalf("NewCatalog() error = %v", err)
	}
	handler := NewServerWithOptions(catalog, nil, Options{Now: func() time.Time { return currentTime }})

	readyResponse := httptest.NewRecorder()
	handler.ServeHTTP(readyResponse, httptest.NewRequest(http.MethodGet, "/readyz", nil))
	if readyResponse.Code != http.StatusOK {
		t.Fatalf("initial status = %d, want %d", readyResponse.Code, http.StatusOK)
	}

	currentTime = currentTime.Add(facility.DynamicInformationFreshnessWindow + time.Minute)
	staleResponse := httptest.NewRecorder()
	handler.ServeHTTP(staleResponse, httptest.NewRequest(http.MethodGet, "/readyz", nil))
	if staleResponse.Code != http.StatusServiceUnavailable || !strings.Contains(staleResponse.Body.String(), `"status":"not_ready"`) {
		t.Fatalf("stale response = %d %s", staleResponse.Code, staleResponse.Body.String())
	}
}

func TestMetricsRecalculateCatalogFreshnessAtScrapeTime(t *testing.T) {
	currentTime := fixedServerTime()
	catalog, err := facility.NewCatalog([]facility.Facility{testFacility()})
	if err != nil {
		t.Fatalf("NewCatalog() error = %v", err)
	}
	handler := NewServerWithOptions(catalog, nil, Options{Now: func() time.Time { return currentTime }})
	currentTime = currentTime.Add(facility.DynamicInformationFreshnessWindow + time.Minute)

	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("metrics status = %d, want %d", response.Code, http.StatusOK)
	}
	for _, wanted := range []string{
		`spot_diggz_catalog_freshness{state="fresh"} 0`,
		`spot_diggz_catalog_freshness{state="stale"} 1`,
	} {
		if !strings.Contains(response.Body.String(), wanted) {
			t.Fatalf("metrics missing %q\n%s", wanted, response.Body.String())
		}
	}
}

func TestServerReturnsRecommendations(t *testing.T) {
	catalog, err := facility.NewCatalog([]facility.Facility{testFacility()})
	if err != nil {
		t.Fatalf("NewCatalog() error = %v", err)
	}
	handler := NewServer(catalog, nil)
	requestBody := `{
		"purpose":"basics",
		"mood":"focused",
		"level":"beginner",
		"availableMinutes":120,
		"transport":"public_transit",
		"origin":{"mode":"specified_location","latitude":34.7025,"longitude":135.4960}
	}`
	request := httptest.NewRequest(http.MethodPost, "/api/recommendations", strings.NewReader(requestBody))
	request.Header.Set("Content-Type", "application/json; charset=utf-8")
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body = %s", response.Code, http.StatusOK, response.Body.String())
	}
	var body recommendation.Response
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(body.Recommendations) != 1 || body.Recommendations[0].Facility.ID != "facility-a" {
		t.Fatalf("recommendations = %#v, want facility-a", body.Recommendations)
	}
}

func TestServerRejectsInvalidRecommendationRequests(t *testing.T) {
	catalog, err := facility.NewCatalog([]facility.Facility{testFacility()})
	if err != nil {
		t.Fatalf("NewCatalog() error = %v", err)
	}
	handler := NewServer(catalog, nil)
	validBody := `{"purpose":"basics","mood":"focused","level":"beginner","availableMinutes":120,"transport":"public_transit","origin":{"mode":"specified_location","latitude":34.7025,"longitude":135.4960}}`

	tests := []struct {
		name        string
		body        string
		contentType string
		wantStatus  int
		wantCode    string
	}{
		{name: "unsupported content type", body: validBody, contentType: "text/plain", wantStatus: http.StatusUnsupportedMediaType, wantCode: "unsupported_media_type"},
		{name: "unknown field", body: strings.Replace(validBody, `"purpose":"basics"`, `"purpose":"basics","unknown":true`, 1), contentType: "application/json", wantStatus: http.StatusBadRequest, wantCode: "invalid_json"},
		{name: "invalid selection", body: strings.Replace(validBody, `"mood":"focused"`, `"mood":"unknown"`, 1), contentType: "application/json", wantStatus: http.StatusBadRequest, wantCode: "invalid_session_input"},
		{name: "too large", body: "{" + strings.Repeat(" ", maxRecommendationRequestBytes+1), contentType: "application/json", wantStatus: http.StatusRequestEntityTooLarge, wantCode: "request_too_large"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, "/api/recommendations", strings.NewReader(test.body))
			request.Header.Set("Content-Type", test.contentType)
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)

			if response.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d; body = %s", response.Code, test.wantStatus, response.Body.String())
			}
			var body errorBody
			if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
				t.Fatalf("decode error response: %v", err)
			}
			if body.Error.Code != test.wantCode {
				t.Fatalf("error code = %q, want %q", body.Error.Code, test.wantCode)
			}
		})
	}
}

func TestServerServesWebUI(t *testing.T) {
	catalog, err := facility.NewCatalog(nil)
	if err != nil {
		t.Fatalf("NewCatalog() error = %v", err)
	}
	response := httptest.NewRecorder()
	NewServer(catalog, nil).ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/", nil))

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	if !strings.Contains(response.Body.String(), "spot-diggz") {
		t.Fatal("response does not contain application name")
	}
	if got, want := response.Header().Get("Content-Security-Policy"), "default-src 'self'; base-uri 'none'; connect-src 'self'; form-action 'self'; frame-ancestors 'none'; frame-src https://www.youtube-nocookie.com; img-src 'self' data:; script-src 'self'; style-src 'self'"; got != want {
		t.Fatalf("Content-Security-Policy = %q, want %q", got, want)
	}
	if got, want := response.Header().Get("Referrer-Policy"), "strict-origin-when-cross-origin"; got != want {
		t.Fatalf("Referrer-Policy = %q, want %q", got, want)
	}
}

func TestServerDoesNotLogRecommendationCoordinates(t *testing.T) {
	catalog, err := facility.NewCatalog([]facility.Facility{testFacility()})
	if err != nil {
		t.Fatalf("NewCatalog() error = %v", err)
	}
	var logs bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logs, nil))
	handler := NewServer(catalog, logger)
	requestBody := `{"purpose":"basics","mood":"focused","level":"beginner","availableMinutes":120,"transport":"public_transit","origin":{"mode":"current_location","latitude":34.712345,"longitude":135.512345}}`
	request := httptest.NewRequest(http.MethodPost, "/api/recommendations", strings.NewReader(requestBody))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	if strings.Contains(logs.String(), "34.712345") || strings.Contains(logs.String(), "135.512345") {
		t.Fatalf("access log contains origin coordinates: %s", logs.String())
	}
	if strings.Contains(response.Body.String(), "34.712345") || strings.Contains(response.Body.String(), "135.512345") {
		t.Fatalf("response contains origin coordinates: %s", response.Body.String())
	}
}

func TestServerSearchesLocationsWithoutLoggingQuery(t *testing.T) {
	catalog, err := facility.NewCatalog(nil)
	if err != nil {
		t.Fatalf("NewCatalog() error = %v", err)
	}
	provider := &stubGeocoder{results: []geocoding.Result{{
		Label: "大阪駅", Location: facility.Location{Latitude: 34.7025, Longitude: 135.4959},
	}}}
	var logs bytes.Buffer
	handler := NewServerWithOptions(catalog, slog.New(slog.NewJSONHandler(&logs, nil)), Options{Geocoder: provider})
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, newLocationSearchRequest(`{"query":"secret-place"}`))

	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), "大阪駅") {
		t.Fatalf("response = %d %s", response.Code, response.Body.String())
	}
	if provider.query != "secret-place" {
		t.Fatalf("query = %q", provider.query)
	}
	if strings.Contains(logs.String(), "secret-place") {
		t.Fatalf("access log contains location query: %s", logs.String())
	}
}

func TestServerReturnsUnavailableWhenLocationSearchIsNotConfigured(t *testing.T) {
	catalog, err := facility.NewCatalog(nil)
	if err != nil {
		t.Fatalf("NewCatalog() error = %v", err)
	}
	response := httptest.NewRecorder()
	NewServer(catalog, nil).ServeHTTP(response, newLocationSearchRequest(`{"query":"大阪駅"}`))

	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusServiceUnavailable)
	}
}

func TestServerDoesNotAcceptLocationSearchInURLQuery(t *testing.T) {
	catalog, err := facility.NewCatalog(nil)
	if err != nil {
		t.Fatalf("NewCatalog() error = %v", err)
	}
	provider := &stubGeocoder{}
	response := httptest.NewRecorder()
	NewServerWithOptions(catalog, nil, Options{Geocoder: provider}).ServeHTTP(
		response,
		httptest.NewRequest(http.MethodGet, "/api/locations/search?q=private-address", nil),
	)
	if response.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusNotFound)
	}
	if provider.calls != 0 {
		t.Fatalf("provider calls = %d, want 0", provider.calls)
	}
}

func TestServerMapsInvalidAndUnavailableLocationSearchErrors(t *testing.T) {
	catalog, err := facility.NewCatalog(nil)
	if err != nil {
		t.Fatalf("NewCatalog() error = %v", err)
	}
	tests := []struct {
		name        string
		providerErr error
		wantStatus  int
		wantCode    string
	}{
		{name: "invalid query", providerErr: geocoding.ErrInvalidQuery, wantStatus: http.StatusBadRequest, wantCode: "invalid_location_query"},
		{name: "provider unavailable", providerErr: geocoding.ErrUnavailable, wantStatus: http.StatusServiceUnavailable, wantCode: "location_search_unavailable"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			response := httptest.NewRecorder()
			handler := NewServerWithOptions(catalog, nil, Options{Geocoder: &stubGeocoder{err: test.providerErr}})
			handler.ServeHTTP(response, newLocationSearchRequest(`{"query":"osaka"}`))
			if response.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d", response.Code, test.wantStatus)
			}
			var body errorBody
			if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
				t.Fatalf("decode error response: %v", err)
			}
			if body.Error.Code != test.wantCode {
				t.Fatalf("error code = %q, want %q", body.Error.Code, test.wantCode)
			}
		})
	}
}

func TestServerValidatesLocationSearchJSON(t *testing.T) {
	catalog, err := facility.NewCatalog(nil)
	if err != nil {
		t.Fatalf("NewCatalog() error = %v", err)
	}
	handler := NewServerWithOptions(catalog, nil, Options{Geocoder: &stubGeocoder{}})
	tests := []struct {
		name        string
		body        string
		contentType string
		wantStatus  int
	}{
		{name: "unknown field", body: `{"query":"osaka","extra":true}`, contentType: "application/json", wantStatus: http.StatusBadRequest},
		{name: "unsupported content type", body: `{"query":"osaka"}`, contentType: "text/plain", wantStatus: http.StatusUnsupportedMediaType},
		{name: "too large", body: "{" + strings.Repeat(" ", maxLocationSearchRequestBytes+1), contentType: "application/json", wantStatus: http.StatusRequestEntityTooLarge},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, "/api/locations/search", strings.NewReader(test.body))
			request.Header.Set("Content-Type", test.contentType)
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d; body = %s", response.Code, test.wantStatus, response.Body.String())
			}
		})
	}
}

func TestServerRateLimitsLocationSearchBeforeCallingProvider(t *testing.T) {
	catalog, err := facility.NewCatalog(nil)
	if err != nil {
		t.Fatalf("NewCatalog() error = %v", err)
	}
	provider := &stubGeocoder{}
	limiter := ratelimit.New(1, 1, fixedServerTime)
	handler := NewServerWithOptions(catalog, nil, Options{
		Geocoder:              provider,
		LocationSearchLimiter: limiter,
		Now:                   fixedServerTime,
	})
	for index, wantStatus := range []int{http.StatusOK, http.StatusTooManyRequests} {
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, newLocationSearchRequest(`{"query":"osaka"}`))
		if response.Code != wantStatus {
			t.Fatalf("request %d status = %d, want %d; body = %s", index, response.Code, wantStatus, response.Body.String())
		}
	}
	if provider.calls != 1 {
		t.Fatalf("provider calls = %d, want 1", provider.calls)
	}
}

func TestServerAcceptsCorrectionAndStoresRetentionMetadata(t *testing.T) {
	catalog, err := facility.NewCatalog([]facility.Facility{testFacility()})
	if err != nil {
		t.Fatalf("NewCatalog() error = %v", err)
	}
	store := correction.NewMemoryStore()
	service, err := correction.NewService(store, fixedServerTime)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	registry := observability.NewRegistry()
	handler := NewServerWithOptions(catalog, nil, Options{CorrectionService: service, Metrics: registry, Now: fixedServerTime})
	body := `{"facilityId":"facility-a","category":"hours","details":"公式ページでは営業時間が変更されています。","evidenceUrl":"https://example.com/official"}`
	request := httptest.NewRequest(http.MethodPost, "/api/corrections", strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	if response.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want %d; body = %s", response.Code, http.StatusAccepted, response.Body.String())
	}
	reports := store.Reports()
	if len(reports) != 1 || reports[0].FacilityID != "facility-a" {
		t.Fatalf("reports = %#v", reports)
	}
	if want := fixedServerTime().AddDate(0, 0, correction.RetentionDays); !reports[0].DeleteAfter.Equal(want) {
		t.Fatalf("DeleteAfter = %s, want %s", reports[0].DeleteAfter, want)
	}
	metricsResponse := httptest.NewRecorder()
	handler.ServeHTTP(metricsResponse, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	if !strings.Contains(metricsResponse.Body.String(), `spot_diggz_product_events_total{event="correction_submitted"} 1`) {
		t.Fatalf("correction metric was not incremented\n%s", metricsResponse.Body.String())
	}
}

func TestServerRejectsCorrectionForUnknownFacility(t *testing.T) {
	catalog, err := facility.NewCatalog([]facility.Facility{testFacility()})
	if err != nil {
		t.Fatalf("NewCatalog() error = %v", err)
	}
	body := `{"facilityId":"missing","category":"hours","details":"公式ページでは営業時間が変更されています。"}`
	request := httptest.NewRequest(http.MethodPost, "/api/corrections", strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	NewServer(catalog, nil).ServeHTTP(response, request)

	if response.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusNotFound)
	}
}

func TestServerRejectsInvalidCorrectionRequests(t *testing.T) {
	catalog, err := facility.NewCatalog([]facility.Facility{testFacility()})
	if err != nil {
		t.Fatalf("NewCatalog() error = %v", err)
	}
	validBody := `{"facilityId":"facility-a","category":"hours","details":"公式ページでは営業時間が変更されています。"}`
	tests := []struct {
		name        string
		body        string
		contentType string
		wantStatus  int
		wantCode    string
	}{
		{name: "unknown field", body: strings.Replace(validBody, `}`, `,"unknown":true}`, 1), contentType: "application/json", wantStatus: http.StatusBadRequest, wantCode: "invalid_json"},
		{name: "contact without consent", body: strings.Replace(validBody, `}`, `,"contact":"user@example.com"}`, 1), contentType: "application/json", wantStatus: http.StatusBadRequest, wantCode: "invalid_correction"},
		{name: "unsupported content type", body: validBody, contentType: "text/plain", wantStatus: http.StatusUnsupportedMediaType, wantCode: "unsupported_media_type"},
		{name: "too large", body: "{" + strings.Repeat(" ", maxCorrectionRequestBytes+1), contentType: "application/json", wantStatus: http.StatusRequestEntityTooLarge, wantCode: "request_too_large"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, "/api/corrections", strings.NewReader(test.body))
			request.Header.Set("Content-Type", test.contentType)
			response := httptest.NewRecorder()
			NewServer(catalog, nil).ServeHTTP(response, request)
			if response.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d; body = %s", response.Code, test.wantStatus, response.Body.String())
			}
			var body errorBody
			if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
				t.Fatalf("decode error response: %v", err)
			}
			if body.Error.Code != test.wantCode {
				t.Fatalf("error code = %q, want %q", body.Error.Code, test.wantCode)
			}
		})
	}
}

func TestServerDoesNotLogCorrectionContentOrContact(t *testing.T) {
	catalog, err := facility.NewCatalog([]facility.Facility{testFacility()})
	if err != nil {
		t.Fatalf("NewCatalog() error = %v", err)
	}
	store := correction.NewMemoryStore()
	service, err := correction.NewService(store, fixedServerTime)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	var logs bytes.Buffer
	handler := NewServerWithOptions(catalog, slog.New(slog.NewJSONHandler(&logs, nil)), Options{
		CorrectionService: service,
		Now:               fixedServerTime,
	})
	body := `{"facilityId":"facility-a","category":"hours","details":"PRIVATE-CORRECTION-CONTENT","contact":"private-contact@example.com","contactConsent":true}`
	request := httptest.NewRequest(http.MethodPost, "/api/corrections", strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	if response.Code != http.StatusAccepted {
		t.Fatalf("status = %d; body = %s", response.Code, response.Body.String())
	}
	for _, forbidden := range []string{"PRIVATE-CORRECTION-CONTENT", "private-contact@example.com"} {
		if strings.Contains(logs.String(), forbidden) {
			t.Fatalf("logs contain private correction value %q: %s", forbidden, logs.String())
		}
	}
}

func TestServerRecordsAllowListedEventsAndExposesMetrics(t *testing.T) {
	catalog, err := facility.NewCatalog([]facility.Facility{testFacility()})
	if err != nil {
		t.Fatalf("NewCatalog() error = %v", err)
	}
	registry := observability.NewRegistry()
	handler := NewServerWithOptions(catalog, nil, Options{Metrics: registry, Now: fixedServerTime})
	for _, event := range []string{
		"result_displayed",
		"video_embed_requested",
		"video_embed_loaded",
		"video_external_opened",
		"social_profile_opened",
	} {
		eventRequest := httptest.NewRequest(http.MethodPost, "/api/events", strings.NewReader(`{"event":"`+event+`"}`))
		eventRequest.Header.Set("Content-Type", "application/json")
		eventResponse := httptest.NewRecorder()
		handler.ServeHTTP(eventResponse, eventRequest)
		if eventResponse.Code != http.StatusAccepted {
			t.Fatalf("event %q status = %d; body = %s", event, eventResponse.Code, eventResponse.Body.String())
		}
	}

	metricsResponse := httptest.NewRecorder()
	handler.ServeHTTP(metricsResponse, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	if metricsResponse.Code != http.StatusOK {
		t.Fatalf("metrics status = %d", metricsResponse.Code)
	}
	for _, wanted := range []string{
		`spot_diggz_product_events_total{event="result_displayed"} 1`,
		`spot_diggz_product_events_total{event="video_embed_requested"} 1`,
		`spot_diggz_product_events_total{event="video_embed_loaded"} 1`,
		`spot_diggz_product_events_total{event="video_external_opened"} 1`,
		`spot_diggz_product_events_total{event="social_profile_opened"} 1`,
		`spot_diggz_http_requests_total{route="/api/events",method="POST",status_class="2xx"} 5`,
		`spot_diggz_catalog_facilities 1`,
	} {
		if !strings.Contains(metricsResponse.Body.String(), wanted) {
			t.Fatalf("metrics missing %q\n%s", wanted, metricsResponse.Body.String())
		}
	}
}

func TestServerRejectsProductEventsWithTargetData(t *testing.T) {
	catalog, err := facility.NewCatalog(nil)
	if err != nil {
		t.Fatalf("NewCatalog() error = %v", err)
	}
	handler := NewServer(catalog, nil)

	request := httptest.NewRequest(http.MethodPost, "/api/events", strings.NewReader(`{"event":"video_embed_requested","facilityId":"facility-a","videoId":"dQw4w9WgXcQ","url":"https://www.youtube.com/watch?v=dQw4w9WgXcQ"}`))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d; body = %s", response.Code, http.StatusBadRequest, response.Body.String())
	}
	var body errorBody
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("decode error response: %v", err)
	}
	if body.Error.Code != "invalid_json" {
		t.Fatalf("error code = %q, want invalid_json", body.Error.Code)
	}
}

func TestServerRejectsUnknownProductEvent(t *testing.T) {
	catalog, err := facility.NewCatalog(nil)
	if err != nil {
		t.Fatalf("NewCatalog() error = %v", err)
	}
	request := httptest.NewRequest(http.MethodPost, "/api/events", strings.NewReader(`{"event":"user-123"}`))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	NewServer(catalog, nil).ServeHTTP(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusBadRequest)
	}
}

func TestServerRejectsServerOwnedProductEvent(t *testing.T) {
	catalog, err := facility.NewCatalog(nil)
	if err != nil {
		t.Fatalf("NewCatalog() error = %v", err)
	}
	request := httptest.NewRequest(http.MethodPost, "/api/events", strings.NewReader(`{"event":"correction_submitted"}`))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	NewServer(catalog, nil).ServeHTTP(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusBadRequest)
	}
}

func TestServerRateLimitsRecommendationRequests(t *testing.T) {
	catalog, err := facility.NewCatalog([]facility.Facility{testFacility()})
	if err != nil {
		t.Fatalf("NewCatalog() error = %v", err)
	}
	limiter := ratelimit.New(1, 1, fixedServerTime)
	handler := NewServerWithOptions(catalog, nil, Options{RecommendationLimiter: limiter, Now: fixedServerTime})
	body := `{"purpose":"basics","mood":"focused","level":"beginner","availableMinutes":120,"transport":"public_transit","origin":{"mode":"specified_location","latitude":34.7025,"longitude":135.4960}}`
	for index, wantStatus := range []int{http.StatusOK, http.StatusTooManyRequests} {
		request := httptest.NewRequest(http.MethodPost, "/api/recommendations", strings.NewReader(body))
		request.Header.Set("Content-Type", "application/json")
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != wantStatus {
			t.Fatalf("request %d status = %d, want %d; body = %s", index, response.Code, wantStatus, response.Body.String())
		}
	}
}

func testFacility() facility.Facility {
	verifiedAt := time.Date(2026, time.July, 15, 0, 0, 0, 0, time.UTC)
	return facility.Facility{
		ID:               "facility-a",
		Name:             "Test Facility",
		Address:          "大阪府大阪市",
		Prefecture:       "大阪府",
		Municipality:     "大阪市",
		Location:         facility.Location{Latitude: 34.6937, Longitude: 135.5023},
		Activities:       []string{"skateboard"},
		Hours:            []facility.OperatingHours{{Day: "daily", Opens: "00:00", Closes: "24:00"}},
		ScheduleNotes:    []string{"臨時変更は公式情報を確認"},
		Price:            "500円",
		Reservation:      "当日受付",
		BeginnerFriendly: true,
		Features:         []string{"flat-area"},
		Rules:            []string{"ヘルメット必須"},
		Access:           facility.Access{Notes: "テスト用アクセス"},
		EnglishTranslation: facility.FacilityEnglishTranslation{
			Name: "Test Facility", Address: "Osaka City, Osaka", ScheduleNotes: []string{"Check the official source for temporary changes."},
			Price: "JPY 500", Reservation: "Register on arrival", Rules: []string{"Helmet required"}, AccessNotes: "Test access",
		},
		SourceURL:         "https://example.com/facilities/a",
		SourceType:        "official",
		Status:            "verified",
		Confidence:        "high",
		VerifiedAt:        verifiedAt,
		DynamicVerifiedAt: verifiedAt,
		StableVerifiedAt:  verifiedAt,
	}
}

func assertCuratedExternalMetadata(t *testing.T, item facility.Facility) {
	t.Helper()
	if item.Media == nil || item.Media.YouTube == nil {
		t.Fatalf("media = %#v, want YouTube metadata", item.Media)
	}
	if got, want := item.Media.YouTube.VideoID, "a1B2c3D4e5F"; got != want {
		t.Fatalf("media.youtube.videoId = %q, want %q", got, want)
	}
	if got, want := item.Media.YouTube.SourceURL, "https://www.youtube.com/watch?v=a1B2c3D4e5F"; got != want {
		t.Fatalf("media.youtube.sourceUrl = %q, want %q", got, want)
	}
	if len(item.SocialLinks) != 2 {
		t.Fatalf("socialLinks = %#v, want Instagram and X profiles", item.SocialLinks)
	}
	if got, want := item.SocialLinks[0].URL, "https://www.instagram.com/spot_diggz/"; got != want {
		t.Fatalf("socialLinks[0].url = %q, want %q", got, want)
	}
	if got, want := item.SocialLinks[1].URL, "https://x.com/spotdiggz"; got != want {
		t.Fatalf("socialLinks[1].url = %q, want %q", got, want)
	}
}

func fixedServerTime() time.Time {
	return time.Date(2026, time.July, 19, 12, 0, 0, 0, time.UTC)
}

type stubGeocoder struct {
	query   string
	results []geocoding.Result
	err     error
	calls   int
}

func (provider *stubGeocoder) Search(_ context.Context, query string) ([]geocoding.Result, error) {
	provider.calls++
	provider.query = query
	return provider.results, provider.err
}

func newLocationSearchRequest(body string) *http.Request {
	request := httptest.NewRequest(http.MethodPost, "/api/locations/search", strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	return request
}
