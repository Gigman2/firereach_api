package content

import (
	"context"
	"fmt"

	"github.com/firereach/api/internal/domain"
)

type ListContent struct {
	repo domain.ContentRepository
}

func NewListContent(r domain.ContentRepository) *ListContent {
	return &ListContent{repo: r}
}

func (uc *ListContent) Execute(ctx context.Context, category, subcategory string) ([]domain.SafetyContent, error) {
	items, err := uc.repo.List(ctx, category, subcategory)
	if err != nil {
		return nil, fmt.Errorf("list content: %w", err)
	}
	return items, nil
}
