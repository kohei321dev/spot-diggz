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

type Facility struct {
	ID               string           `json:"facilityId"`
	Name             string           `json:"name"`
	Address          string           `json:"address"`
	Location         Location         `json:"location"`
	Activities       []string         `json:"activities"`
	Hours            []OperatingHours `json:"hours,omitempty"`
	ScheduleNotes    []string         `json:"scheduleNotes,omitempty"`
	Price            string           `json:"price,omitempty"`
	Reservation      string           `json:"reservation,omitempty"`
	BeginnerFriendly bool             `json:"beginnerFriendly"`
	Features         []string         `json:"features,omitempty"`
	Rules            []string         `json:"rules,omitempty"`
	Access           Access           `json:"access,omitempty"`
	SourceURL        string           `json:"sourceUrl"`
	SourceType       string           `json:"sourceType"`
	Status           string           `json:"status"`
	Confidence       string           `json:"confidence,omitempty"`
	UpdatedAt        *time.Time       `json:"updatedAt,omitempty"`
	VerifiedAt       time.Time        `json:"verifiedAt"`
}
