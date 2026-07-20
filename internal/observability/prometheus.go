package observability

import (
	"net/http"
	"sort"
	"strconv"
	"strings"
)

const PrometheusContentType = "text/plain; version=0.0.4; charset=utf-8"

type prometheusLabel struct {
	name  string
	value string
}

type httpSample struct {
	labels  httpLabels
	metrics requestMetrics
}

// Handler returns a Prometheus text exposition handler backed by the registry.
func (r *Registry) Handler() http.Handler {
	return r
}

// ServeHTTP implements http.Handler. Rendering uses a point-in-time snapshot so
// metric updates are blocked only while the in-memory values are copied.
func (r *Registry) ServeHTTP(w http.ResponseWriter, request *http.Request) {
	w.Header().Set("Content-Type", PrometheusContentType)
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")

	if request.Method == http.MethodHead {
		w.WriteHeader(http.StatusOK)
		return
	}

	_, _ = w.Write([]byte(renderPrometheus(r.snapshot())))
}

func renderPrometheus(snapshot registrySnapshot) string {
	var output strings.Builder

	writeMetricHeader(
		&output,
		"spot_diggz_http_requests_total",
		"counter",
		"Total number of HTTP requests by route template, method, and status class.",
	)
	httpSamples := sortedHTTPSamples(snapshot.httpRequests)
	for _, sample := range httpSamples {
		writeSample(
			&output,
			"spot_diggz_http_requests_total",
			httpPrometheusLabels(sample.labels),
			strconv.FormatUint(sample.metrics.count, 10),
		)
	}

	writeMetricHeader(
		&output,
		"spot_diggz_http_request_duration_seconds",
		"histogram",
		"HTTP request latency in seconds by route template, method, and status class.",
	)
	for _, sample := range httpSamples {
		writeHistogram(
			&output,
			"spot_diggz_http_request_duration_seconds",
			httpPrometheusLabels(sample.labels),
			sample.metrics.latency,
		)
	}

	writeMetricHeader(
		&output,
		"spot_diggz_recommendations_total",
		"counter",
		"Total number of recommendation attempts by result.",
	)
	for _, result := range recommendationResults {
		metrics := snapshot.recommendations[result]
		writeSample(
			&output,
			"spot_diggz_recommendations_total",
			[]prometheusLabel{{name: "result", value: string(result)}},
			strconv.FormatUint(metrics.count, 10),
		)
	}

	writeMetricHeader(
		&output,
		"spot_diggz_external_requests_total",
		"counter",
		"Total number of external provider requests by fixed provider and result.",
	)
	for _, provider := range externalProviders {
		for _, result := range externalResults {
			labels := externalLabels{provider: provider, result: result}
			writeSample(
				&output,
				"spot_diggz_external_requests_total",
				externalPrometheusLabels(labels),
				strconv.FormatUint(snapshot.externalCalls[labels].count, 10),
			)
		}
	}

	writeMetricHeader(
		&output,
		"spot_diggz_external_request_duration_seconds",
		"histogram",
		"External provider request latency in seconds by fixed provider and result.",
	)
	for _, provider := range externalProviders {
		for _, result := range externalResults {
			labels := externalLabels{provider: provider, result: result}
			writeHistogram(
				&output,
				"spot_diggz_external_request_duration_seconds",
				externalPrometheusLabels(labels),
				snapshot.externalCalls[labels].latency,
			)
		}
	}

	writeMetricHeader(
		&output,
		"spot_diggz_recommendation_duration_seconds",
		"histogram",
		"Recommendation latency in seconds by result.",
	)
	for _, result := range recommendationResults {
		metrics := snapshot.recommendations[result]
		writeHistogram(
			&output,
			"spot_diggz_recommendation_duration_seconds",
			[]prometheusLabel{{name: "result", value: string(result)}},
			metrics.latency,
		)
	}

	writeMetricHeader(
		&output,
		"spot_diggz_product_events_total",
		"counter",
		"Total number of allow-listed product journey events.",
	)
	for _, event := range productEvents {
		writeSample(
			&output,
			"spot_diggz_product_events_total",
			[]prometheusLabel{{name: "event", value: string(event)}},
			strconv.FormatUint(snapshot.productEventCounts[event], 10),
		)
	}

	writeMetricHeader(
		&output,
		"spot_diggz_catalog_facilities",
		"gauge",
		"Current number of facilities in the catalog.",
	)
	writeSample(
		&output,
		"spot_diggz_catalog_facilities",
		nil,
		strconv.FormatUint(snapshot.catalogFacilityCount, 10),
	)

	writeMetricHeader(
		&output,
		"spot_diggz_catalog_freshness",
		"gauge",
		"Current number of catalog facilities by freshness state.",
	)
	writeSample(
		&output,
		"spot_diggz_catalog_freshness",
		[]prometheusLabel{{name: "state", value: "fresh"}},
		strconv.FormatUint(snapshot.catalogFreshCount, 10),
	)
	writeSample(
		&output,
		"spot_diggz_catalog_freshness",
		[]prometheusLabel{{name: "state", value: "stale"}},
		strconv.FormatUint(snapshot.catalogStaleCount, 10),
	)

	return output.String()
}

