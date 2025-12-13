package monitor

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

func (r *Repository) Create(ctx context.Context, req *domain.CreateMonitorRequest) (*domain.Monitor, error) {
	query := `
		INSERT INTO monitors (user_id, name, url, method, expected_status, interval_seconds, timeout_seconds)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, user_id, name, url, method, expected_status, interval_seconds, timeout_seconds, is_active, created_at, updated_at
	`

	var monitor domain.Monitor
	err := r.db.QueryRow(ctx, query,
		req.UserID, req.Name, req.URL, req.Method,
		req.ExpectedStatus, req.IntervalSeconds, req.TimeoutSeconds,
	).Scan(
		&monitor.ID, &monitor.UserID, &monitor.Name, &monitor.URL,
		&monitor.Method, &monitor.ExpectedStatus, &monitor.IntervalSeconds,
		&monitor.TimeoutSeconds, &monitor.IsActive, &monitor.CreatedAt, &monitor.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create monitor: %w", err)
	}

	return &monitor, nil
}

func (r *Repository) GetByID(ctx context.Context, id string) (*domain.Monitor, error) {
	query := `
		SELECT id, user_id, name, url, method, expected_status, interval_seconds, 
		       timeout_seconds, is_active, created_at, updated_at
		FROM monitors
		WHERE id = $1
	`

	var monitor domain.Monitor
	err := r.db.QueryRow(ctx, query, id).Scan(
		&monitor.ID, &monitor.UserID, &monitor.Name, &monitor.URL,
		&monitor.Method, &monitor.ExpectedStatus, &monitor.IntervalSeconds,
		&monitor.TimeoutSeconds, &monitor.IsActive, &monitor.CreatedAt, &monitor.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get monitor: %w", err)
	}

	return &monitor, nil
}

func (r *Repository) List(ctx context.Context) ([]*domain.Monitor, error) {
	query := `
		SELECT id, user_id, name, url, method, expected_status, interval_seconds,
		       timeout_seconds, is_active, created_at, updated_at
		FROM monitors
		ORDER BY created_at DESC
	`

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list monitors: %w", err)
	}
	defer rows.Close()

	var monitors []*domain.Monitor
	for rows.Next() {
		var monitor domain.Monitor
		err := rows.Scan(
			&monitor.ID, &monitor.UserID, &monitor.Name, &monitor.URL,
			&monitor.Method, &monitor.ExpectedStatus, &monitor.IntervalSeconds,
			&monitor.TimeoutSeconds, &monitor.IsActive, &monitor.CreatedAt, &monitor.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan monitor: %w", err)
		}
		monitors = append(monitors, &monitor)
	}

	return monitors, nil
}

func (r *Repository) Update(ctx context.Context, id string, req *domain.UpdateMonitorRequest) (*domain.Monitor, error) {
	query := `
		UPDATE monitors
		SET name = COALESCE($2, name),
		    url = COALESCE($3, url),
		    method = COALESCE($4, method),
		    expected_status = COALESCE($5, expected_status),
		    interval_seconds = COALESCE($6, interval_seconds),
		    timeout_seconds = COALESCE($7, timeout_seconds),
		    is_active = COALESCE($8, is_active),
		    updated_at = NOW()
		WHERE id = $1
		RETURNING id, user_id, name, url, method, expected_status, interval_seconds,
		          timeout_seconds, is_active, created_at, updated_at
	`

	var monitor domain.Monitor
	err := r.db.QueryRow(ctx, query,
		id, req.Name, req.URL, req.Method, req.ExpectedStatus,
		req.IntervalSeconds, req.TimeoutSeconds, req.IsActive,
	).Scan(
		&monitor.ID, &monitor.UserID, &monitor.Name, &monitor.URL,
		&monitor.Method, &monitor.ExpectedStatus, &monitor.IntervalSeconds,
		&monitor.TimeoutSeconds, &monitor.IsActive, &monitor.CreatedAt, &monitor.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to update monitor: %w", err)
	}

	return &monitor, nil
}

func (r *Repository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM monitors WHERE id = $1`
	result, err := r.db.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete monitor: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("monitor not found")
	}

	return nil
}

func (r *Repository) GetActiveMonitors(ctx context.Context) ([]*domain.Monitor, error) {
	query := `
		SELECT id, user_id, name, url, method, expected_status, interval_seconds,
		       timeout_seconds, is_active, created_at, updated_at
		FROM monitors
		WHERE is_active = true
		ORDER BY created_at DESC
	`

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get active monitors: %w", err)
	}
	defer rows.Close()

	var monitors []*domain.Monitor
	for rows.Next() {
		var monitor domain.Monitor
		err := rows.Scan(
			&monitor.ID, &monitor.UserID, &monitor.Name, &monitor.URL,
			&monitor.Method, &monitor.ExpectedStatus, &monitor.IntervalSeconds,
			&monitor.TimeoutSeconds, &monitor.IsActive, &monitor.CreatedAt, &monitor.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan monitor: %w", err)
		}
		monitors = append(monitors, &monitor)
	}

	return monitors, nil
}

func (r *Repository) SaveCheck(ctx context.Context, check *domain.MonitorCheck) error {
	query := `
		INSERT INTO monitor_checks (monitor_id, status_code, response_time_ms, is_up, error_message, checked_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`

	_, err := r.db.Exec(ctx, query,
		check.MonitorID, check.StatusCode, check.ResponseTimeMs,
		check.IsUp, check.ErrorMessage, check.CheckedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to save check: %w", err)
	}

	return nil
}

func (r *Repository) GetRecentChecks(ctx context.Context, monitorID string, limit int) ([]*domain.MonitorCheck, error) {
	query := `
		SELECT id, monitor_id, status_code, response_time_ms, is_up, error_message, checked_at
		FROM monitor_checks
		WHERE monitor_id = $1
		ORDER BY checked_at DESC
		LIMIT $2
	`

	rows, err := r.db.Query(ctx, query, monitorID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get recent checks: %w", err)
	}
	defer rows.Close()

	var checks []*domain.MonitorCheck
	for rows.Next() {
		var check domain.MonitorCheck
		err := rows.Scan(
			&check.ID, &check.MonitorID, &check.StatusCode,
			&check.ResponseTimeMs, &check.IsUp, &check.ErrorMessage, &check.CheckedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan check: %w", err)
		}
		checks = append(checks, &check)
	}

	return checks, nil
}

func (r *Repository) GetChecksSince(ctx context.Context, monitorID string, since time.Time) ([]*domain.MonitorCheck, error) {
	query := `
		SELECT id, monitor_id, status_code, response_time_ms, is_up, error_message, checked_at
		FROM monitor_checks
		WHERE monitor_id = $1 AND checked_at >= $2
		ORDER BY checked_at DESC
	`

	rows, err := r.db.Query(ctx, query, monitorID, since)
	if err != nil {
		return nil, fmt.Errorf("failed to get checks since: %w", err)
	}
	defer rows.Close()

	var checks []*domain.MonitorCheck
	for rows.Next() {
		var check domain.MonitorCheck
		err := rows.Scan(
			&check.ID, &check.MonitorID, &check.StatusCode,
			&check.ResponseTimeMs, &check.IsUp, &check.ErrorMessage, &check.CheckedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan check: %w", err)
		}
		checks = append(checks, &check)
	}

	return checks, nil
}