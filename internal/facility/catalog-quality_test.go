package facility

import (
	"encoding/json"
	"math"
	"strings"
	"testing"
	"time"
)

func TestCatalogAcceptsAllJapanesePrefecturesAndRejectsNoncanonicalValues(t *testing.T) {
	prefectures := strings.Fields(JapanesePrefectures)
	if len(prefectures) != 47 {
		t.Fatal("expected all 47 prefectures")
	}
	seen := make(map[string]bool)
	for _, name := range prefectures {
		if seen[name] {
			t.Fatal("duplicate prefecture")
		}
		seen[name] = true
		spot := validFacility()
		spot.Prefecture = name
		if _, err := NewCatalog([]Facility{spot}); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
	}
	for _, name := range []string{"", " ", "東京", "東京県", "大阪", "東京都 ", " 東京都", "北海道 青森県", "California", "都道府県不明"} {
		spot := validFacility()
		spot.Prefecture = name
		if _, err := NewCatalog([]Facility{spot}); err == nil {
			t.Fatalf("accepted invalid prefecture %q", name)
		}
	}
	for _, coordinate := range []float64{math.NaN(), math.Inf(1), math.Inf(-1), 91} {
		spot := validFacility()
		spot.Location.Latitude = coordinate
		if _, err := NewCatalog([]Facility{spot}); err == nil {
			t.Fatal("invalid coordinate accepted")
		}
	}
	for _, coordinate := range []float64{math.NaN(), math.Inf(1), math.Inf(-1), 181} {
		spot := validFacility()
		spot.Location.Longitude = coordinate
		if _, err := NewCatalog([]Facility{spot}); err == nil {
			t.Fatal("invalid longitude accepted")
		}
	}
}

func nonTimetabledStreet() Facility {
	spot := validFacility()
	spot.Genre = GenreStreet
	spot.SkatingPermissionSourceURL = "https://example.com/permission"
	spot.Hours = nil
	spot.HoursStatus = HoursNotApplicable
	spot.HoursSourceURL = "https://example.com/hours-policy"
	spot.GeneralUseStatus = GeneralUseLimited
	spot.AvailabilityNote = "営業時間制の対象外です。利用ルールと現地の案内を確認してください。"
	spot.EnglishTranslation.AvailabilityNote = "No opening-hours system applies. Check the rules and on-site notices."
	return spot
}

func TestCatalogNonKnownHoursRequireEvidenceAndBilingualGuidance(t *testing.T) {
	for _, test := range []struct {
		name   string
		change func(*Facility)
		valid  bool
	}{
		{"verified non-applicable street", func(*Facility) {}, true},
		{"unknown street", func(s *Facility) {
			s.HoursStatus = HoursUnknown
			s.HoursSourceURL = ""
			s.GeneralUseStatus = GeneralUseScheduleCheckRequired
		}, true},
		{"unknown park", func(s *Facility) {
			s.HoursStatus = HoursUnknown
			s.Genre = GenreSkatepark
			s.GeneralUseStatus = GeneralUseScheduleCheckRequired
		}, true},
		{"no status does not mean unrestricted", func(s *Facility) { s.HoursStatus = "" }, false},
		{"known without hours", func(s *Facility) { s.HoursStatus = HoursKnown }, false},
		{"invalid status", func(s *Facility) { s.HoursStatus = "unlimited" }, false},
		{"not applicable park", func(s *Facility) { s.Genre = GenreSkatepark }, false},
		{"missing hours evidence", func(s *Facility) { s.HoursSourceURL = "" }, false},
		{"insecure hours evidence", func(s *Facility) { s.HoursSourceURL = "http://example.com/policy" }, false},
		{"empty port hours evidence", func(s *Facility) { s.HoursSourceURL = "https://example.com:/policy" }, false},
		{"empty IPv6 port hours evidence", func(s *Facility) { s.HoursSourceURL = "https://[::1]:/policy" }, false},
		{"empty hostname hours evidence", func(s *Facility) { s.HoursSourceURL = "https://[]/policy" }, false},
		{"credential-bearing hours evidence", func(s *Facility) { s.HoursSourceURL = "https://user:pass@example.com/policy" }, false},
		{"missing permission", func(s *Facility) { s.SkatingPermissionSourceURL = "" }, false},
		{"missing rule", func(s *Facility) { s.Rules = []string{" "} }, false},
		{"missing Japanese explanation", func(s *Facility) { s.AvailabilityNote = "" }, false},
		{"missing English explanation", func(s *Facility) { s.EnglishTranslation.AvailabilityNote = "" }, false},
		{"contradictory timetable", func(s *Facility) { s.Hours = validFacility().Hours }, false},
		{"contradictory hours basis", func(s *Facility) { s.HoursBasis = HoursBasisOfficial }, false},
		{"unconfirmed general use", func(s *Facility) { s.GeneralUseStatus = "" }, false},
		{"unknown with regular general use", func(s *Facility) { s.HoursStatus = HoursUnknown; s.GeneralUseStatus = GeneralUseRegular }, false},
		{"unknown unclassified", func(s *Facility) {
			s.HoursStatus = HoursUnknown
			s.GeneralUseStatus = GeneralUseScheduleCheckRequired
			s.Genre = ""
		}, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			spot := nonTimetabledStreet()
			test.change(&spot)
			_, err := NewCatalog([]Facility{spot})
			if (err == nil) != test.valid {
				t.Fatalf("valid=%t error=%v", test.valid, err)
			}
		})
	}
}

