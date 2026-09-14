package submission_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/firereach/api/internal/domain"
	"github.com/firereach/api/internal/usecase/mocks"
	"github.com/firereach/api/internal/usecase/submission"
)

func TestReviewSubmission_Approve(t *testing.T) {
	subRepo := &mocks.SubmissionRepo{
		UpdateStatusFunc: func(ctx context.Context, id string, status domain.SubmissionStatus, adminNote string) error {
			if status != domain.SubmissionStatusApproved {
				t.Errorf("expected approved, got %s", status)
			}
			return nil
		},
	}
	stationRepo := &mocks.StationRepo{}

	uc := submission.NewReviewSubmission(subRepo, stationRepo)
	err := uc.Execute(context.Background(), "1", domain.SubmissionStatusApproved, "looks good")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestReviewSubmission_Reject(t *testing.T) {
	subRepo := &mocks.SubmissionRepo{
		UpdateStatusFunc: func(ctx context.Context, id string, status domain.SubmissionStatus, adminNote string) error {
			return nil
		},
	}
	stationRepo := &mocks.StationRepo{}

	uc := submission.NewReviewSubmission(subRepo, stationRepo)
	err := uc.Execute(context.Background(), "1", domain.SubmissionStatusRejected, "duplicate")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestReviewSubmission_InvalidStatus(t *testing.T) {
	subRepo := &mocks.SubmissionRepo{}
	stationRepo := &mocks.StationRepo{}

	uc := submission.NewReviewSubmission(subRepo, stationRepo)
	err := uc.Execute(context.Background(), "1", domain.SubmissionStatusPending, "")
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Errorf("expected ErrInvalidInput, got %v", err)
	}
}

func TestReviewSubmission_NotFound(t *testing.T) {
	subRepo := &mocks.SubmissionRepo{
		UpdateStatusFunc: func(ctx context.Context, id string, status domain.SubmissionStatus, adminNote string) error {
			return domain.ErrNotFound
		},
	}
	stationRepo := &mocks.StationRepo{}

	uc := submission.NewReviewSubmission(subRepo, stationRepo)
	err := uc.Execute(context.Background(), "999", domain.SubmissionStatusApproved, "")
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

// The admin note limit counts characters, as the submission's own text does.
func TestReviewSubmission_AdminNoteLimit(t *testing.T) {
	subRepo := &mocks.SubmissionRepo{
		UpdateStatusFunc: func(ctx context.Context, id string, status domain.SubmissionStatus, adminNote string) error {
			return nil
		},
	}
	uc := submission.NewReviewSubmission(subRepo, &mocks.StationRepo{})

	if err := uc.Execute(context.Background(), "1", domain.SubmissionStatusApproved, strings.Repeat("ɛ", 1000)); err != nil {
		t.Fatalf("1000 characters: unexpected error: %v", err)
	}
	err := uc.Execute(context.Background(), "1", domain.SubmissionStatusApproved, strings.Repeat("a", 1001))
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Errorf("1001 characters: expected ErrInvalidInput, got %v", err)
	}
}
