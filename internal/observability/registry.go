package observability

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"
)

var (
	ErrUnknownProductEvent         = errors.New("unknown product event")
	ErrUnknownRecommendationResult = errors.New("unknown recommendation result")
	ErrUnknownExternalProvider     = errors.New("unknown external provider")
	ErrUnknownExternalResult       = errors.New("unknown external result")
	ErrInvalidCatalogCounts        = errors.New("invalid catalog counts")
)

// ProductEvent is a controlled, low-cardinality product journey event.
type ProductEvent string

const (
	ProductEventInputStarted            ProductEvent = "input_started"
	ProductEventInputCompleted          ProductEvent = "input_completed"
	ProductEventRecommendationCompleted ProductEvent = "recommendation_completed"
	ProductEventResultDisplayed         ProductEvent = "result_displayed"
	ProductEventSourceOpened            ProductEvent = "source_opened"
	ProductEventNavigationOpened        ProductEvent = "navigation_opened"
	ProductEventVideoEmbedDisplayed     ProductEvent = "video_embed_displayed"
	ProductEventVideoEmbedLoaded        ProductEvent = "video_embed_loaded"
	ProductEventVideoExternalOpened     ProductEvent = "video_external_opened"
	ProductEventSocialProfileOpened     ProductEvent = "social_profile_opened"
	ProductEventCorrectionSubmitted     ProductEvent = "correction_submitted"
)

var productEvents = [...]ProductEvent{
	ProductEventInputStarted,
	ProductEventInputCompleted,
	ProductEventRecommendationCompleted,
	ProductEventResultDisplayed,
	ProductEventSourceOpened,
	ProductEventNavigationOpened,
	ProductEventVideoEmbedDisplayed,
	ProductEventVideoEmbedLoaded,
	ProductEventVideoExternalOpened,
	ProductEventSocialProfileOpened,
	ProductEventCorrectionSubmitted,
}

var clientProductEvents = [...]ProductEvent{
	ProductEventInputStarted,
	ProductEventInputCompleted,
	ProductEventRecommendationCompleted,
	ProductEventResultDisplayed,
	ProductEventSourceOpened,
	ProductEventNavigationOpened,
	ProductEventVideoEmbedDisplayed,
	ProductEventVideoEmbedLoaded,
	ProductEventVideoExternalOpened,
	ProductEventSocialProfileOpened,
}

// RecommendationResult is the observable outcome of a recommendation.
type RecommendationResult string

const (
	RecommendationResultSuccess   RecommendationResult = "success"
	RecommendationResultNoResults RecommendationResult = "no_results"
	RecommendationResultError     RecommendationResult = "error"
)

var recommendationResults = [...]RecommendationResult{
	RecommendationResultSuccess,
	RecommendationResultNoResults,
	RecommendationResultError,
}

type ExternalProvider string

const (
	ExternalProviderGoogleRoutes    ExternalProvider = "google_routes"
	ExternalProviderGoogleGeocoding ExternalProvider = "google_geocoding"
)

var externalProviders = [...]ExternalProvider{
	ExternalProviderGoogleRoutes,
	ExternalProviderGoogleGeocoding,
}

type ExternalResult string

const (
	ExternalResultSuccess ExternalResult = "success"
	ExternalResultError   ExternalResult = "error"
)

var externalResults = [...]ExternalResult{
	ExternalResultSuccess,
	ExternalResultError,
}

// The finite bounds are fixed so dashboards remain comparable across releases.
var latencyBucketUpperBoundsSeconds = [...]float64{
	0.005,
	0.01,
	0.025,
	0.05,
	0.1,
	0.25,
	0.5,
	1,
	2.5,
	5,
	10,
}

type histogram struct {
	buckets [len(latencyBucketUpperBoundsSeconds)]uint64
	count   uint64
	sum     float64
}

func (h *histogram) observe(latency time.Duration) {
	seconds := latency.Seconds()
	if seconds < 0 {
		seconds = 0
	}

	h.count++
	h.sum += seconds
	for index, upperBound := range latencyBucketUpperBoundsSeconds {
		if seconds <= upperBound {
			h.buckets[index]++
		}
	}
}

