package botsearch

import (
	"context"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/kohei321dev/spot-diggz/internal/nearby"
)

func unitToken(t *testing.T) string {
	t.Helper()
	seed := sha256.Sum256([]byte("synthetic Bot client test: " + t.Name()))
	return hex.EncodeToString(seed[:])
}

func unitInput() nearby.Input {
	return nearby.Input{Query: "Synthetic test place", Limit: 5, RadiusKm: 10, Sort: nearby.DistanceSort}
}

func unitEmptyResponse() nearby.Response {
	return nearby.Response{
		Status: nearby.StatusDataUnavailable, Matches: []nearby.Match{}, DistanceKind: nearby.DistanceKind,
		Limit: 5, RadiusKm: 10, Sort: nearby.DistanceSort, AvailabilityNote: "訪問前に出典と利用ルールを確認してください。",
	}
}

func unitClient(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()
	server := httptest.NewTLSServer(handler)
	t.Cleanup(server.Close)
	client, err := NewClient(server.URL, unitToken(t))
	if err != nil {
		t.Fatal("create test client failed")
	}
	// Change trust roots only, retaining the real transport's limits and TLS checks.
	roots := x509.NewCertPool()
	roots.AddCert(server.Certificate())
	client.httpClient.Transport.(*http.Transport).TLSClientConfig = &tls.Config{RootCAs: roots}
	t.Cleanup(client.CloseIdleConnections)
	return client
}

func TestClientRejectsUnsafeConfiguration(t *testing.T) {
	for _, origin := range []string{
		"", "http://example.test", "https://", "https:example.test", "https:///path",
		"https://user:password@example.test", "https://example.test/api", "https://example.test//",
		"https://example.test/%2F", "https://example.test?query=private", "https://example.test?",
		"https://example.test#", "https://example.test#fragment", "https://example.test:",
		"https://[fe80::1%25zone]", "https://example.test:bad", "https://example.test:65536",
		"https://example.test:0", " https://example.test",
	} {
		if client, err := NewClient(origin, unitToken(t)); !errors.Is(err, ErrInvalidConfig) || client != nil {
			t.Fatal("unsafe origin accepted")
		}
	}
	for _, token := range []string{"", "Bearer " + unitToken(t), unitToken(t) + "\n", strings.ToUpper(unitToken(t)), strings.Repeat("a", 62), strings.Repeat("z", 64)} {
		if client, err := NewClient("https://example.test", token); !errors.Is(err, ErrInvalidConfig) || client != nil {
			t.Fatal("invalid token shape accepted")
		}
	}
	for _, origin := range []string{"https://example.test", "https://example.test/", "https://example.test:8443", "https://[::1]:8443/"} {
		client, err := NewClient(origin, unitToken(t))
		if err != nil {
			t.Fatal("valid HTTPS origin rejected")
		}
		client.CloseIdleConnections()
		if client.httpClient.Timeout != 8*time.Second || client.httpClient.Jar != nil {
			t.Fatal("client safety defaults changed")
		}
		for _, formatted := range []string{fmt.Sprint(client), fmt.Sprintf("%+v", client), fmt.Sprintf("%#v", client), fmt.Sprintf("%+v", *client), fmt.Sprintf("%#v", *client)} {
			if strings.Contains(formatted, unitToken(t)) || strings.Contains(formatted, origin) {
				t.Fatal("client formatting exposed configuration")
			}
		}
	}
}

func TestClientUsesFixedPOSTAndOnlyStructuredInput(t *testing.T) {
	var calls atomic.Int32
	client := unitClient(t, func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.Method != http.MethodPost || r.URL.Path != searchPath || r.URL.RawQuery != "" ||
			r.Header.Get("Authorization") != "Bearer "+unitToken(t) || r.Header.Get("Content-Type") != "application/json" ||
			r.Header.Get("Accept") != "application/json" || r.Header.Get("Cache-Control") != "no-store" ||
			r.Header.Get("Cookie") != "" || r.Header.Get("Idempotency-Key") != "" {
			t.Error("request contract mismatch")
		}
		body, err := io.ReadAll(r.Body)
		input, decodeErr := nearby.DecodeInput(body)
		if err != nil || decodeErr != nil || input != unitInput() {
			t.Error("input was changed or sent outside JSON")
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(unitEmptyResponse())
	})
	input := unitInput()
	input.Query = "  " + input.Query + "  "
	result, err := client.Search(context.Background(), input)
	if err != nil || result.Status != nearby.StatusDataUnavailable || result.Matches == nil || calls.Load() != 1 {
		t.Fatal("single structured search failed")
	}
}

