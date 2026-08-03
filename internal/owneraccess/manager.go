package owneraccess

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	defaultGitHubOwner       = "kohei321dev"
	defaultAuthorizeURL      = "https://github.com/login/oauth/authorize"
	defaultTokenURL          = "https://github.com/login/oauth/access_token"
	defaultUserURL           = "https://api.github.com/user"
	stateCookieName          = "spotdiggz_oauth_state"
	sessionCookieName        = "spotdiggz_session"
	stateLifetime            = 10 * time.Minute
	sessionLifetime          = 12 * time.Hour
	maximumProviderBodyBytes = 64 * 1024
)

type Config struct {
	Environment        string
	BaseURL            string
	SessionSecret      string
	GitHubClientID     string
	GitHubClientSecret string
	GitHubOwner        string
	DevBypass          bool
	HTTPClient         *http.Client
	Now                func() time.Time
	AuthorizeURL       string
	TokenURL           string
	UserURL            string
}

type Manager struct {
	baseURL            *url.URL
	sessionSecret      []byte
	githubClientID     string
	githubClientSecret string
	githubOwner        string
	devBypass          bool
	configured         bool
	secureCookies      bool
	httpClient         *http.Client
	now                func() time.Time
	authorizeURL       string
	tokenURL           string
	userURL            string
}

type stateClaims struct {
	State        string `json:"state"`
	CodeVerifier string `json:"codeVerifier"`
	ExpiresAt    int64  `json:"expiresAt"`
}

type sessionClaims struct {
	GitHubID    string `json:"githubId"`
	GitHubLogin string `json:"githubLogin"`
	ExpiresAt   int64  `json:"expiresAt"`
}

type githubTokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	Error       string `json:"error"`
}

type githubUser struct {
	ID    int64  `json:"id"`
	Login string `json:"login"`
}

func New(config Config) (*Manager, error) {
	now := config.Now
	if now == nil {
		now = time.Now
	}
	client := config.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 5 * time.Second}
	}
	owner := strings.TrimSpace(config.GitHubOwner)
	if owner == "" {
		owner = defaultGitHubOwner
	}
	baseURL, err := parseBaseURL(config.BaseURL)
	if err != nil {
		return nil, err
	}
	production := strings.EqualFold(strings.TrimSpace(config.Environment), "production")
	if production && baseURL != nil && baseURL.Scheme != "https" {
		return nil, errors.New("production APP_BASE_URL must use HTTPS")
	}
	devBypass := config.DevBypass && !production
	configured := strings.TrimSpace(config.SessionSecret) != "" &&
		strings.TrimSpace(config.GitHubClientID) != "" &&
		strings.TrimSpace(config.GitHubClientSecret) != "" &&
		baseURL != nil
	if production && !configured {
		return nil, errors.New("production owner authentication requires APP_BASE_URL, AUTH_SECRET, GITHUB_CLIENT_ID, and GITHUB_CLIENT_SECRET")
	}
	if configured && len(config.SessionSecret) < 32 {
		return nil, errors.New("AUTH_SECRET must contain at least 32 bytes")
	}

	return &Manager{
		baseURL:            baseURL,
		sessionSecret:      []byte(config.SessionSecret),
		githubClientID:     strings.TrimSpace(config.GitHubClientID),
		githubClientSecret: strings.TrimSpace(config.GitHubClientSecret),
		githubOwner:        owner,
		devBypass:          devBypass,
		configured:         configured,
		secureCookies:      baseURL != nil && baseURL.Scheme == "https",
		httpClient:         client,
		now:                now,
		authorizeURL:       valueOrDefault(config.AuthorizeURL, defaultAuthorizeURL),
		tokenURL:           valueOrDefault(config.TokenURL, defaultTokenURL),
		userURL:            valueOrDefault(config.UserURL, defaultUserURL),
	}, nil
}

