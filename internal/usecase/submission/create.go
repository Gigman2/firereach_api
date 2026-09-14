package submission

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/firereach/api/internal/domain"
)

// maxTextLen bounds suggested_value, note and admin_note. It counts characters,
// not bytes, so a report written in Twi gets the same room as one in English;
// the app caps its report form to match.
const maxTextLen = 1000

// deviceHashPattern accepts what the app generates (d_<time>_<random>, base 36)
// with room for the format to change, and nothing that needs escaping in a log
// line. The hash is a rate-limit bucket key, not a credential.
var deviceHashPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{1,64}$`)

type CreateSubmission struct {
	repo domain.SubmissionRepository
}

func NewCreateSubmission(r domain.SubmissionRepository) *CreateSubmission {
	return &CreateSubmission{repo: r}
}

func (uc *CreateSubmission) Execute(ctx context.Context, s domain.Submission) error {
	if err := validate(s); err != nil {
		return fmt.Errorf("create submission: %w", err)
	}
	s.Status = domain.SubmissionStatusPending
	if err := uc.repo.Create(ctx, s); err != nil {
		return fmt.Errorf("create submission: %w", err)
	}
	return nil
}

func validate(s domain.Submission) error {
	switch {
	case !domain.ValidSubmissionType(s.Type):
		return invalid("type must be one of wrong_phone, wrong_location, missing, closed")
	case strings.TrimSpace(s.SuggestedValue) == "":
		return invalid("suggested_value is required")
	case utf8.RuneCountInString(s.SuggestedValue) > maxTextLen:
		return invalid("suggested_value must be at most 1000 characters")
	case utf8.RuneCountInString(s.Note) > maxTextLen:
		return invalid("note must be at most 1000 characters")
	case !deviceHashPattern.MatchString(s.DeviceHash):
		return invalid("device_hash must be 1 to 64 letters, digits, underscores or hyphens")
	case s.StationID != nil && !domain.ValidID(*s.StationID):
		return invalid("station_id must be a UUID")
	}
	return nil
}

// invalid wraps a message that tells the caller what to fix.
func invalid(msg string) error {
	return fmt.Errorf("%s: %w", msg, domain.ErrInvalidInput)
}
