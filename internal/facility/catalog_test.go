package facility

import (
	"errors"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestNewCatalogRejectsUnverifiedFacility(t *testing.T) {
	item := validFacility()
	item.Status = "candidate"

	_, err := NewCatalog([]Facility{item})
	if !errors.Is(err, ErrInvalidData) {
		t.Fatalf("NewCatalog() error = %v, want ErrInvalidData", err)
	}
}

func TestNewCatalogRejectsDuplicateIDs(t *testing.T) {
	item := validFacility()

	_, err := NewCatalog([]Facility{item, item})
	if !errors.Is(err, ErrDuplicateID) {
		t.Fatalf("NewCatalog() error = %v, want ErrDuplicateID", err)
	}
}

func TestCatalogListFiltersAndSortsByID(t *testing.T) {
	first := validFacility()
	first.ID = "facility-b"
	first.Activities = []string{"skateboard"}
	second := validFacility()
	second.ID = "facility-a"
	second.Activities = []string{"BMX"}

	catalog, err := NewCatalog([]Facility{first, second})
	if err != nil {
		t.Fatalf("NewCatalog() error = %v", err)
	}

	items := catalog.List("SKATEBOARD")
	if len(items) != 1 || items[0].ID != "facility-b" {
		t.Fatalf("List() = %#v, want only facility-b", items)
	}

	all := catalog.List("")
	if len(all) != 2 || all[0].ID != "facility-a" || all[1].ID != "facility-b" {
		t.Fatalf("List() = %#v, want IDs sorted", all)
	}
}

func TestLoadCatalogRejectsTrailingJSON(t *testing.T) {
	_, err := LoadCatalog(strings.NewReader(`{"facilities": []}{"facilities": []}`))
	if err == nil {
		t.Fatal("LoadCatalog() error = nil, want trailing JSON error")
	}
}

func TestNewCatalogRejectsInvalidOperatingHours(t *testing.T) {
	item := validFacility()
	item.Hours = []OperatingHours{{Day: "holiday", Opens: "09:00", Closes: "21:00"}}

	_, err := NewCatalog([]Facility{item})
	if !errors.Is(err, ErrInvalidData) {
		t.Fatalf("NewCatalog() error = %v, want ErrInvalidData", err)
	}
}

func TestNewCatalogRejectsFacilityWithoutOpenHours(t *testing.T) {
	item := validFacility()
	item.Hours = []OperatingHours{{Day: "daily", Closed: true}}

	_, err := NewCatalog([]Facility{item})
	if !errors.Is(err, ErrInvalidData) {
		t.Fatalf("NewCatalog() error = %v, want ErrInvalidData", err)
	}
}

func TestDevelopmentFixtureLoads(t *testing.T) {
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller() could not resolve the test file")
	}
	fixturePath := filepath.Join(filepath.Dir(currentFile), "..", "..", "testdata", "facilities.dev.json")

	catalog, err := LoadCatalogFile(fixturePath)
	if err != nil {
		t.Fatalf("LoadCatalogFile() error = %v", err)
	}
	items := catalog.List("")
	if len(items) != 3 {
		t.Fatalf("List() returned %d facilities, want 3", len(items))
	}
	for _, item := range items {
		if item.SourceType != "test_fixture" || item.Confidence != "test" {
			t.Fatalf("fixture %s is not marked as test data", item.ID)
		}
	}
}

func TestProductionCatalogContainsOnlyVerifiedRealFacilities(t *testing.T) {
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller() could not resolve the test file")
	}
	catalogPath := filepath.Join(filepath.Dir(currentFile), "..", "..", "data", "facilities.json")

	catalog, err := LoadCatalogFile(catalogPath)
	if err != nil {
		t.Fatalf("LoadCatalogFile() error = %v", err)
	}
	items := catalog.List("")
	if len(items) < 3 {
		t.Fatalf("List() returned %d facilities, want at least 3", len(items))
	}
	for _, item := range items {
		if item.SourceType == "test_fixture" || item.Confidence == "test" || strings.Contains(strings.ToUpper(item.Name), "DUMMY") || strings.Contains(item.SourceURL, "example.com") {
			t.Fatalf("production facility %s is marked as test data", item.ID)
		}
		if len(item.ScheduleNotes) == 0 {
			t.Fatalf("production facility %s has no schedule notes", item.ID)
		}
	}
}

func validFacility() Facility {
	return Facility{
		ID:               "facility-a",
		Name:             "Test Facility",
		Address:          "大阪府大阪市",
		Location:         Location{Latitude: 34.6937, Longitude: 135.5023},
		Activities:       []string{"skateboard"},
		Hours:            []OperatingHours{{Day: "daily", Opens: "09:00", Closes: "21:00"}},
		Price:            "500円",
		Reservation:      "当日受付",
		BeginnerFriendly: true,
		Features:         []string{"flat-area"},
		Rules:            []string{"ヘルメット必須"},
		SourceURL:        "https://example.com/facilities/a",
		SourceType:       "official",
		Status:           "verified",
		Confidence:       "high",
		VerifiedAt:       time.Date(2026, time.July, 15, 0, 0, 0, 0, time.UTC),
	}
}