func (manager *Manager) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /auth/github/start", manager.startGitHub)
	mux.HandleFunc("GET /auth/github/callback", manager.finishGitHub)
	mux.HandleFunc("GET /auth/setup", manager.setup)
	mux.HandleFunc("GET /auth/denied", manager.denied)
	mux.HandleFunc("GET /auth/session", manager.session)
	mux.HandleFunc("POST /auth/signout", manager.signOut)
}

func (manager *Manager) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if isPublicPath(r.URL.Path) || manager.devBypass {
			next.ServeHTTP(w, r)
			return
		}
		if !manager.configured {
			if isAPIPath(r.URL.Path) {
				writeAuthError(w, http.StatusServiceUnavailable, "auth_not_configured", "owner authentication is not configured")
				return
			}
			http.Redirect(w, r, "/auth/setup", http.StatusSeeOther)
			return
		}
		if _, ok := manager.authorizedSession(r); ok {
			next.ServeHTTP(w, r)
			return
		}
		if isAPIPath(r.URL.Path) {
			writeAuthError(w, http.StatusUnauthorized, "authentication_required", "GitHub owner authentication is required")
			return
		}
		http.Redirect(w, r, "/auth/github/start", http.StatusSeeOther)
	})
}

func (manager *Manager) startGitHub(w http.ResponseWriter, r *http.Request) {
	if manager.devBypass {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	if !manager.configured {
		http.Redirect(w, r, "/auth/setup", http.StatusSeeOther)
		return
	}
	state, err := randomValue(32)
	if err != nil {
		http.Error(w, "authentication could not be started", http.StatusInternalServerError)
		return
	}
	verifier, err := randomValue(48)
	if err != nil {
		http.Error(w, "authentication could not be started", http.StatusInternalServerError)
		return
	}
	claims := stateClaims{State: state, CodeVerifier: verifier, ExpiresAt: manager.now().Add(stateLifetime).Unix()}
	sealed, err := manager.seal(claims)
	if err != nil {
		http.Error(w, "authentication could not be started", http.StatusInternalServerError)
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     stateCookieName,
		Value:    sealed,
		Path:     "/auth/github/callback",
		MaxAge:   int(stateLifetime.Seconds()),
		HttpOnly: true,
		Secure:   manager.secureCookies,
		SameSite: http.SameSiteLaxMode,
	})

	authorizeURL, err := url.Parse(manager.authorizeURL)
	if err != nil {
		http.Error(w, "authentication provider is not configured", http.StatusInternalServerError)
		return
	}
	query := authorizeURL.Query()
	query.Set("client_id", manager.githubClientID)
	query.Set("redirect_uri", manager.callbackURL())
	query.Set("state", state)
	query.Set("code_challenge", pkceChallenge(verifier))
	query.Set("code_challenge_method", "S256")
	query.Set("allow_signup", "false")
	authorizeURL.RawQuery = query.Encode()
	http.Redirect(w, r, authorizeURL.String(), http.StatusSeeOther)
}

