package chatbot

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/kohei321dev/spot-diggz/internal/recommendation"
	"github.com/kohei321dev/spot-diggz/internal/session"
)

const maximumMessageRunes = 1900

type Recommender interface {
	RecommendContext(context.Context, session.Input) (recommendation.Response, error)
}

type Service struct {
	recommender  Recommender
	defaultInput *session.Input
}

func NewService(recommender Recommender, input session.Input) (*Service, error) {
	if recommender == nil {
		return nil, errors.New("chat recommendation service requires a recommender")
	}
	if err := input.Validate(); err != nil {
		return nil, fmt.Errorf("chat recommendation input: %w", err)
	}
	return &Service{recommender: recommender, defaultInput: &input}, nil
}

func NewDynamicService(recommender Recommender) (*Service, error) {
	if recommender == nil {
		return nil, errors.New("chat recommendation service requires a recommender")
	}
	return &Service{recommender: recommender}, nil
}

func (service *Service) Recommend(ctx context.Context) (string, error) {
	if service.defaultInput == nil {
		return "", errors.New("chat recommendation service has no default input")
	}
	return service.RecommendInput(ctx, *service.defaultInput)
}

func (service *Service) RecommendInput(ctx context.Context, input session.Input) (string, error) {
	response, err := service.RecommendResponse(ctx, input)
	if err != nil {
		return "", err
	}
	return FormatRecommendation(response), nil
}

func (service *Service) RecommendResponse(ctx context.Context, input session.Input) (recommendation.Response, error) {
	if err := input.Validate(); err != nil {
		return recommendation.Response{}, fmt.Errorf("chat recommendation input: %w", err)
	}
	response, err := service.recommender.RecommendContext(ctx, input)
	if err != nil {
		return recommendation.Response{}, fmt.Errorf("recommend spots: %w", err)
	}
	return response, nil
}

func FormatRecommendation(response recommendation.Response) string {
	if len(response.Recommendations) == 0 {
		return "現在の条件で案内できるスポットがありません。条件を変えてWeb版でもう一度検索してください。"
	}

	var message strings.Builder
	message.WriteString("おすすめスポット\n")
	for index, item := range response.Recommendations {
		fmt.Fprintf(&message, "\n%d. %s\n", index+1, item.Facility.Name)
		fmt.Fprintf(&message, "片道約%d分・滑走約%d分\n", item.EstimatedTravelMinutes, item.EstimatedSkateMinutes)
		if len(item.Reasons) > 0 {
			message.WriteString(item.Reasons[0].Message)
			message.WriteByte('\n')
		}
		fmt.Fprintf(&message, "公式情報: %s\n", item.Facility.SourceURL)
	}
	message.WriteString("\n営業時間や貸切は公式情報で確認してください。")
	return truncateMessage(message.String(), maximumMessageRunes)
}

func truncateMessage(message string, maximumRunes int) string {
	runes := []rune(message)
	if len(runes) <= maximumRunes {
		return message
	}
	return string(runes[:maximumRunes-1]) + "…"
}
