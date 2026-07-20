package facility

import "time"

// Location is a facility's public location. It never contains a user's location.
type Location struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

type OperatingHours struct {
	Day    string `json:"day"`
	Opens  string `json:"opens,omitempty"`
	Closes string `json:"closes,omitempty"`
	Closed bool   `json:"closed,omitempty"`
}

type Access struct {
	Stations []string `json:"stations,omitempty"`
	Parking  string   `json:"parking,omitempty"`
	Notes    string   `json:"notes,omitempty"`
}

type FacilityEnglishTranslation struct {
	Name             string   `json:"name"`
	Address          string   `json:"address"`
	ScheduleNotes    []string `json:"scheduleNotes"`
	AvailabilityNote string   `json:"availabilityNote,omitempty"`
	Price            string   `json:"price"`
	Reservation      string   `json:"reservation"`
	Rules            []string `json:"rules"`
	AccessNotes      string   `json:"accessNotes"`
}

type ClosurePeriodType string

const (
	ClosurePeriodOneTime ClosurePeriodType = "one_time"
	ClosurePeriodAnnual  ClosurePeriodType = "annual"
)

// ClosurePeriod is either a one-time YYYY-MM-DD range or an annual MM-DD range.
type ClosurePeriod struct {
	Type   ClosurePeriodType `json:"type"`
	Start  string            `json:"start"`
	End    string            `json:"end"`
	Reason string            `json:"reason"`
}

type Facility struct {
	ID                 string                     `json:"facilityId"`
	Name               string                     `json:"name"`
	Address            string                     `json:"address"`
	Prefecture         string                     `json:"prefecture"`
	Municipality       string                     `json:"municipality"`
	Location           Location                   `json:"location"`
	Activities         []string                   `json:"activities"`
	Hours              []OperatingHours           `json:"hours,omitempty"`
	GeneralUseStatus   string                     `json:"generalUseStatus,omitempty"`
	HoursBasis         string                     `json:"hoursBasis,omitempty"`
	AvailabilityNote   string                     `json:"availabilityNote,omitempty"`
	ScheduleNotes      []string                   `json:"scheduleNotes,omitempty"`
	ClosurePeriods     []ClosurePeriod            `json:"closurePeriods,omitempty"`
	Price              string                     `json:"price,omitempty"`
	Reservation        string                     `json:"reservation,omitempty"`
	BeginnerFriendly   bool                       `json:"beginnerFriendly"`
	Features           []string                   `json:"features,omitempty"`
	Rules              []string                   `json:"rules,omitempty"`
	Access             Access                     `json:"access,omitempty"`
	EnglishTranslation FacilityEnglishTranslation `json:"englishTranslation"`
	SourceURL          string                     `json:"sourceUrl"`
	SourceType         string                     `json:"sourceType"`
	Status             string                     `json:"status"`
	Confidence         string                     `json:"confidence,omitempty"`
	UpdatedAt          *time.Time                 `json:"updatedAt,omitempty"`
	VerifiedAt         time.Time                  `json:"verifiedAt"`
	DynamicVerifiedAt  time.Time                  `json:"dynamicVerifiedAt"`
	StableVerifiedAt   time.Time                  `json:"stableVerifiedAt"`
}

const (
	GeneralUseRegular               = "regular"
	GeneralUseLimited               = "limited"
	GeneralUseScheduleCheckRequired = "schedule_check_required"
	HoursBasisOfficial              = "official"
	HoursBasisConservative          = "conservative"
)
