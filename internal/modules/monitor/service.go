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
		scheduler: nil, // Will be set by SetScheduler
	}
}

func (s *Service) SetScheduler(scheduler SchedulerManager) {
	s.scheduler = scheduler
}

func (s *Service) CreateMonitor(ctx context.Context, req *domain.CreateMonitorRequest) (*domain.Monitor, error) {
	if err := ValidateCreateRequest(req); err != nil {
		return nil, err
	}

	monitor, err := s.repo.Create(ctx, req)
	if err != nil {
		return nil, err
	}

	// Immediately schedule the monitor if scheduler is available
	if s.scheduler != nil && monitor.IsActive {
		if err := s.scheduler.AddMonitor(monitor); err != nil {
			// Log error but don't fail the creation
			// The monitor will be picked up on next restart
			// In production, you might want to implement retry logic
		}
	}

	return monitor, nil
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

	monitor, err := s.repo.Update(ctx, id, req)
	if err != nil {
		return nil, err
	}

	// Update scheduler if available
	if s.scheduler != nil {
		if monitor.IsActive {
			// If active, update the schedule
			if err := s.scheduler.UpdateMonitor(monitor); err != nil {
				// Log error but don't fail the update
			}
		} else {
			// If inactive, remove from scheduler
			s.scheduler.RemoveMonitor(monitor.ID)
		}
	}

	return monitor, nil
}

func (s *Service) DeleteMonitor(ctx context.Context, id string) error {
	// Remove from scheduler first (before database deletion)
	if s.scheduler != nil {
		s.scheduler.RemoveMonitor(id)
	}

	// Then delete from database
	return s.repo.Delete(ctx, id)
}

func (s *Service) GetActiveMonitors(ctx context.Context) ([]*domain.Monitor, error) {
	return s.repo.GetActiveMonitors(ctx)
}

func (s *Service) GetRecentChecks(ctx context.Context, monitorID string, limit int) ([]*domain.MonitorCheck, error) {
	return s.repo.GetRecentChecks(ctx, monitorID, limit)
}