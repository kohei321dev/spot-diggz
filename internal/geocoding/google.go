package geocoding

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/kohei321dev/spot-diggz/internal/facility"
)

const (
	googleGeocodingEndpoint = "https://maps.googleapis.com/maps/api/geocode/json"
	googleGeocodingTimeout  = 4 * time.Second
	googleGeocodingMaxBytes = 512 * 1024
	MaxQueryLength          = 120
	MaxResults              = 5
)

var (
	ErrInvalidQuery = errors.New("invalid geocoding query")
	ErrUnavailable  = errors.New("geocoding provider unavailable")
)

type Result struct {
	Label    string            `json:"label"`
	Location facility.Location `json:"location"`
}

type Provider interface {
	Search(context.Context, string) ([]Result, error)
}

type GoogleProvider struct {
	apiKey   string
	endpoint string
	client   *http.Client
}

func NewGoogleProvider(apiKey string, client *http.Client) (*GoogleProvider, error) {
	return newGoogleProvider(apiKey, googleGeocodingEndpoint, client)
}

func newGoogleProvider(apiKey string, endpoint string, client *http.Client) (*GoogleProvider, error) {
	if strings.TrimSpace(apiKey) == "" {
		return nil, fmt.Errorf("%w: Google Maps API key is required", ErrInvalidQuery)
	}
	parsedEndpoint, err := url.Parse(endpoint)
	if err != nil || parsedEndpoint.Scheme == "" || parsedEndpoint.Host == "" {
		return nil, fmt.Errorf("%w: invalid Google Geocoding endpoint", ErrInvalidQuery)
	}
	if client == nil {
		client = &http.Client{Timeout: googleGeocodingTimeout}
	}
	return &GoogleProvider{apiKey: apiKey, endpoint: endpoint, client: client}, nil
}

func (provider *GoogleProvider) Search(ctx context.Context, query string) ([]Result, error) {
	query = strings.TrimSpace(query)
	if query == "" || len([]rune(query)) > MaxQueryLength || strings.ContainsAny(query, "\r\n\x00") {
		return nil, ErrInvalidQuery
	}

	requestURL, err := url.Parse(provider.endpoint)
	if err != nil {
		return nil, fmt.Errorf("%w: build geocoding request", ErrUnavailable)
	}
	values := requestURL.Query()
	values.Set("address", query)
	values.Set("components", "country:JP")
	values.Set("language", "ja")
	values.Set("region", "jp")
	values.Set("key", provider.apiKey)
	requestURL.RawQuery = values.Encode()

	requestContext, cancel := context.WithTimeout(ctx, googleGeocodingTimeout)
	defer cancel()
	request, err := http.NewRequestWithContext(requestContext, http.MethodGet, requestURL.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("%w: create geocoding request", ErrUnavailable)
	}
	response, err := provider.client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("%w: Google Geocoding request", ErrUnavailable)
	}
	defer response.Body.Close()
	body, err := io.ReadAll(io.LimitReader(response.Body, googleGeocodingMaxBytes+1))
	if err != nil {
		return nil, fmt.Errorf("%w: read Google Geocoding response", ErrUnavailable)
	}
	if len(body) > googleGeocodingMaxBytes || response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: Google Geocoding HTTP response", ErrUnavailable)
	}

	var payload googleResponse
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("%w: decode Google Geocoding response", ErrUnavailable)
	}
	if payload.Status == "ZERO_RESULTS" {
		return []Result{}, nil
	}
	if payload.Status != "OK" {
		return nil, fmt.Errorf("%w: Google Geocoding status %s", ErrUnavailable, payload.Status)
	}

	limit := min(MaxResults, len(payload.Results))
	results := make([]Result, 0, limit)
	for _, item := range payload.Results[:limit] {
		location := facility.Location{
			Latitude:  item.Geometry.Location.Latitude,
			Longitude: item.Geometry.Location.Longitude,
		}
		if strings.TrimSpace(item.FormattedAddress) == "" || !validLocation(location) {
			continue
		}
		results = append(results, Result{Label: item.FormattedAddress, Location: location})
	}
	return results, nil
}

func validLocation(location facility.Location) bool {
	return location.Latitude >= -90 && location.Latitude <= 90 &&
		location.Longitude >= -180 && location.Longitude <= 180
}

type googleResponse struct {
	Status  string         `json:"status"`
	Results []googleResult `json:"results"`
}

type googleResult struct {
	FormattedAddress string         `json:"formatted_address"`
	Geometry         googleGeometry `json:"geometry"`
}

type googleGeometry struct {
	Location googleLocation `json:"location"`
}

type googleLocation struct {
	Latitude  float64 `json:"lat"`
	Longitude float64 `json:"lng"`
}
