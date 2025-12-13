package incident

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shubhbham/uptime-monitor/internal/domain"
)

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Create(ctx context.Context, monitorID string, cause *string) (*domain.Incident, error) {
	query := `
		INSERT INTO incidents (monitor_id, started_at, cause)
		VALUES ($1, NOW(), $2)
		RETURNING id, monitor_id, started_at, resolved_at, cause, is_resolved
	`

	var incident domain.Incident
	err := r.db.QueryRow(ctx, query, monitorID, cause).Scan(
		&incident.ID, &incident.MonitorID, &incident.StartedAt,
		&incident.ResolvedAt, &incident.Cause, &incident.IsResolved,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create incident: %w", err)
	}

	return &incident, nil
}

func (r *Repository) Resolve(ctx context.Context, id int64) error {
	query := `
		UPDATE incidents
		SET resolved_at = NOW(), is_resolved = true
		WHERE id = $1 AND is_resolved = false
	`

	result, err := r.db.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to resolve incident: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("incident not found or already resolved")
	}

	return nil
}

func (r *Repository) GetOpenIncident(ctx context.Context, monitorID string) (*domain.Incident, error) {
	query := `
		SELECT id, monitor_id, started_at, resolved_at, cause, is_resolved
		FROM incidents
		WHERE monitor_id = $1 AND is_resolved = false
		ORDER BY started_at DESC
		LIMIT 1
	`

	var incident domain.Incident
	err := r.db.QueryRow(ctx, query, monitorID).Scan(
		&incident.ID, &incident.MonitorID, &incident.StartedAt,
		&incident.ResolvedAt, &incident.Cause, &incident.IsResolved,
	)

	if err != nil {
		return nil, err
	}

	return &incident, nil
}

func (r *Repository) ListByMonitor(ctx context.Context, monitorID string, limit int) ([]*domain.Incident, error) {
	query := `
		SELECT id, monitor_id, started_at, resolved_at, cause, is_resolved
		FROM incidents
		WHERE monitor_id = $1
		ORDER BY started_at DESC
		LIMIT $2
	`

	rows, err := r.db.Query(ctx, query, monitorID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to list incidents: %w", err)
	}
	defer rows.Close()

	var incidents []*domain.Incident
	for rows.Next() {
		var incident domain.Incident
		err := rows.Scan(
			&incident.ID, &incident.MonitorID, &incident.StartedAt,
			&incident.ResolvedAt, &incident.Cause, &incident.IsResolved,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan incident: %w", err)
		}
		incidents = append(incidents, &incident)
	}

	return incidents, nil
}

func (r *Repository) ListAll(ctx context.Context, limit int) ([]*domain.IncidentWithMonitor, error) {
	query := `
		SELECT i.id, i.monitor_id, i.started_at, i.resolved_at, i.cause, i.is_resolved,
		       m.name, m.url
		FROM incidents i
		JOIN monitors m ON i.monitor_id = m.id
		ORDER BY i.started_at DESC
		LIMIT $1
	`

	rows, err := r.db.Query(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to list all incidents: %w", err)
	}
	defer rows.Close()

	var incidents []*domain.IncidentWithMonitor
	for rows.Next() {
		var incident domain.IncidentWithMonitor
		err := rows.Scan(
			&incident.ID, &incident.MonitorID, &incident.StartedAt,
			&incident.ResolvedAt, &incident.Cause, &incident.IsResolved,
			&incident.MonitorName, &incident.MonitorURL,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan incident: %w", err)
		}
		incidents = append(incidents, &incident)
	}

	return incidents, nil
}

func (r *Repository) GetRecentIncidents(ctx context.Context, since time.Time) ([]*domain.IncidentWithMonitor, error) {
	query := `
		SELECT i.id, i.monitor_id, i.started_at, i.resolved_at, i.cause, i.is_resolved,
		       m.name, m.url
		FROM incidents i
		JOIN monitors m ON i.monitor_id = m.id
		WHERE i.started_at >= $1
		ORDER BY i.started_at DESC
	`

	rows, err := r.db.Query(ctx, query, since)
	if err != nil {
		return nil, fmt.Errorf("failed to get recent incidents: %w", err)
	}
	defer rows.Close()

	var incidents []*domain.IncidentWithMonitor
	for rows.Next() {
		var incident domain.IncidentWithMonitor
		err := rows.Scan(
			&incident.ID, &incident.MonitorID, &incident.StartedAt,
			&incident.ResolvedAt, &incident.Cause, &incident.IsResolved,
			&incident.MonitorName, &incident.MonitorURL,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan incident: %w", err)
		}
		incidents = append(incidents, &incident)
	}

	return incidents, nil
}