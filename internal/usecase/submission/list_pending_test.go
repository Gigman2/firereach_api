package submission_test

import (
	"context"
	"errors"
	"testing"

	"github.com/firereach/api/internal/domain"
	"github.com/firereach/api/internal/usecase/mocks"
	"github.com/firereach/api/internal/usecase/submission"
)

func TestListPending_Success(t *testing.T) {
	repo := &mocks.SubmissionRepo{
		ListPendingFunc: func(ctx context.Context) ([]domain.Submission, error) {
			return []domain.Submission{
				{ID: "1", Status: domain.SubmissionStatusPending},
				{ID: "2", Status: domain.SubmissionStatusPending},
			}, nil
		},
	}

	uc := submission.NewListPending(repo)
	subs, err := uc.Execute(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(subs) != 2 {
		t.Fatalf("expected 2 submissions, got %d", len(subs))
	}
}

func TestListPending_RepoError(t *testing.T) {
	repo := &mocks.SubmissionRepo{
		ListPendingFunc: func(ctx context.Context) ([]domain.Submission, error) {
			return nil, errors.New("db error")
		},
	}

	uc := submission.NewListPending(repo)
	_, err := uc.Execute(context.Background())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
