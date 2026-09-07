package httpapi

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/kohei321dev/spot-diggz/internal/apiauth"
	"github.com/kohei321dev/spot-diggz/internal/facility"
	"github.com/kohei321dev/spot-diggz/internal/geocoding"
	"github.com/kohei321dev/spot-diggz/internal/nearby"
)

func readAPIFixture(t *testing.T, provider geocoding.Provider, logs *bytes.Buffer, changes ...func(*facility.Facility)) (http.Handler, string) {
	t.Helper()
	spot := testFacility()
	spot.Genre = facility.GenreSkatepark
	spot.SkatingPermissionSourceURL = "https://example.com/permission"
	for _, change := range changes {
		change(&spot)
	}
	catalog, err := facility.NewCatalog([]facility.Facility{spot})
	if err != nil {
		t.Fatal(err)
	}
	seed := sha256.Sum256([]byte(t.Name()))
	token := hex.EncodeToString(seed[:])
	digest := sha256.Sum256([]byte(token))
	raw, _ := json.Marshal([]apiauth.Credential{{ClientID: "test-bot", OwnerID: "test-owner", Scope: apiauth.ReadScope, ExpiresAt: fixedServerTime().Add(time.Hour), TokenSHA256: hex.EncodeToString(digest[:])}})
	credentials, err := apiauth.Parse(string(raw), "test-owner", fixedServerTime)
	if err != nil {
		t.Fatal(err)
	}
	return NewReadAPI(catalog, credentials, provider, slog.New(slog.NewJSONHandler(logs, nil)), nil, fixedServerTime), token
}

func TestReadAPIAuthenticatesBeforeProviderAndDoesNotExposeLegacyRoutes(t *testing.T) {
	provider := &stubGeocoder{results: []geocoding.Result{{Label: "Test station", Location: testFacility().Location}}}
	var logs bytes.Buffer
	handler, token := readAPIFixture(t, provider, &logs)
	for _, path := range []string{"/api/facilities/search", "/api/facilities/facility-a"} {
		method := http.MethodGet
		if strings.HasSuffix(path, "search") {
			method = http.MethodPost
		}
		for _, header := range []string{"", "Bearer invalid", "Basic " + token} {
			r := httptest.NewRequest(method, path, strings.NewReader(`{"query":"private-input"}`))
			r.Header.Set("Authorization", header)
			r.Header.Set("Content-Type", "application/json")
			r.Header.Set("Cookie", "spotdiggz_session=not-api-auth")
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, r)
			if w.Code != http.StatusUnauthorized || w.Header().Get("WWW-Authenticate") == "" {
				t.Fatalf("auth status %d", w.Code)
			}
		}
	}
	if provider.calls != 0 {
		t.Fatal("unauthorized request reached provider")
	}
	for _, path := range []string{"/", "/metrics", "/auth/github/login", "/api/facilities", "/api/recommendations", "/api/events", "/api/corrections", "/integrations/slack/commands", "/integrations/discord/interactions"} {
		r := httptest.NewRequest(http.MethodGet, path, nil)
		r.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		if w.Code != http.StatusNotFound {
			t.Fatalf("legacy route %s returned %d", path, w.Code)
		}
	}
	if strings.Contains(logs.String(), token) || strings.Contains(logs.String(), "private-input") {
		t.Fatal("sensitive request logged")
	}
}

func TestReadAPISearchHTTPContractAndPrivacy(t *testing.T) {
	provider := &stubGeocoder{results: []geocoding.Result{{Label: "Private location label", Location: testFacility().Location}}}
	var logs bytes.Buffer
	handler, token := readAPIFixture(t, provider, &logs)
	r := httptest.NewRequest(http.MethodPost, "/api/facilities/search", strings.NewReader(`{"query":"Private query","limit":1,"radiusKm":2}`))
	r.Header.Set("Authorization", "Bearer "+token)
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("search status %d", w.Code)
	}
	var response nearby.Response
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response.Status != nearby.StatusOK || len(response.Matches) != 1 || response.Limit != 1 || response.RadiusKm != 2 {
		t.Fatal("search contract mismatch")
	}
	if w.Header().Get("X-Request-ID") == "" || w.Header().Get("Cache-Control") != "no-store" {
		t.Fatal("missing safety headers")
	}
	for _, private := range []string{token, "Private query", "Private location label"} {
		if strings.Contains(w.Body.String(), private) || strings.Contains(logs.String(), private) {
			t.Fatal("private request echoed")
		}
	}
	r = httptest.NewRequest(http.MethodGet, "/api/facilities/facility-a", nil)
	r.Header.Set("Authorization", "Bearer "+token)
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Fatal("authenticated detail unavailable")
	}
}

