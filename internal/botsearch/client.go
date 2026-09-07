// Package botsearch calls the independent, authenticated nearby search API.
// Platform adapters must verify the human and authorize the owner before calling
// it. This package does not receive mentions, render messages, or persist searches.
package botsearch

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/kohei321dev/spot-diggz/internal/nearby"
)

const (
	requestTimeout   = 8 * time.Second
	maxResponseBytes = 1 << 20
	searchPath       = "/api/facilities/search"
)

// Errors carry no request, endpoint, credential, or upstream diagnostic text.
// None triggers an automatic retry; the adapter owns user guidance and delivery.
var (
	ErrInvalidConfig   = errors.New("invalid Bot API client configuration")
	ErrInvalidInput    = errors.New("invalid Bot API search input")
	ErrUnauthorized    = errors.New("Bot API credential rejected")
	ErrForbidden       = errors.New("Bot API access forbidden")
	ErrRejected        = errors.New("Bot API request rejected")
	ErrRateLimited     = errors.New("Bot API rate limited")
	ErrUnavailable     = errors.New("Bot API unavailable")
	ErrTimeout         = errors.New("Bot API request timed out")
	ErrCanceled        = errors.New("Bot API request canceled")
	ErrInvalidResponse = errors.New("invalid Bot API response")
)

// Client is reusable and safe for concurrent searches. Configuration is immutable
// after construction. Keep the plaintext token in the Bot's runtime secret store.
type Client struct {
	endpoint   string
	token      string
	httpClient *http.Client
}

// NewClient takes an operator-configured HTTPS origin, never a URL from a message.
// Local tests trust a test server certificate; there is no insecure runtime mode.
func NewClient(baseURL, token string) (*Client, error) {
	u, err := url.Parse(baseURL)
	if err != nil || u.Scheme != "https" || u.Opaque != "" || u.User != nil ||
		u.Hostname() == "" || (u.Path != "" && u.Path != "/") || u.RawPath != "" ||
		u.RawQuery != "" || u.ForceQuery || strings.Contains(baseURL, "#") ||
		strings.HasSuffix(u.Host, ":") || strings.Contains(u.Host, "%") {
		return nil, ErrInvalidConfig
	}
	if u.Port() != "" {
		port, err := strconv.Atoi(u.Port())
		if err != nil || port < 1 || port > 65535 {
			return nil, ErrInvalidConfig
		}
	}
	decoded, err := hex.DecodeString(token)
	if err != nil || len(decoded) != 32 || token != strings.ToLower(token) {
		return nil, ErrInvalidConfig
	}
	u.Path = searchPath
	// Use our own transport rather than sharing mutable DefaultClient settings or
	// a cookie jar. The overall timeout includes TLS, headers, and response reads.
	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout: requestTimeout, KeepAlive: 30 * time.Second,
		}).DialContext,
		ForceAttemptHTTP2: true, MaxIdleConns: 10, IdleConnTimeout: 90 * time.Second,
		TLSHandshakeTimeout: requestTimeout, MaxResponseHeaderBytes: 16 << 10,
	}
	return &Client{
		endpoint: u.String(), token: token,
		httpClient: &http.Client{
			Transport: transport, Timeout: requestTimeout,
			CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
		},
	}, nil
}

// String and GoString prevent ordinary diagnostic formatting from exposing config.
func (Client) String() string          { return "SpotDiggz Bot API client (configuration redacted)" }
func (client Client) GoString() string { return client.String() }

func (client *Client) CloseIdleConnections() {
	if client != nil && client.httpClient != nil {
		client.httpClient.CloseIdleConnections()
	}
}

// Search accepts structured API input, not a raw platform message. Use
// nearby.DecodeInput when decoding JSON with omitted optional values. This client
// never fills in trip conditions, expands the radius, or chooses an ambiguous place.
func (client *Client) Search(ctx context.Context, input nearby.Input) (nearby.Response, error) {
	if client == nil || client.httpClient == nil {
		return nearby.Response{}, ErrInvalidConfig
	}
	if ctx == nil || !utf8.ValidString(input.Query) {
		return nearby.Response{}, ErrInvalidInput
	}
	body, err := json.Marshal(input)
	if err != nil {
		return nearby.Response{}, ErrInvalidInput
	}
	// Reuse the API boundary so trimming, Unicode and numeric validation cannot
	// diverge. In particular, zero-valued explicit limits are not defaulted here.
	input, err = nearby.DecodeInput(body)
	if err != nil {
		return nearby.Response{}, ErrInvalidInput
	}
	body, _ = json.Marshal(input)
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, client.endpoint, bytes.NewReader(body))
	if err != nil {
		return nearby.Response{}, ErrInvalidConfig
	}
	request.Header.Set("Authorization", "Bearer "+client.token)
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "application/json")
	request.Header.Set("Cache-Control", "no-store")
	// Do not make the POST replayable by the transport or add idempotency headers.
	request.GetBody = nil
	response, err := client.httpClient.Do(request)
	if err != nil {
		return nearby.Response{}, transportError(ctx, err)
	}
	defer response.Body.Close()
	if err := responseError(response.StatusCode); err != nil {
		// Do not parse, return, log or follow error bodies/headers (including Location).
		return nearby.Response{}, err
	}
	mediaType, params, err := mime.ParseMediaType(response.Header.Get("Content-Type"))
	if err != nil || mediaType != "application/json" ||
		(params["charset"] != "" && !strings.EqualFold(params["charset"], "utf-8")) ||
		response.ContentLength > maxResponseBytes {
		return nearby.Response{}, ErrInvalidResponse
	}
	data, err := io.ReadAll(io.LimitReader(response.Body, maxResponseBytes+1))
	if err != nil {
		return nearby.Response{}, transportError(ctx, err)
	}
	if len(data) > maxResponseBytes {
		return nearby.Response{}, ErrInvalidResponse
	}
	return decodeResponse(data, input)
}

func transportError(ctx context.Context, err error) error {
	if errors.Is(ctx.Err(), context.Canceled) || errors.Is(err, context.Canceled) {
		return ErrCanceled
	}
	var networkError net.Error
	if errors.Is(ctx.Err(), context.DeadlineExceeded) || errors.Is(err, context.DeadlineExceeded) ||
		(errors.As(err, &networkError) && networkError.Timeout()) {
		return ErrTimeout
	}
	// url.Error and TLS/DNS/read errors can contain endpoints or provider text.
	return ErrUnavailable
}

func responseError(status int) error {
	switch status {
	case http.StatusOK:
		return nil
	case http.StatusUnauthorized:
		return ErrUnauthorized
	case http.StatusForbidden:
		return ErrForbidden
	case http.StatusBadRequest, http.StatusRequestEntityTooLarge, http.StatusUnsupportedMediaType:
		return ErrRejected
	case http.StatusTooManyRequests:
		return ErrRateLimited
	default:
		if status >= 500 && status <= 599 {
			return ErrUnavailable
		}
		return ErrInvalidResponse
	}
}
