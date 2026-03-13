package content_test

import (
	"context"
	"errors"
	"testing"

	"github.com/firereach/api/internal/domain"
	"github.com/firereach/api/internal/usecase/content"
	"github.com/firereach/api/internal/usecase/mocks"
)

func TestGetContent_Success(t *testing.T) {
	repo := &mocks.ContentRepo{
		GetByIDFunc: func(ctx context.Context, id string) (*domain.SafetyContent, error) {
			return &domain.SafetyContent{ID: "1", Title: "Burns Treatment"}, nil
		},
	}

	uc := content.NewGetContent(repo)
	item, err := uc.Execute(context.Background(), "1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if item.Title != "Burns Treatment" {
		t.Errorf("expected 'Burns Treatment', got '%s'", item.Title)
	}
}

func TestGetContent_NotFound(t *testing.T) {
	repo := &mocks.ContentRepo{
		GetByIDFunc: func(ctx context.Context, id string) (*domain.SafetyContent, error) {
			return nil, domain.ErrNotFound
		},
	}

	uc := content.NewGetContent(repo)
	_, err := uc.Execute(context.Background(), "999")
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}
