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
