package monitor

import (
	"context"

	"github.com/shubhbham/uptime-monitor/internal/domain"
)

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) CreateMonitor(ctx context.Context, req *domain.CreateMonitorRequest) (*domain.Monitor, error) {
	if err := ValidateCreateRequest(req); err != nil {
		return nil, err
	}

	return s.repo.Create(ctx, req)
}

func (s *Service) GetMonitor(ctx context.Context, id string) (*domain.Monitor, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *Service) ListMonitors(ctx context.Context) ([]*domain.Monitor, error) {
	return s.repo.List(ctx)
}

func (s *Service) UpdateMonitor(ctx context.Context, id string, req *domain.UpdateMonitorRequest) (*domain.Monitor, error) {
	if err := ValidateUpdateRequest(req); err != nil {
		return nil, err
	}

	return s.repo.Update(ctx, id, req)
}

func (s *Service) DeleteMonitor(ctx context.Context, id string) error {
	return s.repo.Delete(ctx, id)
}

func (s *Service) GetActiveMonitors(ctx context.Context) ([]*domain.Monitor, error) {
	return s.repo.GetActiveMonitors(ctx)
}

func (s *Service) GetRecentChecks(ctx context.Context, monitorID string, limit int) ([]*domain.MonitorCheck, error) {
	return s.repo.GetRecentChecks(ctx, monitorID, limit)
}