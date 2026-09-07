package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/kohei321dev/spot-diggz/internal/facility"
)

func TestRunAcceptsCatalogFreshThroughMinimumValidity(t *testing.T) {
	asOf := time.Date(2026, time.July, 20, 0, 0, 0, 0, time.UTC)
	catalogPath := writeTestCatalog(t, asOf.Add(-time.Hour), asOf.Add(-time.Hour))
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := run(
		[]string{"-path", catalogPath},
		func() time.Time { return asOf },
		&stdout,
		&stderr,
	)

	if exitCode != 0 {
		t.Fatalf("run() exit code = %d, want 0; stderr = %q", exitCode, stderr.String())
	}
	if !strings.Contains(stdout.String(), "1 facilities are fresh") {
		t.Errorf("stdout = %q, want successful facility count", stdout.String())
	}
}

func TestRunRejectsCatalogThatExpiresBeforeMinimumValidity(t *testing.T) {
	asOf := time.Date(2026, time.July, 20, 0, 0, 0, 0, time.UTC)
	catalogPath := writeTestCatalog(t, asOf.Add(-25*24*time.Hour), asOf.Add(-time.Hour))
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := run(
		[]string{"-path", catalogPath, "-minimum-validity", "168h"},
		func() time.Time { return asOf },
		&stdout,
		&stderr,
	)

	if exitCode != 1 {
		t.Fatalf("run() exit code = %d, want 1", exitCode)
	}
	if !strings.Contains(stderr.String(), "CATALOGCHECK-TEST(dynamic)") {
		t.Errorf("stderr = %q, want stale dynamic field", stderr.String())
	}
}

func TestRunRejectsNonPositiveMinimumValidity(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := run(
		[]string{"-minimum-validity", "0s"},
		time.Now,
		&stdout,
		&stderr,
	)

	if exitCode != 2 {
		t.Fatalf("run() exit code = %d, want 2", exitCode)
	}
}

func TestRunRequiresSearchableEvidenceWithoutChangingCatalog(t *testing.T) {
	asOf := time.Date(2026, time.July, 20, 0, 0, 0, 0, time.UTC)
	for _, test := range []struct {
		name   string
		change func(*facility.Facility)
		want   int
	}{
		{"known classified park", func(*facility.Facility) {}, 0},
		{"other activity", func(s *facility.Facility) { s.Activities = []string{"BMX"} }, 1},
		{"unclassified", func(s *facility.Facility) { s.Genre = ""; s.SkatingPermissionSourceURL = "" }, 1},
		{"schedule check required", func(s *facility.Facility) { s.GeneralUseStatus = facility.GeneralUseScheduleCheckRequired }, 1},
		{"non-applicable street", func(s *facility.Facility) {
			s.Genre = facility.GenreStreet
			s.HoursStatus = facility.HoursNotApplicable
			s.HoursSourceURL = "https://example.com/hours-policy"
			s.GeneralUseStatus = facility.GeneralUseLimited
			s.Hours = nil
		}, 0},
		{"unknown street", func(s *facility.Facility) {
			s.Genre = facility.GenreStreet
			s.HoursStatus = facility.HoursUnknown
			s.GeneralUseStatus = facility.GeneralUseScheduleCheckRequired
			s.Hours = nil
		}, 1},
		{"expires at validity boundary", func(s *facility.Facility) {
			s.DynamicVerifiedAt = asOf.Add(defaultMinimumValidity - facility.DynamicInformationFreshnessWindow)
			s.VerifiedAt = s.DynamicVerifiedAt
		}, 0},
		{"expires before validity boundary", func(s *facility.Facility) {
			s.DynamicVerifiedAt = asOf.Add(defaultMinimumValidity - facility.DynamicInformationFreshnessWindow - time.Second)
			s.VerifiedAt = s.DynamicVerifiedAt
		}, 1},
	} {
		t.Run(test.name, func(t *testing.T) {
			path := writeTestCatalog(t, asOf.Add(-time.Hour), asOf.Add(-time.Hour), func(s *facility.Facility) {
				s.Genre = facility.GenreSkatepark
				s.SkatingPermissionSourceURL = "https://example.com/permission"
				s.AvailabilityNote = "公式の利用条件を確認してください。"
				s.EnglishTranslation.AvailabilityNote = "Check the official conditions."
				test.change(s)
			})
			before, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			var stdout, stderr bytes.Buffer
			got := run([]string{"-path", path, "-require-searchable"}, func() time.Time { return asOf }, &stdout, &stderr)
			if got != test.want {
				t.Fatalf("exit=%d want=%d stderr=%s", got, test.want, stderr.String())
			}
			if got == 0 && !strings.Contains(stdout.String(), "1 searchable spots") {
				t.Fatal("missing searchable count")
			}
			after, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(before, after) {
				t.Fatal("catalogcheck changed catalog evidence")
			}
		})
	}
}

func writeTestCatalog(t *testing.T, dynamicVerifiedAt, stableVerifiedAt time.Time, changes ...func(*facility.Facility)) string {
	t.Helper()
	item := facility.Facility{
		ID:           "CATALOGCHECK-TEST",
		Name:         "鮮度確認用施設",
		Address:      "大阪府大阪市テスト1-1",
		Prefecture:   "大阪府",
		Municipality: "大阪市",
		Location:     facility.Location{Latitude: 34.6937, Longitude: 135.5023},
		Activities:   []string{"skateboard"},
		Hours:        []facility.OperatingHours{{Day: "daily", Opens: "09:00", Closes: "18:00"}},
		ScheduleNotes: []string{
			"公式情報を確認してください。",
		},
		Price:            "無料",
		Reservation:      "不要",
		BeginnerFriendly: true,
		Features:         []string{"outdoor", "flat-area"},
		Rules:            []string{"施設のルールに従ってください。"},
		Access:           facility.Access{Notes: "テスト用"},
		EnglishTranslation: facility.FacilityEnglishTranslation{
			Name:          "Catalog Check Test Facility",
			Address:       "1-1 Test, Osaka, Osaka",
			ScheduleNotes: []string{"Check the official source."},
			Price:         "Free",
			Reservation:   "Not required",
			Rules:         []string{"Follow the facility rules."},
			AccessNotes:   "Test fixture",
		},
		SourceURL:         "https://example.com/facility",
		SourceType:        "municipality_official",
		Status:            "verified",
		Confidence:        "high",
		VerifiedAt:        dynamicVerifiedAt,
		DynamicVerifiedAt: dynamicVerifiedAt,
		StableVerifiedAt:  stableVerifiedAt,
	}
	for _, change := range changes {
		change(&item)
	}
	payload, err := json.Marshal(struct {
		Facilities []facility.Facility `json:"facilities"`
	}{Facilities: []facility.Facility{item}})
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	path := filepath.Join(t.TempDir(), "facilities.json")
	if err := os.WriteFile(path, payload, 0o600); err != nil {
		t.Fatalf("os.WriteFile() error = %v", err)
	}
	return path
}
