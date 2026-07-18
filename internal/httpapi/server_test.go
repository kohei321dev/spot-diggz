package httpapi

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/kohei321dev/spot-diggz/internal/facility"
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
	if response.Header().Get("Content-Security-Policy") == "" {
		t.Fatal("Content-Security-Policy is empty")
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

func testFacility() facility.Facility {
	return facility.Facility{
		ID:               "facility-a",
		Name:             "Test Facility",
		Address:          "大阪府大阪市",
		Location:         facility.Location{Latitude: 34.6937, Longitude: 135.5023},
		Activities:       []string{"skateboard"},
		Hours:            []facility.OperatingHours{{Day: "daily", Opens: "00:00", Closes: "24:00"}},
		Price:            "500円",
		Reservation:      "当日受付",
		BeginnerFriendly: true,
		Features:         []string{"flat-area"},
		Rules:            []string{"ヘルメット必須"},
		SourceURL:        "https://example.com/facilities/a",
		SourceType:       "official",
		Status:           "verified",
		Confidence:       "high",
		VerifiedAt:       time.Date(2026, time.July, 15, 0, 0, 0, 0, time.UTC),
	}
}
