package spot

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
	"time"
)

type SdzVisibility string

const (
	SdzVisibilityPublic   SdzVisibility = "public"
	SdzVisibilityPrivate  SdzVisibility = "private"
	SdzVisibilityUnlisted SdzVisibility = "unlisted"
)

var validVisibilities = map[SdzVisibility]struct{}{
	SdzVisibilityPublic:   {},
	SdzVisibilityPrivate:  {},
	SdzVisibilityUnlisted: {},
}

type SdzLocation struct {
	Lat float64 `json:"lat"`
	Lng float64 `json:"lng"`
}

type SdzSpot struct {
	SdzSpotID     string        `json:"spotId"`
	Name          string        `json:"name"`
	Description   string        `json:"description,omitempty"`
	SdzLocation   SdzLocation   `json:"location"`
	Tags          []string      `json:"tags"`
	SdzVisibility SdzVisibility `json:"visibility"`
	CreatedAt     time.Time     `json:"createdAt"`
	UpdatedAt     time.Time     `json:"updatedAt"`
	DeletedAt     *time.Time    `json:"deletedAt,omitempty"`
}

type SdzCreateSpotInput struct {
	Name          string        `json:"name"`
	Description   string        `json:"description,omitempty"`
	SdzLocation   *SdzLocation  `json:"location"`
	Tags          []string      `json:"tags,omitempty"`
	SdzVisibility SdzVisibility `json:"visibility,omitempty"`
}

type SdzUpdateSpotInput struct {
	Name          *string        `json:"name,omitempty"`
	Description   *string        `json:"description,omitempty"`
	SdzLocation   *SdzLocation   `json:"location,omitempty"`
	Tags          *[]string      `json:"tags,omitempty"`
	SdzVisibility *SdzVisibility `json:"visibility,omitempty"`
}

type SdzListFilter struct {
	SdzBBox       *SdzBBox
	Tags          []string
	SdzVisibility *SdzVisibility
}

type SdzBBox struct {
	MinLng float64
	MinLat float64
	MaxLng float64
	MaxLat float64
}

type SdzValidationError struct {
	Message string
}

func (e SdzValidationError) Error() string {
	return e.Message
}

func NewSdzSpot(id string, input SdzCreateSpotInput, now time.Time) (SdzSpot, error) {
	name, err := normalizeName(input.Name)
	if err != nil {
		return SdzSpot{}, err
	}
	if input.SdzLocation == nil {
		return SdzSpot{}, SdzValidationError{Message: "location is required"}
	}
	if err := validateLocation(*input.SdzLocation); err != nil {
		return SdzSpot{}, err
	}
	visibility, err := normalizeVisibility(input.SdzVisibility)
	if err != nil {
		return SdzSpot{}, err
	}

	return SdzSpot{
		SdzSpotID:     id,
		Name:          name,
		Description:   strings.TrimSpace(input.Description),
		SdzLocation:   *input.SdzLocation,
		Tags:          normalizeTags(input.Tags),
		SdzVisibility: visibility,
		CreatedAt:     now,
		UpdatedAt:     now,
	}, nil
}

func SdzApplyUpdate(current SdzSpot, input SdzUpdateSpotInput, now time.Time) (SdzSpot, error) {
	if input.Name != nil {
		name, err := normalizeName(*input.Name)
		if err != nil {
			return SdzSpot{}, err
		}
		current.Name = name
	}
	if input.Description != nil {
		current.Description = strings.TrimSpace(*input.Description)
	}
	if input.SdzLocation != nil {
		if err := validateLocation(*input.SdzLocation); err != nil {
			return SdzSpot{}, err
		}
		current.SdzLocation = *input.SdzLocation
	}
	if input.Tags != nil {
		current.Tags = normalizeTags(*input.Tags)
	}
	if input.SdzVisibility != nil {
		visibility, err := normalizeVisibility(*input.SdzVisibility)
		if err != nil {
			return SdzSpot{}, err
		}
		current.SdzVisibility = visibility
	}
	current.UpdatedAt = now
	return current, nil
}

