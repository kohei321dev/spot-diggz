package chatbot

import (
	"context"
	"crypto/ed25519"
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
)

const (
	defaultDiscordCommand  = "spotdiggz"
	defaultDiscordAPIBase  = "https://discord.com/api/v10"
	discordPingType        = 1
	discordCommandType     = 2
	discordPongResponse    = 1
	discordMessageResponse = 4
	discordDeferredMessage = 5
	discordEphemeralFlag   = 64
)

type DiscordConfig struct {
	PublicKeyHex  string
	ApplicationID string
	GuildID       string
	UserID        string
	Command       string
	APIBaseURL    string
	HTTPClient    *http.Client
	Now           func() time.Time
}

type DiscordHandler struct {
	service       *Service
	publicKey     ed25519.PublicKey
	applicationID string
	guildID       string
	userID        string
	command       string
	apiBaseURL    string
	httpClient    *http.Client
	now           func() time.Time
}

type discordInteraction struct {
	Type          int    `json:"type"`
	ApplicationID string `json:"application_id"`
	GuildID       string `json:"guild_id"`
	Token         string `json:"token"`
	Data          struct {
		Name string `json:"name"`
	} `json:"data"`
	Member struct {
		User struct {
			ID string `json:"id"`
		} `json:"user"`
	} `json:"member"`
	User struct {
		ID string `json:"id"`
	} `json:"user"`
}

func NewDiscordHandler(service *Service, config DiscordConfig) (*DiscordHandler, error) {
	if service == nil {
		return nil, errors.New("Discord handler requires a recommendation service")
	}
	publicKey, err := hex.DecodeString(strings.TrimSpace(config.PublicKeyHex))
	if err != nil || len(publicKey) != ed25519.PublicKeySize {
		return nil, errors.New("Discord public key must be a 32-byte hexadecimal value")
	}
	if strings.TrimSpace(config.ApplicationID) == "" || strings.TrimSpace(config.GuildID) == "" || strings.TrimSpace(config.UserID) == "" {
		return nil, errors.New("Discord handler requires application, guild, and user IDs")
	}
	command := strings.TrimSpace(config.Command)
	if command == "" {
		command = defaultDiscordCommand
	}
	apiBaseURL := strings.TrimRight(strings.TrimSpace(config.APIBaseURL), "/")
	if apiBaseURL == "" {
		apiBaseURL = defaultDiscordAPIBase
	}
	parsedBaseURL, err := url.Parse(apiBaseURL)
	if err != nil || parsedBaseURL.Scheme == "" || parsedBaseURL.Host == "" || parsedBaseURL.RawQuery != "" || parsedBaseURL.Fragment != "" {
		return nil, errors.New("Discord API base URL must be an absolute URL")
	}
	client := config.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 5 * time.Second}
	}
	now := config.Now
	if now == nil {
		now = time.Now
	}
	return &DiscordHandler{
		service:       service,
		publicKey:     ed25519.PublicKey(publicKey),
		applicationID: strings.TrimSpace(config.ApplicationID),
		guildID:       strings.TrimSpace(config.GuildID),
		userID:        strings.TrimSpace(config.UserID),
		command:       command,
		apiBaseURL:    apiBaseURL,
		httpClient:    client,
		now:           now,
	}, nil
}

func (handler *DiscordHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || mediaType != "application/json" {
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
		http.Error(w, "invalid Discord signature", http.StatusUnauthorized)
		return
	}
	var interaction discordInteraction
	if err := json.Unmarshal(body, &interaction); err != nil {
		http.Error(w, "invalid interaction payload", http.StatusBadRequest)
		return
	}
	if interaction.Type == discordPingType {
		writeDiscordResponse(w, map[string]any{"type": discordPongResponse})
		return
	}
	if interaction.Type != discordCommandType || interaction.ApplicationID != handler.applicationID || interaction.Data.Name != handler.command {
		writeDiscordMessage(w, "このコマンドは利用できません。")
		return
	}
	userID := interaction.Member.User.ID
	if userID == "" {
		userID = interaction.User.ID
	}
	if interaction.GuildID != handler.guildID || userID != handler.userID {
		writeDiscordMessage(w, "このアカウントではspot-diggzを利用できません。")
		return
	}
	if strings.TrimSpace(interaction.Token) == "" {
		http.Error(w, "interaction token is missing", http.StatusBadRequest)
		return
	}
	writeDiscordResponse(w, map[string]any{"type": discordDeferredMessage, "data": map[string]int{"flags": discordEphemeralFlag}})
	go handler.deliver(interaction.ApplicationID, interaction.Token)
}

func (handler *DiscordHandler) verifyRequest(header http.Header, body []byte) bool {
	signature, err := hex.DecodeString(header.Get("X-Signature-Ed25519"))
	if err != nil || len(signature) != ed25519.SignatureSize {
		return false
	}
	timestamp := header.Get("X-Signature-Timestamp")
	timestampSeconds, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil {
		return false
	}
	requestTime := time.Unix(timestampSeconds, 0)
	if difference := handler.now().Sub(requestTime); difference > chatReplayWindow || difference < -chatReplayWindow {
		return false
	}
	message := append([]byte(timestamp), body...)
	return ed25519.Verify(handler.publicKey, message, signature)
}

func (handler *DiscordHandler) deliver(applicationID string, token string) {
	ctx, cancel := context.WithTimeout(context.Background(), chatDeliveryTimeout)
	defer cancel()
	message, err := handler.service.Recommend(ctx)
	if err != nil {
		message = "おすすめスポットを取得できませんでした。時間をおいて再度お試しください。"
	}
	endpoint := handler.apiBaseURL + "/webhooks/" + url.PathEscape(applicationID) + "/" + url.PathEscape(token) + "/messages/@original"
	payload, err := json.Marshal(map[string]any{
		"content":          message,
		"allowed_mentions": map[string]any{"parse": []string{}},
	})
	if err != nil {
		return
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPatch, endpoint, strings.NewReader(string(payload)))
	if err != nil {
		return
	}
	request.Header.Set("Content-Type", "application/json")
	response, err := handler.httpClient.Do(request)
	if err != nil {
		return
	}
	defer response.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, maximumChatBodyBytes))
}

func writeDiscordMessage(w http.ResponseWriter, message string) {
	writeDiscordResponse(w, map[string]any{
		"type": discordMessageResponse,
		"data": map[string]any{"content": message, "flags": discordEphemeralFlag, "allowed_mentions": map[string]any{"parse": []string{}}},
	})
}

func writeDiscordResponse(w http.ResponseWriter, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(payload)
}
