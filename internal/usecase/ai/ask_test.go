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

// The limits count characters, not bytes, so an app that caps its input at
// 1,000 characters can never send a question this refuses. "ɛ" and "ɔ" are Twi
// letters of two bytes each.
func TestAskAI_LimitsCountCharactersNotBytes(t *testing.T) {
	gw := &mocks.AIGateway{
		AskFunc: func(ctx context.Context, question, topic string, history []domain.Turn) (domain.AIResponse, error) {
			return domain.AIResponse{Kind: domain.KindText, Body: "ok"}, nil
		},
	}
	uc := ai.NewAskAI(gw)

	question := strings.Repeat("ɛ", 1000)
	topic := strings.Repeat("ɔ", 100)
	history := []domain.Turn{{Role: "assistant", Content: strings.Repeat("ɛ", 2000)}}
	if _, err := uc.Execute(context.Background(), question, topic, history); err != nil {
		t.Fatalf("at the limits: unexpected error: %v", err)
	}

	over := []struct {
		name            string
		question, topic string
		history         []domain.Turn
	}{
		{"question", strings.Repeat("ɛ", 1001), "", nil},
		{"topic", "How do I stop a pan fire?", strings.Repeat("ɔ", 101), nil},
		{"history turn", "How do I stop a pan fire?", "", []domain.Turn{{Role: "user", Content: strings.Repeat("ɛ", 2001)}}},
	}
	for _, tt := range over {
		if _, err := uc.Execute(context.Background(), tt.question, tt.topic, tt.history); !errors.Is(err, domain.ErrInvalidInput) {
			t.Errorf("%s one character over: expected ErrInvalidInput, got %v", tt.name, err)
		}
	}
}
