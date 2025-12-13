package utils

import (
	"math"

	"github.com/shubhbham/uptime-monitor/internal/domain"
)

func CalculateUptime(checks []*domain.MonitorCheck) float64 {
	if len(checks) == 0 {
		return 100.0
	}

	upCount := 0
	for _, check := range checks {
		if check.IsUp {
			upCount++
		}
	}

	uptime := (float64(upCount) / float64(len(checks))) * 100.0
	return math.Round(uptime*100) / 100
}

func CalculateAverageResponseTime(checks []*domain.MonitorCheck) *int {
	var total int
	var count int

	for _, check := range checks {
		if check.ResponseTimeMs != nil {
			total += *check.ResponseTimeMs
			count++
		}
	}

	if count == 0 {
		return nil
	}

	avg := total / count
	return &avg
}