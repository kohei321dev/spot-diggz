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

	"github.com/kohei321dev/spot-diggz/internal/facility"
	"github.com/kohei321dev/spot-diggz/internal/recommendation"
	"github.com/kohei321dev/spot-diggz/internal/session"
	"github.com/kohei321dev/spot-diggz/internal/webui"
)

const maxRecommendationRequestBytes = 16 * 1024

type Server struct {
	catalog     *facility.Catalog
	recommender *recommendation.Engine
	logger      *slog.Logger
}

func NewServer(catalog *facility.Catalog, logger *slog.Logger) http.Handler {
	if logger == nil {
		logger = slog.Default()
	}

	server := &Server{
		catalog:     catalog,
		recommender: recommendation.NewEngine(catalog, time.Now),
		logger:      logger,
	}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", server.health)
	mux.HandleFunc("GET /api/facilities", server.listFacilities)
	mux.HandleFunc("GET /api/facilities/{facilityId}", server.getFacility)
	mux.HandleFunc("POST /api/recommendations", server.recommend)
	mux.Handle("GET /", webui.NewHandler())

	return server.withRequestID(server.withAccessLog(server.withSecurityHeaders(mux)))
}

func (s *Server) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
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

func (s *Server) recommend(w http.ResponseWriter, r *http.Request) {
	mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || mediaType != "application/json" {
		writeError(w, http.StatusUnsupportedMediaType, "unsupported_media_type", "Content-Type must be application/json")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxRecommendationRequestBytes)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	var input session.Input
	if err := decoder.Decode(&input); err != nil {
		var maxBytesError *http.MaxBytesError
		if errors.As(err, &maxBytesError) {
			writeError(w, http.StatusRequestEntityTooLarge, "request_too_large", "request body is too large")
			return
		}
		writeError(w, http.StatusBadRequest, "invalid_json", "request body must be valid JSON")
		return
	}

	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		var maxBytesError *http.MaxBytesError
		if errors.As(err, &maxBytesError) {
			writeError(w, http.StatusRequestEntityTooLarge, "request_too_large", "request body is too large")
			return
		}
		writeError(w, http.StatusBadRequest, "invalid_json", "request body must contain one JSON object")
		return
	}

	response, err := s.recommender.Recommend(input)
	if recommendation.IsInvalidInput(err) {
		writeError(w, http.StatusBadRequest, "invalid_session_input", "one or more session selections are invalid")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "recommendation_failed", "recommendation failed")
		return
	}

	writeJSON(w, http.StatusOK, response)
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
		s.logger.InfoContext(r.Context(), "http_request",
			"request_id", requestIDFromContext(r.Context()),
			"method", r.Method,
			"path", r.URL.Path,
			"status", response.status,
			"duration_ms", time.Since(startedAt).Milliseconds(),
		)
	})
}

func (s *Server) withSecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("Permissions-Policy", "geolocation=(self)")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; base-uri 'none'; connect-src 'self'; form-action 'self'; frame-ancestors 'none'; img-src 'self' data:; script-src 'self'; style-src 'self'")
		w.Header().Set("Cache-Control", "no-store")
		next.ServeHTTP(w, r)
	})
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (w *statusRecorder) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
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
