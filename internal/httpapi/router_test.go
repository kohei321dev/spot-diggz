package httpapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/kohei321dev/spot-diggz/internal/spot"
)

func TestSpotLifecycleHTTP(t *testing.T) {
	router := NewSdzRouter(spot.NewSdzMemoryStore())

	createBody := []byte(`{
		"name": "Tokyo Demo Ledge",
		"description": "short ledge near the station",
		"location": {"lat": 35.6812, "lng": 139.7671},
		"tags": ["ledge", "street"],
		"visibility": "public"
	}`)
	createRequest := httptest.NewRequest(http.MethodPost, "/sdz/spots", bytes.NewReader(createBody))
	createRecorder := httptest.NewRecorder()
	router.ServeHTTP(createRecorder, createRequest)
	if createRecorder.Code != http.StatusCreated {
		t.Fatalf("POST /sdz/spots status = %d, body = %s", createRecorder.Code, createRecorder.Body.String())
	}

	var created spot.SdzSpot
	if err := json.Unmarshal(createRecorder.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode created spot: %v", err)
	}
	if created.SdzSpotID == "" {
		t.Fatal("created spot id is empty")
	}

	listRequest := httptest.NewRequest(http.MethodGet, "/sdz/spots?bbox=139,35,140,36&tags=ledge", nil)
	listRecorder := httptest.NewRecorder()
	router.ServeHTTP(listRecorder, listRequest)
	if listRecorder.Code != http.StatusOK {
		t.Fatalf("GET /sdz/spots status = %d, body = %s", listRecorder.Code, listRecorder.Body.String())
	}

	var listed []spot.SdzSpot
	if err := json.Unmarshal(listRecorder.Body.Bytes(), &listed); err != nil {
		t.Fatalf("decode listed spots: %v", err)
	}
	if len(listed) != 1 {
		t.Fatalf("listed spots = %d, want 1", len(listed))
	}

	geoJSONRequest := httptest.NewRequest(http.MethodGet, "/sdz/spots.geojson?bbox=139,35,140,36", nil)
	geoJSONRecorder := httptest.NewRecorder()
	router.ServeHTTP(geoJSONRecorder, geoJSONRequest)
	if geoJSONRecorder.Code != http.StatusOK {
		t.Fatalf("GET /sdz/spots.geojson status = %d", geoJSONRecorder.Code)
	}

	var geoJSON struct {
		Type     string `json:"type"`
		Features []struct {
			Geometry struct {
				Coordinates [2]float64 `json:"coordinates"`
			} `json:"geometry"`
		} `json:"features"`
	}
	if err := json.Unmarshal(geoJSONRecorder.Body.Bytes(), &geoJSON); err != nil {
		t.Fatalf("decode GeoJSON: %v", err)
	}
	if geoJSON.Type != "FeatureCollection" {
		t.Fatalf("GeoJSON type = %q, want FeatureCollection", geoJSON.Type)
	}
	if geoJSON.Features[0].Geometry.Coordinates != [2]float64{139.7671, 35.6812} {
		t.Fatalf("GeoJSON coordinates = %#v", geoJSON.Features[0].Geometry.Coordinates)
	}
}

func TestInvalidCreateReturnsBadRequest(t *testing.T) {
	router := NewSdzRouter(spot.NewSdzMemoryStore())

	request := httptest.NewRequest(http.MethodPost, "/sdz/spots", bytes.NewReader([]byte(`{"name": ""}`)))
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
}
