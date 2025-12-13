package metrics

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shubhbham/uptime-monitor/internal/domain"
)

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

func (r *Repository) UpsertStats(ctx context.Context, stats *domain.MonitorStats) error {
	query := `
		INSERT INTO monitor_stats (monitor_id, uptime_percentage, avg_response_time_ms, last_checked_at, last_status)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (monitor_id)
		DO UPDATE SET
			uptime_percentage = EXCLUDED.uptime_percentage,
			avg_response_time_ms = EXCLUDED.avg_response_time_ms,
			last_checked_at = EXCLUDED.last_checked_at,
			last_status = EXCLUDED.last_status
	`

	_, err := r.db.Exec(ctx, query,
		stats.MonitorID, stats.UptimePercentage, stats.AvgResponseTimeMs,
		stats.LastCheckedAt, stats.LastStatus,
	)

	if err != nil {
		return fmt.Errorf("failed to upsert stats: %w", err)
	}

	return nil
}

func (r *Repository) GetStatsByMonitor(ctx context.Context, monitorID string) (*domain.MonitorStats, error) {
	query := `
		SELECT monitor_id, uptime_percentage, avg_response_time_ms, last_checked_at, last_status
		FROM monitor_stats
		WHERE monitor_id = $1
	`

	var stats domain.MonitorStats
	err := r.db.QueryRow(ctx, query, monitorID).Scan(
		&stats.MonitorID, &stats.UptimePercentage, &stats.AvgResponseTimeMs,
		&stats.LastCheckedAt, &stats.LastStatus,
	)

	if err != nil {
		return nil, err
	}

	return &stats, nil
}

func (r *Repository) GetAllStats(ctx context.Context) ([]*domain.MonitorStats, error) {
	query := `
		SELECT monitor_id, uptime_percentage, avg_response_time_ms, last_checked_at, last_status
		FROM monitor_stats
	`

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get all stats: %w", err)
	}
	defer rows.Close()

	var statsList []*domain.MonitorStats
	for rows.Next() {
		var stats domain.MonitorStats
		err := rows.Scan(
			&stats.MonitorID, &stats.UptimePercentage, &stats.AvgResponseTimeMs,
			&stats.LastCheckedAt, &stats.LastStatus,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan stats: %w", err)
		}
		statsList = append(statsList, &stats)
	}

	return statsList, nil
}

func (r *Repository) DeleteStats(ctx context.Context, monitorID string) error {
	query := `DELETE FROM monitor_stats WHERE monitor_id = $1`
	_, err := r.db.Exec(ctx, query, monitorID)
	return err
}