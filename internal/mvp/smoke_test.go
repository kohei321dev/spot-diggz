package mvp_test

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/kohei321dev/spot-diggz/internal/facility"
	"github.com/kohei321dev/spot-diggz/internal/httpapi"
	"github.com/kohei321dev/spot-diggz/internal/recommendation"
	"github.com/kohei321dev/spot-diggz/internal/session"
)

const smokeRequestTimeout = 3 * time.Second

func TestRunnableMVPFlow(t *testing.T) {
	t.Parallel()

	now := func() time.Time {
		return time.Date(2026, time.July, 19, 12, 0, 0, 0, time.FixedZone("JST", 9*60*60))
	}
	catalogPath := filepath.Join("..", "..", "testdata", "facilities.dev.json")
	catalog, err := facility.LoadCatalogFileAt(catalogPath, now())
	if err != nil {
		t.Fatalf("LoadCatalogFile() error = %v", err)
	}

	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	server := httptest.NewServer(httpapi.NewServerWithOptions(catalog, logger, httpapi.Options{Now: now}))
	t.Cleanup(server.Close)
	client := &http.Client{Timeout: smokeRequestTimeout}

	assertWebUI(t, client, server.URL)
	assertRecommendationFlow(t, client, server.URL)
}

func TestProductionCatalogReturnsOsakaRecommendation(t *testing.T) {
	t.Parallel()

	catalogPath := filepath.Join("..", "..", "data", "facilities.json")
	catalog, err := facility.LoadCatalogFile(catalogPath)
	if err != nil {
		t.Fatalf("LoadCatalogFile() error = %v", err)
	}
	latitude := 34.7025
	longitude := 135.4960
	now := func() time.Time {
		return time.Date(2026, time.July, 20, 13, 0, 0, 0, time.FixedZone("JST", 9*60*60))
	}
	result, err := recommendation.NewEngine(catalog, now).Recommend(session.Input{
		Purpose:          session.PurposeBasics,
		Mood:             session.MoodFocused,
		Level:            session.LevelBeginner,
		AvailableMinutes: 120,
		Transport:        session.TransportPublicTransit,
		Origin: session.Origin{
			Mode:      session.OriginSpecifiedLocation,
			Latitude:  &latitude,
			Longitude: &longitude,
		},
	})
	if err != nil {
		t.Fatalf("Recommend() error = %v", err)
	}
	if len(result.Recommendations) == 0 {
		t.Fatal("Recommend() returned no production facilities")
	}
	for _, item := range result.Recommendations {
		if item.Facility.SourceType == "test_fixture" || strings.Contains(strings.ToUpper(item.Facility.Name), "DUMMY") {
			t.Fatalf("Recommend() returned test facility %s", item.Facility.ID)
		}
		if len(item.Facility.ScheduleNotes) == 0 {
			t.Fatalf("Recommend() returned facility %s without schedule notes", item.Facility.ID)
		}
	}
}

func TestProductionCatalogReturnsRecommendationInEverySupportedPrefecture(t *testing.T) {
	t.Parallel()

	catalogPath := filepath.Join("..", "..", "data", "facilities.json")
	catalog, err := facility.LoadCatalogFile(catalogPath)
	if err != nil {
		t.Fatalf("LoadCatalogFile() error = %v", err)
	}
	now := func() time.Time {
		return time.Date(2026, time.July, 20, 13, 0, 0, 0, time.FixedZone("JST", 9*60*60))
	}
	tests := []struct {
		name       string
		prefecture string
		latitude   float64
		longitude  float64
	}{
		{name: "Osaka", prefecture: "大阪府", latitude: 34.6724, longitude: 135.4966},
		{name: "Hyogo", prefecture: "兵庫県", latitude: 34.7960, longitude: 134.9906},
		{name: "Wakayama", prefecture: "和歌山県", latitude: 33.8616, longitude: 135.1693},
		{name: "Nara", prefecture: "奈良県", latitude: 34.6990, longitude: 135.8274},
		{name: "Tokushima", prefecture: "徳島県", latitude: 34.0655, longitude: 134.5980},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			latitude := test.latitude
			longitude := test.longitude
			result, err := recommendation.NewEngine(catalog, now).Recommend(session.Input{
				Purpose:          session.PurposeBasics,
				Mood:             session.MoodFocused,
				Level:            session.LevelReturning,
				AvailableMinutes: 120,
				Transport:        session.TransportCar,
				Origin: session.Origin{
					Mode:      session.OriginSpecifiedLocation,
					Latitude:  &latitude,
					Longitude: &longitude,
				},
			})
			if err != nil {
				t.Fatalf("Recommend() error = %v", err)
			}
			if len(result.Recommendations) == 0 {
				t.Fatal("Recommend() returned no facilities")
			}
			foundPrefecture := false
			for _, item := range result.Recommendations {
				if item.Facility.Prefecture == test.prefecture {
					foundPrefecture = true
					break
				}
			}
			if !foundPrefecture {
				t.Fatalf("recommendations do not contain %s: %#v", test.prefecture, result.Recommendations)
			}
		})
	}
}