func (manager *Manager) finishGitHub(w http.ResponseWriter, r *http.Request) {
	manager.clearStateCookie(w)
	if !manager.configured {
		http.Redirect(w, r, "/auth/setup", http.StatusSeeOther)
		return
	}
	stateCookie, err := r.Cookie(stateCookieName)
	if err != nil {
		http.Redirect(w, r, "/auth/denied", http.StatusSeeOther)
		return
	}
	var state stateClaims
	if err := manager.unseal(stateCookie.Value, &state); err != nil || state.ExpiresAt < manager.now().Unix() || !hmac.Equal([]byte(state.State), []byte(r.URL.Query().Get("state"))) {
		http.Redirect(w, r, "/auth/denied", http.StatusSeeOther)
		return
	}
	code := strings.TrimSpace(r.URL.Query().Get("code"))
	if code == "" {
		http.Redirect(w, r, "/auth/denied", http.StatusSeeOther)
		return
	}
	token, err := manager.exchangeCode(r, code, state.CodeVerifier)
	if err != nil {
		http.Redirect(w, r, "/auth/denied", http.StatusSeeOther)
		return
	}
	user, err := manager.loadGitHubUser(r, token)
	if err != nil || !strings.EqualFold(user.Login, manager.githubOwner) {
		http.Redirect(w, r, "/auth/denied", http.StatusSeeOther)
		return
	}
	claims := sessionClaims{
		GitHubID:    fmt.Sprintf("%d", user.ID),
		GitHubLogin: user.Login,
		ExpiresAt:   manager.now().Add(sessionLifetime).Unix(),
	}
	sealed, err := manager.seal(claims)
	if err != nil {
		http.Error(w, "session could not be created", http.StatusInternalServerError)
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    sealed,
		Path:     "/",
		MaxAge:   int(sessionLifetime.Seconds()),
		HttpOnly: true,
		Secure:   manager.secureCookies,
		SameSite: http.SameSiteLaxMode,
	})
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (manager *Manager) exchangeCode(r *http.Request, code string, verifier string) (string, error) {
	form := url.Values{
		"client_id":     {manager.githubClientID},
		"client_secret": {manager.githubClientSecret},
		"code":          {code},
		"redirect_uri":  {manager.callbackURL()},
		"code_verifier": {verifier},
	}
	request, err := http.NewRequestWithContext(r.Context(), http.MethodPost, manager.tokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	request.Header.Set("Accept", "application/json")
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	response, err := manager.httpClient.Do(request)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, maximumProviderBodyBytes))
		return "", errors.New("GitHub token exchange failed")
	}
	var payload githubTokenResponse
	if err := json.NewDecoder(io.LimitReader(response.Body, maximumProviderBodyBytes)).Decode(&payload); err != nil {
		return "", err
	}
	if payload.Error != "" || payload.AccessToken == "" {
		return "", errors.New("GitHub token exchange returned no token")
	}
	return payload.AccessToken, nil
}

func (manager *Manager) loadGitHubUser(r *http.Request, accessToken string) (githubUser, error) {
	request, err := http.NewRequestWithContext(r.Context(), http.MethodGet, manager.userURL, nil)
	if err != nil {
		return githubUser{}, err
	}
	request.Header.Set("Accept", "application/vnd.github+json")
	request.Header.Set("Authorization", "Bearer "+accessToken)
	request.Header.Set("User-Agent", "spot-diggz")
	response, err := manager.httpClient.Do(request)
	if err != nil {
		return githubUser{}, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, maximumProviderBodyBytes))
		return githubUser{}, errors.New("GitHub user lookup failed")
	}
	var user githubUser
	if err := json.NewDecoder(io.LimitReader(response.Body, maximumProviderBodyBytes)).Decode(&user); err != nil {
		return githubUser{}, err
	}
	if user.ID <= 0 || strings.TrimSpace(user.Login) == "" {
		return githubUser{}, errors.New("GitHub user response is incomplete")
	}
	return user, nil
}

func (manager *Manager) setup(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusServiceUnavailable)
	_, _ = io.WriteString(w, `<!doctype html><html lang="ja"><meta charset="utf-8"><title>認証設定が必要です</title><main><h1>GitHub認証の設定が必要です</h1><p>APP_BASE_URL、AUTH_SECRET、GITHUB_CLIENT_ID、GITHUB_CLIENT_SECRETをserver-side環境変数へ設定してください。</p><p>ローカル画面確認だけを行う場合はDEV_AUTH_BYPASS=1を使用できます。productionでは無効です。</p></main></html>`)
}

func (manager *Manager) denied(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusForbidden)
	_, _ = io.WriteString(w, `<!doctype html><html lang="ja"><meta charset="utf-8"><title>利用できません</title><main><h1>このアカウントでは利用できません</h1><p>許可されたGitHubアカウントでサインインしてください。</p></main></html>`)
}

