package geocoding

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGoogleProviderSearchesJapaneseLocations(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("address"); got != "神戸駅" {
			t.Errorf("address = %q", got)
		}
		if got := r.URL.Query().Get("components"); got != "country:JP" {
			t.Errorf("components = %q", got)
		}
		if got := r.URL.Query().Get("key"); got != "test-key" {
			t.Errorf("key = %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"status":"OK",
			"results":[{
				"formatted_address":"日本、兵庫県神戸市中央区相生町３丁目 神戸駅",
				"geometry":{"location":{"lat":34.6797,"lng":135.1780}}
			}]
		}`))
	}))
	defer server.Close()

	provider, err := newGoogleProvider("test-key", server.URL, server.Client())
	if err != nil {
		t.Fatalf("newGoogleProvider() error = %v", err)
	}
	results, err := provider.Search(context.Background(), " 神戸駅 ")
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}
	if len(results) != 1 || results[0].Location.Latitude != 34.6797 {
		t.Fatalf("results = %#v", results)
	}
}

func TestGoogleProviderReturnsEmptyForZeroResults(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"status":"ZERO_RESULTS","results":[]}`))
	}))
	defer server.Close()
	provider, err := newGoogleProvider("test-key", server.URL, server.Client())
	if err != nil {
		t.Fatalf("newGoogleProvider() error = %v", err)
	}
	results, err := provider.Search(context.Background(), "存在しない場所")
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}
	if len(results) != 0 {
		t.Fatalf("results = %#v", results)
	}
}

func TestGoogleProviderRejectsInvalidQuery(t *testing.T) {
	provider, err := newGoogleProvider("test-key", "https://example.com", http.DefaultClient)
	if err != nil {
		t.Fatalf("newGoogleProvider() error = %v", err)
	}
	if _, err := provider.Search(context.Background(), "\n"); err == nil {
		t.Fatal("Search() error = nil")
	}
}
