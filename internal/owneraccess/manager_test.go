package owneraccess

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

var fixedTime = time.Date(2026, time.August, 1, 12, 0, 0, 0, time.UTC)

func TestProductionRequiresGitHubAuthenticationConfiguration(t *testing.T) {
	_, err := New(Config{Environment: "production", GitHubOwner: "kohei321dev"})
	if err == nil {
		t.Fatal("New() error = nil, want missing production auth configuration error")
	}
}

func TestProductionIgnoresDevelopmentBypass(t *testing.T) {
	_, err := New(Config{Environment: "production", DevBypass: true})
	if err == nil {
		t.Fatal("New() error = nil, want production configuration error")
	}
}

func TestProductionRequiresHTTPSBaseURL(t *testing.T) {
	_, err := New(Config{
		Environment:        "production",
		BaseURL:            "http://spot.example.com",
		SessionSecret:      strings.Repeat("s", 32),
		GitHubClientID:     "client-id",
		GitHubClientSecret: "client-secret",
	})
	if err == nil || !strings.Contains(err.Error(), "HTTPS") {
		t.Fatalf("New() error = %v, want Production HTTPS requirement", err)
	}
}

func TestBaseURLRejectsPath(t *testing.T) {
	_, err := New(Config{Environment: "development", BaseURL: "http://localhost:8080/app"})
	if err == nil {
		t.Fatal("New() error = nil, want APP_BASE_URL origin validation error")
	}
}

func TestDevelopmentBypassAllowsProtectedRoutes(t *testing.T) {
	manager, err := New(Config{Environment: "development", DevBypass: true, GitHubOwner: "kohei321dev", Now: func() time.Time { return fixedTime }})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	handler := manager.Middleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/recommendations", nil))
	if response.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusNoContent)
	}
}

func TestMissingDevelopmentConfigurationFailsClosed(t *testing.T) {
	manager, err := New(Config{Environment: "development", GitHubOwner: "kohei321dev"})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	handler := manager.Middleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		t.Fatal("protected handler was called")
	}))

	apiResponse := httptest.NewRecorder()
	handler.ServeHTTP(apiResponse, httptest.NewRequest(http.MethodGet, "/api/facilities", nil))
	if apiResponse.Code != http.StatusServiceUnavailable || !strings.Contains(apiResponse.Body.String(), "auth_not_configured") {
		t.Fatalf("API response = %d %s", apiResponse.Code, apiResponse.Body.String())
	}

	pageResponse := httptest.NewRecorder()
	handler.ServeHTTP(pageResponse, httptest.NewRequest(http.MethodGet, "/", nil))
	if pageResponse.Code != http.StatusSeeOther || pageResponse.Header().Get("Location") != "/auth/setup" {
		t.Fatalf("page response = %d location=%q", pageResponse.Code, pageResponse.Header().Get("Location"))
	}
}

func TestGitHubOwnerCompletesOAuthAndReceivesSession(t *testing.T) {
	provider := newGitHubProvider(t, "kohei321dev")
	defer provider.Close()
	manager := newConfiguredManager(t, provider.URL)
	mux := http.NewServeMux()
	manager.RegisterRoutes(mux)
	protected := manager.Middleware(mux)

	startResponse := httptest.NewRecorder()
	protected.ServeHTTP(startResponse, httptest.NewRequest(http.MethodGet, "/auth/github/start", nil))
	if startResponse.Code != http.StatusSeeOther {
		t.Fatalf("start status = %d", startResponse.Code)
	}
	stateCookie := findCookie(t, startResponse.Result(), stateCookieName)
	authorizeLocation, err := url.Parse(startResponse.Header().Get("Location"))
	if err != nil {
		t.Fatalf("parse authorize location: %v", err)
	}
	if authorizeLocation.Query().Get("code_challenge") == "" || authorizeLocation.Query().Get("code_challenge_method") != "S256" {
		t.Fatalf("authorize query = %s", authorizeLocation.RawQuery)
	}

	callbackRequest := httptest.NewRequest(http.MethodGet, "/auth/github/callback?code=test-code&state="+url.QueryEscape(authorizeLocation.Query().Get("state")), nil)
	callbackRequest.AddCookie(stateCookie)
	callbackResponse := httptest.NewRecorder()
	protected.ServeHTTP(callbackResponse, callbackRequest)
	if callbackResponse.Code != http.StatusSeeOther || callbackResponse.Header().Get("Location") != "/" {
		t.Fatalf("callback response = %d location=%q body=%s", callbackResponse.Code, callbackResponse.Header().Get("Location"), callbackResponse.Body.String())
	}
	sessionCookie := findCookie(t, callbackResponse.Result(), sessionCookieName)
	if !sessionCookie.HttpOnly || !sessionCookie.Secure || sessionCookie.SameSite != http.SameSiteLaxMode {
		t.Fatalf("session cookie security attributes = %#v", sessionCookie)
	}

	protectedMux := http.NewServeMux()
	protectedMux.HandleFunc("GET /api/private", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) })
	privateRequest := httptest.NewRequest(http.MethodGet, "/api/private", nil)
	privateRequest.AddCookie(sessionCookie)
	privateResponse := httptest.NewRecorder()
	manager.Middleware(protectedMux).ServeHTTP(privateResponse, privateRequest)
	if privateResponse.Code != http.StatusNoContent {
		t.Fatalf("private status = %d body=%s", privateResponse.Code, privateResponse.Body.String())
	}
}

