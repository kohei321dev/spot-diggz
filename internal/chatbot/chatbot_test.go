package chatbot

import (
	"context"
	"crypto/ed25519"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/kohei321dev/spot-diggz/internal/facility"
	"github.com/kohei321dev/spot-diggz/internal/geocoding"
	"github.com/kohei321dev/spot-diggz/internal/recommendation"
	"github.com/kohei321dev/spot-diggz/internal/session"
)

var fixedChatTime = time.Date(2026, time.August, 1, 12, 0, 0, 0, time.UTC)

type stubRecommender struct {
	response recommendation.Response
	err      error
}

func (stub stubRecommender) RecommendContext(_ context.Context, _ session.Input) (recommendation.Response, error) {
	return stub.response, stub.err
}

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (function roundTripperFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return function(request)
}

func TestServiceFormatsCompactRecommendations(t *testing.T) {
	service := newTestService(t)
	message, err := service.Recommend(context.Background())
	if err != nil {
		t.Fatalf("Recommend() error = %v", err)
	}
	for _, wanted := range []string{"おすすめスポット", "1. Test Skatepark", "片道約20分・滑走約50分", "雨天でも利用しやすい施設です。", "https://example.com/official"} {
		if !strings.Contains(message, wanted) {
			t.Fatalf("message missing %q:\n%s", wanted, message)
		}
	}
}

type stubGeocoder struct {
	results []geocoding.Result
	err     error
}

func (stub stubGeocoder) Search(_ context.Context, _ string) ([]geocoding.Result, error) {
	return stub.results, stub.err
}

