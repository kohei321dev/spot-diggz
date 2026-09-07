package httpapi

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/kohei321dev/spot-diggz/internal/apiauth"
	"github.com/kohei321dev/spot-diggz/internal/facility"
	"github.com/kohei321dev/spot-diggz/internal/geocoding"
	"github.com/kohei321dev/spot-diggz/internal/nearby"
	"github.com/kohei321dev/spot-diggz/internal/observability"
	"github.com/kohei321dev/spot-diggz/internal/ratelimit"
)

const maxSearchRequestBytes = 4096

// NewReadAPI registers an explicit allowlist of routes, independent of Web/OAuth/DB.
// Even callers constructing a handler without credentials cannot bypass auth.
func NewReadAPI(catalog *facility.Catalog, credentials *apiauth.Credentials, geocoder geocoding.Provider, logger *slog.Logger, metrics *observability.Registry, now func() time.Time) http.Handler {
	if now == nil {
		now = time.Now
	}
	if logger == nil {
		logger = slog.Default()
	}
	if metrics == nil {
		metrics = observability.NewRegistry()
	}
	server := &Server{catalog: catalog, logger: logger, metrics: metrics, now: now}
	search := nearby.NewService(catalog, geocoder, now)
	searchLimiter := ratelimit.New(locationSearchRequestsPerMinute, locationSearchRequestBurst, now)
	readLimiter := ratelimit.New(recommendationRequestsPerMinute, recommendationRequestBurst, now)
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", server.health)
	mux.HandleFunc("GET /readyz", func(w http.ResponseWriter, _ *http.Request) {
		if credentials == nil || !search.Ready() {
			writeError(w, http.StatusServiceUnavailable, "not_ready", "verified searchable data and location provider are required")
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "ready"})
	})
	authorize := func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			if credentials == nil {
				writeError(w, http.StatusServiceUnavailable, "api_auth_unavailable", "API authentication is not configured; contact the operator")
				return
			}
			if len(r.Header.Values("Authorization")) != 1 || !credentials.Authorize(r.Header.Get("Authorization")) {
				w.Header().Set("WWW-Authenticate", `Bearer realm="spotdiggz"`)
				writeError(w, http.StatusUnauthorized, "invalid_token", "a valid API client token is required; check or renew the credential")
				return
			}
			if r.URL.RawQuery != "" {
				writeError(w, http.StatusBadRequest, "unexpected_query", "send search parameters in a JSON body, not the URL")
				return
			}
			next(w, r)
		}
	}
	mux.HandleFunc("POST /api/facilities/search", authorize(func(w http.ResponseWriter, r *http.Request) {
		if !searchLimiter.Allow() {
			writeRateLimitError(w)
			return
		}
		var body json.RawMessage
		if !decodeJSONObject(w, r, maxSearchRequestBytes, &body) {
			return
		}
		input, err := nearby.DecodeInput(body)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid_search_input", "check query, genre, limit (1-10), radiusKm (0.1-50), and sort (distance)")
			return
		}
		response, err := search.Search(r.Context(), input)
		if errors.Is(err, nearby.ErrUnavailable) {
			writeError(w, http.StatusServiceUnavailable, "location_search_unavailable", "location search is unavailable; retry later or contact the operator")
			return
		}
		if err != nil {
			writeError(w, http.StatusInternalServerError, "search_failed", "search failed; retry later")
			return
		}
		logger.InfoContext(r.Context(), "nearby_search_completed", "request_id", requestIDFromContext(r.Context()), "result", response.Status, "count", len(response.Matches))
		writeJSON(w, http.StatusOK, response)
	}))
	mux.HandleFunc("GET /api/facilities/{facilityId}", authorize(func(w http.ResponseWriter, r *http.Request) {
		if !readLimiter.Allow() {
			writeRateLimitError(w)
			return
		}
		server.getFacility(w, r)
	}))
	// Unknown URLs never fall through to the legacy Web UI.
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		writeError(w, http.StatusNotFound, "not_found", "endpoint not found; check the API contract")
	})
	return server.withRequestID(server.withAccessLog(server.withMetrics(server.withSecurityHeaders(mux))))
}
