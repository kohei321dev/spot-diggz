package main

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/kohei321dev/spot-diggz/internal/chatbot"
	"github.com/kohei321dev/spot-diggz/internal/geocoding"
	"github.com/kohei321dev/spot-diggz/internal/recommendation"
	"github.com/kohei321dev/spot-diggz/internal/session"
)

type configTestRecommender struct{}

func (configTestRecommender) RecommendContext(context.Context, session.Input) (recommendation.Response, error) {
	return recommendation.Response{}, nil
}

type configTestGeocoder struct{}

func (configTestGeocoder) Search(context.Context, string) ([]geocoding.Result, error) {
	return []geocoding.Result{}, nil
}

func TestBuildChatHandlersDoesNothingWithoutIntegrationConfiguration(t *testing.T) {
	clearChatEnvironment(t)
	slackHandler, discordHandler, err := buildChatHandlers(configTestRecommender{}, nil, nil, time.Now)
	if err != nil {
		t.Fatalf("buildChatHandlers() error = %v", err)
	}
	if slackHandler != nil || discordHandler != nil {
		t.Fatalf("handlers = (%v, %v), want both nil", slackHandler, discordHandler)
	}
}

func TestBuildChatHandlersRejectsPartialSlackConfiguration(t *testing.T) {
	clearChatEnvironment(t)
	t.Setenv("SLACK_SIGNING_SECRET", "secret")
	_, _, err := buildChatHandlers(configTestRecommender{}, nil, nil, time.Now)
	if err == nil || !strings.Contains(err.Error(), "SLACK_BOT_TOKEN") {
		t.Fatalf("error = %v, want missing SLACK_BOT_TOKEN", err)
	}
}

func TestBuildChatHandlersRequiresSlackGeocoder(t *testing.T) {
	clearChatEnvironment(t)
	t.Setenv("SLACK_BOT_TOKEN", "xoxb-test")
	t.Setenv("SLACK_SIGNING_SECRET", "secret")
	t.Setenv("SLACK_TEAM_ID", "T123")
	t.Setenv("SLACK_OWNER_USER_ID", "U123")
	_, _, err := buildChatHandlers(configTestRecommender{}, nil, chatbot.NewMemorySlackRequestStore(), time.Now)
	if err == nil || !strings.Contains(err.Error(), "geocoding provider") {
		t.Fatalf("error = %v, want missing geocoding provider", err)
	}
}

func TestBuildChatHandlersCreatesConfiguredSlackHandler(t *testing.T) {
	clearChatEnvironment(t)
	t.Setenv("SLACK_BOT_TOKEN", "xoxb-test")
	t.Setenv("SLACK_SIGNING_SECRET", "secret")
	t.Setenv("SLACK_TEAM_ID", "T123")
	t.Setenv("SLACK_OWNER_USER_ID", "U123")
	slackHandler, discordHandler, err := buildChatHandlers(configTestRecommender{}, configTestGeocoder{}, chatbot.NewMemorySlackRequestStore(), time.Now)
	if err != nil {
		t.Fatalf("buildChatHandlers() error = %v", err)
	}
	if slackHandler == nil || discordHandler != nil {
		t.Fatalf("handlers = (%v, %v), want Slack only", slackHandler, discordHandler)
	}
}

func clearChatEnvironment(t *testing.T) {
	t.Helper()
	keys := append([]string{}, slackConfigurationKeys...)
	keys = append(keys, discordConfigurationKeys...)
	keys = append(keys,
		"CHAT_DEFAULT_ORIGIN_LATITUDE",
		"CHAT_DEFAULT_ORIGIN_LONGITUDE",
		"CHAT_DEFAULT_PURPOSE",
		"CHAT_DEFAULT_MOOD",
		"CHAT_DEFAULT_LEVEL",
		"CHAT_DEFAULT_AVAILABLE_MINUTES",
		"CHAT_DEFAULT_TRANSPORT",
	)
	for _, key := range keys {
		t.Setenv(key, "")
	}
}