func sortedHTTPSamples(metricsByLabels map[httpLabels]requestMetrics) []httpSample {
	samples := make([]httpSample, 0, len(metricsByLabels))
	for labels, metrics := range metricsByLabels {
		samples = append(samples, httpSample{labels: labels, metrics: metrics})
	}
	sort.Slice(samples, func(firstIndex int, secondIndex int) bool {
		first := samples[firstIndex].labels
		second := samples[secondIndex].labels
		if first.routeTemplate != second.routeTemplate {
			return first.routeTemplate < second.routeTemplate
		}
		if first.method != second.method {
			return first.method < second.method
		}
		return first.statusClass < second.statusClass
	})
	return samples
}

func httpPrometheusLabels(labels httpLabels) []prometheusLabel {
	return []prometheusLabel{
		{name: "route", value: labels.routeTemplate},
		{name: "method", value: labels.method},
		{name: "status_class", value: labels.statusClass},
	}
}

func externalPrometheusLabels(labels externalLabels) []prometheusLabel {
	return []prometheusLabel{
		{name: "provider", value: string(labels.provider)},
		{name: "result", value: string(labels.result)},
	}
}

func writeMetricHeader(output *strings.Builder, name string, metricType string, help string) {
	output.WriteString("# HELP ")
	output.WriteString(name)
	output.WriteByte(' ')
	output.WriteString(help)
	output.WriteByte('\n')
	output.WriteString("# TYPE ")
	output.WriteString(name)
	output.WriteByte(' ')
	output.WriteString(metricType)
	output.WriteByte('\n')
}

func writeHistogram(output *strings.Builder, name string, labels []prometheusLabel, value histogram) {
	for index, upperBound := range latencyBucketUpperBoundsSeconds {
		writeSample(
			output,
			name+"_bucket",
			labelsWith(labels, prometheusLabel{name: "le", value: formatPrometheusFloat(upperBound)}),
			strconv.FormatUint(value.buckets[index], 10),
		)
	}
	writeSample(
		output,
		name+"_bucket",
		labelsWith(labels, prometheusLabel{name: "le", value: "+Inf"}),
		strconv.FormatUint(value.count, 10),
	)
	writeSample(output, name+"_sum", labels, formatPrometheusFloat(value.sum))
	writeSample(output, name+"_count", labels, strconv.FormatUint(value.count, 10))
}

func labelsWith(labels []prometheusLabel, extra prometheusLabel) []prometheusLabel {
	combined := make([]prometheusLabel, 0, len(labels)+1)
	combined = append(combined, labels...)
	return append(combined, extra)
}

func writeSample(output *strings.Builder, name string, labels []prometheusLabel, value string) {
	output.WriteString(name)
	if len(labels) > 0 {
		output.WriteByte('{')
		for index, label := range labels {
			if index > 0 {
				output.WriteByte(',')
			}
			output.WriteString(label.name)
			output.WriteString("=\"")
			output.WriteString(escapePrometheusLabelValue(label.value))
			output.WriteByte('"')
		}
		output.WriteByte('}')
	}
	output.WriteByte(' ')
	output.WriteString(value)
	output.WriteByte('\n')
}

func escapePrometheusLabelValue(value string) string {
	value = strings.ReplaceAll(value, "\\", "\\\\")
	value = strings.ReplaceAll(value, "\n", "\\n")
	return strings.ReplaceAll(value, "\"", "\\\"")
}

func formatPrometheusFloat(value float64) string {
	return strconv.FormatFloat(value, 'g', -1, 64)
}
