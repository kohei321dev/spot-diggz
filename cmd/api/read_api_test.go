package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/kohei321dev/spot-diggz/internal/apiauth"
	"github.com/kohei321dev/spot-diggz/internal/facility"
)

func TestBuildReadAPIRequiresNoOAuthOrDatabaseAndFailsClosed(t *testing.T) {
	now := time.Date(2026, 7, 19, 0, 0, 0, 0, time.UTC)
	catalog, err := facility.LoadCatalogFileAt("../../testdata/facilities.dev.json", now)
	if err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"API_OWNER_ID", "API_CLIENTS_JSON", "GOOGLE_MAPS_API_KEY", "GITHUB_CLIENT_ID", "GITHUB_CLIENT_SECRET", "AUTH_SECRET", "APP_BASE_URL"} {
		t.Setenv(key, "")
	}
	t.Setenv("APP_ENV", "production")
	t.Setenv("DATABASE_URL", "deliberately-unusable")
	if _, err := buildReadAPI(catalog, nil, func() time.Time { return now }); err == nil {
		t.Fatal("missing credentials accepted")
	}
	seed := sha256.Sum256([]byte(t.Name()))
	token := hex.EncodeToString(seed[:])
	digest := sha256.Sum256([]byte(token))
	raw, _ := json.Marshal([]apiauth.Credential{{ClientID: "test-bot", OwnerID: "test-owner", Scope: apiauth.ReadScope, ExpiresAt: now.Add(time.Hour), TokenSHA256: hex.EncodeToString(digest[:])}})
	t.Setenv("API_OWNER_ID", "test-owner")
	t.Setenv("API_CLIENTS_JSON", string(raw))
	handler, err := buildReadAPI(catalog, nil, func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest("GET", "/healthz", nil))
	if w.Code != 200 {
		t.Fatal("API mode requires legacy services")
	}
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest("GET", "/readyz", nil))
	if w.Code != 503 {
		t.Fatal("missing provider falsely ready")
	}
}