func TestReadAPINonTimetabledStreetSearchDetailAndReadiness(t *testing.T) {
	for _, test := range []struct {
		status                 facility.HoursStatus
		wantStatus             string
		wantMatches, wantReady int
	}{
		{facility.HoursNotApplicable, nearby.StatusOK, 1, http.StatusOK},
		{facility.HoursUnknown, nearby.StatusDataUnavailable, 0, http.StatusServiceUnavailable},
	} {
		t.Run(string(test.status), func(t *testing.T) {
			var logs bytes.Buffer
			location := facility.Location{Latitude: 35.0116, Longitude: 135.7681}
			provider := &stubGeocoder{results: []geocoding.Result{{Label: "Test place", Location: location}}}
			handler, token := readAPIFixture(t, provider, &logs, func(s *facility.Facility) {
				s.Prefecture, s.Municipality = "京都府", "京都市"
				s.Address, s.EnglishTranslation.Address = "京都府京都市のテスト住所", "Test address, Kyoto"
				s.Location, s.Genre, s.HoursStatus = location, facility.GenreStreet, test.status
				s.Hours, s.HoursBasis = nil, ""
				s.HoursSourceURL = "https://example.com/hours-policy"
				s.GeneralUseStatus = facility.GeneralUseLimited
				s.AvailabilityNote = "営業時間制の対象外です。利用ルールを確認してください。"
				s.EnglishTranslation.AvailabilityNote = "No opening-hours system applies. Check the rules."
				if test.status == facility.HoursUnknown {
					s.GeneralUseStatus = facility.GeneralUseScheduleCheckRequired
					s.AvailabilityNote = "利用時間は未確認です。予定の確認が必要です。"
					s.EnglishTranslation.AvailabilityNote = "Hours are unknown. Check the schedule."
				}
			})
			request := httptest.NewRequest(http.MethodPost, "/api/facilities/search", strings.NewReader(`{"query":"Test place","genre":"street"}`))
			request.Header.Set("Authorization", "Bearer "+token)
			request.Header.Set("Content-Type", "application/json")
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			var result nearby.Response
			if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
				t.Fatal(err)
			}
			if response.Code != http.StatusOK || result.Status != test.wantStatus || len(result.Matches) != test.wantMatches {
				t.Fatalf("search status=%d result=%s matches=%d", response.Code, result.Status, len(result.Matches))
			}
			if len(result.Matches) == 1 && (result.Matches[0].Facility.HoursStatus != test.status || result.Matches[0].Facility.HoursSourceURL == "" || result.Matches[0].Facility.AvailabilityNote == "") {
				t.Fatal("search lost hours evidence")
			}
			request = httptest.NewRequest(http.MethodGet, "/api/facilities/facility-a", nil)
			request.Header.Set("Authorization", "Bearer "+token)
			response = httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != http.StatusOK {
				t.Fatalf("detail status=%d", response.Code)
			}
			var detail facility.Facility
			if err := json.Unmarshal(response.Body.Bytes(), &detail); err != nil {
				t.Fatal(err)
			}
			if detail.HoursStatus != test.status || detail.AvailabilityNote == "" || detail.EnglishTranslation.AvailabilityNote == "" {
				t.Fatal("detail lost hours state or explanation")
			}
			if strings.Contains(response.Body.String(), `"hours":`) || strings.Contains(response.Body.String(), `"hoursBasis":`) {
				t.Fatal("detail fabricated timetable")
			}
			response = httptest.NewRecorder()
			handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/readyz", nil))
			if response.Code != test.wantReady {
				t.Fatalf("ready=%d want=%d", response.Code, test.wantReady)
			}
		})
	}
}

