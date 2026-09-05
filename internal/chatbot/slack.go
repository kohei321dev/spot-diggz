package chatbot

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/kohei321dev/spot-diggz/internal/geocoding"
)

const (
	defaultSlackCommand  = "/spotdiggz"
	maximumChatBodyBytes = 64 * 1024
	chatReplayWindow     = 5 * time.Minute
	chatDeliveryTimeout  = 12 * time.Second
)

type SlackConfig struct {
	SigningSecret string
	BotToken      string
	TeamID        string
	UserID        string
	Command       string
	Geocoder      geocoding.Provider
	Store         SlackRequestStore
	HTTPClient    *http.Client
	Now           func() time.Time
}

type SlackHandler struct {
	service       *Service
	signingSecret []byte
	botToken      string
	teamID        string
	userID        string
	command       string
	geocoder      geocoding.Provider
	store         SlackRequestStore
	httpClient    *http.Client
	now           func() time.Time
}

func NewSlackHandler(service *Service, config SlackConfig) (*SlackHandler, error) {
	if service == nil {
		return nil, errors.New("Slack handler requires a recommendation service")
	}
	if strings.TrimSpace(config.SigningSecret) == "" || strings.TrimSpace(config.BotToken) == "" || strings.TrimSpace(config.TeamID) == "" || strings.TrimSpace(config.UserID) == "" {
		return nil, errors.New("Slack handler requires signing secret, bot token, team ID, and user ID")
	}
	if config.Geocoder == nil {
		return nil, errors.New("Slack handler requires a geocoding provider")
	}
	if config.Store == nil {
		return nil, errors.New("Slack handler requires a request store")
	}
	command := strings.TrimSpace(config.Command)
	if command == "" {
		command = defaultSlackCommand
	}
	client := config.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 5 * time.Second}
	}
	now := config.Now
	if now == nil {
		now = time.Now
	}
	return &SlackHandler{
		service:       service,
		signingSecret: []byte(config.SigningSecret),
		botToken:      strings.TrimSpace(config.BotToken),
		teamID:        strings.TrimSpace(config.TeamID),
		userID:        strings.TrimSpace(config.UserID),
		command:       command,
		geocoder:      config.Geocoder,
		store:         config.Store,
		httpClient:    client,
		now:           now,
	}, nil
}

func (handler *SlackHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || mediaType != "application/x-www-form-urlencoded" {
		http.Error(w, "unsupported content type", http.StatusUnsupportedMediaType)
		return
	}
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, maximumChatBodyBytes))
	if err != nil {
		var maximumBytesError *http.MaxBytesError
		if errors.As(err, &maximumBytesError) {
			http.Error(w, "request body is too large", http.StatusRequestEntityTooLarge)
			return
		}
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if !handler.verifyRequest(r.Header, body) {
		http.Error(w, "invalid Slack signature", http.StatusUnauthorized)
		return
	}
	form, err := url.ParseQuery(string(body))
	if err != nil {
		http.Error(w, "invalid command payload", http.StatusBadRequest)
		return
	}
	if payload := strings.TrimSpace(form.Get("payload")); payload != "" {
		handler.handleInteraction(w, payload)
		return
	}
	if form.Get("team_id") != handler.teamID || form.Get("user_id") != handler.userID {
		writeSlackResponse(w, "このアカウントではspot-diggzを利用できません。")
		return
	}
	if form.Get("command") != handler.command {
		writeSlackResponse(w, "このコマンドは利用できません。")
		return
	}
	triggerID := strings.TrimSpace(form.Get("trigger_id"))
	channelID := strings.TrimSpace(form.Get("channel_id"))
	if triggerID == "" || channelID == "" {
		http.Error(w, "invalid Slack command payload", http.StatusBadRequest)
		return
	}
	if err := handler.openSearchModal(r.Context(), triggerID, channelID); err != nil {
		writeSlackResponse(w, "条件入力画面を開けませんでした。時間をおいて再度お試しください。")
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (handler *SlackHandler) verifyRequest(header http.Header, body []byte) bool {
	timestampText := header.Get("X-Slack-Request-Timestamp")
	timestamp, err := strconv.ParseInt(timestampText, 10, 64)
	if err != nil {
		return false
	}
	requestTime := time.Unix(timestamp, 0)
	if difference := handler.now().Sub(requestTime); difference > chatReplayWindow || difference < -chatReplayWindow {
		return false
	}
	provided := header.Get("X-Slack-Signature")
	if !strings.HasPrefix(provided, "v0=") {
		return false
	}
	providedBytes, err := hex.DecodeString(strings.TrimPrefix(provided, "v0="))
	if err != nil {
		return false
	}
	mac := hmac.New(sha256.New, handler.signingSecret)
	_, _ = mac.Write([]byte("v0:" + timestampText + ":"))
	_, _ = mac.Write(body)
	return hmac.Equal(providedBytes, mac.Sum(nil))
}

func writeSlackResponse(w http.ResponseWriter, message string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{"response_type": "ephemeral", "text": message})
}
