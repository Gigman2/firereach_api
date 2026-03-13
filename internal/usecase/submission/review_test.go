package submission_test

import (
	"context"
	"errors"
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
