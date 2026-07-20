package travel

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/kohei321dev/spot-diggz/internal/facility"
	"github.com/kohei321dev/spot-diggz/internal/session"
)

const (
	GoogleRoutesKind       = "google_routes"
	googleRoutesEndpoint   = "https://routes.googleapis.com/distanceMatrix/v2:computeRouteMatrix"
	googleResponseMaxBytes = 2 * 1024 * 1024
	googleRequestTimeout   = 4 * time.Second
)

type GoogleRoutesProvider struct {
	apiKey   string
	endpoint string
	client   *http.Client
}

func NewGoogleRoutesProvider(apiKey string, client *http.Client) (*GoogleRoutesProvider, error) {
	return newGoogleRoutesProvider(apiKey, googleRoutesEndpoint, client)
}

func newGoogleRoutesProvider(apiKey string, endpoint string, client *http.Client) (*GoogleRoutesProvider, error) {
	if strings.TrimSpace(apiKey) == "" {
		return nil, fmt.Errorf("%w: Google Routes API key is required", ErrInvalidRequest)
	}
	parsedEndpoint, err := url.Parse(endpoint)
	if err != nil || parsedEndpoint.Scheme == "" || parsedEndpoint.Host == "" {
		return nil, fmt.Errorf("%w: invalid Google Routes endpoint", ErrInvalidRequest)
	}
	if client == nil {
		client = &http.Client{Timeout: googleRequestTimeout}
	}
	return &GoogleRoutesProvider{apiKey: apiKey, endpoint: endpoint, client: client}, nil
}

func (provider *GoogleRoutesProvider) Matrix(ctx context.Context, request Request) ([]Estimate, error) {
	if err := validateRequest(request); err != nil {
		return nil, err
	}

	payload := googleMatrixRequest{
		Origins:      []googleRouteMatrixOrigin{{Waypoint: googleWaypointFor(request.Origin)}},
		Destinations: make([]googleRouteMatrixDestination, 0, len(request.Destinations)),
		TravelMode:   googleTravelMode(request.Transport),
		LanguageCode: "ja",
		RegionCode:   "JP",
		Units:        "METRIC",
	}
	if request.Transport == session.TransportCar {
		payload.RoutingPreference = "TRAFFIC_AWARE"
	}
	for _, destination := range request.Destinations {
		payload.Destinations = append(payload.Destinations, googleRouteMatrixDestination{
			Waypoint: googleWaypointFor(destination.Location),
		})
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("encode Google Routes request: %w", err)
	}
	requestContext, cancel := context.WithTimeout(ctx, googleRequestTimeout)
	defer cancel()
	httpRequest, err := http.NewRequestWithContext(requestContext, http.MethodPost, provider.endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create Google Routes request: %w", err)
	}
	httpRequest.Header.Set("Content-Type", "application/json")
	httpRequest.Header.Set("X-Goog-Api-Key", provider.apiKey)
	httpRequest.Header.Set("X-Goog-FieldMask", "originIndex,destinationIndex,duration,distanceMeters,status,condition")

	response, err := provider.client.Do(httpRequest)
	if err != nil {
		return nil, fmt.Errorf("%w: Google Routes request: %v", ErrProviderUnavailable, err)
	}
	defer response.Body.Close()
	limitedBody := io.LimitReader(response.Body, googleResponseMaxBytes+1)
	responseBody, err := io.ReadAll(limitedBody)
	if err != nil {
		return nil, fmt.Errorf("%w: read Google Routes response", ErrProviderUnavailable)
	}
	if len(responseBody) > googleResponseMaxBytes {
		return nil, fmt.Errorf("%w: Google Routes response is too large", ErrProviderUnavailable)
	}
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: Google Routes returned HTTP %d", ErrProviderUnavailable, response.StatusCode)
	}

	var elements []googleMatrixElement
	if err := json.Unmarshal(responseBody, &elements); err != nil {
		return nil, fmt.Errorf("%w: decode Google Routes response", ErrProviderUnavailable)
	}
	return mapGoogleElements(request.Destinations, elements)
}

func mapGoogleElements(destinations []Destination, elements []googleMatrixElement) ([]Estimate, error) {
	if len(elements) != len(destinations) {
		return nil, fmt.Errorf("%w: Google Routes returned %d of %d matrix elements", ErrProviderUnavailable, len(elements), len(destinations))
	}
	estimates := make([]Estimate, len(destinations))
	seen := make([]bool, len(destinations))
	for _, element := range elements {
		if element.OriginIndex != 0 || element.DestinationIndex < 0 || element.DestinationIndex >= len(destinations) {
			return nil, fmt.Errorf("%w: Google Routes returned an invalid matrix index", ErrProviderUnavailable)
		}
		if seen[element.DestinationIndex] {
			return nil, fmt.Errorf("%w: Google Routes returned a duplicate matrix index", ErrProviderUnavailable)
		}
		if element.Condition != "ROUTE_EXISTS" || element.Status.Code != 0 {
			return nil, fmt.Errorf("%w: Google Routes did not return every route", ErrProviderUnavailable)
		}
		duration, err := time.ParseDuration(element.Duration)
		if err != nil || duration < 0 || element.DistanceMeters < 0 {
			return nil, fmt.Errorf("%w: Google Routes returned invalid distance or duration", ErrProviderUnavailable)
		}
		seen[element.DestinationIndex] = true
		destination := destinations[element.DestinationIndex]
		estimates[element.DestinationIndex] = Estimate{
			FacilityID:    destination.FacilityID,
			DistanceKm:    float64(element.DistanceMeters) / 1000,
			TravelMinutes: int(math.Ceil(duration.Minutes())),
			Kind:          GoogleRoutesKind,
		}
	}
	return estimates, nil
}

func googleTravelMode(transport session.Transport) string {
	switch transport {
	case session.TransportPublicTransit:
		return "TRANSIT"
	case session.TransportCar:
		return "DRIVE"
	case session.TransportBicycle:
		return "BICYCLE"
	case session.TransportWalk:
		return "WALK"
	default:
		return ""
	}
}

func googleWaypointFor(location facility.Location) googleWaypoint {
	return googleWaypoint{Location: googleLocation{LatLng: googleLatLng{
		Latitude:  location.Latitude,
		Longitude: location.Longitude,
	}}}
}

type googleMatrixRequest struct {
	Origins           []googleRouteMatrixOrigin      `json:"origins"`
	Destinations      []googleRouteMatrixDestination `json:"destinations"`
	TravelMode        string                         `json:"travelMode"`
	RoutingPreference string                         `json:"routingPreference,omitempty"`
	LanguageCode      string                         `json:"languageCode"`
	RegionCode        string                         `json:"regionCode"`
	Units             string                         `json:"units"`
}

type googleRouteMatrixOrigin struct {
	Waypoint googleWaypoint `json:"waypoint"`
}

type googleRouteMatrixDestination struct {
	Waypoint googleWaypoint `json:"waypoint"`
}

type googleWaypoint struct {
	Location googleLocation `json:"location"`
}

type googleLocation struct {
	LatLng googleLatLng `json:"latLng"`
}

type googleLatLng struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

type googleMatrixElement struct {
	OriginIndex      int          `json:"originIndex"`
	DestinationIndex int          `json:"destinationIndex"`
	Duration         string       `json:"duration"`
	DistanceMeters   int          `json:"distanceMeters"`
	Status           googleStatus `json:"status"`
	Condition        string       `json:"condition"`
}

type googleStatus struct {
	Code int `json:"code"`
}
