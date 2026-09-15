package domain

import (
	"context"
	"time"
)

type SubmissionStatus string

const (
	SubmissionStatusPending  SubmissionStatus = "pending"
	SubmissionStatusApproved SubmissionStatus = "approved"
	SubmissionStatusRejected SubmissionStatus = "rejected"
)

// What a submission reports. These strings are the wire contract with the app
// (SubmissionType in app/src/lib/submissionsApi.ts) and are stored verbatim, so
// renaming one changes what admins see in the review queue.
const (
	SubmissionWrongPhone    = "wrong_phone"
	SubmissionWrongLocation = "wrong_location"
	SubmissionMissing       = "missing"
	SubmissionClosed        = "closed"
)

// ValidSubmissionType reports whether t is one of the types above.
func ValidSubmissionType(t string) bool {
	switch t {
	case SubmissionWrongPhone, SubmissionWrongLocation, SubmissionMissing, SubmissionClosed:
		return true
	}
	return false
}

type Submission struct {
	ID             string
	StationID      *string
	Type           string
	SuggestedValue string
	Note           string
	DeviceHash     string
	Status         SubmissionStatus
	SubmittedAt    time.Time
	ReviewedAt     *time.Time
	AdminNote      string
}

type SubmissionRepository interface {
	Create(ctx context.Context, s Submission) error
	ListPending(ctx context.Context) ([]Submission, error)
	UpdateStatus(ctx context.Context, id string, status SubmissionStatus, adminNote string) error
}
