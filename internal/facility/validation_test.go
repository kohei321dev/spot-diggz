package facility

import (
	"bytes"
	"encoding/json"
	"errors"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestNewCatalogAcceptsStructuredClosurePeriods(t *testing.T) {
	item := validFacility()
	item.ClosurePeriods = []ClosurePeriod{
		{
			Type:   ClosurePeriodOneTime,
			Start:  "2026-08-10",
			End:    "2026-08-12",
			Reason: "大会開催のため",
		},
		{
			Type:   ClosurePeriodAnnual,
			Start:  "12-29",
			End:    "01-03",
			Reason: "年末年始休場",
		},
		{
			Type:   ClosurePeriodAnnual,
			Start:  "02-29",
			End:    "02-29",
			Reason: "うるう日の設備点検",
		},
	}

	catalog, err := NewCatalog([]Facility{item})
	if err != nil {
		t.Fatalf("NewCatalog() error = %v", err)
	}
	stored, err := catalog.Find(item.ID)
	if err != nil {
		t.Fatalf("Find() error = %v", err)
	}
	if len(stored.ClosurePeriods) != len(item.ClosurePeriods) {
		t.Fatalf("len(ClosurePeriods) = %d, want %d", len(stored.ClosurePeriods), len(item.ClosurePeriods))
	}
}

func TestNewCatalogRejectsInvalidClosurePeriods(t *testing.T) {
	tests := []struct {
		name   string
		period ClosurePeriod
	}{
		{
			name:   "missing reason",
			period: ClosurePeriod{Type: ClosurePeriodOneTime, Start: "2026-08-10", End: "2026-08-12"},
		},
		{
			name:   "unsupported type",
			period: ClosurePeriod{Type: "weekly", Start: "08-10", End: "08-12", Reason: "test"},
		},
		{
			name:   "invalid one-time date",
			period: ClosurePeriod{Type: ClosurePeriodOneTime, Start: "2026-02-30", End: "2026-03-01", Reason: "test"},
		},
		{
			name:   "non-padded one-time date",
			period: ClosurePeriod{Type: ClosurePeriodOneTime, Start: "2026-2-01", End: "2026-02-02", Reason: "test"},
		},
		{
			name:   "reversed one-time range",
			period: ClosurePeriod{Type: ClosurePeriodOneTime, Start: "2026-08-12", End: "2026-08-10", Reason: "test"},
		},
		{
			name:   "invalid annual date",
			period: ClosurePeriod{Type: ClosurePeriodAnnual, Start: "02-30", End: "03-01", Reason: "test"},
		},
		{
			name:   "non-padded annual date",
			period: ClosurePeriod{Type: ClosurePeriodAnnual, Start: "2-01", End: "02-02", Reason: "test"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			item := validFacility()
			item.ClosurePeriods = []ClosurePeriod{test.period}

			_, err := NewCatalog([]Facility{item})
			if !errors.Is(err, ErrInvalidData) {
				t.Fatalf("NewCatalog() error = %v, want ErrInvalidData", err)
			}
		})
	}
}

func TestNewCatalogRejectsMissingRegion(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*Facility)
	}{
		{name: "prefecture", mutate: func(item *Facility) { item.Prefecture = " " }},
		{name: "municipality", mutate: func(item *Facility) { item.Municipality = "" }},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			item := validFacility()
			test.mutate(&item)

			_, err := NewCatalog([]Facility{item})
			if !errors.Is(err, ErrInvalidData) {
				t.Fatalf("NewCatalog() error = %v, want ErrInvalidData", err)
			}
		})
	}
}