func TestSlackHandlerVerifiesOwnerAndOpensSearchModal(t *testing.T) {
	delivered := make(chan string, 1)
	client := &http.Client{Transport: roundTripperFunc(func(request *http.Request) (*http.Response, error) {
		body, _ := io.ReadAll(request.Body)
		delivered <- string(body)
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"ok":true}`)), Header: make(http.Header)}, nil
	})}
	config := newTestSlackConfig(client)
	handler, err := NewSlackHandler(newTestService(t), config)
	if err != nil {
		t.Fatalf("NewSlackHandler() error = %v", err)
	}
	body := url.Values{
		"team_id":    {"T123"},
		"user_id":    {"U123"},
		"command":    {"/spotdiggz"},
		"trigger_id": {"trigger-123"},
		"channel_id": {"C123"},
	}.Encode()
	request := signedSlackRequest(t, body, "slack-secret", fixedChatTime)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("initial response = %d %s", response.Code, response.Body.String())
	}
	select {
	case payload := <-delivered:
		if !strings.Contains(payload, `"callback_id":"spotdiggz_search"`) || !strings.Contains(payload, "出発地・行きたいエリア") {
			t.Fatalf("delivered payload = %s", payload)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Slack modal was not opened")
	}
}

func TestSlackHandlerRejectsInvalidSignatureAndNonOwner(t *testing.T) {
	handler, err := NewSlackHandler(newTestService(t), newTestSlackConfig(nil))
	if err != nil {
		t.Fatalf("NewSlackHandler() error = %v", err)
	}
	body := url.Values{"team_id": {"T123"}, "user_id": {"someone-else"}, "command": {"/spotdiggz"}, "trigger_id": {"trigger-123"}, "channel_id": {"C123"}}.Encode()

	invalidRequest := httptest.NewRequest(http.MethodPost, "/integrations/slack/commands", strings.NewReader(body))
	invalidRequest.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	invalidResponse := httptest.NewRecorder()
	handler.ServeHTTP(invalidResponse, invalidRequest)
	if invalidResponse.Code != http.StatusUnauthorized {
		t.Fatalf("invalid signature status = %d", invalidResponse.Code)
	}

	nonOwnerResponse := httptest.NewRecorder()
	handler.ServeHTTP(nonOwnerResponse, signedSlackRequest(t, body, "slack-secret", fixedChatTime))
	if nonOwnerResponse.Code != http.StatusOK || !strings.Contains(nonOwnerResponse.Body.String(), "利用できません") {
		t.Fatalf("non-owner response = %d %s", nonOwnerResponse.Code, nonOwnerResponse.Body.String())
	}
}

func TestSlackHandlerRejectsReplayedRequest(t *testing.T) {
	config := newTestSlackConfig(http.DefaultClient)
	config.Now = func() time.Time { return fixedChatTime.Add(chatReplayWindow + time.Second) }
	handler, err := NewSlackHandler(newTestService(t), config)
	if err != nil {
		t.Fatalf("NewSlackHandler() error = %v", err)
	}
	body := url.Values{
		"team_id": {"T123"}, "user_id": {"U123"}, "command": {"/spotdiggz"},
		"trigger_id": {"trigger-1"}, "channel_id": {"C123"},
	}.Encode()
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, signedSlackRequest(t, body, "slack-secret", fixedChatTime))
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusUnauthorized)
	}
}

func TestSlackModalSubmissionReturnsCandidatesWithoutPersistenceActions(t *testing.T) {
	posted := make(chan []byte, 1)
	client := &http.Client{Transport: roundTripperFunc(func(request *http.Request) (*http.Response, error) {
		body, _ := io.ReadAll(request.Body)
		if request.URL.Host == "slack.com" && request.URL.Path == "/api/chat.postEphemeral" {
			posted <- body
			return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"ok":true}`)), Header: make(http.Header)}, nil
		}
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"ok":true}`)), Header: make(http.Header)}, nil
	})}
	store := NewMemorySlackRequestStore()
	config := newTestSlackConfig(client)
	config.Store = store
	handler, err := NewSlackHandler(newTestService(t), config)
	if err != nil {
		t.Fatalf("NewSlackHandler() error = %v", err)
	}

	submission := `{
		"type":"view_submission",
		"team":{"id":"T123"},
		"user":{"id":"U123"},
		"view":{
			"id":"V123",
			"callback_id":"spotdiggz_search",
			"private_metadata":"{\"channel_id\":\"C123\"}",
			"state":{"values":{
				"location":{"location":{"value":"大阪駅"}},
				"available_minutes":{"available_minutes":{"selected_option":{"value":"120"}}},
				"transport":{"transport":{"selected_option":{"value":"public_transit"}}},
				"level":{"level":{"selected_option":{"value":"beginner"}}},
				"purpose":{"purpose":{"selected_option":{"value":"basics"}}},
				"mood":{"mood":{"selected_option":{"value":"focused"}}}
			}}
		}
	}`
	submissionBody := url.Values{"payload": {submission}}.Encode()
	submissionResponse := httptest.NewRecorder()
	handler.ServeHTTP(submissionResponse, signedSlackRequest(t, submissionBody, "slack-secret", fixedChatTime))
	if submissionResponse.Code != http.StatusOK || !strings.Contains(submissionResponse.Body.String(), `"response_action":"clear"`) {
		t.Fatalf("submission response = %d %s", submissionResponse.Code, submissionResponse.Body.String())
	}

	var candidatePayload []byte
	select {
	case candidatePayload = <-posted:
	case <-time.After(2 * time.Second):
		t.Fatal("Slack candidates were not posted")
	}
	payloadText := string(candidatePayload)
	if strings.Contains(payloadText, "spotdiggz_save") || strings.Contains(payloadText, "spotdiggz_reject") || strings.Contains(payloadText, "Lists") {
		t.Fatalf("candidate payload contains a persistence action: %s", payloadText)
	}
	if !strings.Contains(payloadText, "spotdiggz_official_0") || !strings.Contains(payloadText, "spotdiggz_route_0") {
		t.Fatalf("candidate payload is missing navigation actions: %s", payloadText)
	}
}

func TestDiscordHandlerVerifiesOwnerAndDeliversRecommendation(t *testing.T) {
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("GenerateKey() error = %v", err)
	}
	delivered := make(chan string, 1)
	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch || r.URL.Path != "/webhooks/123456/interaction-token/messages/@original" {
			t.Fatalf("delivery request = %s %s", r.Method, r.URL.Path)
		}
		body, _ := io.ReadAll(r.Body)
		delivered <- string(body)
		w.WriteHeader(http.StatusOK)
	}))
	defer apiServer.Close()
	handler, err := NewDiscordHandler(newTestService(t), DiscordConfig{
		PublicKeyHex:  hex.EncodeToString(publicKey),
		ApplicationID: "123456",
		GuildID:       "654321",
		UserID:        "777777",
		APIBaseURL:    apiServer.URL,
		Now:           func() time.Time { return fixedChatTime },
	})
	if err != nil {
		t.Fatalf("NewDiscordHandler() error = %v", err)
	}
	payload := `{"type":2,"application_id":"123456","guild_id":"654321","token":"interaction-token","data":{"name":"spotdiggz"},"member":{"user":{"id":"777777"}}}`
	request := signedDiscordRequest(t, payload, privateKey)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"type":5`) || !strings.Contains(response.Body.String(), `"flags":64`) {
		t.Fatalf("initial response = %d %s", response.Code, response.Body.String())
	}
	select {
	case deliveredPayload := <-delivered:
		if !strings.Contains(deliveredPayload, "Test Skatepark") || !strings.Contains(deliveredPayload, "allowed_mentions") {
			t.Fatalf("delivered payload = %s", deliveredPayload)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Discord recommendation was not delivered")
	}
}

