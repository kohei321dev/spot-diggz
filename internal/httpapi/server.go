package httpapi

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"mime"
	"net/http"
	"strings"
	"time"

	"github.com/kohei321dev/spot-diggz/internal/correction"
	"github.com/kohei321dev/spot-diggz/internal/facility"
	"github.com/kohei321dev/spot-diggz/internal/geocoding"
	"github.com/kohei321dev/spot-diggz/internal/observability"
	"github.com/kohei321dev/spot-diggz/internal/ratelimit"
	"github.com/kohei321dev/spot-diggz/internal/recommendation"
	"github.com/kohei321dev/spot-diggz/internal/session"
	"github.com/kohei321dev/spot-diggz/internal/webui"
)

const (
	maxRecommendationRequestBytes   = 16 * 1024
	maxLocationSearchRequestBytes   = 1024
	maxCorrectionRequestBytes       = 8 * 1024
	maxEventRequestBytes            = 1024
	recommendationRequestsPerMinute = 60
	recommendationRequestBurst      = 10
	locationSearchRequestsPerMinute = 30
	locationSearchRequestBurst      = 5
	eventRequestsPerMinute          = 300
	eventRequestBurst               = 60
	correctionRequestsPerMinute     = 30
	correctionRequestBurst          = 10
)

type Options struct {
	Recommender           *recommendation.Engine
	Geocoder              geocoding.Provider
	CorrectionService     *correction.Service
	Metrics               *observability.Registry
	RecommendationLimiter *ratelimit.Limiter
	LocationSearchLimiter *ratelimit.Limiter
	CorrectionLimiter     *ratelimit.Limiter
	EventLimiter          *ratelimit.Limiter
	Now                   func() time.Time
}

type Server struct {
	catalog               *facility.Catalog
	recommender           *recommendation.Engine
	geocoder              geocoding.Provider
	correctionService     *correction.Service
	metrics               *observability.Registry
	recommendationLimiter *ratelimit.Limiter
	locationSearchLimiter *ratelimit.Limiter
	correctionLimiter     *ratelimit.Limiter
	eventLimiter          *ratelimit.Limiter
	logger                *slog.Logger
	now                   func() time.Time
}

func NewServer(catalog *facility.Catalog, logger *slog.Logger) http.Handler {
	return NewServerWithOptions(catalog, logger, Options{})
}

func NewServerWithOptions(catalog *facility.Catalog, logger *slog.Logger, options Options) http.Handler {
	if logger == nil {
		logger = slog.Default()
	}
	now := options.Now
	if now == nil {
		now = time.Now
	}
	if options.Recommender == nil {
		options.Recommender = recommendation.NewEngine(catalog, now)
	}
	if options.CorrectionService == nil {
		options.CorrectionService, _ = correction.NewService(correction.NewMemoryStore(), now)
	}
	if options.Metrics == nil {
		options.Metrics = observability.NewRegistry()
	}
	if options.RecommendationLimiter == nil {
		options.RecommendationLimiter = ratelimit.New(recommendationRequestsPerMinute, recommendationRequestBurst, now)
	}
	if options.LocationSearchLimiter == nil {
		options.LocationSearchLimiter = ratelimit.New(locationSearchRequestsPerMinute, locationSearchRequestBurst, now)
	}
	if options.CorrectionLimiter == nil {
		options.CorrectionLimiter = ratelimit.New(correctionRequestsPerMinute, correctionRequestBurst, now)
	}
	if options.EventLimiter == nil {
		options.EventLimiter = ratelimit.New(eventRequestsPerMinute, eventRequestBurst, now)
	}

	facilityCount, freshCount, staleCount := catalogFreshnessCounts(catalog, now())
	_ = options.Metrics.SetCatalog(facilityCount, freshCount, staleCount)

	server := &Server{
		catalog:               catalog,
		recommender:           options.Recommender,
		geocoder:              options.Geocoder,
		correctionService:     options.CorrectionService,
		metrics:               options.Metrics,
		recommendationLimiter: options.RecommendationLimiter,
		locationSearchLimiter: options.LocationSearchLimiter,
		correctionLimiter:     options.CorrectionLimiter,
		eventLimiter:          options.EventLimiter,
		logger:                logger,
		now:                   now,
	}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", server.health)
	mux.HandleFunc("GET /readyz", server.ready)
	mux.HandleFunc("GET /api/facilities", server.listFacilities)
	mux.HandleFunc("GET /api/facilities/{facilityId}", server.getFacility)
	mux.HandleFunc("POST /api/locations/search", server.searchLocations)
	mux.HandleFunc("POST /api/recommendations", server.recommend)
	mux.HandleFunc("POST /api/corrections", server.submitCorrection)
	mux.HandleFunc("POST /api/events", server.recordProductEvent)
	mux.HandleFunc("GET /metrics", server.serveMetrics)
	mux.Handle("GET /", webui.NewHandler())

	return server.withRequestID(server.withAccessLog(server.withMetrics(server.withSecurityHeaders(mux))))
}