func TestClientInvalidInputNeverCallsAPI(t *testing.T) {
	var calls atomic.Int32
	client := unitClient(t, func(w http.ResponseWriter, r *http.Request) { calls.Add(1) })
	for _, mutate := range []func(*nearby.Input){
		func(i *nearby.Input) { i.Query = "   " },
		func(i *nearby.Input) { i.Query = "a\nb" },
		func(i *nearby.Input) { i.Query = string([]byte{0xff}) },
		func(i *nearby.Input) { i.Query = strings.Repeat("あ", 121) },
		func(i *nearby.Input) { i.Limit = 0 },
		func(i *nearby.Input) { i.Limit = 11 },
		func(i *nearby.Input) { i.RadiusKm = 0.09 },
		func(i *nearby.Input) { i.RadiusKm = 50.1 },
		func(i *nearby.Input) { i.RadiusKm = math.NaN() },
		func(i *nearby.Input) { i.RadiusKm = math.Inf(1) },
		func(i *nearby.Input) { i.Sort = "price" },
		func(i *nearby.Input) { i.Genre = "stree" },
	} {
		input := unitInput()
		mutate(&input)
		if _, err := client.Search(context.Background(), input); !errors.Is(err, ErrInvalidInput) {
			t.Fatal("invalid input was not rejected locally")
		}
	}
	if _, err := client.Search(nil, unitInput()); !errors.Is(err, ErrInvalidInput) {
		t.Fatal("nil context accepted")
	}
	if calls.Load() != 0 {
		t.Fatal("invalid input reached API")
	}
	for _, client := range []*Client{nil, {}} {
		if _, err := client.Search(context.Background(), unitInput()); !errors.Is(err, ErrInvalidConfig) {
			t.Fatal("unconfigured client accepted")
		}
		client.CloseIdleConnections()
	}
}

func TestClientRejectsEveryRedirectWithoutForwarding(t *testing.T) {
	var targetCalls atomic.Int32
	target := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { targetCalls.Add(1) }))
	defer target.Close()
	for _, status := range []int{301, 302, 303, 307, 308} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			for _, destination := range []string{target.URL, "/api/facilities/search?private=redirect"} {
				var calls atomic.Int32
				client := unitClient(t, func(w http.ResponseWriter, r *http.Request) {
					calls.Add(1)
					w.Header().Set("Location", destination)
					w.WriteHeader(status)
				})
				if _, err := client.Search(context.Background(), unitInput()); !errors.Is(err, ErrInvalidResponse) || calls.Load() != 1 {
					t.Fatal("redirect was accepted or retried")
				}
			}
		})
	}
	if targetCalls.Load() != 0 {
		t.Fatal("redirect target received request")
	}
}

func TestClientClassifiesHTTPFailuresWithoutBodyOrRetry(t *testing.T) {
	for _, test := range []struct {
		status int
		want   error
	}{
		{400, ErrRejected}, {401, ErrUnauthorized}, {403, ErrForbidden}, {413, ErrRejected}, {415, ErrRejected},
		{429, ErrRateLimited}, {500, ErrUnavailable}, {502, ErrUnavailable}, {503, ErrUnavailable}, {504, ErrUnavailable},
		{404, ErrInvalidResponse}, {201, ErrInvalidResponse}, {204, ErrInvalidResponse},
	} {
		t.Run(fmt.Sprint(test.status), func(t *testing.T) {
			var calls atomic.Int32
			client := unitClient(t, func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				w.Header().Set("Retry-After", "60")
				w.WriteHeader(test.status)
				_, _ = io.WriteString(w, "synthetic private upstream diagnostic")
			})
			result, err := client.Search(context.Background(), unitInput())
			if !errors.Is(err, test.want) || result.Status != "" || result.Matches != nil || calls.Load() != 1 {
				t.Fatal("failure classification, empty result, or single attempt violated")
			}
			if strings.Contains(err.Error(), "synthetic private") {
				t.Fatal("upstream diagnostic leaked")
			}
		})
	}
}

