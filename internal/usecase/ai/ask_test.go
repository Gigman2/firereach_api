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
		AskFunc: func(ctx context.Context, question, topic string, history []domain.Turn) (domain.AIResponse, error) {
			return domain.AIResponse{Kind: domain.KindText, Body: "Keep calm and evacuate immediately."}, nil
		},
	}

	uc := ai.NewAskAI(gw)
	answer, err := uc.Execute(context.Background(), "What do I do if there's a fire?", "hazard", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if answer.Body == "" {
		t.Error("expected non-empty answer body")
	}
}

func TestAskAI_EmptyQuestion(t *testing.T) {
	gw := &mocks.AIGateway{}

	uc := ai.NewAskAI(gw)
	_, err := uc.Execute(context.Background(), "", "", nil)
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Errorf("expected ErrInvalidInput, got %v", err)
	}
}

func TestAskAI_QuestionTooLong(t *testing.T) {
	gw := &mocks.AIGateway{}

	uc := ai.NewAskAI(gw)
	longQ := strings.Repeat("a", 1001)
	_, err := uc.Execute(context.Background(), longQ, "", nil)
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Errorf("expected ErrInvalidInput, got %v", err)
	}
}

func TestAskAI_TopicTooLong(t *testing.T) {
	gw := &mocks.AIGateway{}

	uc := ai.NewAskAI(gw)
	longTopic := strings.Repeat("a", 101)
	_, err := uc.Execute(context.Background(), "What do I do if there's a fire?", longTopic, nil)
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Errorf("expected ErrInvalidInput, got %v", err)
	}
}

func TestAskAI_GatewayError(t *testing.T) {
	gw := &mocks.AIGateway{
		AskFunc: func(ctx context.Context, question, topic string, history []domain.Turn) (domain.AIResponse, error) {
			return domain.AIResponse{}, errors.New("api error")
		},
	}

	uc := ai.NewAskAI(gw)
	_, err := uc.Execute(context.Background(), "How to treat burns?", "first_aid", nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestAskAI_RateLimited(t *testing.T) {
	gw := &mocks.AIGateway{
		AskFunc: func(ctx context.Context, question, topic string, history []domain.Turn) (domain.AIResponse, error) {
			return domain.AIResponse{}, domain.ErrRateLimited
		},
	}

	uc := ai.NewAskAI(gw)
	_, err := uc.Execute(context.Background(), "question", "topic", nil)
	if !errors.Is(err, domain.ErrRateLimited) {
		t.Errorf("expected ErrRateLimited, got %v", err)
	}
}
