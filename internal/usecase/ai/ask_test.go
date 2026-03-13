package ai_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/firereach/api/internal/domain"
	"github.com/firereach/api/internal/usecase/ai"
	"github.com/firereach/api/internal/usecase/mocks"
)

func TestAskAI_Success(t *testing.T) {
	gw := &mocks.AIGateway{
		AskFunc: func(ctx context.Context, question, topic string) (string, error) {
			return "Keep calm and evacuate immediately.", nil
		},
	}

	uc := ai.NewAskAI(gw)
	answer, err := uc.Execute(context.Background(), "What do I do if there's a fire?", "hazard")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if answer == "" {
		t.Error("expected non-empty answer")
	}
}

func TestAskAI_EmptyQuestion(t *testing.T) {
	gw := &mocks.AIGateway{}

	uc := ai.NewAskAI(gw)
	_, err := uc.Execute(context.Background(), "", "")
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Errorf("expected ErrInvalidInput, got %v", err)
	}
}

func TestAskAI_QuestionTooLong(t *testing.T) {
	gw := &mocks.AIGateway{}

	uc := ai.NewAskAI(gw)
	longQ := strings.Repeat("a", 1001)
	_, err := uc.Execute(context.Background(), longQ, "")
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Errorf("expected ErrInvalidInput, got %v", err)
	}
}

func TestAskAI_GatewayError(t *testing.T) {
	gw := &mocks.AIGateway{
		AskFunc: func(ctx context.Context, question, topic string) (string, error) {
			return "", errors.New("api error")
		},
	}

	uc := ai.NewAskAI(gw)
	_, err := uc.Execute(context.Background(), "How to treat burns?", "first_aid")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestAskAI_RateLimited(t *testing.T) {
	gw := &mocks.AIGateway{
		AskFunc: func(ctx context.Context, question, topic string) (string, error) {
			return "", domain.ErrRateLimited
		},
	}

	uc := ai.NewAskAI(gw)
	_, err := uc.Execute(context.Background(), "question", "topic")
	if !errors.Is(err, domain.ErrRateLimited) {
		t.Errorf("expected ErrRateLimited, got %v", err)
	}
}