func TestDiscordHandlerSupportsPingAndRejectsNonOwner(t *testing.T) {
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("GenerateKey() error = %v", err)
	}
	handler, err := NewDiscordHandler(newTestService(t), DiscordConfig{
		PublicKeyHex:  hex.EncodeToString(publicKey),
		ApplicationID: "123456",
		GuildID:       "654321",
		UserID:        "777777",
		Now:           func() time.Time { return fixedChatTime },
	})
	if err != nil {
		t.Fatalf("NewDiscordHandler() error = %v", err)
	}

	pingResponse := httptest.NewRecorder()
	handler.ServeHTTP(pingResponse, signedDiscordRequest(t, `{"type":1}`, privateKey))
	if pingResponse.Code != http.StatusOK || !strings.Contains(pingResponse.Body.String(), `"type":1`) {
		t.Fatalf("ping response = %d %s", pingResponse.Code, pingResponse.Body.String())
	}

	nonOwnerPayload := `{"type":2,"application_id":"123456","guild_id":"654321","token":"interaction-token","data":{"name":"spotdiggz"},"member":{"user":{"id":"someone-else"}}}`
	nonOwnerResponse := httptest.NewRecorder()
	handler.ServeHTTP(nonOwnerResponse, signedDiscordRequest(t, nonOwnerPayload, privateKey))
	if nonOwnerResponse.Code != http.StatusOK || !strings.Contains(nonOwnerResponse.Body.String(), "利用できません") || !strings.Contains(nonOwnerResponse.Body.String(), `"flags":64`) {
		t.Fatalf("non-owner response = %d %s", nonOwnerResponse.Code, nonOwnerResponse.Body.String())
	}
}

func TestDiscordHandlerRejectsInvalidSignature(t *testing.T) {
	publicKey, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("GenerateKey() error = %v", err)
	}
	handler, err := NewDiscordHandler(newTestService(t), DiscordConfig{
		PublicKeyHex:  hex.EncodeToString(publicKey),
		ApplicationID: "123456",
		GuildID:       "654321",
		UserID:        "777777",
		Now:           func() time.Time { return fixedChatTime },
	})
	if err != nil {
		t.Fatalf("NewDiscordHandler() error = %v", err)
	}
	request := httptest.NewRequest(http.MethodPost, "/integrations/discord/interactions", strings.NewReader(`{"type":1}`))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("X-Signature-Timestamp", "1234567890")
	request.Header.Set("X-Signature-Ed25519", strings.Repeat("0", ed25519.SignatureSize*2))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusUnauthorized)
	}
}

func TestDiscordHandlerRejectsReplayedRequest(t *testing.T) {
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("GenerateKey() error = %v", err)
	}
	handler, err := NewDiscordHandler(newTestService(t), DiscordConfig{
		PublicKeyHex:  hex.EncodeToString(publicKey),
		ApplicationID: "123456",
		GuildID:       "654321",
		UserID:        "777777",
		Now:           func() time.Time { return fixedChatTime.Add(chatReplayWindow + time.Second) },
	})
	if err != nil {
		t.Fatalf("NewDiscordHandler() error = %v", err)
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, signedDiscordRequest(t, `{"type":1}`, privateKey))
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusUnauthorized)
	}
}

