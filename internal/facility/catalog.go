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

type catalogFile struct {
	Facilities []Facility `json:"facilities"`
}

// Catalog is an immutable, read-only snapshot loaded when the application starts.
type Catalog struct {
	facilities map[string]Facility
}

func LoadCatalogFile(path string) (*Catalog, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open facility catalog %q: %w", path, err)
	}
	defer file.Close()

	return LoadCatalog(file)
}

func LoadCatalog(reader io.Reader) (*Catalog, error) {
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

	return NewCatalog(data.Facilities)
}

func NewCatalog(facilities []Facility) (*Catalog, error) {
	byID := make(map[string]Facility, len(facilities))
	for _, item := range facilities {
		if err := validateFacility(item); err != nil {
			return nil, err
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
	if strings.TrimSpace(item.Price) == "" || strings.TrimSpace(item.Reservation) == "" {
		return fmt.Errorf("%w: price and reservation guidance are required for %s", ErrInvalidData, item.ID)
	}
	if len(item.Features) == 0 || len(item.Rules) == 0 {
		return fmt.Errorf("%w: features and rules are required for %s", ErrInvalidData, item.ID)
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
	if item.VerifiedAt.IsZero() {
		return fmt.Errorf("%w: verifiedAt is required for %s", ErrInvalidData, item.ID)
	}
	return nil
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