func TestReadAPIIndependentHTTPClientNeedsOnlyBearerAndJSON(t *testing.T) {
	var logs bytes.Buffer
	provider := &stubGeocoder{results: []geocoding.Result{{Label: "Test station", Location: testFacility().Location}}}
	handler, token := readAPIFixture(t, provider, &logs)
	server := httptest.NewServer(handler)
	defer server.Close()
	request, err := http.NewRequest(http.MethodPost, server.URL+"/api/facilities/search", strings.NewReader(`{"query":"Test station"}`))
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Authorization", "Bearer "+token)
	request.Header.Set("Content-Type", "application/json")
	response, err := server.Client().Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	var result nearby.Response
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != http.StatusOK || result.Status != nearby.StatusOK || len(result.Matches) != 1 {
		t.Fatal("independent HTTP client could not search")
	}
}

func TestReadAPIRejectsDuplicateAuthorizationHeaders(t *testing.T) {
	var logs bytes.Buffer
	provider := &stubGeocoder{}
	handler, token := readAPIFixture(t, provider, &logs)
	request := httptest.NewRequest(http.MethodPost, "/api/facilities/search", strings.NewReader(`{"query":"test"}`))
	request.Header.Add("Authorization", "Bearer "+token)
	request.Header.Add("Authorization", "Bearer "+token)
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusUnauthorized || provider.calls != 0 {
		t.Fatal("duplicate authorization accepted")
	}
}

func TestReadAPIRejectsInvalidRequestsAndProviderFailure(t *testing.T) {
	for _, test := range []struct {
		name, body, contentType, path string
		status                        int
	}{
		{"missing query", `{}`, "application/json", "/api/facilities/search", 400},
		{"duplicate key", `{"query":"a","query":"b"}`, "application/json", "/api/facilities/search", 400},
		{"null", `null`, "application/json", "/api/facilities/search", 400},
		{"wrong media", `{"query":"a"}`, "text/plain", "/api/facilities/search", 415},
		{"oversized", `{"query":"` + strings.Repeat("a", maxSearchRequestBytes) + `"}`, "application/json", "/api/facilities/search", 413},
		{"URL input", `{"query":"a"}`, "application/json", "/api/facilities/search?query=private-query", 400},
		{"provider failure", `{"query":"a"}`, "application/json", "/api/facilities/search", 503},
	} {
		t.Run(test.name, func(t *testing.T) {
			var logs bytes.Buffer
			provider := &stubGeocoder{err: errors.New("private provider details")}
			handler, token := readAPIFixture(t, provider, &logs)
			r := httptest.NewRequest(http.MethodPost, test.path, strings.NewReader(test.body))
			r.Header.Set("Authorization", "Bearer "+token)
			r.Header.Set("Content-Type", test.contentType)
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, r)
			if w.Code != test.status {
				t.Fatalf("status=%d want=%d", w.Code, test.status)
			}
			if test.status != 503 && provider.calls != 0 {
				t.Fatal("invalid request reached provider")
			}
			if strings.Contains(logs.String(), "private-query") || strings.Contains(w.Body.String(), "private provider") {
				t.Fatal("private diagnostics exposed")
			}
		})
	}
}

func TestReadAPIRateLimitAndReadiness(t *testing.T) {
	var logs bytes.Buffer
	handler, token := readAPIFixture(t, &stubGeocoder{}, &logs)
	for i := 0; i <= locationSearchRequestBurst; i++ {
		r := httptest.NewRequest(http.MethodPost, "/api/facilities/search", strings.NewReader(`{"query":"test"}`))
		r.Header.Set("Authorization", "Bearer "+token)
		r.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		if i == locationSearchRequestBurst && (w.Code != 429 || w.Header().Get("Retry-After") != "60") {
			t.Fatal("rate limit not enforced")
		}
	}
	for _, path := range []string{"/healthz", "/readyz"} {
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, httptest.NewRequest(http.MethodGet, path, nil))
		if w.Code != 200 {
			t.Fatalf("%s not healthy", path)
		}
	}
	missing := NewReadAPI(nil, nil, nil, slog.New(slog.NewJSONHandler(&logs, nil)), nil, fixedServerTime)
	for _, path := range []string{"/readyz", "/api/facilities/facility-a"} {
		w := httptest.NewRecorder()
		missing.ServeHTTP(w, httptest.NewRequest(http.MethodGet, path, nil))
		if w.Code != 503 {
			t.Fatal("missing config not fail-closed")
		}
	}
}
