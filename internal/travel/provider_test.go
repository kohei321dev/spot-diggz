package travel

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/kohei321dev/spot-diggz/internal/facility"
	"github.com/kohei321dev/spot-diggz/internal/session"
)

func TestStraightLineProviderReturnsEveryDestination(t *testing.T) {
	request := validRequest()
	estimates, err := NewStraightLineProvider().Matrix(context.Background(), request)
	if err != nil {
		t.Fatalf("Matrix() error = %v", err)
	}
	if len(estimates) != len(request.Destinations) {
		t.Fatalf("estimate count = %d, want %d", len(estimates), len(request.Destinations))
	}
	if estimates[0].Kind != StraightLineKind || estimates[0].TravelMinutes <= 0 || estimates[0].DistanceKm <= 0 {
		t.Fatalf("estimate = %#v", estimates[0])
	}
}

func TestGoogleRoutesProviderMapsMatrixResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Goog-Api-Key") != "test-key" {
			t.Errorf("API key header = %q", r.Header.Get("X-Goog-Api-Key"))
		}
		if !strings.Contains(r.Header.Get("X-Goog-FieldMask"), "duration") {
			t.Errorf("field mask = %q", r.Header.Get("X-Goog-FieldMask"))
		}
		var request googleMatrixRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if request.TravelMode != "TRANSIT" || len(request.Destinations) != 2 {
			t.Fatalf("request = %#v", request)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[
			{"originIndex":0,"destinationIndex":1,"duration":"901s","distanceMeters":7500,"status":{},"condition":"ROUTE_EXISTS"},
			{"originIndex":0,"destinationIndex":0,"duration":"600s","distanceMeters":4200,"status":{},"condition":"ROUTE_EXISTS"}
		]`))
	}))
	defer server.Close()

	provider, err := newGoogleRoutesProvider("test-key", server.URL, server.Client())
	if err != nil {
		t.Fatalf("newGoogleRoutesProvider() error = %v", err)
	}
	estimates, err := provider.Matrix(context.Background(), validRequest())
	if err != nil {
		t.Fatalf("Matrix() error = %v", err)
	}
	if estimates[0].FacilityID != "facility-a" || estimates[0].TravelMinutes != 10 || estimates[0].DistanceKm != 4.2 {
		t.Fatalf("first estimate = %#v", estimates[0])
	}
	if estimates[1].FacilityID != "facility-b" || estimates[1].TravelMinutes != 16 || estimates[1].Kind != GoogleRoutesKind {
		t.Fatalf("second estimate = %#v", estimates[1])
	}
}

func TestGoogleRoutesProviderOmitsDepartureTimeForImmediateRequests(t *testing.T) {
	tests := []struct {
		transport session.Transport
		wantMode  string
	}{
		{transport: session.TransportPublicTransit, wantMode: "TRANSIT"},
		{transport: session.TransportCar, wantMode: "DRIVE"},
		{transport: session.TransportBicycle, wantMode: "BICYCLE"},
		{transport: session.TransportWalk, wantMode: "WALK"},
	}

	for _, test := range tests {
		t.Run(string(test.transport), func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var payload map[string]any
				if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
					t.Fatalf("decode request: %v", err)
				}
				if _, exists := payload["departureTime"]; exists {
					t.Fatalf("request includes departureTime: %#v", payload)
				}
				if got := payload["travelMode"]; got != test.wantMode {
					t.Fatalf("travelMode = %#v, want %q", got, test.wantMode)
				}
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`[
					{"originIndex":0,"destinationIndex":0,"duration":"600s","distanceMeters":4200,"status":{},"condition":"ROUTE_EXISTS"},
					{"originIndex":0,"destinationIndex":1,"duration":"900s","distanceMeters":7500,"status":{},"condition":"ROUTE_EXISTS"}
				]`))
			}))
			defer server.Close()

			provider, err := newGoogleRoutesProvider("test-key", server.URL, server.Client())
			if err != nil {
				t.Fatalf("newGoogleRoutesProvider() error = %v", err)
			}
			request := validRequest()
			request.Transport = test.transport
			if _, err := provider.Matrix(context.Background(), request); err != nil {
				t.Fatalf("Matrix() error = %v", err)
			}
		})
	}
}

func TestFallbackProviderUsesStraightLineWhenGoogleFails(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "unavailable", http.StatusServiceUnavailable)
	}))
	defer server.Close()
	primary, err := newGoogleRoutesProvider("test-key", server.URL, server.Client())
	if err != nil {
		t.Fatalf("newGoogleRoutesProvider() error = %v", err)
	}
	provider, err := NewFallbackProvider(primary, NewStraightLineProvider())
	if err != nil {
		t.Fatalf("NewFallbackProvider() error = %v", err)
	}

	estimates, err := provider.Matrix(context.Background(), validRequest())
	if err != nil {
		t.Fatalf("Matrix() error = %v", err)
	}
	if estimates[0].Kind != StraightLineKind {
		t.Fatalf("kind = %q, want %q", estimates[0].Kind, StraightLineKind)
	}
}

func TestProviderRejectsInvalidRequests(t *testing.T) {
	request := validRequest()
	request.Destinations[1].FacilityID = request.Destinations[0].FacilityID
	if _, err := NewStraightLineProvider().Matrix(context.Background(), request); err == nil {
		t.Fatal("Matrix() error = nil, want invalid request")
	}
}

func validRequest() Request {
	return Request{
		Origin:      facility.Location{Latitude: 34.7025, Longitude: 135.4960},
		Transport:   session.TransportPublicTransit,
		DepartureAt: time.Date(2026, time.July, 19, 9, 0, 0, 0, time.FixedZone("JST", 9*60*60)),
		Destinations: []Destination{
			{FacilityID: "facility-a", Location: facility.Location{Latitude: 34.68, Longitude: 135.50}},
			{FacilityID: "facility-b", Location: facility.Location{Latitude: 34.65, Longitude: 135.48}},
		},
	}
}
