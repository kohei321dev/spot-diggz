package recommendation

import (
	"context"
	"errors"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/kohei321dev/spot-diggz/internal/facility"
	"github.com/kohei321dev/spot-diggz/internal/session"
	"github.com/kohei321dev/spot-diggz/internal/travel"
)

const (
	maxRecommendations             = 3
	maxTravelShareDivisor          = 3
	distanceRoundingFactor         = 10.0
	minutesPerDay                  = 24 * 60
	japanStandardTimeOffsetSeconds = 9 * 60 * 60
	purposeFeatureScore            = 25
	beginnerFriendlyScore          = 30
	generalFriendlyScore           = 10
	focusedMoodScore               = 10
	easygoingFeatureScore          = 8
	challengeFeatureScore          = 10
	skateTimeScoreDivisor          = 10
	TravelEstimateNotice           = "移動時間は実経路providerを利用できない場合、直線距離による概算へ切り替わります。各候補の表示種別と外部ナビを確認してください。"
	GoogleRoutesEstimateNotice     = "移動時間はGoogle Maps Routes APIの実経路に基づく目安です。運休、渋滞、経路変更は外部ナビで確認してください。"
	AvailabilityNotice             = "表示は公式情報の通常営業時間に基づきます。臨時休場、貸切、天候による変更は情報源で確認してください。"
)

var japanStandardTime = time.FixedZone("JST", japanStandardTimeOffsetSeconds)

type Reason struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type Item struct {
	Facility               facility.Facility `json:"facility"`
	Reasons                []Reason          `json:"reasons"`
	DistanceKm             float64           `json:"distanceKm"`
	EstimatedTravelMinutes int               `json:"estimatedTravelMinutes"`
	EstimatedSkateMinutes  int               `json:"estimatedSkateMinutes"`
	ArrivalAt              time.Time         `json:"arrivalAt"`
	FacilityClosesAt       time.Time         `json:"facilityClosesAt"`
	SessionEndsAt          time.Time         `json:"sessionEndsAt"`
	TravelEstimateKind     string            `json:"travelEstimateKind"`
}

type Response struct {
	Recommendations    []Item `json:"recommendations"`
	TravelEstimateNote string `json:"travelEstimateNote"`
	AvailabilityNote   string `json:"availabilityNote"`
}

type Engine struct {
	catalog        *facility.Catalog
	now            func() time.Time
	travelProvider travel.Provider
}

func NewEngine(catalog *facility.Catalog, now func() time.Time) *Engine {
	return NewEngineWithTravelProvider(catalog, now, travel.NewStraightLineProvider())
}

func NewEngineWithTravelProvider(catalog *facility.Catalog, now func() time.Time, provider travel.Provider) *Engine {
	if now == nil {
		now = time.Now
	}
	if provider == nil {
		provider = travel.NewStraightLineProvider()
	}
	return &Engine{catalog: catalog, now: now, travelProvider: provider}
}

func (engine *Engine) Recommend(input session.Input) (Response, error) {
	return engine.RecommendContext(context.Background(), input)
}