func (manager *Manager) session(w http.ResponseWriter, r *http.Request) {
	if manager.devBypass {
		writeJSON(w, http.StatusOK, map[string]string{"githubLogin": manager.githubOwner, "role": "owner", "mode": "development_bypass"})
		return
	}
	claims, ok := manager.authorizedSession(r)
	if !ok {
		writeAuthError(w, http.StatusUnauthorized, "authentication_required", "GitHub owner authentication is required")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"githubLogin": claims.GitHubLogin, "role": "owner"})
}

func (manager *Manager) signOut(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{Name: sessionCookieName, Value: "", Path: "/", MaxAge: -1, HttpOnly: true, Secure: manager.secureCookies, SameSite: http.SameSiteLaxMode})
	http.Redirect(w, r, "/auth/github/start", http.StatusSeeOther)
}

func (manager *Manager) authorizedSession(r *http.Request) (sessionClaims, bool) {
	cookie, err := r.Cookie(sessionCookieName)
	if err != nil {
		return sessionClaims{}, false
	}
	var claims sessionClaims
	if err := manager.unseal(cookie.Value, &claims); err != nil {
		return sessionClaims{}, false
	}
	if claims.ExpiresAt < manager.now().Unix() || !strings.EqualFold(claims.GitHubLogin, manager.githubOwner) || claims.GitHubID == "" {
		return sessionClaims{}, false
	}
	return claims, true
}

func (manager *Manager) seal(value any) (string, error) {
	payload, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	encodedPayload := base64.RawURLEncoding.EncodeToString(payload)
	mac := hmac.New(sha256.New, manager.sessionSecret)
	_, _ = mac.Write([]byte(encodedPayload))
	signature := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	return encodedPayload + "." + signature, nil
}

func (manager *Manager) unseal(value string, destination any) error {
	parts := strings.Split(value, ".")
	if len(parts) != 2 {
		return errors.New("invalid sealed value")
	}
	providedSignature, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return errors.New("invalid sealed signature")
	}
	mac := hmac.New(sha256.New, manager.sessionSecret)
	_, _ = mac.Write([]byte(parts[0]))
	if !hmac.Equal(providedSignature, mac.Sum(nil)) {
		return errors.New("sealed signature does not match")
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return errors.New("invalid sealed payload")
	}
	if err := json.Unmarshal(payload, destination); err != nil {
		return errors.New("invalid sealed claims")
	}
	return nil
}

func (manager *Manager) callbackURL() string {
	return strings.TrimRight(manager.baseURL.String(), "/") + "/auth/github/callback"
}

func (manager *Manager) clearStateCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{Name: stateCookieName, Value: "", Path: "/auth/github/callback", MaxAge: -1, HttpOnly: true, Secure: manager.secureCookies, SameSite: http.SameSiteLaxMode})
}

func parseBaseURL(raw string) (*url.URL, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, nil
	}
	parsed, err := url.Parse(strings.TrimRight(strings.TrimSpace(raw), "/"))
	if err != nil || parsed.Host == "" || parsed.User != nil || (parsed.Scheme != "https" && parsed.Scheme != "http") || (parsed.Path != "" && parsed.Path != "/") || parsed.RawQuery != "" || parsed.Fragment != "" {
		return nil, errors.New("APP_BASE_URL must be an absolute HTTP(S) origin without credentials, path, query, or fragment")
	}
	return parsed, nil
}

func randomValue(size int) (string, error) {
	buffer := make([]byte, size)
	if _, err := rand.Read(buffer); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buffer), nil
}

func pkceChallenge(verifier string) string {
	digest := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(digest[:])
}

func valueOrDefault(value string, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return strings.TrimSpace(value)
}

func isPublicPath(path string) bool {
	return path == "/healthz" || path == "/readyz" || path == "/metrics" || strings.HasPrefix(path, "/auth/") || strings.HasPrefix(path, "/integrations/")
}

func isAPIPath(path string) bool {
	return path == "/api" || strings.HasPrefix(path, "/api/")
}

func writeAuthError(w http.ResponseWriter, status int, code string, message string) {
	writeJSON(w, status, map[string]any{"error": map[string]string{"code": code, "message": message}})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