func (s *Server) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) ready(w http.ResponseWriter, _ *http.Request) {
	freshCount := s.refreshCatalogMetrics()
	if freshCount == 0 {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "not_ready"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ready"})
}

func (s *Server) serveMetrics(w http.ResponseWriter, r *http.Request) {
	s.refreshCatalogMetrics()
	s.metrics.Handler().ServeHTTP(w, r)
}

func (s *Server) refreshCatalogMetrics() int {
	facilityCount, freshCount, staleCount := catalogFreshnessCounts(s.catalog, s.now())
	_ = s.metrics.SetCatalog(facilityCount, freshCount, staleCount)
	return freshCount
}

func catalogFreshnessCounts(catalog *facility.Catalog, now time.Time) (facilityCount int, freshCount int, staleCount int) {
	facilities := catalog.List("")
	for _, item := range facilities {
		if facility.IsDynamicInformationFresh(item.DynamicVerifiedAt, now) &&
			facility.IsStableInformationFresh(item.StableVerifiedAt, now) {
			freshCount++
		}
	}
	return len(facilities), freshCount, len(facilities) - freshCount
}

func (s *Server) listFacilities(w http.ResponseWriter, r *http.Request) {
	activity := strings.TrimSpace(r.URL.Query().Get("activity"))
	if len(activity) > 50 {
		writeError(w, http.StatusBadRequest, "invalid_query", "activity is too long")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"facilities": s.catalog.List(activity),
	})
}

func (s *Server) getFacility(w http.ResponseWriter, r *http.Request) {
	item, err := s.catalog.Find(r.PathValue("facilityId"))
	if errors.Is(err, facility.ErrNotFound) {
		writeError(w, http.StatusNotFound, "facility_not_found", "facility was not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "facility_lookup_failed", "facility lookup failed")
		return
	}

	writeJSON(w, http.StatusOK, item)
}

type locationSearchRequest struct {
	Query string `json:"query"`
}

func (s *Server) searchLocations(w http.ResponseWriter, r *http.Request) {
	if s.geocoder == nil {
		writeError(w, http.StatusServiceUnavailable, "location_search_unavailable", "location search is not configured")
		return
	}
	if !s.locationSearchLimiter.Allow() {
		writeRateLimitError(w)
		return
	}

	var input locationSearchRequest
	if !decodeJSONObject(w, r, maxLocationSearchRequestBytes, &input) {
		return
	}
	results, err := s.geocoder.Search(r.Context(), input.Query)
	if errors.Is(err, geocoding.ErrInvalidQuery) {
		writeError(w, http.StatusBadRequest, "invalid_location_query", "location query is invalid")
		return
	}
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, "location_search_unavailable", "location search is temporarily unavailable")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"results": results})
}

func (s *Server) recommend(w http.ResponseWriter, r *http.Request) {
	if !s.recommendationLimiter.Allow() {
		writeRateLimitError(w)
		return
	}

	var input session.Input
	if !decodeJSONObject(w, r, maxRecommendationRequestBytes, &input) {
		return
	}

	startedAt := time.Now()
	response, err := s.recommender.RecommendContext(r.Context(), input)
	result := observability.RecommendationResultSuccess
	if recommendation.IsInvalidInput(err) {
		result = observability.RecommendationResultError
		_ = s.metrics.ObserveRecommendation(result, time.Since(startedAt))
		writeError(w, http.StatusBadRequest, "invalid_session_input", "one or more session selections are invalid")
		return
	}
	if err != nil {
		result = observability.RecommendationResultError
		_ = s.metrics.ObserveRecommendation(result, time.Since(startedAt))
		writeError(w, http.StatusServiceUnavailable, "recommendation_unavailable", "recommendation is temporarily unavailable")
		return
	}
	if len(response.Recommendations) == 0 {
		result = observability.RecommendationResultNoResults
	}
	_ = s.metrics.ObserveRecommendation(result, time.Since(startedAt))

	writeJSON(w, http.StatusOK, response)
}

func (s *Server) submitCorrection(w http.ResponseWriter, r *http.Request) {
	if !s.correctionLimiter.Allow() {
		writeRateLimitError(w)
		return
	}

	var submission correction.Submission
	if !decodeJSONObject(w, r, maxCorrectionRequestBytes, &submission) {
		return
	}
	if _, err := s.catalog.Find(submission.FacilityID); errors.Is(err, facility.ErrNotFound) {
		writeError(w, http.StatusNotFound, "facility_not_found", "facility was not found")
		return
	}

	receipt, err := s.correctionService.Submit(r.Context(), submission)
	if errors.Is(err, correction.ErrInvalidSubmission) {
		writeError(w, http.StatusBadRequest, "invalid_correction", "one or more correction fields are invalid")
		return
	}
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, "correction_store_unavailable", "correction could not be stored")
		return
	}
	_ = s.metrics.ObserveProductEvent(observability.ProductEventCorrectionSubmitted)
	s.logger.InfoContext(r.Context(), "correction_received",
		"request_id", requestIDFromContext(r.Context()),
		"report_id", receipt.ReportID,
		"facility_id", submission.FacilityID,
		"category", submission.Category,
	)
	writeJSON(w, http.StatusAccepted, receipt)
}

