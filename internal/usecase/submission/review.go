package submission

import (
	"context"
	"fmt"
	"unicode/utf8"

	"github.com/firereach/api/internal/domain"
)

type ReviewSubmission struct {
	subRepo     domain.SubmissionRepository
	stationRepo domain.StationRepository
}

func NewReviewSubmission(sr domain.SubmissionRepository, str domain.StationRepository) *ReviewSubmission {
	return &ReviewSubmission{subRepo: sr, stationRepo: str}
}

func (uc *ReviewSubmission) Execute(ctx context.Context, id string, status domain.SubmissionStatus, adminNote string) error {
	if status != domain.SubmissionStatusApproved && status != domain.SubmissionStatusRejected {
		return fmt.Errorf("review submission: %w", domain.ErrInvalidInput)
	}
	if utf8.RuneCountInString(adminNote) > maxTextLen {
		return fmt.Errorf("review submission: %w", invalid("admin_note must be at most 1000 characters"))
	}
	if err := uc.subRepo.UpdateStatus(ctx, id, status, adminNote); err != nil {
		return fmt.Errorf("review submission: %w", err)
	}
	return nil
}
