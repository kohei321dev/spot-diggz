package session

import (
	"errors"
	"testing"
)

func TestInputValidateAcceptsSupportedSelections(t *testing.T) {
	latitude := 34.7025
	longitude := 135.4960
	input := Input{
		Purpose:          PurposeBasics,
		Mood:             MoodFocused,
		Level:            LevelBeginner,
		AvailableMinutes: 120,
		Transport:        TransportPublicTransit,
		Origin: Origin{
			Mode:      OriginCurrentLocation,
			Latitude:  &latitude,
			Longitude: &longitude,
		},
	}

	if err := input.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
}

func TestInputValidateRejectsUnsupportedOrMissingValues(t *testing.T) {
	latitude := 34.7025
	longitude := 135.4960
	validInput := Input{
		Purpose:          PurposeBasics,
		Mood:             MoodFocused,
		Level:            LevelBeginner,
		AvailableMinutes: 120,
		Transport:        TransportPublicTransit,
		Origin: Origin{
			Mode:      OriginSpecifiedLocation,
			Latitude:  &latitude,
			Longitude: &longitude,
		},
	}

	tests := []struct {
		name   string
		modify func(*Input)
	}{
		{name: "purpose", modify: func(input *Input) { input.Purpose = "unknown" }},
		{name: "mood", modify: func(input *Input) { input.Mood = "unknown" }},
		{name: "level", modify: func(input *Input) { input.Level = "unknown" }},
		{name: "duration", modify: func(input *Input) { input.AvailableMinutes = 90 }},
		{name: "transport", modify: func(input *Input) { input.Transport = "unknown" }},
		{name: "origin mode", modify: func(input *Input) { input.Origin.Mode = "unknown" }},
		{name: "missing latitude", modify: func(input *Input) { input.Origin.Latitude = nil }},
		{name: "latitude range", modify: func(input *Input) { invalid := 91.0; input.Origin.Latitude = &invalid }},
		{name: "longitude range", modify: func(input *Input) { invalid := 181.0; input.Origin.Longitude = &invalid }},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			input := validInput
			test.modify(&input)
			if err := input.Validate(); !errors.Is(err, ErrInvalidInput) {
				t.Fatalf("Validate() error = %v, want ErrInvalidInput", err)
			}
		})
	}
}
