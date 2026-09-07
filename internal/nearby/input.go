package nearby

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"math"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/kohei321dev/spot-diggz/internal/facility"
	"github.com/kohei321dev/spot-diggz/internal/geocoding"
)

const (
	DefaultLimit    = 5
	MaxLimit        = 10
	DefaultRadiusKm = 10.0
	MinRadiusKm     = 0.1
	MaxRadiusKm     = 50.0
	DistanceSort    = "distance"
)

var ErrInvalidInput = errors.New("invalid nearby search input")

type Input struct {
	Query    string         `json:"query"`
	Genre    facility.Genre `json:"genre,omitempty"`
	Limit    int            `json:"limit"`
	RadiusKm float64        `json:"radiusKm"`
	Sort     string         `json:"sort"`
}

// DecodeInput rejects duplicate keys and explicit nulls as well as unknown fields.
// Values are never included in diagnostics, since query may be a private location.
func DecodeInput(body []byte) (Input, error) {
	if !utf8.Valid(body) {
		return Input{}, ErrInvalidInput
	}
	input := Input{Limit: DefaultLimit, RadiusKm: DefaultRadiusKm, Sort: DistanceSort}
	decoder := json.NewDecoder(bytes.NewReader(body))
	start, err := decoder.Token()
	if err != nil || start != json.Delim('{') {
		return Input{}, ErrInvalidInput
	}
	seen := make(map[string]bool)
	for decoder.More() {
		keyToken, err := decoder.Token()
		if err != nil {
			return Input{}, ErrInvalidInput
		}
		key, ok := keyToken.(string)
		if !ok || seen[key] {
			return Input{}, ErrInvalidInput
		}
		seen[key] = true
		var value json.RawMessage
		if err := decoder.Decode(&value); err != nil || bytes.Equal(bytes.TrimSpace(value), []byte("null")) {
			return Input{}, ErrInvalidInput
		}
		switch key {
		case "query":
			err = json.Unmarshal(value, &input.Query)
		case "genre":
			err = json.Unmarshal(value, &input.Genre)
		case "limit":
			err = json.Unmarshal(value, &input.Limit)
		case "radiusKm":
			err = json.Unmarshal(value, &input.RadiusKm)
		case "sort":
			err = json.Unmarshal(value, &input.Sort)
		default:
			return Input{}, ErrInvalidInput
		}
		if err != nil {
			return Input{}, ErrInvalidInput
		}
		if key == "genre" && input.Genre == "" {
			return Input{}, ErrInvalidInput
		}
	}
	if end, err := decoder.Token(); err != nil || end != json.Delim('}') {
		return Input{}, ErrInvalidInput
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		return Input{}, ErrInvalidInput
	}
	if strings.IndexFunc(input.Query, unicode.IsControl) >= 0 {
		return Input{}, ErrInvalidInput
	}
	input.Query = strings.TrimSpace(input.Query)
	return input, input.Validate()
}

func (input Input) Validate() error {
	if input.Query == "" || len([]rune(input.Query)) > geocoding.MaxQueryLength || strings.IndexFunc(input.Query, unicode.IsControl) >= 0 ||
		(input.Genre != "" && input.Genre != facility.GenreSkatepark && input.Genre != facility.GenreStreet) ||
		input.Limit < 1 || input.Limit > MaxLimit || math.IsNaN(input.RadiusKm) || math.IsInf(input.RadiusKm, 0) ||
		input.RadiusKm < MinRadiusKm || input.RadiusKm > MaxRadiusKm || input.Sort != DistanceSort {
		return ErrInvalidInput
	}
	return nil
}
