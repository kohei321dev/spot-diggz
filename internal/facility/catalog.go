package facility

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"sort"
	"strings"
	"time"
)

var (
	ErrDuplicateID = errors.New("duplicate facility id")
	ErrInvalidData = errors.New("invalid facility data")
	ErrNotFound    = errors.New("facility not found")
)

var supportedPrefectures = map[string]struct{}{
	"大阪府":  {},
	"兵庫県":  {},
	"和歌山県": {},
	"奈良県":  {},
	"徳島県":  {},
}

const (
	youTubeProvider      = "youtube"
	youTubeCanonicalHost = "www.youtube.com"
)

type catalogFile struct {
	Facilities []Facility `json:"facilities"`
}

// Catalog is an immutable, read-only snapshot loaded when the application starts.
type Catalog struct {
	facilities map[string]Facility
}

func LoadCatalogFile(path string) (*Catalog, error) {
	return loadCatalogFile(path, nil)
}

// LoadCatalogFileAt rejects verification timestamps later than asOf.
func LoadCatalogFileAt(path string, asOf time.Time) (*Catalog, error) {
	return loadCatalogFile(path, &asOf)
}

func loadCatalogFile(path string, asOf *time.Time) (*Catalog, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open facility catalog %q: %w", path, err)
	}
	defer file.Close()

	return loadCatalog(file, asOf)
}

func LoadCatalog(reader io.Reader) (*Catalog, error) {
	return loadCatalog(reader, nil)
}

// LoadCatalogAt rejects verification timestamps later than asOf.
func LoadCatalogAt(reader io.Reader, asOf time.Time) (*Catalog, error) {
	return loadCatalog(reader, &asOf)
}

func loadCatalog(reader io.Reader, asOf *time.Time) (*Catalog, error) {
	var data catalogFile
	decoder := json.NewDecoder(reader)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&data); err != nil {
		return nil, fmt.Errorf("decode facility catalog: %w", err)
	}

	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			return nil, fmt.Errorf("%w: multiple JSON values", ErrInvalidData)
		}
		return nil, fmt.Errorf("decode trailing facility catalog data: %w", err)
	}

	return newCatalog(data.Facilities, asOf)
}

func NewCatalog(facilities []Facility) (*Catalog, error) {
	return newCatalog(facilities, nil)
}

// NewCatalogAt rejects verification timestamps later than asOf.
func NewCatalogAt(facilities []Facility, asOf time.Time) (*Catalog, error) {
	return newCatalog(facilities, &asOf)
}

func newCatalog(facilities []Facility, asOf *time.Time) (*Catalog, error) {
	byID := make(map[string]Facility, len(facilities))
	for _, item := range facilities {
		if err := validateFacility(item); err != nil {
			return nil, err
		}
		if asOf != nil {
			if err := validateFacilityAt(item, *asOf); err != nil {
				return nil, err
			}
		}
		if _, exists := byID[item.ID]; exists {
			return nil, fmt.Errorf("%w: %s", ErrDuplicateID, item.ID)
		}
		byID[item.ID] = item
	}

	return &Catalog{facilities: byID}, nil
}

func (c *Catalog) List(activity string) []Facility {
	items := make([]Facility, 0, len(c.facilities))
	for _, item := range c.facilities {
		if activity != "" && !containsIgnoreCase(item.Activities, activity) {
			continue
		}
		items = append(items, item)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].ID < items[j].ID })
	return items
}

func (c *Catalog) Find(id string) (Facility, error) {
	item, ok := c.facilities[id]
	if !ok {
		return Facility{}, ErrNotFound
	}
	return item, nil
}

