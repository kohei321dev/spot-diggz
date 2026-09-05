package chatbot

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/kohei321dev/spot-diggz/internal/recommendation"
	"github.com/kohei321dev/spot-diggz/internal/session"
)

const (
	slackSearchCallbackID     = "spotdiggz_search"
	slackRequestLifetime      = time.Hour
	slackAPIRequestTimeout    = 2500 * time.Millisecond
	maximumSlackAPIBytes      = 256 * 1024
	maximumSlackLocationRunes = 120
)

type slackInteractionPayload struct {
	Type string `json:"type"`
	Team struct {
		ID string `json:"id"`
	} `json:"team"`
	User struct {
		ID string `json:"id"`
	} `json:"user"`
	View struct {
		ID              string `json:"id"`
		CallbackID      string `json:"callback_id"`
		PrivateMetadata string `json:"private_metadata"`
		State           struct {
			Values map[string]map[string]slackViewStateValue `json:"values"`
		} `json:"state"`
	} `json:"view"`
	Actions []struct {
		ActionID string `json:"action_id"`
	} `json:"actions"`
}

type slackViewStateValue struct {
	Value          string `json:"value"`
	SelectedOption *struct {
		Value string `json:"value"`
	} `json:"selected_option"`
}

type slackModalMetadata struct {
	ChannelID string `json:"channel_id"`
}

type slackSearchInput struct {
	LocationQuery    string
	AvailableMinutes int
	Transport        session.Transport
	Level            session.Level
	Purpose          session.Purpose
	Mood             session.Mood
	ChannelID        string
}

func (handler *SlackHandler) openSearchModal(parent context.Context, triggerID string, channelID string) error {
	metadata, err := json.Marshal(slackModalMetadata{ChannelID: channelID})
	if err != nil {
		return fmt.Errorf("encode Slack modal metadata: %w", err)
	}
	payload := map[string]any{
		"trigger_id": triggerID,
		"view": map[string]any{
			"type":             "modal",
			"callback_id":      slackSearchCallbackID,
			"private_metadata": string(metadata),
			"title":            slackPlainText("スポット検索"),
			"submit":           slackPlainText("検索する"),
			"close":            slackPlainText("閉じる"),
			"blocks": []any{
				slackTextInput("location", "location", "出発地・行きたいエリア", "例: 梅田駅、難波周辺", false, maximumSlackLocationRunes),
				slackSelectInput("available_minutes", "available_minutes", "使える時間", []slackOption{
					{Text: "60分", Value: "60"}, {Text: "120分", Value: "120"},
					{Text: "180分", Value: "180"}, {Text: "240分", Value: "240"},
				}, "120"),
				slackSelectInput("transport", "transport", "交通手段", []slackOption{
					{Text: "電車・バス", Value: string(session.TransportPublicTransit)},
					{Text: "車", Value: string(session.TransportCar)},
					{Text: "自転車", Value: string(session.TransportBicycle)},
					{Text: "徒歩", Value: string(session.TransportWalk)},
				}, string(session.TransportPublicTransit)),
				slackSelectInput("level", "level", "レベル", []slackOption{
					{Text: "初心者", Value: string(session.LevelBeginner)},
					{Text: "久しぶり", Value: string(session.LevelReturning)},
					{Text: "中級者", Value: string(session.LevelIntermediate)},
				}, string(session.LevelBeginner)),
				slackSelectInput("purpose", "purpose", "やりたいこと", []slackOption{
					{Text: "基礎練習", Value: string(session.PurposeBasics)},
					{Text: "ストリート", Value: string(session.PurposeStreet)},
					{Text: "ランプ・トランジション", Value: string(session.PurposeTransition)},
				}, string(session.PurposeBasics)),
				slackSelectInput("mood", "mood", "今日の気分", []slackOption{
					{Text: "集中して練習", Value: string(session.MoodFocused)},
					{Text: "気軽に滑る", Value: string(session.MoodEasygoing)},
					{Text: "挑戦する", Value: string(session.MoodChallenge)},
				}, string(session.MoodFocused)),
			},
		},
	}
	ctx, cancel := context.WithTimeout(parent, slackAPIRequestTimeout)
	defer cancel()
	return handler.callSlackAPI(ctx, "views.open", payload)
}

