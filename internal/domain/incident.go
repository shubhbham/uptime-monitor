package domain

import "time"

type Incident struct {
	ID         int64      `json:"id"`
	MonitorID  string     `json:"monitor_id"`
	StartedAt  time.Time  `json:"started_at"`
	ResolvedAt *time.Time `json:"resolved_at,omitempty"`
	Cause      *string    `json:"cause,omitempty"`
	IsResolved bool       `json:"is_resolved"`
}

type IncidentWithMonitor struct {
	Incident
	MonitorName string `json:"monitor_name"`
	MonitorURL  string `json:"monitor_url"`
}