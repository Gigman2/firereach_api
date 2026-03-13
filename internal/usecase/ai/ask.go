package ai

import (
	"context"
	"fmt"

	"github.com/firereach/api/internal/domain"
)

type AskAI struct {
	gateway domain.AIGateway
}

func NewAskAI(gw domain.AIGateway) *AskAI {
	return &AskAI{gateway: gw}
}

func (uc *AskAI) Execute(ctx context.Context, question, topic string) (string, error) {
	if len(question) == 0 || len(question) > 1000 {
		return "", fmt.Errorf("ask ai: %w", domain.ErrInvalidInput)
	}
	resp, err := uc.gateway.Ask(ctx, question, topic)
	if err != nil {
		return "", fmt.Errorf("ask ai: %w", err)
	}
	return resp, nil
}