func (handler *SlackHandler) handleInteraction(w http.ResponseWriter, rawPayload string) {
	var payload slackInteractionPayload
	if err := json.Unmarshal([]byte(rawPayload), &payload); err != nil {
		http.Error(w, "invalid Slack interaction payload", http.StatusBadRequest)
		return
	}
	if payload.Team.ID != handler.teamID || payload.User.ID != handler.userID {
		writeSlackResponse(w, "このアカウントではspot-diggzを利用できません。")
		return
	}
	switch payload.Type {
	case "view_submission":
		handler.handleViewSubmission(w, payload)
	case "block_actions":
		handler.handleBlockAction(w, payload)
	default:
		http.Error(w, "unsupported Slack interaction", http.StatusBadRequest)
	}
}

func (handler *SlackHandler) handleViewSubmission(w http.ResponseWriter, payload slackInteractionPayload) {
	if payload.View.CallbackID != slackSearchCallbackID || strings.TrimSpace(payload.View.ID) == "" {
		http.Error(w, "invalid Slack view submission", http.StatusBadRequest)
		return
	}
	input, fieldErrors := parseSlackSearchInput(payload.View)
	if len(fieldErrors) > 0 {
		writeSlackJSON(w, map[string]any{"response_action": "errors", "errors": fieldErrors})
		return
	}
	requestID, err := newSlackRequestID()
	if err != nil {
		writeSlackJSON(w, map[string]any{"response_action": "errors", "errors": map[string]string{"location": "検索を開始できませんでした。"}})
		return
	}
	now := handler.now().UTC()
	created, err := handler.store.Begin(context.Background(), SlackRecommendationRequest{
		RequestID:      requestID,
		SourceEventKey: handler.slackSourceEventKey(payload.View.ID),
		Status:         SlackRequestGenerating,
		CreatedAt:      now,
		UpdatedAt:      now,
		ExpiresAt:      now.Add(slackRequestLifetime),
	})
	if err != nil {
		writeSlackJSON(w, map[string]any{"response_action": "errors", "errors": map[string]string{"location": "検索を開始できませんでした。"}})
		return
	}
	writeSlackJSON(w, map[string]any{"response_action": "clear"})
	if created {
		go handler.processSlackSearch(requestID, payload.User.ID, input)
	}
}

func (handler *SlackHandler) processSlackSearch(requestID string, userID string, search slackSearchInput) {
	ctx, cancel := context.WithTimeout(context.Background(), chatDeliveryTimeout)
	defer cancel()
	fail := func(message string) {
		_ = handler.store.MarkFailed(context.Background(), requestID, handler.now())
		_ = handler.postEphemeral(context.Background(), search.ChannelID, userID, message, nil)
	}

	locations, err := handler.geocoder.Search(ctx, search.LocationQuery)
	if err != nil || len(locations) == 0 {
		fail("出発地を確認できませんでした。駅名や地名を変えてもう一度お試しください。")
		return
	}
	latitude := locations[0].Location.Latitude
	longitude := locations[0].Location.Longitude
	input := session.Input{
		Purpose:          search.Purpose,
		Mood:             search.Mood,
		Level:            search.Level,
		AvailableMinutes: search.AvailableMinutes,
		Transport:        search.Transport,
		Origin: session.Origin{
			Mode:      session.OriginSpecifiedLocation,
			Latitude:  &latitude,
			Longitude: &longitude,
		},
	}
	response, err := handler.service.RecommendResponse(ctx, input)
	if err != nil {
		fail("おすすめスポットを取得できませんでした。時間をおいて再度お試しください。")
		return
	}
	text, blocks := handler.slackRecommendationMessage(response, search.Transport)
	if err := handler.postEphemeral(context.Background(), search.ChannelID, userID, text, blocks); err != nil {
		_ = handler.store.MarkFailed(context.Background(), requestID, handler.now())
		return
	}
	_ = handler.store.MarkDelivered(context.Background(), requestID, handler.now())
}

func (handler *SlackHandler) handleBlockAction(w http.ResponseWriter, payload slackInteractionPayload) {
	if len(payload.Actions) != 1 {
		http.Error(w, "invalid Slack block action", http.StatusBadRequest)
		return
	}
	action := payload.Actions[0]
	if strings.HasPrefix(action.ActionID, "spotdiggz_official_") || strings.HasPrefix(action.ActionID, "spotdiggz_route_") {
		w.WriteHeader(http.StatusOK)
		return
	}
	http.Error(w, "unsupported Slack action", http.StatusBadRequest)
}

