package domain

import "time"

type MonitorStats struct {
	MonitorID           string     `json:"monitor_id"`
	UptimePercentage    float64    `json:"uptime_percentage"`
	AvgResponseTimeMs   *int       `json:"avg_response_time_ms,omitempty"`
	LastCheckedAt       *time.Time `json:"last_checked_at,omitempty"`
	LastStatus          *bool      `json:"last_status,omitempty"`
}

type UptimeStats struct {
	Last24Hours float64 `json:"last_24_hours"`
	Last7Days   float64 `json:"last_7_days"`
	Last30Days  float64 `json:"last_30_days"`
}