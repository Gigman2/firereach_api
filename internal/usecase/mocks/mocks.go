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
	AskFunc func(ctx context.Context, question, topic string, history []domain.Turn) (domain.AIResponse, error)
}

func (m *AIGateway) Ask(ctx context.Context, question, topic string, history []domain.Turn) (domain.AIResponse, error) {
	return m.AskFunc(ctx, question, topic, history)
}

// AdminRepo is a mock implementation of domain.AdminRepository.
type AdminRepo struct {
	GetByEmailFunc  func(ctx context.Context, email string) (*domain.AdminUser, error)
	CreateFunc      func(ctx context.Context, email, passwordHash string) (*domain.AdminUser, error)
	ListFunc        func(ctx context.Context) ([]domain.AdminUser, error)
	CountFunc       func(ctx context.Context) (int, error)
	CreateFirstFunc func(ctx context.Context, email, passwordHash string) (*domain.AdminUser, error)
}

func (m *AdminRepo) GetByEmail(ctx context.Context, email string) (*domain.AdminUser, error) {
	return m.GetByEmailFunc(ctx, email)
}
func (m *AdminRepo) Create(ctx context.Context, email, passwordHash string) (*domain.AdminUser, error) {
	return m.CreateFunc(ctx, email, passwordHash)
}
func (m *AdminRepo) List(ctx context.Context) ([]domain.AdminUser, error) {
	return m.ListFunc(ctx)
}
func (m *AdminRepo) Count(ctx context.Context) (int, error) {
	return m.CountFunc(ctx)
}
func (m *AdminRepo) CreateFirst(ctx context.Context, email, passwordHash string) (*domain.AdminUser, error) {
	return m.CreateFirstFunc(ctx, email, passwordHash)
}