func (engine *Engine) RecommendContext(ctx context.Context, input session.Input) (Response, error) {
	if err := input.Validate(); err != nil {
		return Response{}, err
	}

	maxTravelMinutes := input.AvailableMinutes / maxTravelShareDivisor
	now := engine.now().In(japanStandardTime).Truncate(time.Minute)
	eligibleFacilities := make([]facility.Facility, 0)
	destinations := make([]travel.Destination, 0)
	for _, item := range engine.catalog.List("skateboard") {
		if item.GeneralUseStatus == facility.GeneralUseScheduleCheckRequired {
			continue
		}
		if input.Level == session.LevelBeginner && !item.BeginnerFriendly {
			continue
		}
		if !facility.IsDynamicInformationFresh(item.DynamicVerifiedAt, now) ||
			!facility.IsStableInformationFresh(item.StableVerifiedAt, now) {
			continue
		}
		eligibleFacilities = append(eligibleFacilities, item)
		destinations = append(destinations, travel.Destination{FacilityID: item.ID, Location: item.Location})
	}
	if len(destinations) == 0 {
		return emptyResponse(), nil
	}

	estimates, err := engine.travelProvider.Matrix(ctx, travel.Request{
		Origin: facility.Location{
			Latitude:  *input.Origin.Latitude,
			Longitude: *input.Origin.Longitude,
		},
		Destinations: destinations,
		Transport:    input.Transport,
		DepartureAt:  now,
	})
	if err != nil {
		return Response{}, fmt.Errorf("estimate travel times: %w", err)
	}
	estimatesByFacilityID := make(map[string]travel.Estimate, len(estimates))
	for _, estimate := range estimates {
		estimatesByFacilityID[estimate.FacilityID] = estimate
	}

	candidates := make([]scoredItem, 0)
	usedGoogleRoutes := false
	for _, item := range eligibleFacilities {
		estimate, exists := estimatesByFacilityID[item.ID]
		if !exists {
			continue
		}
		travelMinutes := estimate.TravelMinutes
		if travelMinutes > maxTravelMinutes {
			continue
		}

		arrivalAt := now.Add(time.Duration(travelMinutes) * time.Minute)
		facilityClosesAt, isOpen := openUntilAtFacility(item, arrivalAt)
		if !isOpen {
			continue
		}

		returnDepartureAt := now.Add(time.Duration(input.AvailableMinutes-travelMinutes) * time.Minute)
		sessionEndsAt := minTime(facilityClosesAt, returnDepartureAt)
		estimatedSkateMinutes := int(sessionEndsAt.Sub(arrivalAt) / time.Minute)
		if estimatedSkateMinutes <= 0 {
			continue
		}

		score, reasons := scoreFacility(item, input, travelMinutes, maxTravelMinutes, estimatedSkateMinutes)
		usedGoogleRoutes = usedGoogleRoutes || estimate.Kind == travel.GoogleRoutesKind
		candidates = append(candidates, scoredItem{
			score: score,
			item: Item{
				Facility:               item,
				Reasons:                reasons,
				DistanceKm:             math.Round(estimate.DistanceKm*distanceRoundingFactor) / distanceRoundingFactor,
				EstimatedTravelMinutes: travelMinutes,
				EstimatedSkateMinutes:  estimatedSkateMinutes,
				ArrivalAt:              arrivalAt,
				FacilityClosesAt:       facilityClosesAt,
				SessionEndsAt:          sessionEndsAt,
				TravelEstimateKind:     estimate.Kind,
			},
		})
	}

	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].score != candidates[j].score {
			return candidates[i].score > candidates[j].score
		}
		if candidates[i].item.DistanceKm != candidates[j].item.DistanceKm {
			return candidates[i].item.DistanceKm < candidates[j].item.DistanceKm
		}
		return candidates[i].item.Facility.ID < candidates[j].item.Facility.ID
	})

	limit := min(maxRecommendations, len(candidates))
	recommendations := make([]Item, 0, limit)
	for _, candidate := range candidates[:limit] {
		recommendations = append(recommendations, candidate.item)
	}

	response := Response{
		Recommendations:    recommendations,
		TravelEstimateNote: TravelEstimateNotice,
		AvailabilityNote:   AvailabilityNotice,
	}
	if usedGoogleRoutes {
		response.TravelEstimateNote = GoogleRoutesEstimateNotice
	}
	return response, nil
}

func emptyResponse() Response {
	return Response{
		Recommendations:    []Item{},
		TravelEstimateNote: TravelEstimateNotice,
		AvailabilityNote:   AvailabilityNotice,
	}
}

type scoredItem struct {
	score int
	item  Item
}

