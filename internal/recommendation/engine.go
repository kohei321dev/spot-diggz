package recommendation

import (
	"errors"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/kohei321dev/spot-diggz/internal/facility"
	"github.com/kohei321dev/spot-diggz/internal/session"
)

const (
	maxRecommendations             = 3
	maxTravelShareDivisor          = 3
	earthRadiusKm                  = 6371.0
	distanceRoundingFactor         = 10.0
	minutesPerHour                 = 60.0
	minutesPerDay                  = 24 * 60
	japanStandardTimeOffsetSeconds = 9 * 60 * 60
	purposeFeatureScore            = 25
	beginnerFriendlyScore          = 30
	generalFriendlyScore           = 10
	focusedMoodScore               = 10
	easygoingFeatureScore          = 8
	challengeFeatureScore          = 10
	publicTransitSpeedKmPerHour    = 22.0
	publicTransitOverheadMinutes   = 12
	carSpeedKmPerHour              = 30.0
	carOverheadMinutes             = 8
	bicycleSpeedKmPerHour          = 13.0
	bicycleOverheadMinutes         = 3
	walkSpeedKmPerHour             = 4.5
	TravelEstimateKind             = "straight_line"
	TravelEstimateNotice           = "移動時間は直線距離による概算です。実際の経路と所要時間は外部ナビで確認してください。"
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
	TravelEstimateKind     string            `json:"travelEstimateKind"`
}

type Response struct {
	Recommendations    []Item `json:"recommendations"`
	TravelEstimateNote string `json:"travelEstimateNote"`
}

type Engine struct {
	catalog *facility.Catalog
	now     func() time.Time
}

func NewEngine(catalog *facility.Catalog, now func() time.Time) *Engine {
	if now == nil {
		now = time.Now
	}
	return &Engine{catalog: catalog, now: now}
}

func (engine *Engine) Recommend(input session.Input) (Response, error) {
	if err := input.Validate(); err != nil {
		return Response{}, err
	}

	originLatitude := *input.Origin.Latitude
	originLongitude := *input.Origin.Longitude
	maxTravelMinutes := input.AvailableMinutes / maxTravelShareDivisor
	now := engine.now().In(japanStandardTime)
	candidates := make([]scoredItem, 0)

	for _, item := range engine.catalog.List("skateboard") {
		if input.Level == session.LevelBeginner && !item.BeginnerFriendly {
			continue
		}

		distanceKm := haversineKm(originLatitude, originLongitude, item.Location.Latitude, item.Location.Longitude)
		travelMinutes := estimateTravelMinutes(distanceKm, input.Transport)
		if travelMinutes > maxTravelMinutes {
			continue
		}

		arrivalAt := now.Add(time.Duration(travelMinutes) * time.Minute)
		if !isOpenAt(item.Hours, arrivalAt) {
			continue
		}

		score, reasons := scoreFacility(item, input, travelMinutes, maxTravelMinutes)
		candidates = append(candidates, scoredItem{
			score: score,
			item: Item{
				Facility:               item,
				Reasons:                reasons,
				DistanceKm:             math.Round(distanceKm*distanceRoundingFactor) / distanceRoundingFactor,
				EstimatedTravelMinutes: travelMinutes,
				TravelEstimateKind:     TravelEstimateKind,
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

	return Response{
		Recommendations:    recommendations,
		TravelEstimateNote: TravelEstimateNotice,
	}, nil
}

type scoredItem struct {
	score int
	item  Item
}

type travelProfile struct {
	speedKmPerHour  float64
	overheadMinutes int
}

func estimateTravelMinutes(distanceKm float64, transport session.Transport) int {
	profile := travelProfileFor(transport)
	return int(math.Ceil(distanceKm/profile.speedKmPerHour*minutesPerHour)) + profile.overheadMinutes
}

func travelProfileFor(transport session.Transport) travelProfile {
	switch transport {
	case session.TransportPublicTransit:
		return travelProfile{speedKmPerHour: publicTransitSpeedKmPerHour, overheadMinutes: publicTransitOverheadMinutes}
	case session.TransportCar:
		return travelProfile{speedKmPerHour: carSpeedKmPerHour, overheadMinutes: carOverheadMinutes}
	case session.TransportBicycle:
		return travelProfile{speedKmPerHour: bicycleSpeedKmPerHour, overheadMinutes: bicycleOverheadMinutes}
	case session.TransportWalk:
		return travelProfile{speedKmPerHour: walkSpeedKmPerHour}
	default:
		return travelProfile{}
	}
}

func scoreFacility(item facility.Facility, input session.Input, travelMinutes int, maxTravelMinutes int) (int, []Reason) {
	matchedPurposeFeatures := matchingFeatures(item.Features, purposeFeatures(input.Purpose))
	score := (maxTravelMinutes - travelMinutes) + len(matchedPurposeFeatures)*purposeFeatureScore
	reasons := make([]Reason, 0, 4)

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
		matched := matchingFeatures(item.Features, []string{"stairs", "handrail", "mini-ramp", "bowl"})
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
		return []string{"stairs", "handrail"}
	case session.PurposeTransition:
		return []string{"mini-ramp", "bowl"}
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
	minutes := at.Hour()*60 + at.Minute()
	for _, period := range hours {
		opens, openErr := parseClockMinutes(period.Opens)
		closes, closeErr := parseClockMinutes(period.Closes)
		if period.Closed || openErr != nil || closeErr != nil {
			continue
		}

		if opens < closes && dayMatches(period.Day, at) && minutes >= opens && minutes < closes {
			return true
		}
		if opens > closes {
			if dayMatches(period.Day, at) && minutes >= opens {
				return true
			}
			if dayMatches(period.Day, at.Add(-24*time.Hour)) && minutes < closes {
				return true
			}
		}
	}
	return false
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

func IsInvalidInput(err error) bool {
	return errors.Is(err, session.ErrInvalidInput)
}