func TestHoursStatusRoundTripsWithoutInventingTimetable(t *testing.T) {
	known := validFacility()
	known.HoursStatus = HoursKnown
	unknown := nonTimetabledStreet()
	unknown.HoursStatus = HoursUnknown
	unknown.GeneralUseStatus = GeneralUseScheduleCheckRequired
	for _, spot := range []Facility{validFacility(), known, nonTimetabledStreet(), unknown} {
		raw, err := json.Marshal(struct {
			Facilities []Facility `json:"facilities"`
		}{[]Facility{spot}})
		if err != nil {
			t.Fatal(err)
		}
		catalog, err := LoadCatalog(strings.NewReader(string(raw)))
		if err != nil {
			t.Fatal(err)
		}
		loaded, _ := catalog.Find(spot.ID)
		if loaded.HoursStatus != spot.HoursStatus || len(loaded.Hours) != len(spot.Hours) {
			t.Fatal("hours contract did not round trip")
		}
		if !HasKnownOperatingHours(spot) && strings.Contains(string(raw), `"hours":`) {
			t.Fatal("fabricated timetable serialized")
		}
	}
}

func TestNearbyEligibilityRejectsUnknownStaleAndUnverifiedRecords(t *testing.T) {
	now := validFacility().VerifiedAt.Add(time.Hour)
	for _, test := range []struct {
		name   string
		change func(*Facility)
		want   bool
	}{
		{"non-applicable is searchable", func(*Facility) {}, true},
		{"unclassified", func(s *Facility) { s.Genre = "" }, false},
		{"unverified", func(s *Facility) { s.Status = "candidate" }, false},
		{"missing permission", func(s *Facility) { s.SkatingPermissionSourceURL = "" }, false},
		{"hours unknown", func(s *Facility) { s.HoursStatus = HoursUnknown }, false},
		{"general-use schedule unknown", func(s *Facility) { s.GeneralUseStatus = GeneralUseScheduleCheckRequired }, false},
		{"dynamic stale", func(s *Facility) { s.DynamicVerifiedAt = now.Add(-DynamicInformationFreshnessWindow - time.Second) }, false},
		{"stable stale", func(s *Facility) { s.StableVerifiedAt = now.Add(-StableInformationFreshnessWindow - time.Second) }, false},
		{"future verification", func(s *Facility) { s.DynamicVerifiedAt = now.Add(time.Second) }, false},
		{"other activity", func(s *Facility) { s.Activities = []string{"BMX"} }, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			spot := nonTimetabledStreet()
			test.change(&spot)
			if IsNearbySearchable(spot, now) != test.want {
				t.Fatal("eligibility mismatch")
			}
		})
	}
}
