package facility

import "testing"

func TestCatalogRequiresPermissionEvidenceWhenGenreIsSet(t *testing.T) {
	for _, genre := range []Genre{GenreSkatepark, GenreStreet} {
		spot := validFacility()
		spot.Genre = genre
		if _, err := NewCatalog([]Facility{spot}); err == nil {
			t.Fatal("genre without permission accepted")
		}
		for _, source := range []string{"http://example.com/rules", "https://user:password@example.com/rules", "javascript:alert(1)"} {
			spot.SkatingPermissionSourceURL = source
			if _, err := NewCatalog([]Facility{spot}); err == nil {
				t.Fatal("unsafe permission source accepted")
			}
		}
		spot.SkatingPermissionSourceURL = "https://example.com/rules"
		if _, err := NewCatalog([]Facility{spot}); err != nil {
			t.Fatal(err)
		}
	}
	legacy := validFacility()
	if _, err := NewCatalog([]Facility{legacy}); err != nil {
		t.Fatal("legacy catalog rejected")
	}
	legacy.Genre = "stree"
	if _, err := NewCatalog([]Facility{legacy}); err == nil {
		t.Fatal("unknown genre accepted")
	}
}
