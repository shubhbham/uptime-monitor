package scheduler

import (
	"context"
	"log"
	"time"

	"github.com/shubhbham/uptime-monitor/internal/domain"
	"github.com/shubhbham/uptime-monitor/internal/worker"
)

type MonitorCheckJob struct {
	monitor         *domain.Monitor
	checker         *worker.HTTPChecker
	monitorRepo     MonitorRepository
	incidentRepo    IncidentRepository
	metricsService  MetricsService
}

type MonitorRepository interface {
	SaveCheck(ctx context.Context, check *domain.MonitorCheck) error
}

type IncidentRepository interface {
	GetOpenIncident(ctx context.Context, monitorID string) (*domain.Incident, error)
	Create(ctx context.Context, monitorID string, cause *string) (*domain.Incident, error)
	Resolve(ctx context.Context, id int64) error
}

type MetricsService interface {
	UpdateStats(ctx context.Context, monitorID string) error
}

func NewMonitorCheckJob(
	monitor *domain.Monitor,
	checker *worker.HTTPChecker,
	monitorRepo MonitorRepository,
	incidentRepo IncidentRepository,
	metricsService MetricsService,
) *MonitorCheckJob {
	return &MonitorCheckJob{
		monitor:        monitor,
		checker:        checker,
		monitorRepo:    monitorRepo,
		incidentRepo:   incidentRepo,
		metricsService: metricsService,
	}
}

func (j *MonitorCheckJob) Run() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	result := j.checker.Check(ctx, j.monitor)

	check := &domain.MonitorCheck{
		MonitorID:      j.monitor.ID,
		StatusCode:     result.StatusCode,
		ResponseTimeMs: result.ResponseTimeMs,
		IsUp:           result.IsUp,
		ErrorMessage:   result.ErrorMessage,
		CheckedAt:      time.Now(),
	}

	if err := j.monitorRepo.SaveCheck(ctx, check); err != nil {
		log.Printf("Failed to save check for monitor %s: %v", j.monitor.ID, err)
		return
	}

	j.handleIncidents(ctx, result)

	if err := j.metricsService.UpdateStats(ctx, j.monitor.ID); err != nil {
		log.Printf("Failed to update stats for monitor %s: %v", j.monitor.ID, err)
	}
}

func (j *MonitorCheckJob) handleIncidents(ctx context.Context, result *domain.CheckResult) {
	openIncident, err := j.incidentRepo.GetOpenIncident(ctx, j.monitor.ID)
	
	if !result.IsUp {
		if err != nil {
			if _, createErr := j.incidentRepo.Create(ctx, j.monitor.ID, result.ErrorMessage); createErr != nil {
				log.Printf("Failed to create incident for monitor %s: %v", j.monitor.ID, createErr)
			} else {
				log.Printf("Created incident for monitor %s: %s", j.monitor.Name, j.monitor.URL)
			}
		}
	} else {
		if openIncident != nil {
			if resolveErr := j.incidentRepo.Resolve(ctx, openIncident.ID); resolveErr != nil {
				log.Printf("Failed to resolve incident %d: %v", openIncident.ID, resolveErr)
			} else {
				log.Printf("Resolved incident for monitor %s: %s", j.monitor.Name, j.monitor.URL)
			}
		}
	}
}