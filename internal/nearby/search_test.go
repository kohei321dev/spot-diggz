package nearby

import (
	"context"
	"errors"
	"math"
	"testing"
	"time"

	"github.com/kohei321dev/spot-diggz/internal/facility"
	"github.com/kohei321dev/spot-diggz/internal/geocoding"
)

type placeProvider struct {
	places []geocoding.Result
	err    error
}

func (p placeProvider) Search(context.Context, string) ([]geocoding.Result, error) {
	return p.places, p.err
}

func searchFixture(t *testing.T) (facility.Facility, time.Time) {
	t.Helper()
	now := time.Date(2026, 7, 19, 0, 0, 0, 0, time.UTC)
	catalog, err := facility.LoadCatalogFileAt("../../testdata/facilities.dev.json", now)
	if err != nil {
		t.Fatal(err)
	}
	spot := catalog.List("skateboard")[0]
	spot.Genre = facility.GenreSkatepark
	spot.SkatingPermissionSourceURL = "https://example.com/test-permission"
	spot.GeneralUseStatus = facility.GeneralUseRegular
	spot.DynamicVerifiedAt = now
	spot.StableVerifiedAt = now
	return spot, now
}

func TestSearchFiltersRadiusGenreFreshnessAndSortsBeforeLimit(t *testing.T) {
	spot, now := searchFixture(t)
	spot.ID = "b"
	tie := spot
	tie.ID = "a"
	street := spot
	street.ID = "street"
	street.Genre = facility.GenreStreet
	street.Location.Latitude += 0.01
	far := spot
	far.ID = "far"
	far.Location.Latitude += 1
	stale := spot
	stale.ID = "stale"
	stale.DynamicVerifiedAt = now.Add(-facility.DynamicInformationFreshnessWindow - time.Second)
	unclassified := spot
	unclassified.ID = "unknown"
	unclassified.Genre = ""
	catalog, err := facility.NewCatalog([]facility.Facility{spot, tie, street, far, stale, unclassified})
	if err != nil {
		t.Fatal(err)
	}
	service := NewService(catalog, placeProvider{places: []geocoding.Result{{Label: "Test station", Location: spot.Location}}}, func() time.Time { return now })
	input, _ := DecodeInput([]byte(`{"query":"Test station","limit":2,"radiusKm":5}`))
	response, err := service.Search(context.Background(), input)
	if err != nil || response.Status != StatusOK || len(response.Matches) != 2 || response.Matches[0].Facility.ID != "a" || response.Matches[1].Facility.ID != "b" {
		t.Fatalf("unexpected search result: status=%s count=%d err=%v", response.Status, len(response.Matches), err)
	}
	input.Genre = facility.GenreStreet
	response, err = service.Search(context.Background(), input)
	if err != nil || len(response.Matches) != 1 || response.Matches[0].Facility.ID != "street" || response.Matches[0].DistanceKm <= 0 {
		t.Fatal("genre filtering failed")
	}
	input.RadiusKm = MinRadiusKm
	response, err = service.Search(context.Background(), input)
	if err != nil || len(response.Matches) != 0 || response.Status != StatusDataUnavailable {
		t.Fatal("radius was expanded or insufficient evidence hidden")
	}
	if !service.Ready() {
		t.Fatal("valid fixture not ready")
	}
}

func TestSearchDistinguishesLocationAndDataOutcomes(t *testing.T) {
	spot, now := searchFixture(t)
	input, _ := DecodeInput([]byte(`{"query":"Test station","genre":"street"}`))
	for _, test := range []struct {
		name, status string
		change       func(*facility.Facility)
		places       []geocoding.Result
		providerErr  error
		unavailable  bool
	}{
		{name: "genre no match", status: StatusNoMatches},
		{name: "unclassified", status: StatusDataUnavailable, change: func(f *facility.Facility) { f.Genre = "" }},
		{name: "stale", status: StatusDataUnavailable, change: func(f *facility.Facility) {
			f.StableVerifiedAt = now.Add(-facility.StableInformationFreshnessWindow - time.Second)
		}},
		{name: "outside coverage", status: StatusDataUnavailable, change: func(f *facility.Facility) { f.Location.Latitude += 1 }},
		{name: "ambiguous", status: StatusLocationAmbiguous, places: []geocoding.Result{{Label: "Station A", Location: spot.Location}, {Label: "Station B", Location: spot.Location}}},
		{name: "not found", status: StatusLocationNotFound, places: []geocoding.Result{}},
		{name: "provider error", providerErr: errors.New("private provider details"), unavailable: true},
		{name: "invalid provider", places: []geocoding.Result{{Label: "bad", Location: facility.Location{Latitude: math.NaN()}}}, unavailable: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			fixture := spot
			if test.change != nil {
				test.change(&fixture)
			}
			catalog, err := facility.NewCatalog([]facility.Facility{fixture})
			if err != nil {
				t.Fatal(err)
			}
			places := test.places
			if places == nil {
				places = []geocoding.Result{{Label: "Test station", Location: spot.Location}}
			}
			service := NewService(catalog, placeProvider{places, test.providerErr}, func() time.Time { return now })
			response, err := service.Search(context.Background(), input)
			if test.unavailable {
				if !errors.Is(err, ErrUnavailable) {
					t.Fatal("provider failure not propagated")
				}
				return
			}
			if err != nil || response.Status != test.status || response.Matches == nil || len(response.Matches) != 0 {
				t.Fatalf("status=%s err=%v", response.Status, err)
			}
		})
	}
	if _, err := NewService(nil, nil, nil).Search(context.Background(), input); !errors.Is(err, ErrUnavailable) {
		t.Fatal("missing dependencies accepted")
	}
}

func TestDistanceUsesKilometersAndIncludesExactRadius(t *testing.T) {
	spot, now := searchFixture(t)
	center := spot.Location
	center.Latitude -= 0.01
	distance := distanceKm(center, spot.Location)
	if math.Abs(distance-1.1119492664) > 0.000001 {
		t.Fatal("incorrect kilometer conversion")
	}
	catalog, err := facility.NewCatalog([]facility.Facility{spot})
	if err != nil {
		t.Fatal(err)
	}
	service := NewService(catalog, placeProvider{places: []geocoding.Result{{Label: "Test station", Location: center}}}, func() time.Time { return now })
	input := Input{Query: "test", Limit: 1, RadiusKm: distance, Sort: DistanceSort}
	response, err := service.Search(context.Background(), input)
	if err != nil || len(response.Matches) != 1 {
		t.Fatal("radius boundary excluded")
	}
	input.RadiusKm = math.Nextafter(distance, 0)
	response, err = service.Search(context.Background(), input)
	if err != nil || len(response.Matches) != 0 {
		t.Fatal("out-of-radius candidate included")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := service.Search(ctx, input); !errors.Is(err, ErrUnavailable) {
		t.Fatal("cancelled request accepted")
	}
}
