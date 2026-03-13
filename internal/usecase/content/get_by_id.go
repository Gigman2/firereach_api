package content

import (
	"context"
	"fmt"

	"github.com/firereach/api/internal/domain"
)

type GetContent struct {
	repo domain.ContentRepository
}

func NewGetContent(r domain.ContentRepository) *GetContent {
	return &GetContent{repo: r}
}

func (uc *GetContent) Execute(ctx context.Context, id string) (*domain.SafetyContent, error) {
	item, err := uc.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get content: %w", err)
	}
	return item, nil
}