func SdzParseBBox(raw string) (*SdzBBox, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, nil
	}
	parts := strings.Split(raw, ",")
	if len(parts) != 4 {
		return nil, SdzValidationError{Message: "bbox must be minLng,minLat,maxLng,maxLat"}
	}
	values := make([]float64, 4)
	for i, part := range parts {
		value, err := strconv.ParseFloat(strings.TrimSpace(part), 64)
		if err != nil {
			return nil, SdzValidationError{Message: "bbox contains a non-number value"}
		}
		values[i] = value
	}
	bbox := SdzBBox{
		MinLng: values[0],
		MinLat: values[1],
		MaxLng: values[2],
		MaxLat: values[3],
	}
	if err := validateBBox(bbox); err != nil {
		return nil, err
	}
	return &bbox, nil
}

func SdzParseVisibility(raw string) (*SdzVisibility, error) {
	if strings.TrimSpace(raw) == "" {
		visibility := SdzVisibilityPublic
		return &visibility, nil
	}
	visibility, err := normalizeVisibility(SdzVisibility(strings.TrimSpace(raw)))
	if err != nil {
		return nil, err
	}
	return &visibility, nil
}

func SdzSplitTags(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	return normalizeTags(strings.Split(raw, ","))
}

func (b SdzBBox) Contains(location SdzLocation) bool {
	return location.Lng >= b.MinLng &&
		location.Lng <= b.MaxLng &&
		location.Lat >= b.MinLat &&
		location.Lat <= b.MaxLat
}

func NewSdzID() (string, error) {
	var bytes [12]byte
	if _, err := rand.Read(bytes[:]); err != nil {
		return "", fmt.Errorf("generate spot id: %w", err)
	}
	return "spot_" + hex.EncodeToString(bytes[:]), nil
}

func normalizeName(value string) (string, error) {
	name := strings.TrimSpace(value)
	if name == "" {
		return "", SdzValidationError{Message: "name is required"}
	}
	if len([]rune(name)) > 120 {
		return "", SdzValidationError{Message: "name must be 120 characters or fewer"}
	}
	return name, nil
}

func normalizeVisibility(value SdzVisibility) (SdzVisibility, error) {
	if value == "" {
		return SdzVisibilityPublic, nil
	}
	value = SdzVisibility(strings.TrimSpace(string(value)))
	if _, ok := validVisibilities[value]; !ok {
		return "", SdzValidationError{Message: "visibility must be public, private, or unlisted"}
	}
	return value, nil
}

func normalizeTags(tags []string) []string {
	seen := map[string]struct{}{}
	result := make([]string, 0, len(tags))
	for _, tag := range tags {
		normalized := strings.TrimSpace(tag)
		if normalized == "" {
			continue
		}
		key := strings.ToLower(normalized)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, normalized)
	}
	return result
}

func validateLocation(location SdzLocation) error {
	if location.Lat < -90 || location.Lat > 90 {
		return SdzValidationError{Message: "location.lat must be between -90 and 90"}
	}
	if location.Lng < -180 || location.Lng > 180 {
		return SdzValidationError{Message: "location.lng must be between -180 and 180"}
	}
	return nil
}

func validateBBox(bbox SdzBBox) error {
	if err := validateLocation(SdzLocation{Lat: bbox.MinLat, Lng: bbox.MinLng}); err != nil {
		return err
	}
	if err := validateLocation(SdzLocation{Lat: bbox.MaxLat, Lng: bbox.MaxLng}); err != nil {
		return err
	}
	if bbox.MinLng > bbox.MaxLng {
		return SdzValidationError{Message: "bbox minLng must be less than or equal to maxLng"}
	}
	if bbox.MinLat > bbox.MaxLat {
		return SdzValidationError{Message: "bbox minLat must be less than or equal to maxLat"}
	}
	return nil
}