func TestChatHandlersRejectOversizedBodies(t *testing.T) {
	slackHandler, err := NewSlackHandler(newTestService(t), newTestSlackConfig(nil))
	if err != nil {
		t.Fatalf("NewSlackHandler() error = %v", err)
	}
	slackRequest := httptest.NewRequest(http.MethodPost, "/integrations/slack/commands", strings.NewReader(strings.Repeat("x", maximumChatBodyBytes+1)))
	slackRequest.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	slackResponse := httptest.NewRecorder()
	slackHandler.ServeHTTP(slackResponse, slackRequest)
	if slackResponse.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("Slack status = %d, want %d", slackResponse.Code, http.StatusRequestEntityTooLarge)
	}

	publicKey, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("GenerateKey() error = %v", err)
	}
	discordHandler, err := NewDiscordHandler(newTestService(t), DiscordConfig{
		PublicKeyHex:  hex.EncodeToString(publicKey),
		ApplicationID: "123456",
		GuildID:       "654321",
		UserID:        "777777",
	})
	if err != nil {
		t.Fatalf("NewDiscordHandler() error = %v", err)
	}
	discordRequest := httptest.NewRequest(http.MethodPost, "/integrations/discord/interactions", strings.NewReader(strings.Repeat("x", maximumChatBodyBytes+1)))
	discordRequest.Header.Set("Content-Type", "application/json")
	discordResponse := httptest.NewRecorder()
	discordHandler.ServeHTTP(discordResponse, discordRequest)
	if discordResponse.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("Discord status = %d, want %d", discordResponse.Code, http.StatusRequestEntityTooLarge)
	}
}

func newTestService(t *testing.T) *Service {
	t.Helper()
	latitude := 34.7025
	longitude := 135.4960
	input := session.Input{
		Purpose:          session.PurposeBasics,
		Mood:             session.MoodFocused,
		Level:            session.LevelBeginner,
		AvailableMinutes: 120,
		Transport:        session.TransportPublicTransit,
		Origin: session.Origin{
			Mode:      session.OriginSpecifiedLocation,
			Latitude:  &latitude,
			Longitude: &longitude,
		},
	}
	service, err := NewService(stubRecommender{response: recommendation.Response{Recommendations: []recommendation.Item{{
		Facility:               facility.Facility{ID: "facility-a", Name: "Test Skatepark", SourceURL: "https://example.com/official", Location: facility.Location{Latitude: 34.7, Longitude: 135.5}},
		Reasons:                []recommendation.Reason{{Code: "weather", Message: "雨天でも利用しやすい施設です。"}},
		EstimatedTravelMinutes: 20,
		EstimatedSkateMinutes:  50,
	}}}}, input)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	return service
}

func newTestSlackConfig(client *http.Client) SlackConfig {
	return SlackConfig{
		SigningSecret: "slack-secret",
		BotToken:      "xoxb-test-token-value-for-tests",
		TeamID:        "T123",
		UserID:        "U123",
		Geocoder: stubGeocoder{results: []geocoding.Result{{
			Label:    "大阪駅",
			Location: facility.Location{Latitude: 34.7025, Longitude: 135.4960},
		}}},
		Store:      NewMemorySlackRequestStore(),
		HTTPClient: client,
		Now:        func() time.Time { return fixedChatTime },
	}
}

func signedSlackRequest(t *testing.T, body string, secret string, at time.Time) *http.Request {
	t.Helper()
	timestamp := strconv.FormatInt(at.Unix(), 10)
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte("v0:" + timestamp + ":" + body))
	request := httptest.NewRequest(http.MethodPost, "/integrations/slack/commands", strings.NewReader(body))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.Header.Set("X-Slack-Request-Timestamp", timestamp)
	request.Header.Set("X-Slack-Signature", "v0="+hex.EncodeToString(mac.Sum(nil)))
	return request
}

func signedDiscordRequest(t *testing.T, body string, privateKey ed25519.PrivateKey) *http.Request {
	t.Helper()
	timestamp := "1785585600"
	signature := ed25519.Sign(privateKey, append([]byte(timestamp), []byte(body)...))
	request := httptest.NewRequest(http.MethodPost, "/integrations/discord/interactions", strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("X-Signature-Timestamp", timestamp)
	request.Header.Set("X-Signature-Ed25519", hex.EncodeToString(signature))
	return request
}