func validateFacility(item Facility) error {
	if strings.TrimSpace(item.ID) == "" || strings.TrimSpace(item.Name) == "" || strings.TrimSpace(item.Address) == "" {
		return fmt.Errorf("%w: facilityId, name, and address are required", ErrInvalidData)
	}
	if strings.TrimSpace(item.Prefecture) == "" || strings.TrimSpace(item.Municipality) == "" {
		return fmt.Errorf("%w: prefecture and municipality are required for %s", ErrInvalidData, item.ID)
	}
	if _, supported := supportedPrefectures[item.Prefecture]; !supported {
		return fmt.Errorf("%w: prefecture is outside the MVP scope for %s", ErrInvalidData, item.ID)
	}
	if item.Location.Latitude < -90 || item.Location.Latitude > 90 || item.Location.Longitude < -180 || item.Location.Longitude > 180 {
		return fmt.Errorf("%w: location is out of range for %s", ErrInvalidData, item.ID)
	}
	if len(item.Activities) == 0 {
		return fmt.Errorf("%w: activities are required for %s", ErrInvalidData, item.ID)
	}
	if len(item.Hours) == 0 {
		return fmt.Errorf("%w: operating hours are required for %s", ErrInvalidData, item.ID)
	}
	if err := validateOperatingHours(item.ID, item.Hours); err != nil {
		return err
	}
	if err := validateAvailability(item); err != nil {
		return err
	}
	if err := validateClosurePeriods(item.ID, item.ClosurePeriods); err != nil {
		return err
	}
	if strings.TrimSpace(item.Price) == "" || strings.TrimSpace(item.Reservation) == "" {
		return fmt.Errorf("%w: price and reservation guidance are required for %s", ErrInvalidData, item.ID)
	}
	if len(item.Features) == 0 || len(item.Rules) == 0 {
		return fmt.Errorf("%w: features and rules are required for %s", ErrInvalidData, item.ID)
	}
	if err := validateEnglishTranslation(item); err != nil {
		return err
	}
	if item.Status != "verified" {
		return fmt.Errorf("%w: only verified facilities can be published: %s", ErrInvalidData, item.ID)
	}
	if strings.TrimSpace(item.SourceType) == "" || strings.TrimSpace(item.Confidence) == "" {
		return fmt.Errorf("%w: sourceType and confidence are required for %s", ErrInvalidData, item.ID)
	}
	parsedURL, err := url.ParseRequestURI(item.SourceURL)
	if err != nil || (parsedURL.Scheme != "https" && parsedURL.Scheme != "http") || parsedURL.Host == "" {
		return fmt.Errorf("%w: sourceUrl must be an absolute HTTP(S) URL for %s", ErrInvalidData, item.ID)
	}
	if err := validateFacilityMedia(item.ID, item.Media); err != nil {
		return err
	}
	if err := validateSocialLinks(item.ID, item.SocialLinks); err != nil {
		return err
	}
	if item.VerifiedAt.IsZero() {
		return fmt.Errorf("%w: verifiedAt is required for %s", ErrInvalidData, item.ID)
	}
	if item.DynamicVerifiedAt.IsZero() || item.StableVerifiedAt.IsZero() {
		return fmt.Errorf("%w: dynamicVerifiedAt and stableVerifiedAt are required for %s", ErrInvalidData, item.ID)
	}
	return nil
}

func validateAvailability(item Facility) error {
	if item.GeneralUseStatus != "" && item.GeneralUseStatus != GeneralUseRegular &&
		item.GeneralUseStatus != GeneralUseLimited && item.GeneralUseStatus != GeneralUseScheduleCheckRequired {
		return fmt.Errorf("%w: unsupported generalUseStatus %q for %s", ErrInvalidData, item.GeneralUseStatus, item.ID)
	}
	if item.HoursBasis != "" && item.HoursBasis != HoursBasisOfficial && item.HoursBasis != HoursBasisConservative {
		return fmt.Errorf("%w: unsupported hoursBasis %q for %s", ErrInvalidData, item.HoursBasis, item.ID)
	}
	if item.GeneralUseStatus == GeneralUseScheduleCheckRequired && strings.TrimSpace(item.AvailabilityNote) == "" {
		return fmt.Errorf("%w: availabilityNote is required when generalUseStatus is %q for %s", ErrInvalidData, GeneralUseScheduleCheckRequired, item.ID)
	}
	return nil
}

func validateFacilityMedia(facilityID string, media *FacilityMedia) error {
	if media == nil {
		return nil
	}
	if media.YouTube == nil {
		return fmt.Errorf("%w: media.youtube is required when media is set for %s", ErrInvalidData, facilityID)
	}

	video := media.YouTube
	if video.Provider != youTubeProvider {
		return fmt.Errorf("%w: unsupported media provider %q for %s", ErrInvalidData, video.Provider, facilityID)
	}
	if !isYouTubeVideoID(video.VideoID) {
		return fmt.Errorf("%w: invalid YouTube videoId for %s", ErrInvalidData, facilityID)
	}
	if strings.TrimSpace(video.Title) == "" || strings.TrimSpace(video.SelectionReason) == "" {
		return fmt.Errorf("%w: YouTube title and selectionReason are required for %s", ErrInvalidData, facilityID)
	}
	if !isCanonicalYouTubeWatchURL(video.SourceURL, video.VideoID) {
		return fmt.Errorf("%w: YouTube sourceUrl must be the canonical HTTPS watch URL for %s", ErrInvalidData, facilityID)
	}
	if video.SelectedAt.IsZero() || video.VerifiedAt.IsZero() {
		return fmt.Errorf("%w: YouTube selectedAt and verifiedAt are required for %s", ErrInvalidData, facilityID)
	}
	if video.SelectedAt.After(video.VerifiedAt) {
		return fmt.Errorf("%w: YouTube selectedAt must not be later than verifiedAt for %s", ErrInvalidData, facilityID)
	}
	return nil
}

