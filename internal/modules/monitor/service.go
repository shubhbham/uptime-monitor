package monitor

import (
	"context"

	"github.com/shubhbham/uptime-monitor/internal/domain"
)

type SchedulerManager interface {
	AddMonitor(monitor *domain.Monitor) error
	RemoveMonitor(monitorID string)
	UpdateMonitor(monitor *domain.Monitor) error
}

type Service struct {
	repo      *Repository
	scheduler SchedulerManager
}

func NewService(repo *Repository) *Service {
	return &Service{
		repo:      repo,
		scheduler: nil,
	}
}

func (s *Service) SetScheduler(scheduler SchedulerManager) {
	s.scheduler = scheduler
}

func (s *Service) CreateMonitor(ctx context.Context, userID string, req *domain.CreateMonitorRequest) (*domain.Monitor, error) {
	if err := ValidateCreateRequest(req); err != nil {
		return nil, err
	}

	// Set the user_id from authenticated context
	req.UserID = &userID

	monitor, err := s.repo.Create(ctx, req)
	if err != nil {
		return nil, err
	}

	if s.scheduler != nil && monitor.IsActive {
		if err := s.scheduler.AddMonitor(monitor); err != nil {
			// Log but don't fail
		}
	}

	return monitor, nil
}

func (s *Service) GetMonitor(ctx context.Context, id, userID string) (*domain.Monitor, error) {
	return s.repo.GetByIDForUser(ctx, id, userID)
}

func (s *Service) ListMonitors(ctx context.Context, userID string) ([]*domain.Monitor, error) {
	return s.repo.ListForUser(ctx, userID)
}

func (s *Service) UpdateMonitor(ctx context.Context, id, userID string, req *domain.UpdateMonitorRequest) (*domain.Monitor, error) {
	if err := ValidateUpdateRequest(req); err != nil {
		return nil, err
	}

	monitor, err := s.repo.UpdateForUser(ctx, id, userID, req)
	if err != nil {
		return nil, err
	}

	if s.scheduler != nil {
		if monitor.IsActive {
			if err := s.scheduler.UpdateMonitor(monitor); err != nil {
				// Log but don't fail
			}
		} else {
			s.scheduler.RemoveMonitor(monitor.ID)
		}
	}

	return monitor, nil
}

func (s *Service) DeleteMonitor(ctx context.Context, id, userID string) error {
	if s.scheduler != nil {
		s.scheduler.RemoveMonitor(id)
	}

	return s.repo.DeleteForUser(ctx, id, userID)
}

func (s *Service) GetActiveMonitors(ctx context.Context) ([]*domain.Monitor, error) {
	return s.repo.GetActiveMonitors(ctx)
}

func (s *Service) GetRecentChecks(ctx context.Context, monitorID string, limit int) ([]*domain.MonitorCheck, error) {
	return s.repo.GetRecentChecks(ctx, monitorID, limit)
}