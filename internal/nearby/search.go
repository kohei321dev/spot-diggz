// Package nearby finds verified spots around a resolved place without planning a trip.
package nearby

import (
	"context"
	"errors"
	"math"
	"sort"
	"strings"
	"time"
	"unicode"

	"github.com/kohei321dev/spot-diggz/internal/facility"
	"github.com/kohei321dev/spot-diggz/internal/geocoding"
)

const (
	StatusOK                = "ok"
	StatusNoMatches         = "no_matches"
	StatusDataUnavailable   = "data_unavailable"
	StatusLocationNotFound  = "location_not_found"
	StatusLocationAmbiguous = "location_ambiguous"
	DistanceKind            = "straight_line"
	earthRadiusKm           = 6371.0
	searchTimeout           = 5 * time.Second
	maxPlaceLabelLength     = 300
)

var ErrUnavailable = errors.New("nearby location provider unavailable")

type Match struct {
	Facility   facility.Facility `json:"facility"`
	DistanceKm float64           `json:"distanceKm"`
}

type PlaceCandidate struct {
	Label string `json:"label"`
}

type Response struct {
	Status             string           `json:"status"`
	Matches            []Match          `json:"matches"`
	LocationCandidates []PlaceCandidate `json:"locationCandidates,omitempty"`
	DistanceKind       string           `json:"distanceKind"`
	Limit              int              `json:"limit"`
	RadiusKm           float64          `json:"radiusKm"`
	Sort               string           `json:"sort"`
	AvailabilityNote   string           `json:"availabilityNote"`
}

type Service struct {
	catalog  *facility.Catalog
	geocoder geocoding.Provider
	now      func() time.Time
}

func NewService(catalog *facility.Catalog, geocoder geocoding.Provider, now func() time.Time) *Service {
	if now == nil {
		now = time.Now
	}
	return &Service{catalog: catalog, geocoder: geocoder, now: now}
}

func (service *Service) Search(ctx context.Context, input Input) (Response, error) {
	if err := input.Validate(); err != nil {
		return Response{}, err
	}
	if service.catalog == nil || service.geocoder == nil {
		return Response{}, ErrUnavailable
	}
	ctx, cancel := context.WithTimeout(ctx, searchTimeout)
	defer cancel()
	places, err := service.geocoder.Search(ctx, input.Query)
	if err != nil || ctx.Err() != nil {
		return Response{}, ErrUnavailable
	}
	response := Response{
		Status: StatusLocationNotFound, Matches: []Match{}, DistanceKind: DistanceKind,
		Limit: input.Limit, RadiusKm: input.RadiusKm, Sort: input.Sort,
		AvailabilityNote: "周辺の登録情報です。現在の営業・滑走可否を保証しません。訪問前に出典と利用ルールを確認してください。",
	}
	if len(places) == 0 {
		return response, nil
	}
	// Validate the provider boundary instead of treating invalid data as no results.
	for _, place := range places {
		if !validPlace(place) {
			return Response{}, ErrUnavailable
		}
	}
	if len(places) > 1 {
		response.Status = StatusLocationAmbiguous
		for _, place := range places[:min(len(places), geocoding.MaxResults)] {
			response.LocationCandidates = append(response.LocationCandidates, PlaceCandidate{Label: place.Label})
		}
		return response, nil
	}
	center := places[0].Location
	nearbyRecords := 0
	insufficientEvidence := false
	now := service.now()
	for _, spot := range service.catalog.List("skateboard") {
		distance := distanceKm(center, spot.Location)
		if distance > input.RadiusKm {
			continue
		}
		nearbyRecords++
		if !isSearchable(spot, now) {
			insufficientEvidence = true
			continue
		}
		if input.Genre != "" && spot.Genre != input.Genre {
			continue
		}
		response.Matches = append(response.Matches, Match{Facility: spot, DistanceKm: distance})
	}
	sort.Slice(response.Matches, func(i, j int) bool {
		left, right := response.Matches[i], response.Matches[j]
		if left.DistanceKm != right.DistanceKm {
			return left.DistanceKm < right.DistanceKm
		}
		return left.Facility.ID < right.Facility.ID
	})
	if len(response.Matches) > input.Limit {
		response.Matches = response.Matches[:input.Limit]
	}
	response.Status = StatusOK
	if len(response.Matches) == 0 {
		response.Status = StatusNoMatches
		if nearbyRecords == 0 || insufficientEvidence {
			response.Status = StatusDataUnavailable
		}
	}
	return response, nil
}

func isSearchable(spot facility.Facility, now time.Time) bool {
	return (spot.Genre == facility.GenreSkatepark || spot.Genre == facility.GenreStreet) &&
		spot.SkatingPermissionSourceURL != "" && spot.Status == "verified" &&
		spot.GeneralUseStatus != facility.GeneralUseScheduleCheckRequired &&
		facility.IsDynamicInformationFresh(spot.DynamicVerifiedAt, now) &&
		facility.IsStableInformationFresh(spot.StableVerifiedAt, now)
}

func (service *Service) Ready() bool {
	if service == nil || service.catalog == nil || service.geocoder == nil {
		return false
	}
	for _, spot := range service.catalog.List("skateboard") {
		if isSearchable(spot, service.now()) {
			return true
		}
	}
	return false
}

func validPlace(place geocoding.Result) bool {
	return strings.TrimSpace(place.Label) != "" && len([]rune(place.Label)) <= maxPlaceLabelLength &&
		strings.IndexFunc(place.Label, unicode.IsControl) < 0 &&
		!math.IsNaN(place.Location.Latitude) && !math.IsNaN(place.Location.Longitude) &&
		place.Location.Latitude >= -90 && place.Location.Latitude <= 90 &&
		place.Location.Longitude >= -180 && place.Location.Longitude <= 180
}

func distanceKm(from, to facility.Location) float64 {
	latDelta := (to.Latitude - from.Latitude) * math.Pi / 180
	lonDelta := (to.Longitude - from.Longitude) * math.Pi / 180
	a := math.Pow(math.Sin(latDelta/2), 2) + math.Cos(from.Latitude*math.Pi/180)*math.Cos(to.Latitude*math.Pi/180)*math.Pow(math.Sin(lonDelta/2), 2)
	// Rounding can otherwise put antipodal values just outside the square-root domain.
	a = max(0, min(1, a))
	return 2 * earthRadiusKm * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
}