func TestClientCancellationAndTimeoutCoverHeadersAndBody(t *testing.T) {
	for _, flushHeaders := range []bool{false, true} {
		t.Run(fmt.Sprint(flushHeaders), func(t *testing.T) {
			release := make(chan struct{})
			defer close(release)
			client := unitClient(t, func(w http.ResponseWriter, r *http.Request) {
				_, _ = io.Copy(io.Discard, r.Body)
				if flushHeaders {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusOK)
					w.(http.Flusher).Flush()
				}
				select {
				case <-r.Context().Done():
				case <-release:
				}
			})
			client.httpClient.Timeout = 40 * time.Millisecond
			if _, err := client.Search(context.Background(), unitInput()); !errors.Is(err, ErrTimeout) {
				t.Fatal("HTTP deadline not classified")
			}
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			if _, err := client.Search(ctx, unitInput()); !errors.Is(err, ErrCanceled) {
				t.Fatal("caller cancellation not classified")
			}
			ctx, cancel = context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
			defer cancel()
			if _, err := client.Search(ctx, unitInput()); !errors.Is(err, ErrTimeout) {
				t.Fatal("caller deadline not classified")
			}
		})
	}
}

func TestClientRejectsWrongContentTypeAndOversizedBody(t *testing.T) {
	for _, test := range []struct {
		name, contentType, body string
		chunked                 bool
	}{
		{"missing type", "", `{}`, false},
		{"HTML", "text/html", `<html></html>`, false},
		{"non UTF8 charset", "application/json; charset=shift_jis", `{}`, false},
		{"oversize length", "application/json", strings.Repeat(" ", maxResponseBytes+1), false},
		{"oversize chunked", "application/json", strings.Repeat(" ", maxResponseBytes+1), true},
	} {
		t.Run(test.name, func(t *testing.T) {
			client := unitClient(t, func(w http.ResponseWriter, r *http.Request) {
				w.Header()["Content-Type"] = []string{test.contentType}
				if test.chunked {
					w.(http.Flusher).Flush()
				} else {
					w.Header().Set("Content-Length", fmt.Sprint(len(test.body)))
				}
				_, _ = io.WriteString(w, test.body)
			})
			if _, err := client.Search(context.Background(), unitInput()); !errors.Is(err, ErrInvalidResponse) {
				t.Fatal("unsafe response type or size accepted")
			}
		})
	}
}

type unitRoundTrip func(*http.Request) (*http.Response, error)

func (fn unitRoundTrip) RoundTrip(r *http.Request) (*http.Response, error) { return fn(r) }

func TestClientTransportFailureNeverExposesDiagnostic(t *testing.T) {
	client, err := NewClient("https://synthetic-private-endpoint.test", unitToken(t))
	if err != nil {
		t.Fatal("client setup failed")
	}
	client.httpClient.Transport = unitRoundTrip(func(r *http.Request) (*http.Response, error) {
		if r.GetBody != nil || r.Header.Get("Idempotency-Key") != "" {
			t.Error("request became replayable")
		}
		return nil, errors.New("synthetic private transport diagnostic: " + unitToken(t))
	})
	if _, err := client.Search(context.Background(), unitInput()); err != ErrUnavailable {
		t.Fatal("raw transport error escaped sanitization")
	}
}

func TestClientRejectsUntrustedTLSBeforeSendingCredential(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { calls.Add(1) }))
	server.Config.ErrorLog = log.New(io.Discard, "", 0)
	server.StartTLS()
	defer server.Close()
	client, err := NewClient(server.URL, unitToken(t))
	if err != nil {
		t.Fatal("client setup failed")
	}
	defer client.CloseIdleConnections()
	if _, err := client.Search(context.Background(), unitInput()); err != ErrUnavailable || calls.Load() != 0 {
		t.Fatal("untrusted server received application credentials or TLS error escaped sanitization")
	}
}

func TestClientBoundsResponseHeadersWithItsOwnTransport(t *testing.T) {
	client := unitClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Synthetic-Oversized", strings.Repeat("a", 17<<10))
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(unitEmptyResponse())
	})
	if _, err := client.Search(context.Background(), unitInput()); err != ErrUnavailable {
		t.Fatal("oversized headers were accepted or raw parser error leaked")
	}
}