func TestNewCatalogRejectsIncompleteEnglishTranslation(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*Facility)
	}{
		{name: "name", mutate: func(item *Facility) { item.EnglishTranslation.Name = "" }},
		{name: "address", mutate: func(item *Facility) { item.EnglishTranslation.Address = " " }},
		{name: "price", mutate: func(item *Facility) { item.EnglishTranslation.Price = "" }},
		{name: "reservation", mutate: func(item *Facility) { item.EnglishTranslation.Reservation = "" }},
		{name: "access notes", mutate: func(item *Facility) { item.EnglishTranslation.AccessNotes = "" }},
		{name: "missing schedule note", mutate: func(item *Facility) { item.EnglishTranslation.ScheduleNotes = nil }},
		{name: "blank schedule note", mutate: func(item *Facility) { item.EnglishTranslation.ScheduleNotes = []string{" "} }},
		{name: "extra schedule note", mutate: func(item *Facility) {
			item.EnglishTranslation.ScheduleNotes = append(item.EnglishTranslation.ScheduleNotes, "Extra note.")
		}},
		{name: "missing rule", mutate: func(item *Facility) { item.EnglishTranslation.Rules = nil }},
		{name: "blank rule", mutate: func(item *Facility) { item.EnglishTranslation.Rules = []string{""} }},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			item := validFacility()
			test.mutate(&item)

			_, err := NewCatalog([]Facility{item})
			if !errors.Is(err, ErrInvalidData) {
				t.Fatalf("NewCatalog() error = %v, want ErrInvalidData", err)
			}
		})
	}
}

func TestNewCatalogAcceptsOptionalCuratedMediaAndSocialLinks(t *testing.T) {
	withoutExternalMetadata := validFacility()
	if _, err := NewCatalog([]Facility{withoutExternalMetadata}); err != nil {
		t.Fatalf("NewCatalog() error = %v, want media and social links to remain optional", err)
	}

	item := validFacility()
	item.Media = validCuratedMedia()
	item.SocialLinks = validSocialLinks()

	catalog, err := NewCatalog([]Facility{item})
	if err != nil {
		t.Fatalf("NewCatalog() error = %v", err)
	}
	stored, err := catalog.Find(item.ID)
	if err != nil {
		t.Fatalf("Find() error = %v", err)
	}
	if stored.Media == nil || stored.Media.YouTube == nil || stored.Media.YouTube.VideoID != item.Media.YouTube.VideoID {
		t.Fatalf("stored media = %#v, want YouTube video metadata", stored.Media)
	}
	if len(stored.SocialLinks) != len(item.SocialLinks) {
		t.Fatalf("stored social links = %#v, want %#v", stored.SocialLinks, item.SocialLinks)
	}
}

func TestNewCatalogRejectsInvalidCuratedMedia(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*Facility)
	}{
		{name: "empty media", mutate: func(item *Facility) { item.Media = &FacilityMedia{} }},
		{name: "unsupported provider", mutate: func(item *Facility) {
			item.Media = validCuratedMedia()
			item.Media.YouTube.Provider = "vimeo"
		}},
		{name: "invalid video ID", mutate: func(item *Facility) {
			item.Media = validCuratedMedia()
			item.Media.YouTube.VideoID = "too-short"
		}},
		{name: "missing title", mutate: func(item *Facility) {
			item.Media = validCuratedMedia()
			item.Media.YouTube.Title = " "
		}},
		{name: "mismatched watch URL", mutate: func(item *Facility) {
			item.Media = validCuratedMedia()
			item.Media.YouTube.SourceURL = "https://www.youtube.com/watch?v=zYxWvUtSrQp"
		}},
		{name: "noncanonical watch URL", mutate: func(item *Facility) {
			item.Media = validCuratedMedia()
			item.Media.YouTube.SourceURL = "https://youtu.be/a1B2c3D4e5F"
		}},
		{name: "missing selectedAt", mutate: func(item *Facility) {
			item.Media = validCuratedMedia()
			item.Media.YouTube.SelectedAt = time.Time{}
		}},
		{name: "missing verifiedAt", mutate: func(item *Facility) {
			item.Media = validCuratedMedia()
			item.Media.YouTube.VerifiedAt = time.Time{}
		}},
		{name: "selected after verification", mutate: func(item *Facility) {
			item.Media = validCuratedMedia()
			item.Media.YouTube.SelectedAt = item.Media.YouTube.VerifiedAt.Add(time.Nanosecond)
		}},
		{name: "missing selection reason", mutate: func(item *Facility) {
			item.Media = validCuratedMedia()
			item.Media.YouTube.SelectionReason = ""
		}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			item := validFacility()
			test.mutate(&item)

			_, err := NewCatalog([]Facility{item})
			if !errors.Is(err, ErrInvalidData) {
				t.Fatalf("NewCatalog() error = %v, want ErrInvalidData", err)
			}
		})
	}
}

