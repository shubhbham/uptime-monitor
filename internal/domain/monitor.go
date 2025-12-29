package domain

import (
	"time"
)

type Monitor struct {
	ID              string    `json:"id"`
	UserID          *string   `json:"user_id,omitempty"`
	Name            string    `json:"name"`
	URL             string    `json:"url"`
	Method          string    `json:"method"`
	ExpectedStatus  int       `json:"expected_status"`
	IntervalSeconds int       `json:"interval_seconds"`
	TimeoutSeconds  int       `json:"timeout_seconds"`
	IsActive        bool      `json:"is_active"`
	UserEmail       string    `json:"-"` // Internal use only
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type MonitorCheck struct {
	ID             int64      `json:"id"`
	MonitorID      string     `json:"monitor_id"`
	StatusCode     *int       `json:"status_code,omitempty"`
	ResponseTimeMs *int       `json:"response_time_ms,omitempty"`
	IsUp           bool       `json:"is_up"`
	ErrorMessage   *string    `json:"error_message,omitempty"`
	CheckedAt      time.Time  `json:"checked_at"`
}

type CreateMonitorRequest struct {
	Name            string  `json:"name"`
	URL             string  `json:"url"`
	Method          string  `json:"method"`
	ExpectedStatus  int     `json:"expected_status"`
	IntervalSeconds int     `json:"interval_seconds"`
	TimeoutSeconds  int     `json:"timeout_seconds"`
	UserID          *string `json:"user_id,omitempty"`
}

type UpdateMonitorRequest struct {
	Name            *string `json:"name,omitempty"`
	URL             *string `json:"url,omitempty"`
	Method          *string `json:"method,omitempty"`
	ExpectedStatus  *int    `json:"expected_status,omitempty"`
	IntervalSeconds *int    `json:"interval_seconds,omitempty"`
	TimeoutSeconds  *int    `json:"timeout_seconds,omitempty"`
	IsActive        *bool   `json:"is_active,omitempty"`
}

type MonitorWithStats struct {
	Monitor
	Stats *MonitorStats `json:"stats,omitempty"`
}

type CheckResult struct {
	StatusCode     *int
	ResponseTimeMs *int
	IsUp           bool
	ErrorMessage   *string
}