package recommendation

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/kohei321dev/spot-diggz/internal/facility"
	"github.com/kohei321dev/spot-diggz/internal/session"
	"github.com/kohei321dev/spot-diggz/internal/travel"
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
	if response.AvailabilityNote == "" {
		t.Fatal("AvailabilityNote is empty")
	}
	if len(response.Recommendations[0].Reasons) == 0 {
		t.Fatal("Reasons is empty")
	}
	first := response.Recommendations[0]
	wantSkateMinutes := validInput().AvailableMinutes - 2*first.EstimatedTravelMinutes
	if first.EstimatedSkateMinutes != wantSkateMinutes {
		t.Fatalf("EstimatedSkateMinutes = %d, want %d", first.EstimatedSkateMinutes, wantSkateMinutes)
	}
	if first.ArrivalAt.IsZero() || first.FacilityClosesAt.IsZero() || first.SessionEndsAt.IsZero() {
		t.Fatal("session timing fields must not be zero")
	}
}

func TestEngineExcludesFacilitiesOutsideHardConditions(t *testing.T) {
	closed := recommendationFacility("closed", 34.7020, 135.4970, true, []string{"flat-area"})
	closed.Hours = []facility.OperatingHours{{Day: "Wednesday", Opens: "09:00", Closes: "21:00"}}
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

func TestEngineExcludesFacilityRequiringScheduleCheck(t *testing.T) {
	needsScheduleCheck := recommendationFacility("needs-schedule-check", 34.7020, 135.4970, true, []string{"flat-area"})
	needsScheduleCheck.GeneralUseStatus = facility.GeneralUseScheduleCheckRequired
	needsScheduleCheck.AvailabilityNote = "来場前に公式予定を確認してください。"
	eligible := recommendationFacility("eligible", 34.7030, 135.4970, true, []string{"flat-area"})

	engine := NewEngine(newTestCatalog(t, []facility.Facility{needsScheduleCheck, eligible}), fixedTestTime)
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

func TestEngineLimitsSkateTimeToFacilityClosing(t *testing.T) {
	closingSoon := recommendationFacility("closing-soon", 34.7020, 135.4970, true, []string{"flat-area"})
	closingSoon.Hours = []facility.OperatingHours{{Day: "Thursday", Opens: "09:00", Closes: "20:30"}}
	engine := NewEngine(newTestCatalog(t, []facility.Facility{closingSoon}), fixedTestTime)

	response, err := engine.Recommend(validInput())
	if err != nil {
		t.Fatalf("Recommend() error = %v", err)
	}
	if len(response.Recommendations) != 1 {
		t.Fatalf("recommendation count = %d, want 1", len(response.Recommendations))
	}

	item := response.Recommendations[0]
	wantClose := time.Date(2026, time.July, 16, 20, 30, 0, 0, japanStandardTime)
	if !item.FacilityClosesAt.Equal(wantClose) {
		t.Fatalf("FacilityClosesAt = %s, want %s", item.FacilityClosesAt, wantClose)
	}
	if !item.SessionEndsAt.Equal(wantClose) {
		t.Fatalf("SessionEndsAt = %s, want %s", item.SessionEndsAt, wantClose)
	}
	wantSkateMinutes := int(wantClose.Sub(item.ArrivalAt) / time.Minute)
	if item.EstimatedSkateMinutes != wantSkateMinutes {
		t.Fatalf("EstimatedSkateMinutes = %d, want %d", item.EstimatedSkateMinutes, wantSkateMinutes)
	}
}

func TestEngineTreatsSpecificClosedDayAsOverrideForDailyHours(t *testing.T) {
	item := recommendationFacility("thursday-closed", 34.7020, 135.4970, true, []string{"flat-area"})
	item.Hours = append(item.Hours, facility.OperatingHours{Day: "thursday", Closed: true})
	engine := NewEngine(newTestCatalog(t, []facility.Facility{item}), fixedTestTime)

	response, err := engine.Recommend(validInput())
	if err != nil {
		t.Fatalf("Recommend() error = %v", err)
	}
	if len(response.Recommendations) != 0 {
		t.Fatalf("recommendations = %#v, want none", response.Recommendations)
	}
}

func TestEngineExcludesFacilityDuringOneTimeClosure(t *testing.T) {
	item := recommendationFacility("one-time-closure", 34.7020, 135.4970, true, []string{"flat-area"})
	item.ClosurePeriods = []facility.ClosurePeriod{{
		Type: facility.ClosurePeriodOneTime, Start: "2026-07-16", End: "2026-07-16", Reason: "点検",
	}}
	engine := NewEngine(newTestCatalog(t, []facility.Facility{item}), fixedTestTime)

	response, err := engine.Recommend(validInput())
	if err != nil {
		t.Fatalf("Recommend() error = %v", err)
	}
	if len(response.Recommendations) != 0 {
		t.Fatalf("recommendations = %#v, want none", response.Recommendations)
	}
}

func TestEngineExcludesFacilityDuringCrossYearAnnualClosure(t *testing.T) {
	item := recommendationFacility("annual-closure", 34.7020, 135.4970, true, []string{"flat-area"})
	item.ClosurePeriods = []facility.ClosurePeriod{{
		Type: facility.ClosurePeriodAnnual, Start: "12-29", End: "01-03", Reason: "年末年始休場",
	}}
	now := time.Date(2026, time.December, 30, 12, 0, 0, 0, japanStandardTime)
	item.DynamicVerifiedAt = now.Add(-24 * time.Hour)
	item.StableVerifiedAt = now.Add(-24 * time.Hour)
	engine := NewEngine(newTestCatalog(t, []facility.Facility{item}), func() time.Time { return now })

	response, err := engine.Recommend(validInput())
	if err != nil {
		t.Fatalf("Recommend() error = %v", err)
	}
	if len(response.Recommendations) != 0 {
		t.Fatalf("recommendations = %#v, want none", response.Recommendations)
	}
}

func TestEngineExcludesFacilityWithStaleInformation(t *testing.T) {
	item := recommendationFacility("stale", 34.7020, 135.4970, true, []string{"flat-area"})
	item.DynamicVerifiedAt = fixedTestTime().Add(-facility.DynamicInformationFreshnessWindow - time.Minute)
	engine := NewEngine(newTestCatalog(t, []facility.Facility{item}), fixedTestTime)

	response, err := engine.Recommend(validInput())
	if err != nil {
		t.Fatalf("Recommend() error = %v", err)
	}
	if len(response.Recommendations) != 0 {
		t.Fatalf("recommendations = %#v, want none", response.Recommendations)
	}
}

func TestEngineUsesTravelProviderEstimatesAndNotice(t *testing.T) {
	catalog := newTestCatalog(t, []facility.Facility{
		recommendationFacility("slower", 34.7020, 135.4970, true, []string{"flat-area"}),
		recommendationFacility("faster", 34.8020, 135.5970, true, []string{"flat-area"}),
	})
	provider := staticTravelProvider{estimates: []travel.Estimate{
		{FacilityID: "slower", DistanceKm: 2, TravelMinutes: 18, Kind: travel.GoogleRoutesKind},
		{FacilityID: "faster", DistanceKm: 10, TravelMinutes: 5, Kind: travel.GoogleRoutesKind},
	}}
	engine := NewEngineWithTravelProvider(catalog, fixedTestTime, provider)

	response, err := engine.Recommend(validInput())
	if err != nil {
		t.Fatalf("Recommend() error = %v", err)
	}
	if len(response.Recommendations) != 2 || response.Recommendations[0].Facility.ID != "faster" {
		t.Fatalf("recommendations = %#v, want faster first", response.Recommendations)
	}
	if response.Recommendations[0].TravelEstimateKind != travel.GoogleRoutesKind {
		t.Fatalf("TravelEstimateKind = %q", response.Recommendations[0].TravelEstimateKind)
	}
	if response.TravelEstimateNote != GoogleRoutesEstimateNotice {
		t.Fatalf("TravelEstimateNote = %q", response.TravelEstimateNote)
	}
}

func TestEngineReturnsTravelProviderError(t *testing.T) {
	providerError := errors.New("provider failed")
	engine := NewEngineWithTravelProvider(
		newTestCatalog(t, []facility.Facility{recommendationFacility("facility", 34.7020, 135.4970, true, []string{"flat-area"})}),
		fixedTestTime,
		staticTravelProvider{err: providerError},
	)

	_, err := engine.Recommend(validInput())
	if !errors.Is(err, providerError) {
		t.Fatalf("Recommend() error = %v, want provider error", err)
	}
}

func TestOpenUntilAtReturnsNextDayForOvernightHours(t *testing.T) {
	hours := []facility.OperatingHours{{Day: "Thursday", Opens: "22:00", Closes: "02:00"}}
	at := time.Date(2026, time.July, 17, 1, 0, 0, 0, japanStandardTime)

	got, isOpen := openUntilAt(hours, at)
	if !isOpen {
		t.Fatal("openUntilAt() isOpen = false, want true")
	}
	want := time.Date(2026, time.July, 17, 2, 0, 0, 0, japanStandardTime)
	if !got.Equal(want) {
		t.Fatalf("openUntilAt() = %s, want %s", got, want)
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
	verifiedAt := time.Date(2026, time.July, 15, 0, 0, 0, 0, time.UTC)
	return facility.Facility{
		ID:                 id,
		Name:               "Test Facility " + id,
		Address:            "大阪府大阪市",
		Prefecture:         "大阪府",
		Municipality:       "大阪市",
		Location:           facility.Location{Latitude: latitude, Longitude: longitude},
		Activities:         []string{"skateboard"},
		Hours:              []facility.OperatingHours{{Day: "daily", Opens: "00:00", Closes: "24:00"}},
		ScheduleNotes:      []string{"臨時変更は公式情報を確認"},
		Price:              "500円",
		Reservation:        "当日受付",
		BeginnerFriendly:   beginnerFriendly,
		Features:           features,
		Rules:              []string{"ヘルメット必須"},
		Access:             facility.Access{Notes: "テスト用アクセス"},
		EnglishTranslation: facility.FacilityEnglishTranslation{Name: "Test Facility " + id, Address: "Osaka City, Osaka", ScheduleNotes: []string{"Check the official source for temporary changes."}, Price: "JPY 500", Reservation: "Register on arrival", Rules: []string{"Helmet required"}, AccessNotes: "Test access"},
		SourceURL:          "https://example.com/facilities/" + id,
		SourceType:         "official",
		Status:             "verified",
		Confidence:         "high",
		VerifiedAt:         verifiedAt,
		DynamicVerifiedAt:  verifiedAt,
		StableVerifiedAt:   verifiedAt,
	}
}

type staticTravelProvider struct {
	estimates []travel.Estimate
	err       error
}

func (provider staticTravelProvider) Matrix(_ context.Context, _ travel.Request) ([]travel.Estimate, error) {
	return provider.estimates, provider.err
}

func fixedTestTime() time.Time {
	return time.Date(2026, time.July, 16, 20, 0, 0, 0, japanStandardTime)
}
