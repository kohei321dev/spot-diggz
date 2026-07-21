package correction

import (
	"testing"
	"time"
)

func TestNewPostgresStoreRejectsEmptyURL(t *testing.T) {
	_, err := NewPostgresStore("  ", time.Now())
	if err == nil {
		t.Fatal("NewPostgresStore() error = nil, want invalid configuration error")
	}
	if !contains(err.Error(), "database URL is required") {
		t.Fatalf("NewPostgresStore() error = %v", err)
	}
}
