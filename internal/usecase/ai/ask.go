package ai

import (
	"context"
	"fmt"
	"unicode/utf8"

	"github.com/firereach/api/internal/domain"
)

// Bounds on a question and its history, so a long or crafted chat cannot blow
// the model's token budget or the request size. Lengths count characters, not
// bytes, so the app can cap its input box at the same number and never send a
// question this refuses. The gateway further trims to the most recent turns
// for the actual API call.
const (
	maxQuestionLen    = 1000
	maxTopicLen       = 100
	maxHistoryTurns   = 50
	maxTurnContentLen = 2000
)

type AskAI struct {
	gateway domain.AIGateway
}

func NewAskAI(gw domain.AIGateway) *AskAI {
	return &AskAI{gateway: gw}
}

func (uc *AskAI) Execute(ctx context.Context, question, topic string, history []domain.Turn) (domain.AIResponse, error) {
	if question == "" || utf8.RuneCountInString(question) > maxQuestionLen {
		return domain.AIResponse{}, fmt.Errorf("ask ai: %w", domain.ErrInvalidInput)
	}
	if utf8.RuneCountInString(topic) > maxTopicLen {
		return domain.AIResponse{}, fmt.Errorf("ask ai: %w", domain.ErrInvalidInput)
	}
	if len(history) > maxHistoryTurns {
		return domain.AIResponse{}, fmt.Errorf("ask ai: %w", domain.ErrInvalidInput)
	}
	for _, t := range history {
		if utf8.RuneCountInString(t.Content) > maxTurnContentLen {
			return domain.AIResponse{}, fmt.Errorf("ask ai: %w", domain.ErrInvalidInput)
		}
	}
	resp, err := uc.gateway.Ask(ctx, question, topic, history)
	if err != nil {
		return domain.AIResponse{}, fmt.Errorf("ask ai: %w", err)
	}
	return resp, nil
}
