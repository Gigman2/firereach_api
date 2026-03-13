package mocks

import (
	"context"

	"github.com/firereach/api/internal/domain"
)

// StationRepo is a mock implementation of domain.StationRepository.
type StationRepo struct {
	ListActiveFunc func(ctx context.Context) ([]domain.Station, error)
	GetByIDFunc    func(ctx context.Context, id string) (*domain.Station, error)
	CreateFunc     func(ctx context.Context, s domain.Station) error
	UpdateFunc     func(ctx context.Context, s domain.Station) error
}

func (m *StationRepo) ListActive(ctx context.Context) ([]domain.Station, error) {
	return m.ListActiveFunc(ctx)
}
func (m *StationRepo) GetByID(ctx context.Context, id string) (*domain.Station, error) {
	return m.GetByIDFunc(ctx, id)
}
func (m *StationRepo) Create(ctx context.Context, s domain.Station) error {
	return m.CreateFunc(ctx, s)
}
func (m *StationRepo) Update(ctx context.Context, s domain.Station) error {
	return m.UpdateFunc(ctx, s)
}

// SubmissionRepo is a mock implementation of domain.SubmissionRepository.
type SubmissionRepo struct {
	CreateFunc       func(ctx context.Context, s domain.Submission) error
	ListPendingFunc  func(ctx context.Context) ([]domain.Submission, error)
	UpdateStatusFunc func(ctx context.Context, id string, status domain.SubmissionStatus, adminNote string) error
}

func (m *SubmissionRepo) Create(ctx context.Context, s domain.Submission) error {
	return m.CreateFunc(ctx, s)
}
func (m *SubmissionRepo) ListPending(ctx context.Context) ([]domain.Submission, error) {
	return m.ListPendingFunc(ctx)
}
func (m *SubmissionRepo) UpdateStatus(ctx context.Context, id string, status domain.SubmissionStatus, adminNote string) error {
	return m.UpdateStatusFunc(ctx, id, status, adminNote)
}

// ContentRepo is a mock implementation of domain.ContentRepository.
type ContentRepo struct {
	ListFunc    func(ctx context.Context, category, subcategory string) ([]domain.SafetyContent, error)
	GetByIDFunc func(ctx context.Context, id string) (*domain.SafetyContent, error)
}

func (m *ContentRepo) List(ctx context.Context, category, subcategory string) ([]domain.SafetyContent, error) {
	return m.ListFunc(ctx, category, subcategory)
}
func (m *ContentRepo) GetByID(ctx context.Context, id string) (*domain.SafetyContent, error) {
	return m.GetByIDFunc(ctx, id)
}

// AIGateway is a mock implementation of domain.AIGateway.
type AIGateway struct {
	AskFunc func(ctx context.Context, question, topic string) (string, error)
}

func (m *AIGateway) Ask(ctx context.Context, question, topic string) (string, error) {
	return m.AskFunc(ctx, question, topic)
}