func scoreFacility(item facility.Facility, input session.Input, travelMinutes int, maxTravelMinutes int, estimatedSkateMinutes int) (int, []Reason) {
	matchedPurposeFeatures := matchingFeatures(item.Features, purposeFeatures(input.Purpose))
	score := (maxTravelMinutes - travelMinutes) +
		len(matchedPurposeFeatures)*purposeFeatureScore +
		estimatedSkateMinutes/skateTimeScoreDivisor
	reasons := make([]Reason, 0, 5)

	if len(matchedPurposeFeatures) > 0 {
		reasons = append(reasons, Reason{Code: "purpose_match", Message: purposeReason(input.Purpose)})
	}

	if item.BeginnerFriendly {
		if input.Level == session.LevelBeginner {
			score += beginnerFriendlyScore
			reasons = append(reasons, Reason{Code: "beginner_friendly", Message: "初心者向けとして確認されている施設です。"})
		} else {
			score += generalFriendlyScore
		}
	}

	score, reasons = applyMoodScore(score, reasons, item, input.Mood, matchedPurposeFeatures)
	reasons = append(reasons, Reason{
		Code:    "session_time_estimate",
		Message: "往復の概算移動と通常営業時間を考慮すると約" + strconv.Itoa(estimatedSkateMinutes) + "分滑れる見込みです。",
	})
	reasons = append(reasons, Reason{
		Code:    "travel_estimate",
		Message: "選択した交通手段で片道約" + strconv.Itoa(travelMinutes) + "分の概算です。",
	})

	return score, reasons
}

func applyMoodScore(score int, reasons []Reason, item facility.Facility, mood session.Mood, matchedPurposeFeatures []string) (int, []Reason) {
	switch mood {
	case session.MoodFocused:
		if len(matchedPurposeFeatures) > 0 {
			score += focusedMoodScore
			reasons = append(reasons, Reason{Code: "mood_match", Message: "目的の練習に集中しやすい設備があります。"})
		}
	case session.MoodEasygoing:
		matched := matchingFeatures(item.Features, []string{"indoor", "roof", "lighting", "rental"})
		if len(matched) > 0 {
			score += len(matched) * easygoingFeatureScore
			reasons = append(reasons, Reason{Code: "mood_match", Message: "気軽に滑りやすい設備条件があります。"})
		}
	case session.MoodChallenge:
		matched := matchingFeatures(item.Features, []string{"stairs", "handrail", "rail", "ledge", "bank", "mini-ramp", "quarter-ramp", "bowl"})
		if len(matched) > 0 {
			score += len(matched) * challengeFeatureScore
			reasons = append(reasons, Reason{Code: "mood_match", Message: "挑戦向けのセクションがあります。"})
		}
	}

	return score, reasons
}

func purposeFeatures(purpose session.Purpose) []string {
	switch purpose {
	case session.PurposeBasics:
		return []string{"flat-area"}
	case session.PurposeStreet:
		return []string{"stairs", "handrail", "rail", "ledge", "bank"}
	case session.PurposeTransition:
		return []string{"mini-ramp", "quarter-ramp", "bowl"}
	default:
		return nil
	}
}

func purposeReason(purpose session.Purpose) string {
	switch purpose {
	case session.PurposeBasics:
		return "基礎練習に合うフラットエリアがあります。"
	case session.PurposeStreet:
		return "ストリート練習に合うセクションがあります。"
	case session.PurposeTransition:
		return "ランプやボウルの練習に合う設備があります。"
	default:
		return "選択した目的に合う設備があります。"
	}
}

func matchingFeatures(features []string, targets []string) []string {
	matched := make([]string, 0)
	for _, target := range targets {
		for _, feature := range features {
			if strings.EqualFold(feature, target) {
				matched = append(matched, target)
				break
			}
		}
	}
	return matched
}

func isOpenAt(hours []facility.OperatingHours, at time.Time) bool {
	_, isOpen := openUntilAt(hours, at)
	return isOpen
}

func openUntilAtFacility(item facility.Facility, at time.Time) (time.Time, bool) {
	if isClosedByPeriod(item.ClosurePeriods, at) {
		return time.Time{}, false
	}
	return openUntilAt(item.Hours, at)
}

