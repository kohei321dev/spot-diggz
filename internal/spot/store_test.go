package spot

import (
	"context"
	"testing"
)

func TestMemoryStoreCreateListUpdateDelete(t *testing.T) {
	store := NewSdzMemoryStore()

	created, err := store.Create(context.Background(), SdzCreateSpotInput{
		Name:        " Demo Ledge ",
		SdzLocation: &SdzLocation{Lat: 35.6812, Lng: 139.7671},
		Tags:        []string{"ledge", "street", "ledge"},
	})
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}
	if created.SdzSpotID == "" {
		t.Fatal("Create returned empty spot id")
	}
	if created.Name != "Demo Ledge" {
		t.Fatalf("name was not normalized: %q", created.Name)
	}
	if created.SdzVisibility != SdzVisibilityPublic {
		t.Fatalf("default visibility = %q, want %q", created.SdzVisibility, SdzVisibilityPublic)
	}
	if len(created.Tags) != 2 {
		t.Fatalf("tags were not deduplicated: %#v", created.Tags)
	}

	bbox := SdzBBox{MinLng: 139, MinLat: 35, MaxLng: 140, MaxLat: 36}
	visibility := SdzVisibilityPublic
	listed, err := store.List(context.Background(), SdzListFilter{
		SdzBBox:       &bbox,
		Tags:          []string{"ledge"},
		SdzVisibility: &visibility,
	})
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}
	if len(listed) != 1 {
		t.Fatalf("List returned %d spots, want 1", len(listed))
	}

	nextName := "Updated Ledge"
	updated, err := store.Update(context.Background(), created.SdzSpotID, SdzUpdateSpotInput{Name: &nextName})
	if err != nil {
		t.Fatalf("Update returned error: %v", err)
	}
	if updated.Name != nextName {
		t.Fatalf("updated name = %q, want %q", updated.Name, nextName)
	}

	if err := store.Delete(context.Background(), created.SdzSpotID); err != nil {
		t.Fatalf("Delete returned error: %v", err)
	}
	if _, err := store.Get(context.Background(), created.SdzSpotID); err != SdzErrNotFound {
		t.Fatalf("Get after Delete error = %v, want SdzErrNotFound", err)
	}
}

func TestCreateRequiresLocation(t *testing.T) {
	store := NewSdzMemoryStore()

	_, err := store.Create(context.Background(), SdzCreateSpotInput{Name: "No SdzLocation"})
	if err == nil {
		t.Fatal("Create returned nil error")
	}
	if _, ok := err.(SdzValidationError); !ok {
		t.Fatalf("Create error type = %T, want SdzValidationError", err)
	}
}
