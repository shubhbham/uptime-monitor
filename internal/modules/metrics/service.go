package metrics

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shubhbham/uptime-monitor/internal/domain"
	"github.com/shubhbham/uptime-monitor/internal/utils"
)

type Service struct {
	repo *Repository
	db   *pgxpool.Pool
}

func NewService(repo *Repository, db *pgxpool.Pool) *Service {
	return &Service{
		repo: repo,
		db:   db,
	}
}

func (s *Service) GetStats(ctx context.Context, monitorID string) (*domain.MonitorStats, error) {
	return s.repo.GetStatsByMonitor(ctx, monitorID)
}

func (s *Service) GetAllStats(ctx context.Context) ([]*domain.MonitorStats, error) {
	return s.repo.GetAllStats(ctx)
}

func (s *Service) UpdateStats(ctx context.Context, monitorID string) error {
	now := time.Now()
	last24h := now.Add(-24 * time.Hour)

	query := `
		SELECT status_code, response_time_ms, is_up, checked_at
		FROM monitor_checks
		WHERE monitor_id = $1 AND checked_at >= $2
		ORDER BY checked_at DESC
	`

	rows, err := s.db.Query(ctx, query, monitorID, last24h)
	if err != nil {
		return err
	}
	defer rows.Close()

	var checks []*domain.MonitorCheck
	for rows.Next() {
		var check domain.MonitorCheck
		check.MonitorID = monitorID
		err := rows.Scan(&check.StatusCode, &check.ResponseTimeMs, &check.IsUp, &check.CheckedAt)
		if err != nil {
			return err
		}
		checks = append(checks, &check)
	}

	if len(checks) == 0 {
		return nil
	}

	uptime := utils.CalculateUptime(checks)
	
	var totalResponseTime int
	var responseCount int
	for _, check := range checks {
		if check.ResponseTimeMs != nil {
			totalResponseTime += *check.ResponseTimeMs
			responseCount++
		}
	}

	var avgResponseTime *int
	if responseCount > 0 {
		avg := totalResponseTime / responseCount
		avgResponseTime = &avg
	}

	lastCheck := checks[0]
	stats := &domain.MonitorStats{
		MonitorID:         monitorID,
		UptimePercentage:  uptime,
		AvgResponseTimeMs: avgResponseTime,
		LastCheckedAt:     &lastCheck.CheckedAt,
		LastStatus:        &lastCheck.IsUp,
	}

	return s.repo.UpsertStats(ctx, stats)
}

func (s *Service) GetUptimeStats(ctx context.Context, monitorID string) (*domain.UptimeStats, error) {
	now := time.Now()

	uptime24h, err := s.calculateUptimeForPeriod(ctx, monitorID, now.Add(-24*time.Hour))
	if err != nil {
		return nil, err
	}

	uptime7d, err := s.calculateUptimeForPeriod(ctx, monitorID, now.Add(-7*24*time.Hour))
	if err != nil {
		return nil, err
	}

	uptime30d, err := s.calculateUptimeForPeriod(ctx, monitorID, now.Add(-30*24*time.Hour))
	if err != nil {
		return nil, err
	}

	return &domain.UptimeStats{
		Last24Hours: uptime24h,
		Last7Days:   uptime7d,
		Last30Days:  uptime30d,
	}, nil
}

func (s *Service) calculateUptimeForPeriod(ctx context.Context, monitorID string, since time.Time) (float64, error) {
	query := `
		SELECT is_up
		FROM monitor_checks
		WHERE monitor_id = $1 AND checked_at >= $2
		ORDER BY checked_at DESC
	`

	rows, err := s.db.Query(ctx, query, monitorID, since)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	var checks []*domain.MonitorCheck
	for rows.Next() {
		var check domain.MonitorCheck
		err := rows.Scan(&check.IsUp)
		if err != nil {
			return 0, err
		}
		checks = append(checks, &check)
	}

	if len(checks) == 0 {
		return 100.0, nil
	}

	return utils.CalculateUptime(checks), nil
}