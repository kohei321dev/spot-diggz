package recommendation

import (
	"testing"
	"time"

	"github.com/kohei321dev/spot-diggz/internal/facility"
	"github.com/kohei321dev/spot-diggz/internal/session"
)

func TestEngineReturnsAtMostThreeRankedRecommendations(t *testing.T) {
	catalog := newTestCatalog(t, []facility.Facility{
		recommendationFacility("facility-d", 34.7010, 135.4970, true, []string{"flat-area"}),
		recommendationFacility("facility-b", 34.7030, 135.4970, true, []string{"flat-area", "lighting"}),
		recommendationFacility("facility-c", 34.7040, 135.4970, true, []string{"mini-ramp"}),
		recommendationFacility("facility-a", 34.7020, 135.4970, true, []string{"flat-area", "indoor"}),
	})
	engine := NewEngine(catalog, fixedTestTime)

	response, err := engine.Recommend(validInput())
	if err != nil {
		t.Fatalf("Recommend() error = %v", err)
	}
	if len(response.Recommendations) != 3 {
		t.Fatalf("recommendation count = %d, want 3", len(response.Recommendations))
	}
	if response.Recommendations[0].Facility.ID != "facility-a" {
		t.Fatalf("first facility = %q, want facility-a", response.Recommendations[0].Facility.ID)
	}
	if response.TravelEstimateNote == "" {
		t.Fatal("TravelEstimateNote is empty")
	}
	if len(response.Recommendations[0].Reasons) == 0 {
		t.Fatal("Reasons is empty")
	}
}

func TestEngineExcludesFacilitiesOutsideHardConditions(t *testing.T) {
	closed := recommendationFacility("closed", 34.7020, 135.4970, true, []string{"flat-area"})
	closed.Hours = []facility.OperatingHours{{Day: "daily", Closed: true}}
	catalog := newTestCatalog(t, []facility.Facility{
		recommendationFacility("eligible", 34.7020, 135.4970, true, []string{"flat-area"}),
		recommendationFacility("advanced", 34.7020, 135.4970, false, []string{"flat-area"}),
		recommendationFacility("far-away", 35.6812, 139.7671, true, []string{"flat-area"}),
		closed,
	})
	engine := NewEngine(catalog, fixedTestTime)

	response, err := engine.Recommend(validInput())
	if err != nil {
		t.Fatalf("Recommend() error = %v", err)
	}
	if len(response.Recommendations) != 1 || response.Recommendations[0].Facility.ID != "eligible" {
		t.Fatalf("recommendations = %#v, want only eligible", response.Recommendations)
	}
}

func TestIsOpenAtSupportsDailyAndOvernightHours(t *testing.T) {
	tests := []struct {
		name  string
		hours []facility.OperatingHours
		at    time.Time
		want  bool
	}{
		{
			name:  "daily open",
			hours: []facility.OperatingHours{{Day: "daily", Opens: "09:00", Closes: "21:00"}},
			at:    time.Date(2026, time.July, 16, 20, 0, 0, 0, japanStandardTime),
			want:  true,
		},
		{
			name:  "at close",
			hours: []facility.OperatingHours{{Day: "daily", Opens: "09:00", Closes: "21:00"}},
			at:    time.Date(2026, time.July, 16, 21, 0, 0, 0, japanStandardTime),
			want:  false,
		},
		{
			name:  "overnight after midnight",
			hours: []facility.OperatingHours{{Day: "Thursday", Opens: "22:00", Closes: "02:00"}},
			at:    time.Date(2026, time.July, 17, 1, 0, 0, 0, japanStandardTime),
			want:  true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := isOpenAt(test.hours, test.at); got != test.want {
				t.Fatalf("isOpenAt() = %t, want %t", got, test.want)
			}
		})
	}
}

func validInput() session.Input {
	latitude := 34.7025
	longitude := 135.4960
	return session.Input{
		Purpose:          session.PurposeBasics,
		Mood:             session.MoodEasygoing,
		Level:            session.LevelBeginner,
		AvailableMinutes: 120,
		Transport:        session.TransportPublicTransit,
		Origin: session.Origin{
			Mode:      session.OriginSpecifiedLocation,
			Latitude:  &latitude,
			Longitude: &longitude,
		},
	}
}

func newTestCatalog(t *testing.T, facilities []facility.Facility) *facility.Catalog {
	t.Helper()
	catalog, err := facility.NewCatalog(facilities)
	if err != nil {
		t.Fatalf("NewCatalog() error = %v", err)
	}
	return catalog
}

func recommendationFacility(id string, latitude float64, longitude float64, beginnerFriendly bool, features []string) facility.Facility {
	return facility.Facility{
		ID:               id,
		Name:             "Test Facility " + id,
		Address:          "大阪府大阪市",
		Location:         facility.Location{Latitude: latitude, Longitude: longitude},
		Activities:       []string{"skateboard"},
		Hours:            []facility.OperatingHours{{Day: "daily", Opens: "00:00", Closes: "24:00"}},
		BeginnerFriendly: beginnerFriendly,
		Features:         features,
		SourceURL:        "https://example.com/facilities/" + id,
		SourceType:       "official",
		Status:           "verified",
		VerifiedAt:       time.Date(2026, time.July, 15, 0, 0, 0, 0, time.UTC),
	}
}

func fixedTestTime() time.Time {
	return time.Date(2026, time.July, 16, 20, 0, 0, 0, japanStandardTime)
}