func TestGitHubNonOwnerIsDenied(t *testing.T) {
	provider := newGitHubProvider(t, "someone-else")
	defer provider.Close()
	manager := newConfiguredManager(t, provider.URL)
	mux := http.NewServeMux()
	manager.RegisterRoutes(mux)

	startResponse := httptest.NewRecorder()
	mux.ServeHTTP(startResponse, httptest.NewRequest(http.MethodGet, "/auth/github/start", nil))
	stateCookie := findCookie(t, startResponse.Result(), stateCookieName)
	authorizeLocation, _ := url.Parse(startResponse.Header().Get("Location"))
	callbackRequest := httptest.NewRequest(http.MethodGet, "/auth/github/callback?code=test-code&state="+url.QueryEscape(authorizeLocation.Query().Get("state")), nil)
	callbackRequest.AddCookie(stateCookie)
	callbackResponse := httptest.NewRecorder()
	mux.ServeHTTP(callbackResponse, callbackRequest)
	if callbackResponse.Code != http.StatusSeeOther || callbackResponse.Header().Get("Location") != "/auth/denied" {
		t.Fatalf("callback response = %d location=%q", callbackResponse.Code, callbackResponse.Header().Get("Location"))
	}
	for _, cookie := range callbackResponse.Result().Cookies() {
		if cookie.Name == sessionCookieName && cookie.MaxAge >= 0 {
			t.Fatalf("non-owner received session cookie: %#v", cookie)
		}
	}
}

func TestTamperedSessionIsRejected(t *testing.T) {
	manager := newConfiguredManager(t, "https://provider.example")
	handler := manager.Middleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) }))
	request := httptest.NewRequest(http.MethodGet, "/api/private", nil)
	request.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "tampered.value"})
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusUnauthorized || !strings.Contains(response.Body.String(), "authentication_required") {
		t.Fatalf("response = %d %s", response.Code, response.Body.String())
	}
}

func newConfiguredManager(t *testing.T, providerURL string) *Manager {
	t.Helper()
	manager, err := New(Config{
		Environment:        "production",
		BaseURL:            "https://spot.example.com",
		SessionSecret:      strings.Repeat("s", 32),
		GitHubClientID:     "client-id",
		GitHubClientSecret: "client-secret",
		GitHubOwner:        "kohei321dev",
		Now:                func() time.Time { return fixedTime },
		AuthorizeURL:       providerURL + "/authorize",
		TokenURL:           providerURL + "/token",
		UserURL:            providerURL + "/user",
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	return manager
}

func newGitHubProvider(t *testing.T, login string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/token":
			if r.Method != http.MethodPost {
				t.Fatalf("token method = %s", r.Method)
			}
			if err := r.ParseForm(); err != nil {
				t.Fatalf("ParseForm() error = %v", err)
			}
			if r.Form.Get("code") != "test-code" || r.Form.Get("code_verifier") == "" {
				t.Fatalf("token form = %#v", r.Form)
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]string{"access_token": "provider-token", "token_type": "bearer"})
		case "/user":
			if r.Header.Get("Authorization") != "Bearer provider-token" {
				t.Fatalf("Authorization = %q", r.Header.Get("Authorization"))
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{"id": 12345, "login": login})
		default:
			http.NotFound(w, r)
		}
	}))
}

func findCookie(t *testing.T, response *http.Response, name string) *http.Cookie {
	t.Helper()
	for _, cookie := range response.Cookies() {
		if cookie.Name == name && cookie.MaxAge >= 0 {
			return cookie
		}
	}
	t.Fatalf("cookie %q was not set", name)
	return nil
}