type productEventRequest struct {
	Event observability.ProductEvent `json:"event"`
}

func (s *Server) recordProductEvent(w http.ResponseWriter, r *http.Request) {
	if !s.eventLimiter.Allow() {
		writeRateLimitError(w)
		return
	}

	var input productEventRequest
	if !decodeJSONObject(w, r, maxEventRequestBytes, &input) {
		return
	}
	if err := s.metrics.ObserveClientProductEvent(input.Event); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_product_event", "product event is not supported")
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]string{"status": "accepted"})
}

func decodeJSONObject(w http.ResponseWriter, r *http.Request, maxBytes int64, destination any) bool {
	mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || mediaType != "application/json" {
		writeError(w, http.StatusUnsupportedMediaType, "unsupported_media_type", "Content-Type must be application/json")
		return false
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		var maxBytesError *http.MaxBytesError
		if errors.As(err, &maxBytesError) {
			writeError(w, http.StatusRequestEntityTooLarge, "request_too_large", "request body is too large")
			return false
		}
		writeError(w, http.StatusBadRequest, "invalid_json", "request body must be valid JSON")
		return false
	}

	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		var maxBytesError *http.MaxBytesError
		if errors.As(err, &maxBytesError) {
			writeError(w, http.StatusRequestEntityTooLarge, "request_too_large", "request body is too large")
			return false
		}
		writeError(w, http.StatusBadRequest, "invalid_json", "request body must contain one JSON object")
		return false
	}
	return true
}

func writeRateLimitError(w http.ResponseWriter) {
	w.Header().Set("Retry-After", "60")
	writeError(w, http.StatusTooManyRequests, "rate_limited", "too many requests; retry later")
}

func (s *Server) withRequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := newRequestID()
		w.Header().Set("X-Request-ID", requestID)
		next.ServeHTTP(w, r.WithContext(withRequestIDValue(r.Context(), requestID)))
	})
}

func (s *Server) withAccessLog(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		startedAt := time.Now()
		response := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(response, r)
		routeTemplate := r.Pattern
		if routeTemplate == "" {
			routeTemplate = "unknown"
		}
		s.logger.InfoContext(r.Context(), "http_request",
			"request_id", requestIDFromContext(r.Context()),
			"method", r.Method,
			"route", routeTemplate,
			"status", response.status,
			"duration_ms", time.Since(startedAt).Milliseconds(),
		)
	})
}

func (s *Server) withMetrics(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		startedAt := time.Now()
		response := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(response, r)
		s.metrics.ObserveHTTP(r.Pattern, r.Method, response.status, time.Since(startedAt))
	})
}

func (s *Server) withSecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		w.Header().Set("Permissions-Policy", "geolocation=(self)")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; base-uri 'none'; connect-src 'self'; form-action 'self'; frame-ancestors 'none'; frame-src https://www.youtube-nocookie.com; img-src 'self' data:; script-src 'self'; style-src 'self'")
		w.Header().Set("Cache-Control", "no-store")
		next.ServeHTTP(w, r)
	})
}

type statusRecorder struct {
	http.ResponseWriter
	status      int
	wroteHeader bool
}

func (w *statusRecorder) WriteHeader(status int) {
	if w.wroteHeader {
		return
	}
	w.wroteHeader = true
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

func (w *statusRecorder) Write(payload []byte) (int, error) {
	if !w.wroteHeader {
		w.WriteHeader(http.StatusOK)
	}
	return w.ResponseWriter.Write(payload)
}

type requestIDContextKey struct{}

func withRequestIDValue(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, requestIDContextKey{}, requestID)
}

func requestIDFromContext(ctx context.Context) string {
	requestID, ok := ctx.Value(requestIDContextKey{}).(string)
	if !ok {
		return "unknown"
	}
	return requestID
}

func newRequestID() string {
	var bytes [16]byte
	if _, err := rand.Read(bytes[:]); err != nil {
		return "request-id-unavailable"
	}
	return hex.EncodeToString(bytes[:])
}

type errorBody struct {
	Error apiError `json:"error"`
}

type apiError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func writeError(w http.ResponseWriter, status int, code string, message string) {
	writeJSON(w, status, errorBody{Error: apiError{Code: code, Message: message}})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
