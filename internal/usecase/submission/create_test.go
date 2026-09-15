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

// validSubmission is a report the app could really send. Each test changes one
// field, so a rejection can only come from that field.
func validSubmission() domain.Submission {
	return domain.Submission{
		Type:           "wrong_phone",
		SuggestedValue: "0302999999",
		DeviceHash:     "d_m4k2p1_8f3kq9z2x1",
	}
}

func acceptingRepo(saved *domain.Submission) *mocks.SubmissionRepo {
	return &mocks.SubmissionRepo{
		CreateFunc: func(ctx context.Context, s domain.Submission) error {
			if saved != nil {
				*saved = s
			}
			return nil
		},
	}
}

func TestCreateSubmission_Success(t *testing.T) {
	var saved domain.Submission
	err := submission.NewCreateSubmission(acceptingRepo(&saved)).Execute(context.Background(), validSubmission())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if saved.Status != domain.SubmissionStatusPending {
		t.Errorf("expected status 'pending', got '%s'", saved.Status)
	}
}

// The four types the app sends (SubmissionType in app/src/lib/submissionsApi.ts).
func TestCreateSubmission_AcceptsEveryTypeTheAppSends(t *testing.T) {
	for _, typ := range []string{"wrong_phone", "wrong_location", "missing", "closed"} {
		s := validSubmission()
		s.Type = typ
		if err := submission.NewCreateSubmission(acceptingRepo(nil)).Execute(context.Background(), s); err != nil {
			t.Errorf("type %q: unexpected error: %v", typ, err)
		}
	}
}

func TestCreateSubmission_RejectsInvalidFields(t *testing.T) {
	notAUUID := "station-1"
	empty := ""
	tests := []struct {
		name   string
		change func(*domain.Submission)
	}{
		{"missing type", func(s *domain.Submission) { s.Type = "" }},
		{"unknown type", func(s *domain.Submission) { s.Type = "phone_correction" }},
		{"missing suggested value", func(s *domain.Submission) { s.SuggestedValue = "" }},
		{"blank suggested value", func(s *domain.Submission) { s.SuggestedValue = "   " }},
		{"suggested value over 1000 characters", func(s *domain.Submission) { s.SuggestedValue = strings.Repeat("a", 1001) }},
		{"note over 1000 characters", func(s *domain.Submission) { s.Note = strings.Repeat("a", 1001) }},
		{"missing device hash", func(s *domain.Submission) { s.DeviceHash = "" }},
		{"device hash over 64 characters", func(s *domain.Submission) { s.DeviceHash = strings.Repeat("a", 65) }},
		{"device hash with a space", func(s *domain.Submission) { s.DeviceHash = "d_abc def" }},
		{"station id that is not a UUID", func(s *domain.Submission) { s.StationID = &notAUUID }},
		{"empty station id", func(s *domain.Submission) { s.StationID = &empty }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := validSubmission()
			tt.change(&s)
			repo := &mocks.SubmissionRepo{
				CreateFunc: func(ctx context.Context, _ domain.Submission) error {
					t.Error("an invalid submission reached the repository")
					return nil
				},
			}
			err := submission.NewCreateSubmission(repo).Execute(context.Background(), s)
			if !errors.Is(err, domain.ErrInvalidInput) {
				t.Errorf("expected ErrInvalidInput, got %v", err)
			}
		})
	}
}

// The text limits count characters, not bytes. "ɛ" and "ɔ" are Twi letters of
// two bytes each, and a 1,000-character note written in Twi must not be refused.
func TestCreateSubmission_TextLimitsCountCharacters(t *testing.T) {
	stationID := "5a0c5e1e-0000-4000-8000-000000000001"
	s := validSubmission()
	s.Type = "missing"
	s.SuggestedValue = strings.Repeat("ɛ", 1000)
	s.Note = strings.Repeat("ɔ", 1000)
	s.StationID = &stationID
	if err := submission.NewCreateSubmission(acceptingRepo(nil)).Execute(context.Background(), s); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCreateSubmission_RepoError(t *testing.T) {
	repo := &mocks.SubmissionRepo{
		CreateFunc: func(ctx context.Context, s domain.Submission) error {
			return errors.New("db error")
		},
	}
	err := submission.NewCreateSubmission(repo).Execute(context.Background(), validSubmission())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
