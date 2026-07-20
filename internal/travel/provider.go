package travel

import (
	"context"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/kohei321dev/spot-diggz/internal/facility"
	"github.com/kohei321dev/spot-diggz/internal/session"
)

const MaxDestinations = 100

var (
	ErrInvalidRequest      = errors.New("invalid travel request")
	ErrProviderUnavailable = errors.New("travel provider unavailable")
)

type Destination struct {
	FacilityID string
	Location   facility.Location
}

type Request struct {
	Origin       facility.Location
	Destinations []Destination
	Transport    session.Transport
	DepartureAt  time.Time
}

type Estimate struct {
	FacilityID    string
	DistanceKm    float64
	TravelMinutes int
	Kind          string
}

type Provider interface {
	Matrix(context.Context, Request) ([]Estimate, error)
}

type FallbackProvider struct {
	primary  Provider
	fallback Provider
}

func NewFallbackProvider(primary Provider, fallback Provider) (*FallbackProvider, error) {
	if primary == nil || fallback == nil {
		return nil, fmt.Errorf("%w: primary and fallback providers are required", ErrInvalidRequest)
	}
	return &FallbackProvider{primary: primary, fallback: fallback}, nil
}

func (provider *FallbackProvider) Matrix(ctx context.Context, request Request) ([]Estimate, error) {
	estimates, err := provider.primary.Matrix(ctx, request)
	if err == nil {
		return estimates, nil
	}

	fallbackEstimates, fallbackErr := provider.fallback.Matrix(ctx, request)
	if fallbackErr != nil {
		return nil, fmt.Errorf("primary provider: %v; fallback provider: %w", err, fallbackErr)
	}
	return fallbackEstimates, nil
}

func validateRequest(request Request) error {
	if !validLocation(request.Origin) {
		return fmt.Errorf("%w: origin coordinates", ErrInvalidRequest)
	}
	if len(request.Destinations) == 0 || len(request.Destinations) > MaxDestinations {
		return fmt.Errorf("%w: destinations must contain 1 to %d items", ErrInvalidRequest, MaxDestinations)
	}
	if !validTransport(request.Transport) {
		return fmt.Errorf("%w: transport", ErrInvalidRequest)
	}
	if request.DepartureAt.IsZero() {
		return fmt.Errorf("%w: departure time", ErrInvalidRequest)
	}

	seen := make(map[string]struct{}, len(request.Destinations))
	for _, destination := range request.Destinations {
		if destination.FacilityID == "" || !validLocation(destination.Location) {
			return fmt.Errorf("%w: destination", ErrInvalidRequest)
		}
		if _, exists := seen[destination.FacilityID]; exists {
			return fmt.Errorf("%w: duplicate destination %s", ErrInvalidRequest, destination.FacilityID)
		}
		seen[destination.FacilityID] = struct{}{}
	}
	return nil
}

func validLocation(location facility.Location) bool {
	return !math.IsNaN(location.Latitude) && !math.IsInf(location.Latitude, 0) &&
		!math.IsNaN(location.Longitude) && !math.IsInf(location.Longitude, 0) &&
		location.Latitude >= -90 && location.Latitude <= 90 &&
		location.Longitude >= -180 && location.Longitude <= 180
}

func validTransport(transport session.Transport) bool {
	switch transport {
	case session.TransportPublicTransit, session.TransportCar, session.TransportBicycle, session.TransportWalk:
		return true
	default:
		return false
	}
}
