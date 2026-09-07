package facility

import "time"

// IsNearbySearchable is shared by search, readiness and release checks so unknown
// availability cannot pass a looser operational check than the actual API.
// It expects a validated catalog record and does not assert that it is open now.
func IsNearbySearchable(spot Facility, asOf time.Time) bool {
	return containsIgnoreCase(spot.Activities, "skateboard") &&
		(spot.Genre == GenreSkatepark || spot.Genre == GenreStreet) &&
		spot.SkatingPermissionSourceURL != "" && spot.Status == "verified" &&
		spot.HoursStatus != HoursUnknown &&
		spot.GeneralUseStatus != GeneralUseScheduleCheckRequired &&
		IsDynamicInformationFresh(spot.DynamicVerifiedAt, asOf) &&
		IsStableInformationFresh(spot.StableVerifiedAt, asOf)
}
