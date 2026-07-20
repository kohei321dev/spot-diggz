package travel

import (
	"context"
	"math"
)

const (
	StraightLineKind             = "straight_line"
	earthRadiusKm                = 6371.0
	minutesPerHour               = 60.0
	publicTransitSpeedKmPerHour  = 22.0
	publicTransitOverheadMinutes = 12
	carSpeedKmPerHour            = 30.0
	carOverheadMinutes           = 8
	bicycleSpeedKmPerHour        = 13.0
	bicycleOverheadMinutes       = 3
	walkSpeedKmPerHour           = 4.5
)

type StraightLineProvider struct{}

func NewStraightLineProvider() *StraightLineProvider {
	return &StraightLineProvider{}
}

func (provider *StraightLineProvider) Matrix(_ context.Context, request Request) ([]Estimate, error) {
	if err := validateRequest(request); err != nil {
		return nil, err
	}

	profile := profileFor(request)
	estimates := make([]Estimate, 0, len(request.Destinations))
	for _, destination := range request.Destinations {
		distanceKm := haversineKm(
			request.Origin.Latitude,
			request.Origin.Longitude,
			destination.Location.Latitude,
			destination.Location.Longitude,
		)
		estimates = append(estimates, Estimate{
			FacilityID:    destination.FacilityID,
			DistanceKm:    distanceKm,
			TravelMinutes: int(math.Ceil(distanceKm/profile.speedKmPerHour*minutesPerHour)) + profile.overheadMinutes,
			Kind:          StraightLineKind,
		})
	}
	return estimates, nil
}

type travelProfile struct {
	speedKmPerHour  float64
	overheadMinutes int
}

func profileFor(request Request) travelProfile {
	switch request.Transport {
	case "public_transit":
		return travelProfile{speedKmPerHour: publicTransitSpeedKmPerHour, overheadMinutes: publicTransitOverheadMinutes}
	case "car":
		return travelProfile{speedKmPerHour: carSpeedKmPerHour, overheadMinutes: carOverheadMinutes}
	case "bicycle":
		return travelProfile{speedKmPerHour: bicycleSpeedKmPerHour, overheadMinutes: bicycleOverheadMinutes}
	case "walk":
		return travelProfile{speedKmPerHour: walkSpeedKmPerHour}
	default:
		return travelProfile{}
	}
}

func haversineKm(fromLatitude float64, fromLongitude float64, toLatitude float64, toLongitude float64) float64 {
	toRadians := func(degrees float64) float64 { return degrees * math.Pi / 180 }
	latitudeDelta := toRadians(toLatitude - fromLatitude)
	longitudeDelta := toRadians(toLongitude - fromLongitude)
	fromLatitudeRadians := toRadians(fromLatitude)
	toLatitudeRadians := toRadians(toLatitude)

	a := math.Sin(latitudeDelta/2)*math.Sin(latitudeDelta/2) +
		math.Cos(fromLatitudeRadians)*math.Cos(toLatitudeRadians)*
			math.Sin(longitudeDelta/2)*math.Sin(longitudeDelta/2)
	return earthRadiusKm * 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
}