func parseSlackSearchInput(view struct {
	ID              string `json:"id"`
	CallbackID      string `json:"callback_id"`
	PrivateMetadata string `json:"private_metadata"`
	State           struct {
		Values map[string]map[string]slackViewStateValue `json:"values"`
	} `json:"state"`
}) (slackSearchInput, map[string]string) {
	errorsByBlock := make(map[string]string)
	location := strings.TrimSpace(slackStateValue(view.State.Values, "location", "location"))
	if location == "" || len([]rune(location)) > maximumSlackLocationRunes || strings.ContainsAny(location, "\r\n\x00") {
		errorsByBlock["location"] = "駅名または地名を120文字以内で入力してください。"
	}
	minutes, err := strconv.Atoi(slackStateValue(view.State.Values, "available_minutes", "available_minutes"))
	if err != nil {
		errorsByBlock["available_minutes"] = "使える時間を選択してください。"
	}
	var metadata slackModalMetadata
	if err := json.Unmarshal([]byte(view.PrivateMetadata), &metadata); err != nil || !validSlackChannelID(metadata.ChannelID) {
		errorsByBlock["location"] = "検索元のSlack channelを確認できませんでした。"
	}
	input := slackSearchInput{
		LocationQuery:    location,
		AvailableMinutes: minutes,
		Transport:        session.Transport(slackStateValue(view.State.Values, "transport", "transport")),
		Level:            session.Level(slackStateValue(view.State.Values, "level", "level")),
		Purpose:          session.Purpose(slackStateValue(view.State.Values, "purpose", "purpose")),
		Mood:             session.Mood(slackStateValue(view.State.Values, "mood", "mood")),
		ChannelID:        metadata.ChannelID,
	}
	placeholderLatitude := 0.0
	placeholderLongitude := 0.0
	validationInput := session.Input{
		Purpose: input.Purpose, Mood: input.Mood, Level: input.Level,
		AvailableMinutes: input.AvailableMinutes, Transport: input.Transport,
		Origin: session.Origin{Mode: session.OriginSpecifiedLocation, Latitude: &placeholderLatitude, Longitude: &placeholderLongitude},
	}
	if err := validationInput.Validate(); err != nil {
		errorsByBlock["available_minutes"] = "検索条件を選び直してください。"
	}
	return input, errorsByBlock
}

func slackStateValue(values map[string]map[string]slackViewStateValue, blockID string, actionID string) string {
	value := values[blockID][actionID]
	if value.SelectedOption != nil {
		return value.SelectedOption.Value
	}
	return value.Value
}

func (handler *SlackHandler) slackRecommendationMessage(response recommendation.Response, transport session.Transport) (string, []any) {
	if len(response.Recommendations) == 0 {
		return "現在の条件で案内できるスポットがありません。条件を変えてもう一度お試しください。", nil
	}
	blocks := make([]any, 0, len(response.Recommendations)*2+2)
	blocks = append(blocks, map[string]any{"type": "header", "text": slackPlainText("おすすめスポット")})
	for index, item := range response.Recommendations {
		reason := "条件に合う候補です。"
		if len(item.Reasons) > 0 {
			reason = item.Reasons[0].Message
		}
		text := fmt.Sprintf("*No.%d %s*\n片道約%d分・滑走約%d分\n%s",
			index+1, escapeSlackText(item.Facility.Name), item.EstimatedTravelMinutes,
			item.EstimatedSkateMinutes, escapeSlackText(reason))
		blocks = append(blocks, map[string]any{"type": "section", "text": map[string]any{"type": "mrkdwn", "text": text}})
		actions := []any{
			map[string]any{"type": "button", "action_id": "spotdiggz_official_" + strconv.Itoa(index), "text": slackPlainText("公式情報"), "url": item.Facility.SourceURL},
			map[string]any{"type": "button", "action_id": "spotdiggz_route_" + strconv.Itoa(index), "text": slackPlainText("ここに行く"), "url": slackNavigationURL(item, transport)},
		}
		blocks = append(blocks, map[string]any{"type": "actions", "block_id": "candidate_" + strconv.Itoa(index), "elements": actions})
	}
	return FormatRecommendation(response), blocks
}