func openUntilAt(hours []facility.OperatingHours, at time.Time) (time.Time, bool) {
	minutes := at.Hour()*60 + at.Minute()
	startOfDay := time.Date(at.Year(), at.Month(), at.Day(), 0, 0, 0, 0, at.Location())
	var latestClose time.Time
	for _, period := range matchingOperatingHours(hours, at) {
		opens, openErr := parseClockMinutes(period.Opens)
		closes, closeErr := parseClockMinutes(period.Closes)
		if openErr != nil || closeErr != nil {
			continue
		}

		var closesAt time.Time
		if opens < closes && minutes >= opens && minutes < closes {
			closesAt = startOfDay.Add(time.Duration(closes) * time.Minute)
		}
		if opens > closes {
			if minutes >= opens {
				closesAt = startOfDay.Add(time.Duration(minutesPerDay+closes) * time.Minute)
			}
		}

		if closesAt.After(latestClose) {
			latestClose = closesAt
		}
	}
	for _, period := range matchingOperatingHours(hours, at.AddDate(0, 0, -1)) {
		opens, openErr := parseClockMinutes(period.Opens)
		closes, closeErr := parseClockMinutes(period.Closes)
		if openErr != nil || closeErr != nil || opens <= closes || minutes >= closes {
			continue
		}
		closesAt := startOfDay.Add(time.Duration(closes) * time.Minute)
		if closesAt.After(latestClose) {
			latestClose = closesAt
		}
	}

	return latestClose, !latestClose.IsZero()
}

func matchingOperatingHours(hours []facility.OperatingHours, at time.Time) []facility.OperatingHours {
	matched := make([]facility.OperatingHours, 0)
	maximumSpecificity := -1
	for _, period := range hours {
		if !dayMatches(period.Day, at) {
			continue
		}
		specificity := daySpecificity(period.Day)
		if specificity > maximumSpecificity {
			matched = matched[:0]
			maximumSpecificity = specificity
		}
		if specificity == maximumSpecificity {
			matched = append(matched, period)
		}
	}
	for _, period := range matched {
		if period.Closed {
			return nil
		}
	}
	return matched
}

func daySpecificity(rule string) int {
	switch strings.ToLower(strings.TrimSpace(rule)) {
	case "daily":
		return 1
	case "weekday", "weekend":
		return 2
	default:
		return 3
	}
}

func isClosedByPeriod(periods []facility.ClosurePeriod, at time.Time) bool {
	date := time.Date(at.Year(), at.Month(), at.Day(), 0, 0, 0, 0, at.Location())
	for _, period := range periods {
		switch period.Type {
		case facility.ClosurePeriodOneTime:
			start, startErr := time.ParseInLocation("2006-01-02", period.Start, at.Location())
			end, endErr := time.ParseInLocation("2006-01-02", period.End, at.Location())
			if startErr == nil && endErr == nil && !date.Before(start) && !date.After(end) {
				return true
			}
		case facility.ClosurePeriodAnnual:
			currentMonthDay := int(at.Month())*100 + at.Day()
			startMonthDay, startOK := parseMonthDayNumber(period.Start)
			endMonthDay, endOK := parseMonthDayNumber(period.End)
			if !startOK || !endOK {
				continue
			}
			if startMonthDay <= endMonthDay && currentMonthDay >= startMonthDay && currentMonthDay <= endMonthDay {
				return true
			}
			if startMonthDay > endMonthDay && (currentMonthDay >= startMonthDay || currentMonthDay <= endMonthDay) {
				return true
			}
		}
	}
	return false
}

func parseMonthDayNumber(value string) (int, bool) {
	parsed, err := time.Parse("01-02", value)
	if err != nil || parsed.Format("01-02") != value {
		return 0, false
	}
	return int(parsed.Month())*100 + parsed.Day(), true
}

func minTime(first time.Time, second time.Time) time.Time {
	if first.Before(second) {
		return first
	}
	return second
}

func parseClockMinutes(value string) (int, error) {
	if value == "24:00" {
		return minutesPerDay, nil
	}
	parsed, err := time.Parse("15:04", value)
	if err != nil {
		return 0, err
	}
	return parsed.Hour()*60 + parsed.Minute(), nil
}

func dayMatches(rule string, at time.Time) bool {
	switch strings.ToLower(strings.TrimSpace(rule)) {
	case "daily":
		return true
	case "weekday":
		return at.Weekday() >= time.Monday && at.Weekday() <= time.Friday
	case "weekend":
		return at.Weekday() == time.Saturday || at.Weekday() == time.Sunday
	default:
		return strings.EqualFold(rule, at.Weekday().String())
	}
}

func IsInvalidInput(err error) bool {
	return errors.Is(err, session.ErrInvalidInput)
}
