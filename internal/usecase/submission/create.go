package submission

import (
	"context"
	"fmt"

	"github.com/firereach/api/internal/domain"
)

type CreateSubmission struct {
	repo domain.SubmissionRepository
}

func NewCreateSubmission(r domain.SubmissionRepository) *CreateSubmission {
	return &CreateSubmission{repo: r}
}

func (uc *CreateSubmission) Execute(ctx context.Context, s domain.Submission) error {
	if s.Type == "" || s.SuggestedValue == "" {
		return fmt.Errorf("create submission: %w", domain.ErrInvalidInput)
	}
	s.Status = domain.SubmissionStatusPending
	if err := uc.repo.Create(ctx, s); err != nil {
		return fmt.Errorf("create submission: %w", err)
	}
	return nil
}