func validateSocialLinks(facilityID string, socialLinks []SocialLink) error {
	if len(socialLinks) > 2 {
		return fmt.Errorf("%w: at most one Instagram and one X profile are allowed for %s", ErrInvalidData, facilityID)
	}

	seenPlatforms := make(map[SocialPlatform]struct{}, len(socialLinks))
	for _, link := range socialLinks {
		if link.Platform != SocialPlatformInstagram && link.Platform != SocialPlatformX {
			return fmt.Errorf("%w: unsupported social platform %q for %s", ErrInvalidData, link.Platform, facilityID)
		}
		if _, exists := seenPlatforms[link.Platform]; exists {
			return fmt.Errorf("%w: duplicate social platform %q for %s", ErrInvalidData, link.Platform, facilityID)
		}
		seenPlatforms[link.Platform] = struct{}{}

		if !isCanonicalSocialProfileURL(link.Platform, link.URL) {
			return fmt.Errorf("%w: invalid %s profile URL for %s", ErrInvalidData, link.Platform, facilityID)
		}
		if link.VerifiedAt.IsZero() {
			return fmt.Errorf("%w: social verifiedAt is required for %s", ErrInvalidData, facilityID)
		}
	}
	return nil
}

func isYouTubeVideoID(value string) bool {
	if len(value) != 11 {
		return false
	}
	for _, character := range value {
		if (character < 'a' || character > 'z') &&
			(character < 'A' || character > 'Z') &&
			(character < '0' || character > '9') &&
			character != '_' && character != '-' {
			return false
		}
	}
	return true
}

func isCanonicalYouTubeWatchURL(value string, videoID string) bool {
	parsedURL, ok := parseCanonicalHTTPSURL(value)
	if !ok || parsedURL.Host != youTubeCanonicalHost || parsedURL.Path != "/watch" || parsedURL.Fragment != "" {
		return false
	}
	return parsedURL.RawQuery == "v="+videoID
}

func isCanonicalSocialProfileURL(platform SocialPlatform, value string) bool {
	parsedURL, ok := parseCanonicalHTTPSURL(value)
	if !ok || parsedURL.RawQuery != "" || parsedURL.Fragment != "" {
		return false
	}

	hostname := parsedURL.Hostname()
	profileHandle := strings.Trim(parsedURL.Path, "/")
	if profileHandle == "" || strings.Contains(profileHandle, "/") {
		return false
	}

	switch platform {
	case SocialPlatformInstagram:
		return (hostname == "instagram.com" || hostname == "www.instagram.com") && isInstagramProfileHandle(profileHandle)
	case SocialPlatformX:
		return (hostname == "x.com" || hostname == "www.x.com") && isXProfileHandle(profileHandle)
	default:
		return false
	}
}

func parseCanonicalHTTPSURL(value string) (*url.URL, bool) {
	parsedURL, err := url.ParseRequestURI(value)
	if err != nil || parsedURL.Scheme != "https" || parsedURL.Host == "" || parsedURL.User != nil || parsedURL.Port() != "" {
		return nil, false
	}
	return parsedURL, true
}

func isInstagramProfileHandle(value string) bool {
	for _, character := range value {
		if (character < 'a' || character > 'z') &&
			(character < 'A' || character > 'Z') &&
			(character < '0' || character > '9') &&
			character != '_' && character != '.' {
			return false
		}
	}
	return true
}

func isXProfileHandle(value string) bool {
	if len(value) > 15 {
		return false
	}
	for _, character := range value {
		if (character < 'a' || character > 'z') &&
			(character < 'A' || character > 'Z') &&
			(character < '0' || character > '9') &&
			character != '_' {
			return false
		}
	}
	return true
}

func validateFacilityAt(item Facility, asOf time.Time) error {
	if asOf.IsZero() {
		return fmt.Errorf("%w: validation reference time is required", ErrInvalidData)
	}

	type verificationTime struct {
		name  string
		value time.Time
	}
	verificationTimes := []verificationTime{
		{name: "verifiedAt", value: item.VerifiedAt},
		{name: "dynamicVerifiedAt", value: item.DynamicVerifiedAt},
		{name: "stableVerifiedAt", value: item.StableVerifiedAt},
	}
	if item.UpdatedAt != nil {
		verificationTimes = append(verificationTimes, verificationTime{name: "updatedAt", value: *item.UpdatedAt})
	}
	if item.Media != nil && item.Media.YouTube != nil {
		verificationTimes = append(verificationTimes,
			verificationTime{name: "media.youtube.selectedAt", value: item.Media.YouTube.SelectedAt},
			verificationTime{name: "media.youtube.verifiedAt", value: item.Media.YouTube.VerifiedAt},
		)
	}
	for index, link := range item.SocialLinks {
		verificationTimes = append(verificationTimes, verificationTime{
			name:  fmt.Sprintf("socialLinks[%d].verifiedAt", index),
			value: link.VerifiedAt,
		})
	}

	for _, candidate := range verificationTimes {
		if candidate.value.After(asOf) {
			return fmt.Errorf("%w: %s must not be later than the validation time for %s", ErrInvalidData, candidate.name, item.ID)
		}
	}
	return nil
}

