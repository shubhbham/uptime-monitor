package incident

import (
	"context"
	"time"

	"github.com/shubhbham/uptime-monitor/internal/domain"
)

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) CreateIncident(ctx context.Context, monitorID string, cause *string) (*domain.Incident, error) {
	return s.repo.Create(ctx, monitorID, cause)
}

func (s *Service) ResolveIncident(ctx context.Context, id int64) error {
	return s.repo.Resolve(ctx, id)
}

func (s *Service) GetOpenIncident(ctx context.Context, monitorID string) (*domain.Incident, error) {
	return s.repo.GetOpenIncident(ctx, monitorID)
}

func (s *Service) ListIncidentsByMonitor(ctx context.Context, monitorID string, limit int) ([]*domain.Incident, error) {
	return s.repo.ListByMonitor(ctx, monitorID, limit)
}

func (s *Service) ListAllIncidents(ctx context.Context, limit int) ([]*domain.IncidentWithMonitor, error) {
	return s.repo.ListAll(ctx, limit)
}

func (s *Service) GetRecentIncidents(ctx context.Context, since time.Time) ([]*domain.IncidentWithMonitor, error) {
	return s.repo.GetRecentIncidents(ctx, since)
}