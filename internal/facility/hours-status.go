package facility

import (
	"fmt"
	"strings"
)

type HoursStatus string

const (
	HoursKnown         HoursStatus = "known"
	HoursNotApplicable HoursStatus = "not_applicable"
	HoursUnknown       HoursStatus = "unknown"
)

func HasKnownOperatingHours(spot Facility) bool {
	return spot.HoursStatus == "" || spot.HoursStatus == HoursKnown
}

func validateHoursStatus(spot Facility) error {
	if spot.HoursSourceURL != "" {
		if _, ok := parseCanonicalHTTPSURL(spot.HoursSourceURL); !ok {
			return fmt.Errorf("%w: hoursSourceUrl must use canonical HTTPS", ErrInvalidData)
		}
	}
	if HasKnownOperatingHours(spot) {
		return validateOperatingHours(spot.ID, spot.Hours)
	}
	if spot.HoursStatus != HoursNotApplicable && spot.HoursStatus != HoursUnknown {
		return fmt.Errorf("%w: unsupported hoursStatus", ErrInvalidData)
	}
	if len(spot.Hours) != 0 || spot.HoursBasis != "" {
		return fmt.Errorf("%w: non-known hours cannot include a timetable or hoursBasis", ErrInvalidData)
	}
	if spot.Genre == "" || strings.TrimSpace(spot.AvailabilityNote) == "" || strings.TrimSpace(spot.EnglishTranslation.AvailabilityNote) == "" {
		return fmt.Errorf("%w: non-known hours require genre and bilingual availabilityNote", ErrInvalidData)
	}
	if spot.HoursStatus == HoursUnknown {
		if spot.GeneralUseStatus != GeneralUseScheduleCheckRequired {
			return fmt.Errorf("%w: unknown hours require schedule_check_required", ErrInvalidData)
		}
		return nil
	}
	if spot.Genre != GenreStreet || spot.HoursSourceURL == "" ||
		(spot.GeneralUseStatus != GeneralUseRegular && spot.GeneralUseStatus != GeneralUseLimited) {
		return fmt.Errorf("%w: not_applicable hours require street, hours evidence and confirmed general use", ErrInvalidData)
	}
	return nil
}
