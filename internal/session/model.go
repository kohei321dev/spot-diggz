package session

import (
	"errors"
	"fmt"
	"math"
)

type Purpose string

const (
	PurposeBasics     Purpose = "basics"
	PurposeStreet     Purpose = "street"
	PurposeTransition Purpose = "transition"
)

type Mood string

const (
	MoodFocused   Mood = "focused"
	MoodEasygoing Mood = "easygoing"
	MoodChallenge Mood = "challenge"
)

type Level string

const (
	LevelBeginner     Level = "beginner"
	LevelReturning    Level = "returning"
	LevelIntermediate Level = "intermediate"
)

type Transport string

const (
	TransportPublicTransit Transport = "public_transit"
	TransportCar           Transport = "car"
	TransportBicycle       Transport = "bicycle"
	TransportWalk          Transport = "walk"
)

type OriginMode string

const (
	OriginCurrentLocation   OriginMode = "current_location"
	OriginSpecifiedLocation OriginMode = "specified_location"
)

type Origin struct {
	Mode      OriginMode `json:"mode"`
	Latitude  *float64   `json:"latitude"`
	Longitude *float64   `json:"longitude"`
}

type Input struct {
	Purpose          Purpose   `json:"purpose"`
	Mood             Mood      `json:"mood"`
	Level            Level     `json:"level"`
	AvailableMinutes int       `json:"availableMinutes"`
	Transport        Transport `json:"transport"`
	Origin           Origin    `json:"origin"`
}

var ErrInvalidInput = errors.New("invalid session input")

func (input Input) Validate() error {
	if !isPurpose(input.Purpose) {
		return invalidField("purpose")
	}
	if !isMood(input.Mood) {
		return invalidField("mood")
	}
	if !isLevel(input.Level) {
		return invalidField("level")
	}
	if !isAvailableMinutes(input.AvailableMinutes) {
		return invalidField("availableMinutes")
	}
	if !isTransport(input.Transport) {
		return invalidField("transport")
	}
	if input.Origin.Mode != OriginCurrentLocation && input.Origin.Mode != OriginSpecifiedLocation {
		return invalidField("origin.mode")
	}
	if input.Origin.Latitude == nil || input.Origin.Longitude == nil {
		return invalidField("origin coordinates")
	}
	if !isFinite(*input.Origin.Latitude) || *input.Origin.Latitude < -90 || *input.Origin.Latitude > 90 {
		return invalidField("origin.latitude")
	}
	if !isFinite(*input.Origin.Longitude) || *input.Origin.Longitude < -180 || *input.Origin.Longitude > 180 {
		return invalidField("origin.longitude")
	}

	return nil
}

func invalidField(field string) error {
	return fmt.Errorf("%w: %s", ErrInvalidInput, field)
}

func isPurpose(value Purpose) bool {
	return value == PurposeBasics || value == PurposeStreet || value == PurposeTransition
}

func isMood(value Mood) bool {
	return value == MoodFocused || value == MoodEasygoing || value == MoodChallenge
}

func isLevel(value Level) bool {
	return value == LevelBeginner || value == LevelReturning || value == LevelIntermediate
}

func isAvailableMinutes(value int) bool {
	return value == 60 || value == 120 || value == 180 || value == 240
}

func isTransport(value Transport) bool {
	return value == TransportPublicTransit || value == TransportCar || value == TransportBicycle || value == TransportWalk
}

func isFinite(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0)
}