func assertWebUI(t *testing.T, client *http.Client, baseURL string) {
	t.Helper()

	response, err := client.Get(baseURL + "/")
	if err != nil {
		t.Fatalf("GET / error = %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		t.Fatalf("GET / status = %d, want %d", response.StatusCode, http.StatusOK)
	}
	page, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("read UI response: %v", err)
	}
	for _, marker := range []string{
		`id="quick-search-button"`,
		`class="mood-action"`,
		`id="condition-details"`,
		`id="recommendation-form"`,
		`name="originMode"`,
		`id="purpose"`,
		`id="mood"`,
		`id="level"`,
		`id="available-minutes"`,
		`id="transport"`,
		`id="availability-notice"`,
	} {
		if !bytes.Contains(page, []byte(marker)) {
			t.Fatalf("UI does not contain %s", marker)
		}
	}

	assetResponse, err := client.Get(baseURL + "/assets/app.js")
	if err != nil {
		t.Fatalf("GET /assets/app.js error = %v", err)
	}
	defer assetResponse.Body.Close()
	asset, err := io.ReadAll(assetResponse.Body)
	if err != nil {
		t.Fatalf("read JavaScript response: %v", err)
	}
	if assetResponse.StatusCode != http.StatusOK {
		t.Fatalf("GET /assets/app.js status = %d, want %d", assetResponse.StatusCode, http.StatusOK)
	}
	for _, marker := range []string{
		"/api/recommendations",
		"https://www.google.com/maps/dir/",
		"window.localStorage",
		"createPrimaryRecommendation",
	} {
		if !bytes.Contains(asset, []byte(marker)) {
			t.Fatalf("JavaScript asset does not contain %q", marker)
		}
	}
}

func assertRecommendationFlow(t *testing.T, client *http.Client, baseURL string) {
	t.Helper()

	latitude := 34.7025
	longitude := 135.4960
	input := session.Input{
		Purpose:          session.PurposeBasics,
		Mood:             session.MoodFocused,
		Level:            session.LevelBeginner,
		AvailableMinutes: 120,
		Transport:        session.TransportPublicTransit,
		Origin: session.Origin{
			Mode:      session.OriginSpecifiedLocation,
			Latitude:  &latitude,
			Longitude: &longitude,
		},
	}
	body, err := json.Marshal(input)
	if err != nil {
		t.Fatalf("marshal recommendation input: %v", err)
	}

	request, err := http.NewRequest(http.MethodPost, baseURL+"/api/recommendations", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("NewRequest() error = %v", err)
	}
	request.Header.Set("Content-Type", "application/json")
	response, err := client.Do(request)
	if err != nil {
		t.Fatalf("POST /api/recommendations error = %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		errorBody, _ := io.ReadAll(response.Body)
		t.Fatalf("POST /api/recommendations status = %d, want %d; body = %s", response.StatusCode, http.StatusOK, errorBody)
	}
	responseBody, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("read recommendation response: %v", err)
	}
	if strings.Contains(string(responseBody), "34.7025") || strings.Contains(string(responseBody), "135.496") {
		t.Fatal("recommendation response contains origin coordinates")
	}

	var result recommendation.Response
	if err := json.Unmarshal(responseBody, &result); err != nil {
		t.Fatalf("decode recommendation response: %v", err)
	}
	if len(result.Recommendations) == 0 || len(result.Recommendations) > 3 {
		t.Fatalf("recommendation count = %d, want 1 to 3", len(result.Recommendations))
	}
	first := result.Recommendations[0]
	if first.Facility.ID != "DEV-F001" {
		t.Fatalf("first facility = %q, want DEV-F001", first.Facility.ID)
	}
	if len(first.Reasons) == 0 || first.Facility.SourceURL == "" || first.Facility.VerifiedAt.IsZero() {
		t.Fatal("recommendation is missing reasons, source URL, or verification date")
	}
	if first.EstimatedSkateMinutes <= 0 || first.ArrivalAt.IsZero() || first.FacilityClosesAt.IsZero() || first.SessionEndsAt.IsZero() {
		t.Fatal("recommendation is missing session timing")
	}
	if result.TravelEstimateNote == "" {
		t.Fatal("travel estimate notice is empty")
	}
	if result.AvailabilityNote == "" {
		t.Fatal("availability notice is empty")
	}
}
