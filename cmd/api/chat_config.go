package main

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/kohei321dev/spot-diggz/internal/chatbot"
	"github.com/kohei321dev/spot-diggz/internal/geocoding"
	"github.com/kohei321dev/spot-diggz/internal/session"
)

const (
	defaultChatPurpose          = session.PurposeBasics
	defaultChatMood             = session.MoodFocused
	defaultChatLevel            = session.LevelBeginner
	defaultChatAvailableMinutes = 120
	defaultChatTransport        = session.TransportPublicTransit
)

var (
	slackConfigurationKeys   = []string{"SLACK_BOT_TOKEN", "SLACK_SIGNING_SECRET", "SLACK_TEAM_ID", "SLACK_OWNER_USER_ID"}
	discordConfigurationKeys = []string{
		"DISCORD_PUBLIC_KEY",
		"DISCORD_APPLICATION_ID",
		"DISCORD_GUILD_ID",
		"DISCORD_OWNER_USER_ID",
	}
)

func buildChatHandlers(recommender chatbot.Recommender, geocoder geocoding.Provider, slackStore chatbot.SlackRequestStore, now func() time.Time) (http.Handler, http.Handler, error) {
	slackRequested := anyEnvironmentValue(slackConfigurationKeys)
	discordRequested := anyEnvironmentValue(discordConfigurationKeys)
	if !slackRequested && !discordRequested {
		return nil, nil, nil
	}
	if slackRequested {
		if err := requireEnvironmentValues(slackConfigurationKeys); err != nil {
			return nil, nil, err
		}
	}
	if discordRequested {
		if err := requireEnvironmentValues(discordConfigurationKeys); err != nil {
			return nil, nil, err
		}
	}
	var service *chatbot.Service
	var err error
	if discordRequested {
		input, inputErr := loadChatRecommendationInput()
		if inputErr != nil {
			return nil, nil, inputErr
		}
		service, err = chatbot.NewService(recommender, input)
	} else {
		service, err = chatbot.NewDynamicService(recommender)
	}
	if err != nil {
		return nil, nil, err
	}

	var slackHandler http.Handler
	if slackRequested {
		if geocoder == nil {
			return nil, nil, errors.New("Slack integration requires a geocoding provider")
		}
		if slackStore == nil {
			return nil, nil, errors.New("Slack integration requires a request store")
		}
		slackHandler, err = chatbot.NewSlackHandler(service, chatbot.SlackConfig{
			SigningSecret: os.Getenv("SLACK_SIGNING_SECRET"),
			BotToken:      os.Getenv("SLACK_BOT_TOKEN"),
			TeamID:        os.Getenv("SLACK_TEAM_ID"),
			UserID:        os.Getenv("SLACK_OWNER_USER_ID"),
			Geocoder:      geocoder,
			Store:         slackStore,
			Now:           now,
		})
		if err != nil {
			return nil, nil, err
		}
	}

	var discordHandler http.Handler
	if discordRequested {
		discordHandler, err = chatbot.NewDiscordHandler(service, chatbot.DiscordConfig{
			PublicKeyHex:  os.Getenv("DISCORD_PUBLIC_KEY"),
			ApplicationID: os.Getenv("DISCORD_APPLICATION_ID"),
			GuildID:       os.Getenv("DISCORD_GUILD_ID"),
			UserID:        os.Getenv("DISCORD_OWNER_USER_ID"),
			Now:           now,
		})
		if err != nil {
			return nil, nil, err
		}
	}
	return slackHandler, discordHandler, nil
}

func loadChatRecommendationInput() (session.Input, error) {
	latitude, err := requiredFloatEnvironment("CHAT_DEFAULT_ORIGIN_LATITUDE")
	if err != nil {
		return session.Input{}, err
	}
	longitude, err := requiredFloatEnvironment("CHAT_DEFAULT_ORIGIN_LONGITUDE")
	if err != nil {
		return session.Input{}, err
	}
	availableMinutes := defaultChatAvailableMinutes
	if rawMinutes := strings.TrimSpace(os.Getenv("CHAT_DEFAULT_AVAILABLE_MINUTES")); rawMinutes != "" {
		parsedMinutes, parseErr := strconv.Atoi(rawMinutes)
		if parseErr != nil {
			return session.Input{}, errors.New("CHAT_DEFAULT_AVAILABLE_MINUTES must be an integer")
		}
		availableMinutes = parsedMinutes
	}
	input := session.Input{
		Purpose:          session.Purpose(envOrDefault("CHAT_DEFAULT_PURPOSE", string(defaultChatPurpose))),
		Mood:             session.Mood(envOrDefault("CHAT_DEFAULT_MOOD", string(defaultChatMood))),
		Level:            session.Level(envOrDefault("CHAT_DEFAULT_LEVEL", string(defaultChatLevel))),
		AvailableMinutes: availableMinutes,
		Transport:        session.Transport(envOrDefault("CHAT_DEFAULT_TRANSPORT", string(defaultChatTransport))),
		Origin: session.Origin{
			Mode:      session.OriginSpecifiedLocation,
			Latitude:  &latitude,
			Longitude: &longitude,
		},
	}
	if err := input.Validate(); err != nil {
		return session.Input{}, fmt.Errorf("chat default recommendation input is invalid: %w", err)
	}
	return input, nil
}

func requiredFloatEnvironment(key string) (float64, error) {
	rawValue := strings.TrimSpace(os.Getenv(key))
	if rawValue == "" {
		return 0, fmt.Errorf("%s is required when a chat integration is enabled", key)
	}
	value, err := strconv.ParseFloat(rawValue, 64)
	if err != nil {
		return 0, fmt.Errorf("%s must be a decimal number", key)
	}
	return value, nil
}

func anyEnvironmentValue(keys []string) bool {
	for _, key := range keys {
		if strings.TrimSpace(os.Getenv(key)) != "" {
			return true
		}
	}
	return false
}

func requireEnvironmentValues(keys []string) error {
	for _, key := range keys {
		if strings.TrimSpace(os.Getenv(key)) == "" {
			return fmt.Errorf("%s is required when the integration is enabled", key)
		}
	}
	return nil
}
