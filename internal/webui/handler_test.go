package webui

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHandlerServesApplicationAndAssets(t *testing.T) {
	handler := NewHandler()
	tests := []struct {
		path        string
		contentType string
		contains    []string
	}{
		{
			path:        "/",
			contentType: "text/html",
			contains: []string{
				`id="quick-search-button"`,
				`id="locale-switch"`,
				`id="location-search-results"`,
				`id="correction-dialog"`,
			},
		},
		{
			path:        "/assets/app.css",
			contentType: "text/css",
			contains:    []string{".mood-action-grid", ".location-search-result", ".correction-dialog"},
		},
		{
			path:        "/assets/app.js",
			contentType: "text/javascript",
			contains: []string{
				"spotdiggz.locale.v1",
				"englishTranslation",
				"/api/locations/search",
				"/api/corrections",
				"/api/events",
				"google_routes",
				"tokushima-station",
			},
		},
	}

	for _, test := range tests {
		t.Run(test.path, func(t *testing.T) {
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, test.path, nil))
			if response.Code != http.StatusOK {
				t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
			}
			if got := response.Header().Get("Content-Type"); !strings.HasPrefix(got, test.contentType) {
				t.Fatalf("Content-Type = %q, want prefix %q", got, test.contentType)
			}
			for _, marker := range test.contains {
				if !strings.Contains(response.Body.String(), marker) {
					t.Fatalf("body does not contain %q", marker)
				}
			}
		})
	}
}

func TestHandlerReturnsNotFoundForUnknownPath(t *testing.T) {
	response := httptest.NewRecorder()
	NewHandler().ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/unknown", nil))
	if response.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusNotFound)
	}
}