type httpLabels struct {
	routeTemplate string
	method        string
	statusClass   string
}

type requestMetrics struct {
	count   uint64
	latency histogram
}

type recommendationMetrics struct {
	count   uint64
	latency histogram
}

type externalLabels struct {
	provider ExternalProvider
	result   ExternalResult
}

type externalMetrics struct {
	count   uint64
	latency histogram
}

// Registry owns the in-process metrics state. Its zero value is ready for use.
type Registry struct {
	mu sync.RWMutex

	httpRequests         map[httpLabels]*requestMetrics
	recommendations      map[RecommendationResult]*recommendationMetrics
	externalCalls        map[externalLabels]*externalMetrics
	productEventCounts   map[ProductEvent]uint64
	catalogFacilityCount uint64
	catalogFreshCount    uint64
	catalogStaleCount    uint64
}

// ObserveExternalCall records only fixed provider and result values. Request
// payloads, locations, queries, and provider error text are never labels.
func (r *Registry) ObserveExternalCall(provider ExternalProvider, result ExternalResult, latency time.Duration) error {
	if !isKnownExternalProvider(provider) {
		return fmt.Errorf("%w: %q", ErrUnknownExternalProvider, provider)
	}
	if !isKnownExternalResult(result) {
		return fmt.Errorf("%w: %q", ErrUnknownExternalResult, result)
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if r.externalCalls == nil {
		r.externalCalls = make(map[externalLabels]*externalMetrics)
	}
	labels := externalLabels{provider: provider, result: result}
	metrics := r.externalCalls[labels]
	if metrics == nil {
		metrics = &externalMetrics{}
		r.externalCalls[labels] = metrics
	}
	metrics.count++
	metrics.latency.observe(latency)
	return nil
}

func NewRegistry() *Registry {
	return &Registry{}
}

// ObserveHTTP records only a route template, a normalized method, and a status
// class. Callers must pass the matched route pattern, never r.URL.Path.
func (r *Registry) ObserveHTTP(routeTemplate string, method string, statusCode int, latency time.Duration) {
	method = normalizedMethod(method)
	labels := httpLabels{
		routeTemplate: normalizedRouteTemplate(routeTemplate, method),
		method:        method,
		statusClass:   statusClass(statusCode),
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if r.httpRequests == nil {
		r.httpRequests = make(map[httpLabels]*requestMetrics)
	}
	metrics := r.httpRequests[labels]
	if metrics == nil {
		metrics = &requestMetrics{}
		r.httpRequests[labels] = metrics
	}
	metrics.count++
	metrics.latency.observe(latency)
}

// ObserveRecommendation records a controlled result bucket and its latency.
func (r *Registry) ObserveRecommendation(result RecommendationResult, latency time.Duration) error {
	if !isKnownRecommendationResult(result) {
		return fmt.Errorf("%w: %q", ErrUnknownRecommendationResult, result)
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if r.recommendations == nil {
		r.recommendations = make(map[RecommendationResult]*recommendationMetrics)
	}
	metrics := r.recommendations[result]
	if metrics == nil {
		metrics = &recommendationMetrics{}
		r.recommendations[result] = metrics
	}
	metrics.count++
	metrics.latency.observe(latency)
	return nil
}

// ObserveProductEvent rejects values outside the defined product journey.
func (r *Registry) ObserveProductEvent(event ProductEvent) error {
	if !isKnownProductEvent(event) {
		return fmt.Errorf("%w: %q", ErrUnknownProductEvent, event)
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if r.productEventCounts == nil {
		r.productEventCounts = make(map[ProductEvent]uint64)
	}
	r.productEventCounts[event]++
	return nil
}

// ObserveClientProductEvent rejects server-owned events such as a successful
// correction write, which must only be counted after the operation completes.
func (r *Registry) ObserveClientProductEvent(event ProductEvent) error {
	if !isKnownClientProductEvent(event) {
		return fmt.Errorf("%w: %q", ErrUnknownProductEvent, event)
	}
	return r.ObserveProductEvent(event)
}

// SetCatalog atomically replaces the catalog gauges. Every facility must be
// classified as either fresh or stale.
func (r *Registry) SetCatalog(facilityCount int, freshCount int, staleCount int) error {
	if facilityCount < 0 || freshCount < 0 || staleCount < 0 {
		return ErrInvalidCatalogCounts
	}
	if uint64(freshCount)+uint64(staleCount) != uint64(facilityCount) {
		return ErrInvalidCatalogCounts
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	r.catalogFacilityCount = uint64(facilityCount)
	r.catalogFreshCount = uint64(freshCount)
	r.catalogStaleCount = uint64(staleCount)
	return nil
}

func normalizedRouteTemplate(routeTemplate string, method string) string {
	if separatorIndex := strings.IndexByte(routeTemplate, ' '); separatorIndex > 0 && strings.EqualFold(routeTemplate[:separatorIndex], method) {
		routeTemplate = routeTemplate[separatorIndex+1:]
	}
	if routeTemplate == "" {
		return "unknown"
	}
	return routeTemplate
}

func normalizedMethod(method string) string {
	method = strings.ToUpper(strings.TrimSpace(method))
	if method == "" {
		return "UNKNOWN"
	}
	return method
}

func statusClass(statusCode int) string {
	if statusCode < 100 || statusCode > 599 {
		return "unknown"
	}
	return strconv.Itoa(statusCode/100) + "xx"
}

func isKnownRecommendationResult(result RecommendationResult) bool {
	for _, knownResult := range recommendationResults {
		if result == knownResult {
			return true
		}
	}
	return false
}

func isKnownProductEvent(event ProductEvent) bool {
	for _, knownEvent := range productEvents {
		if event == knownEvent {
			return true
		}
	}
	return false
}

func isKnownClientProductEvent(event ProductEvent) bool {
	for _, knownEvent := range clientProductEvents {
		if event == knownEvent {
			return true
		}
	}
	return false
}

func isKnownExternalProvider(provider ExternalProvider) bool {
	for _, knownProvider := range externalProviders {
		if provider == knownProvider {
			return true
		}
	}
	return false
}

func isKnownExternalResult(result ExternalResult) bool {
	for _, knownResult := range externalResults {
		if result == knownResult {
			return true
		}
	}
	return false
}

type registrySnapshot struct {
	httpRequests         map[httpLabels]requestMetrics
	recommendations      map[RecommendationResult]recommendationMetrics
	externalCalls        map[externalLabels]externalMetrics
	productEventCounts   map[ProductEvent]uint64
	catalogFacilityCount uint64
	catalogFreshCount    uint64
	catalogStaleCount    uint64
}

func (r *Registry) snapshot() registrySnapshot {
	r.mu.RLock()
	defer r.mu.RUnlock()

	snapshot := registrySnapshot{
		httpRequests:       make(map[httpLabels]requestMetrics, len(r.httpRequests)),
		recommendations:    make(map[RecommendationResult]recommendationMetrics, len(recommendationResults)),
		externalCalls:      make(map[externalLabels]externalMetrics, len(externalProviders)*len(externalResults)),
		productEventCounts: make(map[ProductEvent]uint64, len(productEvents)),

		catalogFacilityCount: r.catalogFacilityCount,
		catalogFreshCount:    r.catalogFreshCount,
		catalogStaleCount:    r.catalogStaleCount,
	}
	for labels, metrics := range r.httpRequests {
		snapshot.httpRequests[labels] = *metrics
	}
	for _, result := range recommendationResults {
		if metrics := r.recommendations[result]; metrics != nil {
			snapshot.recommendations[result] = *metrics
			continue
		}
		snapshot.recommendations[result] = recommendationMetrics{}
	}
	for _, provider := range externalProviders {
		for _, result := range externalResults {
			labels := externalLabels{provider: provider, result: result}
			if metrics := r.externalCalls[labels]; metrics != nil {
				snapshot.externalCalls[labels] = *metrics
				continue
			}
			snapshot.externalCalls[labels] = externalMetrics{}
		}
	}
	for _, event := range productEvents {
		snapshot.productEventCounts[event] = r.productEventCounts[event]
	}
	return snapshot
}
