package ai

import (
	"context"
	"fmt"

	"github.com/firereach/api/internal/domain"
)

// Bounds on conversation history, so a long or crafted chat cannot blow the
// model's token budget or the request size. The gateway further trims to the
// most recent turns for the actual API call.
const (
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
	if len(question) == 0 || len(question) > 1000 {
		return domain.AIResponse{}, fmt.Errorf("ask ai: %w", domain.ErrInvalidInput)
	}
	if len(topic) > 100 {
		return domain.AIResponse{}, fmt.Errorf("ask ai: %w", domain.ErrInvalidInput)
	}
	if len(history) > maxHistoryTurns {
		return domain.AIResponse{}, fmt.Errorf("ask ai: %w", domain.ErrInvalidInput)
	}
	for _, t := range history {
		if len(t.Content) > maxTurnContentLen {
			return domain.AIResponse{}, fmt.Errorf("ask ai: %w", domain.ErrInvalidInput)
		}
	}
	resp, err := uc.gateway.Ask(ctx, question, topic, history)
	if err != nil {
		return domain.AIResponse{}, fmt.Errorf("ask ai: %w", err)
	}
	return resp, nil
}