func TestNewCatalogRejectsInvalidSocialLinks(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*Facility)
	}{
		{name: "unsupported platform", mutate: func(item *Facility) {
			item.SocialLinks = []SocialLink{{Platform: "youtube", URL: "https://www.youtube.com/@spotdiggz", VerifiedAt: curatedReviewTime}}
		}},
		{name: "non-HTTPS URL", mutate: func(item *Facility) {
			item.SocialLinks = validSocialLinks()
			item.SocialLinks[0].URL = "http://www.instagram.com/spot_diggz/"
		}},
		{name: "unsupported host", mutate: func(item *Facility) {
			item.SocialLinks = validSocialLinks()
			item.SocialLinks[0].URL = "https://instagram.example/spot_diggz/"
		}},
		{name: "Instagram post URL", mutate: func(item *Facility) {
			item.SocialLinks = validSocialLinks()
			item.SocialLinks[0].URL = "https://www.instagram.com/p/a1B2c3D4e5F/"
		}},
		{name: "X URL with query", mutate: func(item *Facility) {
			item.SocialLinks = validSocialLinks()
			item.SocialLinks[1].URL = "https://x.com/spotdiggz?utm_source=catalog"
		}},
		{name: "duplicate platform", mutate: func(item *Facility) {
			item.SocialLinks = validSocialLinks()
			item.SocialLinks[1] = SocialLink{
				Platform:   SocialPlatformInstagram,
				URL:        "https://instagram.com/spotdiggz_jp/",
				VerifiedAt: curatedReviewTime,
			}
		}},
		{name: "too many profiles", mutate: func(item *Facility) {
			item.SocialLinks = append(validSocialLinks(), SocialLink{
				Platform:   "other",
				URL:        "https://example.com/profile",
				VerifiedAt: curatedReviewTime,
			})
		}},
		{name: "missing verifiedAt", mutate: func(item *Facility) {
			item.SocialLinks = validSocialLinks()
			item.SocialLinks[0].VerifiedAt = time.Time{}
		}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			item := validFacility()
			test.mutate(&item)

			_, err := NewCatalog([]Facility{item})
			if !errors.Is(err, ErrInvalidData) {
				t.Fatalf("NewCatalog() error = %v, want ErrInvalidData", err)
			}
		})
	}
}

func TestLoadCatalogRejectsArbitraryMediaEmbedURL(t *testing.T) {
	item := validFacility()
	item.Media = validCuratedMedia()
	payload, err := json.Marshal(catalogFile{Facilities: []Facility{item}})
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	payload = bytes.Replace(
		payload,
		[]byte(`"selectedAt"`),
		[]byte(`"embedUrl":"https://www.youtube-nocookie.com/embed/a1B2c3D4e5F","selectedAt"`),
		1,
	)

	if _, err := LoadCatalog(bytes.NewReader(payload)); err == nil {
		t.Fatal("LoadCatalog() error = nil, want unknown media embed URL to be rejected")
	}
}

func TestNewCatalogRejectsMissingSplitVerificationTime(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*Facility)
	}{
		{name: "dynamic", mutate: func(item *Facility) { item.DynamicVerifiedAt = time.Time{} }},
		{name: "stable", mutate: func(item *Facility) { item.StableVerifiedAt = time.Time{} }},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			item := validFacility()
			test.mutate(&item)

			_, err := NewCatalog([]Facility{item})
			if !errors.Is(err, ErrInvalidData) {
				t.Fatalf("NewCatalog() error = %v, want ErrInvalidData", err)
			}
		})
	}
}

func TestLoadCatalogRejectsInvalidVerificationDate(t *testing.T) {
	item := validFacility()
	item.DynamicVerifiedAt = time.Date(2026, time.July, 14, 0, 0, 0, 0, time.UTC)
	payload, err := json.Marshal(catalogFile{Facilities: []Facility{item}})
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	validTimestamp := []byte(`"dynamicVerifiedAt":"2026-07-14T00:00:00Z"`)
	invalidTimestamp := []byte(`"dynamicVerifiedAt":"2026-02-30T00:00:00Z"`)
	if !bytes.Contains(payload, validTimestamp) {
		t.Fatalf("catalog JSON does not contain expected dynamicVerifiedAt: %s", payload)
	}
	payload = bytes.Replace(payload, validTimestamp, invalidTimestamp, 1)

	if _, err := LoadCatalog(bytes.NewReader(payload)); err == nil {
		t.Fatal("LoadCatalog() error = nil, want invalid verification date error")
	}
}

