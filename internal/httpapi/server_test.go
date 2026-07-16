package httpapi

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/kohei321dev/spot-diggz/internal/facility"
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
	requestURL := "/api/facilities?activity=" + "a" + "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
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

func testFacility() facility.Facility {
	return facility.Facility{
		ID:               "facility-a",
		Name:             "Test Facility",
		Address:          "大阪府大阪市",
		Location:         facility.Location{Latitude: 34.6937, Longitude: 135.5023},
		Activities:       []string{"skateboard"},
		BeginnerFriendly: true,
		SourceURL:        "https://example.com/facilities/a",
		SourceType:       "official",
		Status:           "verified",
		VerifiedAt:       time.Date(2026, time.July, 15, 0, 0, 0, 0, time.UTC),
	}
}
