package submission_test

import (
	"context"
	"errors"
	"testing"

	"github.com/firereach/api/internal/domain"
	"github.com/firereach/api/internal/usecase/mocks"
	"github.com/firereach/api/internal/usecase/submission"
)

func TestCreateSubmission_Success(t *testing.T) {
	var saved domain.Submission
	repo := &mocks.SubmissionRepo{
		CreateFunc: func(ctx context.Context, s domain.Submission) error {
			saved = s
			return nil
		},
	}

	uc := submission.NewCreateSubmission(repo)
	err := uc.Execute(context.Background(), domain.Submission{
		Type:           "phone_correction",
		SuggestedValue: "+233201234567",
		DeviceHash:     "abc123",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if saved.Status != domain.SubmissionStatusPending {
		t.Errorf("expected status 'pending', got '%s'", saved.Status)
	}
}

func TestCreateSubmission_MissingType(t *testing.T) {
	repo := &mocks.SubmissionRepo{}

	uc := submission.NewCreateSubmission(repo)
	err := uc.Execute(context.Background(), domain.Submission{
		SuggestedValue: "+233201234567",
	})
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Errorf("expected ErrInvalidInput, got %v", err)
	}
}

func TestCreateSubmission_MissingSuggestedValue(t *testing.T) {
	repo := &mocks.SubmissionRepo{}

	uc := submission.NewCreateSubmission(repo)
	err := uc.Execute(context.Background(), domain.Submission{
		Type: "phone_correction",
	})
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Errorf("expected ErrInvalidInput, got %v", err)
	}
}

func TestCreateSubmission_RepoError(t *testing.T) {
	repo := &mocks.SubmissionRepo{
		CreateFunc: func(ctx context.Context, s domain.Submission) error {
			return errors.New("db error")
		},
	}

	uc := submission.NewCreateSubmission(repo)
	err := uc.Execute(context.Background(), domain.Submission{
		Type:           "phone_correction",
		SuggestedValue: "+233201234567",
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