func TestNewCatalogAtRejectsFutureTimestamps(t *testing.T) {
	asOf := time.Date(2026, time.July, 19, 12, 0, 0, 0, time.UTC)
	future := asOf.Add(time.Nanosecond)
	tests := []struct {
		name   string
		mutate func(*Facility)
	}{
		{name: "verifiedAt", mutate: func(item *Facility) { item.VerifiedAt = future }},
		{name: "dynamicVerifiedAt", mutate: func(item *Facility) { item.DynamicVerifiedAt = future }},
		{name: "stableVerifiedAt", mutate: func(item *Facility) { item.StableVerifiedAt = future }},
		{name: "updatedAt", mutate: func(item *Facility) { item.UpdatedAt = &future }},
		{name: "media selectedAt", mutate: func(item *Facility) {
			item.Media = validCuratedMedia()
			item.Media.YouTube.SelectedAt = future
		}},
		{name: "media verifiedAt", mutate: func(item *Facility) {
			item.Media = validCuratedMedia()
			item.Media.YouTube.VerifiedAt = future
		}},
		{name: "social verifiedAt", mutate: func(item *Facility) {
			item.SocialLinks = validSocialLinks()
			item.SocialLinks[0].VerifiedAt = future
		}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			item := validFacility()
			test.mutate(&item)

			_, err := NewCatalogAt([]Facility{item}, asOf)
			if !errors.Is(err, ErrInvalidData) {
				t.Fatalf("NewCatalogAt() error = %v, want ErrInvalidData", err)
			}
		})
	}
}

func TestNewCatalogAtRejectsZeroReferenceTime(t *testing.T) {
	_, err := NewCatalogAt([]Facility{validFacility()}, time.Time{})
	if !errors.Is(err, ErrInvalidData) {
		t.Fatalf("NewCatalogAt() error = %v, want ErrInvalidData", err)
	}
}

func TestNewCatalogAtAcceptsTimestampsEqualToReferenceTime(t *testing.T) {
	asOf := time.Date(2026, time.July, 19, 12, 0, 0, 0, time.UTC)
	item := validFacility()
	item.VerifiedAt = asOf
	item.DynamicVerifiedAt = asOf
	item.StableVerifiedAt = asOf
	item.UpdatedAt = &asOf
	item.Media = validCuratedMedia()
	item.Media.YouTube.SelectedAt = asOf
	item.Media.YouTube.VerifiedAt = asOf
	item.SocialLinks = validSocialLinks()
	for index := range item.SocialLinks {
		item.SocialLinks[index].VerifiedAt = asOf
	}

	if _, err := NewCatalogAt([]Facility{item}, asOf); err != nil {
		t.Fatalf("NewCatalogAt() error = %v, want timestamps equal to reference time to be valid", err)
	}
}

func TestNewCatalogAtAcceptsStaleDataAndFutureClosure(t *testing.T) {
	asOf := time.Date(2026, time.July, 19, 12, 0, 0, 0, time.UTC)
	item := validFacility()
	item.VerifiedAt = asOf.Add(-31 * 24 * time.Hour)
	item.DynamicVerifiedAt = asOf.Add(-31 * 24 * time.Hour)
	item.StableVerifiedAt = asOf.Add(-181 * 24 * time.Hour)
	item.ClosurePeriods = []ClosurePeriod{
		{
			Type:   ClosurePeriodOneTime,
			Start:  "2027-01-01",
			End:    "2027-01-02",
			Reason: "予定された設備点検",
		},
	}

	if _, err := NewCatalogAt([]Facility{item}, asOf); err != nil {
		t.Fatalf("NewCatalogAt() error = %v, want stale data and future closure to remain structurally valid", err)
	}
}

