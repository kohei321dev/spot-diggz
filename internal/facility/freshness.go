package facility

import "time"

const (
	DynamicInformationFreshnessWindow = 30 * 24 * time.Hour
	StableInformationFreshnessWindow  = 180 * 24 * time.Hour
)

func IsDynamicInformationFresh(verifiedAt, asOf time.Time) bool {
	return isInformationFresh(verifiedAt, asOf, DynamicInformationFreshnessWindow)
}

func IsStableInformationFresh(verifiedAt, asOf time.Time) bool {
	return isInformationFresh(verifiedAt, asOf, StableInformationFreshnessWindow)
}

func isInformationFresh(verifiedAt, asOf time.Time, freshnessWindow time.Duration) bool {
	if verifiedAt.IsZero() || asOf.IsZero() || verifiedAt.After(asOf) {
		return false
	}
	return asOf.Sub(verifiedAt) <= freshnessWindow
}
