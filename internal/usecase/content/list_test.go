package content_test

import (
	"context"
	"errors"
	"testing"

	"github.com/firereach/api/internal/domain"
	"github.com/firereach/api/internal/usecase/content"
	"github.com/firereach/api/internal/usecase/mocks"
)

func TestListContent_Success(t *testing.T) {
	repo := &mocks.ContentRepo{
		ListFunc: func(ctx context.Context, category, subcategory string) ([]domain.SafetyContent, error) {
			return []domain.SafetyContent{
				{ID: "1", Category: "hazard", Title: "Electrical Fires"},
				{ID: "2", Category: "hazard", Title: "Cooking Fires"},
			}, nil
		},
	}

	uc := content.NewListContent(repo)
	items, err := uc.Execute(context.Background(), "hazard", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}
}

func TestListContent_RepoError(t *testing.T) {
	repo := &mocks.ContentRepo{
		ListFunc: func(ctx context.Context, category, subcategory string) ([]domain.SafetyContent, error) {
			return nil, errors.New("db error")
		},
	}

	uc := content.NewListContent(repo)
	_, err := uc.Execute(context.Background(), "", "")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