func (handler *SlackHandler) postEphemeral(ctx context.Context, channelID string, userID string, text string, blocks []any) error {
	payload := map[string]any{"channel": channelID, "user": userID, "text": text}
	if len(blocks) > 0 {
		payload["blocks"] = blocks
	}
	return handler.callSlackAPI(ctx, "chat.postEphemeral", payload)
}

func (handler *SlackHandler) callSlackAPI(parent context.Context, method string, payload any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("encode Slack API request: %w", err)
	}
	ctx, cancel := context.WithTimeout(parent, slackAPIRequestTimeout)
	defer cancel()
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://slack.com/api/"+method, strings.NewReader(string(body)))
	if err != nil {
		return fmt.Errorf("create Slack API request: %w", err)
	}
	request.Header.Set("Authorization", "Bearer "+handler.botToken)
	request.Header.Set("Content-Type", "application/json; charset=utf-8")
	response, err := handler.httpClient.Do(request)
	if err != nil {
		return fmt.Errorf("call Slack API: %w", err)
	}
	defer response.Body.Close()
	responseBody, err := io.ReadAll(io.LimitReader(response.Body, maximumSlackAPIBytes+1))
	if err != nil || len(responseBody) > maximumSlackAPIBytes || response.StatusCode != http.StatusOK {
		return errors.New("Slack API response failed")
	}
	var result struct {
		OK bool `json:"ok"`
	}
	if err := json.Unmarshal(responseBody, &result); err != nil || !result.OK {
		return errors.New("Slack API rejected request")
	}
	return nil
}

type slackOption struct {
	Text  string
	Value string
}

func slackPlainText(text string) map[string]any {
	return map[string]any{"type": "plain_text", "text": text, "emoji": true}
}

func slackTextInput(blockID string, actionID string, label string, placeholder string, optional bool, maxLength int) map[string]any {
	return map[string]any{
		"type": "input", "block_id": blockID, "optional": optional,
		"label": slackPlainText(label),
		"element": map[string]any{
			"type": "plain_text_input", "action_id": actionID,
			"placeholder": slackPlainText(placeholder), "max_length": maxLength,
		},
	}
}

func slackSelectInput(blockID string, actionID string, label string, options []slackOption, initialValue string) map[string]any {
	elements := make([]any, 0, len(options))
	var initial any
	for _, option := range options {
		item := map[string]any{"text": slackPlainText(option.Text), "value": option.Value}
		elements = append(elements, item)
		if option.Value == initialValue {
			initial = item
		}
	}
	return map[string]any{
		"type": "input", "block_id": blockID, "label": slackPlainText(label),
		"element": map[string]any{
			"type": "static_select", "action_id": actionID,
			"options": elements, "initial_option": initial,
		},
	}
}

func slackNavigationURL(item recommendation.Item, transport session.Transport) string {
	values := url.Values{}
	values.Set("api", "1")
	values.Set("destination", strconv.FormatFloat(item.Facility.Location.Latitude, 'f', 6, 64)+","+strconv.FormatFloat(item.Facility.Location.Longitude, 'f', 6, 64))
	travelModes := map[session.Transport]string{
		session.TransportPublicTransit: "transit", session.TransportCar: "driving",
		session.TransportBicycle: "bicycling", session.TransportWalk: "walking",
	}
	if travelMode := travelModes[transport]; travelMode != "" {
		values.Set("travelmode", travelMode)
	}
	return "https://www.google.com/maps/dir/?" + values.Encode()
}

func escapeSlackText(value string) string {
	replacer := strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;")
	return replacer.Replace(value)
}

func newSlackRequestID() (string, error) {
	var bytes [16]byte
	if _, err := rand.Read(bytes[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes[:]), nil
}

func (handler *SlackHandler) slackSourceEventKey(viewID string) string {
	mac := hmac.New(sha256.New, handler.signingSecret)
	_, _ = mac.Write([]byte(viewID))
	return hex.EncodeToString(mac.Sum(nil))
}

func validSlackChannelID(value string) bool {
	if len(value) < 3 || len(value) > 32 {
		return false
	}
	for _, character := range value {
		if (character < 'A' || character > 'Z') && (character < '0' || character > '9') {
			return false
		}
	}
	return true
}

func writeSlackJSON(w http.ResponseWriter, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(payload)
}
