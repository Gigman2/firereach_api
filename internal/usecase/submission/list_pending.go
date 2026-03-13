package submission

import (
	"context"
	"fmt"

	"github.com/firereach/api/internal/domain"
)

type ListPending struct {
	repo domain.SubmissionRepository
}

func NewListPending(r domain.SubmissionRepository) *ListPending {
	return &ListPending{repo: r}
}

func (uc *ListPending) Execute(ctx context.Context) ([]domain.Submission, error) {
	subs, err := uc.repo.ListPending(ctx)
	if err != nil {
		return nil, fmt.Errorf("list pending submissions: %w", err)
	}
	return subs, nil
}