func TestProductionCatalogPreservesVerifiedFacilityFacts(t *testing.T) {
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller() could not resolve the test file")
	}
	catalogPath := filepath.Join(filepath.Dir(currentFile), "..", "..", "data", "facilities.json")
	asOf := time.Date(2026, time.July, 21, 0, 0, 0, 0, time.FixedZone("JST", 9*60*60))
	catalog, err := LoadCatalogFileAt(catalogPath, asOf)
	if err != nil {
		t.Fatalf("LoadCatalogFileAt() error = %v", err)
	}

	nara, err := catalog.Find("NAR-F001")
	if err != nil {
		t.Fatalf("Find(NAR-F001) error = %v", err)
	}
	for _, date := range []string{"2026-07-21", "2026-08-12"} {
		if !hasOneTimeClosure(nara.ClosurePeriods, date) {
			t.Errorf("NAR-F001 closurePeriods does not contain one_time closure for %s", date)
		}
	}

	omiya, err := catalog.Find("WAK-F003")
	if err != nil {
		t.Fatalf("Find(WAK-F003) error = %v", err)
	}
	assertIncludesFeatures(t, omiya.Features, "outdoor", "flat-area")
	assertExcludesFeatures(t, omiya.Features, "bank", "rail")

	okinosu, err := catalog.Find("TKS-F001")
	if err != nil {
		t.Fatalf("Find(TKS-F001) error = %v", err)
	}
	assertIncludesFeatures(t, okinosu.Features, "outdoor", "rooftop")
	assertExcludesFeatures(t, okinosu.Features, "indoor", "roof")
	if !containsText(okinosu.ScheduleNotes, "屋上") || !containsText(okinosu.ScheduleNotes, "天候不良") {
		t.Errorf("TKS-F001 scheduleNotes = %q, want rooftop and weather guidance", okinosu.ScheduleNotes)
	}
	if !containsText(okinosu.EnglishTranslation.ScheduleNotes, "outdoor rooftop") || !containsText(okinosu.EnglishTranslation.ScheduleNotes, "weather") {
		t.Errorf("TKS-F001 English scheduleNotes = %q, want rooftop and weather guidance", okinosu.EnglishTranslation.ScheduleNotes)
	}
}

func hasOneTimeClosure(periods []ClosurePeriod, date string) bool {
	for _, period := range periods {
		if period.Type == ClosurePeriodOneTime && period.Start == date && period.End == date {
			return true
		}
	}
	return false
}

func assertIncludesFeatures(t *testing.T, features []string, expected ...string) {
	t.Helper()
	for _, feature := range expected {
		if !containsIgnoreCase(features, feature) {
			t.Errorf("features = %q, want %q", features, feature)
		}
	}
}

func assertExcludesFeatures(t *testing.T, features []string, excluded ...string) {
	t.Helper()
	for _, feature := range excluded {
		if containsIgnoreCase(features, feature) {
			t.Errorf("features = %q, must not contain %q", features, feature)
		}
	}
}

func containsText(values []string, text string) bool {
	for _, value := range values {
		if strings.Contains(strings.ToLower(value), strings.ToLower(text)) {
			return true
		}
	}
	return false
}

var curatedReviewTime = time.Date(2026, time.July, 15, 0, 0, 0, 0, time.UTC)

func validCuratedMedia() *FacilityMedia {
	return &FacilityMedia{
		YouTube: &YouTubeVideo{
			Provider:        youTubeProvider,
			VideoID:         "a1B2c3D4e5F",
			Title:           "Test Facility overview",
			SourceURL:       "https://www.youtube.com/watch?v=a1B2c3D4e5F",
			SelectedAt:      curatedReviewTime,
			VerifiedAt:      curatedReviewTime,
			SelectionReason: "施設のセクションを確認できるため",
		},
	}
}

func validSocialLinks() []SocialLink {
	return []SocialLink{
		{
			Platform:   SocialPlatformInstagram,
			URL:        "https://www.instagram.com/spot_diggz/",
			VerifiedAt: curatedReviewTime,
		},
		{
			Platform:   SocialPlatformX,
			URL:        "https://x.com/spotdiggz",
			VerifiedAt: curatedReviewTime,
		},
	}
}
