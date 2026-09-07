package botsearch

import (
	"bytes"
	"encoding/json"
	"io"
	"math"
	"reflect"
	"slices"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/kohei321dev/spot-diggz/internal/facility"
	"github.com/kohei321dev/spot-diggz/internal/nearby"
)

// Pointers distinguish a real zero-distance match from missing/null wire fields.
type responseMatch struct {
	Facility   *facility.Facility `json:"facility"`
	DistanceKm *float64           `json:"distanceKm"`
}

type responseBody struct {
	Status             string                  `json:"status"`
	Matches            []responseMatch         `json:"matches"`
	LocationCandidates []nearby.PlaceCandidate `json:"locationCandidates,omitempty"`
	DistanceKind       string                  `json:"distanceKind"`
	Limit              int                     `json:"limit"`
	RadiusKm           float64                 `json:"radiusKm"`
	Sort               string                  `json:"sort"`
	AvailabilityNote   string                  `json:"availabilityNote"`
}

func decodeResponse(data []byte, input nearby.Input) (nearby.Response, error) {
	if !utf8.Valid(data) {
		return nearby.Response{}, ErrInvalidResponse
	}
	// Check exact names and required fields before Go's permissive struct decoder
	// can merge case-folded keys or turn missing public facts into zero values.
	check := json.NewDecoder(bytes.NewReader(data))
	if !validWireValue(check, reflect.TypeFor[responseBody](), 0) {
		return nearby.Response{}, ErrInvalidResponse
	}
	if _, err := check.Token(); err != io.EOF {
		return nearby.Response{}, ErrInvalidResponse
	}
	var wire responseBody
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&wire); err != nil || wire.Matches == nil ||
		wire.DistanceKind != nearby.DistanceKind || wire.Limit != input.Limit ||
		wire.RadiusKm != input.RadiusKm || wire.Sort != input.Sort ||
		!validResponseText(wire.AvailabilityNote, 1000) {
		return nearby.Response{}, ErrInvalidResponse
	}
	switch wire.Status {
	case nearby.StatusOK:
		if len(wire.Matches) == 0 || len(wire.Matches) > input.Limit || len(wire.LocationCandidates) != 0 {
			return nearby.Response{}, ErrInvalidResponse
		}
	case nearby.StatusLocationAmbiguous:
		if len(wire.Matches) != 0 || len(wire.LocationCandidates) < 2 || len(wire.LocationCandidates) > 5 {
			return nearby.Response{}, ErrInvalidResponse
		}
		for _, place := range wire.LocationCandidates {
			if !validResponseText(place.Label, 300) {
				return nearby.Response{}, ErrInvalidResponse
			}
		}
	case nearby.StatusNoMatches, nearby.StatusDataUnavailable, nearby.StatusLocationNotFound:
		if len(wire.Matches) != 0 || len(wire.LocationCandidates) != 0 {
			return nearby.Response{}, ErrInvalidResponse
		}
	default:
		return nearby.Response{}, ErrInvalidResponse
	}
	result := nearby.Response{
		Status: wire.Status, Matches: []nearby.Match{}, LocationCandidates: wire.LocationCandidates,
		DistanceKind: wire.DistanceKind, Limit: wire.Limit, RadiusKm: wire.RadiusKm,
		Sort: wire.Sort, AvailabilityNote: wire.AvailabilityNote,
	}
	spots := make([]facility.Facility, 0, len(wire.Matches))
	for i, match := range wire.Matches {
		if match.Facility == nil || match.DistanceKm == nil || math.IsNaN(*match.DistanceKm) ||
			math.IsInf(*match.DistanceKm, 0) || *match.DistanceKm < 0 || *match.DistanceKm > input.RadiusKm {
			return nearby.Response{}, ErrInvalidResponse
		}
		spot := *match.Facility
		if (spot.Genre != facility.GenreSkatepark && spot.Genre != facility.GenreStreet) ||
			!slices.ContainsFunc(spot.Activities, func(activity string) bool { return strings.EqualFold(activity, "skateboard") }) ||
			(input.Genre != "" && spot.Genre != input.Genre) || spot.HoursStatus == facility.HoursUnknown ||
			spot.GeneralUseStatus == facility.GeneralUseScheduleCheckRequired {
			return nearby.Response{}, ErrInvalidResponse
		}
		if i > 0 {
			previous := result.Matches[i-1]
			if previous.DistanceKm > *match.DistanceKm ||
				(previous.DistanceKm == *match.DistanceKm && previous.Facility.ID >= spot.ID) {
				return nearby.Response{}, ErrInvalidResponse
			}
		}
		spots = append(spots, spot)
		result.Matches = append(result.Matches, nearby.Match{Facility: spot, DistanceKm: *match.DistanceKm})
	}
	// Reuse structural validation (including hours evidence and unique IDs), but
	// leave freshness/time and geographic selection to the API's authoritative clock.
	if _, err := facility.NewCatalog(spots); err != nil {
		return nearby.Response{}, ErrInvalidResponse
	}
	return result, nil
}

func validResponseText(value string, maxRunes int) bool {
	return strings.TrimSpace(value) != "" && utf8.RuneCountInString(value) <= maxRunes &&
		strings.IndexFunc(value, unicode.IsControl) < 0
}

// Validate the wire shape using the shared models' JSON tags. This is not a second
// catalog schema: value/enum/evidence rules stay in facility.NewCatalog. Non-omitempty
// fields must be present, names are exact, and each object key can occur only once.
func validWireValue(decoder *json.Decoder, shape reflect.Type, depth int) bool {
	if depth > 32 {
		return false
	}
	for shape.Kind() == reflect.Pointer {
		shape = shape.Elem()
	}
	token, err := decoder.Token()
	if err != nil {
		return false
	}
	if token == nil {
		return false // The API emits arrays or omits optional fields, never null facts.
	}
	if shape == reflect.TypeFor[time.Time]() {
		_, ok := token.(string)
		return ok // The subsequent time.Time decoder checks RFC 3339.
	}
	switch shape.Kind() {
	case reflect.Struct:
		if token != json.Delim('{') {
			return false
		}
		fields := make(map[string]reflect.StructField)
		required := make(map[string]bool)
		for i := 0; i < shape.NumField(); i++ {
			field := shape.Field(i)
			name, options, _ := strings.Cut(field.Tag.Get("json"), ",")
			if name == "" || name == "-" || !field.IsExported() {
				continue
			}
			fields[name] = field
			required[name] = !slices.Contains(strings.Split(options, ","), "omitempty")
		}
		seen := make(map[string]bool)
		for decoder.More() {
			keyToken, err := decoder.Token()
			key, ok := keyToken.(string)
			if err != nil || !ok {
				return false
			}
			field, exists := fields[key]
			if !exists || seen[key] || !validWireValue(decoder, field.Type, depth+1) {
				return false
			}
			seen[key] = true
		}
		for key, isRequired := range required {
			if isRequired && !seen[key] {
				return false
			}
		}
		end, err := decoder.Token()
		return err == nil && end == json.Delim('}')
	case reflect.Slice:
		if token != json.Delim('[') {
			return false
		}
		for decoder.More() {
			if !validWireValue(decoder, shape.Elem(), depth+1) {
				return false
			}
		}
		end, err := decoder.Token()
		return err == nil && end == json.Delim(']')
	case reflect.String:
		_, ok := token.(string)
		return ok
	case reflect.Bool:
		_, ok := token.(bool)
		return ok
	case reflect.Int, reflect.Float64:
		_, ok := token.(float64)
		return ok // The typed decoder rejects fractional/overflowing integers.
	default:
		return false
	}
}