func validateEnglishTranslation(item Facility) error {
	translation := item.EnglishTranslation
	if strings.TrimSpace(translation.Name) == "" ||
		strings.TrimSpace(translation.Address) == "" ||
		strings.TrimSpace(translation.Price) == "" ||
		strings.TrimSpace(translation.Reservation) == "" ||
		strings.TrimSpace(translation.AccessNotes) == "" {
		return fmt.Errorf("%w: complete English name, address, price, reservation, and access notes are required for %s", ErrInvalidData, item.ID)
	}
	if len(item.ScheduleNotes) == 0 || len(translation.ScheduleNotes) != len(item.ScheduleNotes) || containsBlank(translation.ScheduleNotes) {
		return fmt.Errorf("%w: every schedule note requires an English translation for %s", ErrInvalidData, item.ID)
	}
	if len(translation.Rules) != len(item.Rules) || containsBlank(translation.Rules) {
		return fmt.Errorf("%w: every rule requires an English translation for %s", ErrInvalidData, item.ID)
	}
	return nil
}

func validateClosurePeriods(facilityID string, periods []ClosurePeriod) error {
	for index, period := range periods {
		if strings.TrimSpace(period.Reason) == "" {
			return fmt.Errorf("%w: closure period %d requires a reason for %s", ErrInvalidData, index, facilityID)
		}

		switch period.Type {
		case ClosurePeriodOneTime:
			start, startOK := parseDate(period.Start)
			end, endOK := parseDate(period.End)
			if !startOK || !endOK {
				return fmt.Errorf("%w: one-time closure period %d must use YYYY-MM-DD dates for %s", ErrInvalidData, index, facilityID)
			}
			if end.Before(start) {
				return fmt.Errorf("%w: closure period %d ends before it starts for %s", ErrInvalidData, index, facilityID)
			}
		case ClosurePeriodAnnual:
			if !isMonthDay(period.Start) || !isMonthDay(period.End) {
				return fmt.Errorf("%w: annual closure period %d must use MM-DD dates for %s", ErrInvalidData, index, facilityID)
			}
		default:
			return fmt.Errorf("%w: unsupported closure period type %q for %s", ErrInvalidData, period.Type, facilityID)
		}
	}
	return nil
}

func parseDate(value string) (time.Time, bool) {
	parsed, err := time.Parse("2006-01-02", value)
	return parsed, err == nil && parsed.Format("2006-01-02") == value
}

func isMonthDay(value string) bool {
	parsed, err := time.Parse("2006-01-02", "2000-"+value)
	return err == nil && parsed.Format("01-02") == value
}

func containsBlank(values []string) bool {
	for _, value := range values {
		if strings.TrimSpace(value) == "" {
			return true
		}
	}
	return false
}

func validateOperatingHours(facilityID string, hours []OperatingHours) error {
	validDays := map[string]bool{
		"daily": true, "weekday": true, "weekend": true,
		"monday": true, "tuesday": true, "wednesday": true, "thursday": true,
		"friday": true, "saturday": true, "sunday": true,
	}
	hasOpenWindow := false
	for _, period := range hours {
		if !validDays[strings.ToLower(strings.TrimSpace(period.Day))] {
			return fmt.Errorf("%w: invalid operating-hours day %q for %s", ErrInvalidData, period.Day, facilityID)
		}
		if period.Closed {
			continue
		}
		if !isClock(period.Opens, false) || !isClock(period.Closes, true) || period.Opens == period.Closes {
			return fmt.Errorf("%w: invalid operating-hours window for %s", ErrInvalidData, facilityID)
		}
		hasOpenWindow = true
	}
	if !hasOpenWindow {
		return fmt.Errorf("%w: at least one open operating-hours window is required for %s", ErrInvalidData, facilityID)
	}
	return nil
}

func isClock(value string, allowEndOfDay bool) bool {
	if value == "24:00" {
		return allowEndOfDay
	}
	_, err := time.Parse("15:04", value)
	return err == nil
}

func containsIgnoreCase(values []string, target string) bool {
	for _, value := range values {
		if strings.EqualFold(value, target) {
			return true
		}
	}
	return false
}
