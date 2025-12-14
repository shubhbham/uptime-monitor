package scheduler

import (
	"context"
	"errors"
	"log"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/shubhbham/uptime-monitor/internal/domain"
	"github.com/shubhbham/uptime-monitor/internal/worker"
)

type MonitorCheckJob struct {
	monitor         *domain.Monitor
	checker         *worker.HTTPChecker
	monitorRepo     MonitorRepository
	incidentRepo    IncidentRepository
	metricsService  MetricsService
	ctx             context.Context
}

type MonitorRepository interface {
	SaveCheck(ctx context.Context, check *domain.MonitorCheck) error
	GetByID(ctx context.Context, id string) (*domain.Monitor, error)
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
	ctx context.Context,
) *MonitorCheckJob {
	return &MonitorCheckJob{
		monitor:        monitor,
		checker:        checker,
		monitorRepo:    monitorRepo,
		incidentRepo:   incidentRepo,
		metricsService: metricsService,
		ctx:            ctx,
	}
}

func (j *MonitorCheckJob) Run() {
	// Check if job should stop (monitor deleted or context cancelled)
	select {
	case <-j.ctx.Done():
		log.Printf("Job cancelled for monitor %s", j.monitor.ID)
		return
	default:
	}

	// Create a timeout context for this specific check
	checkCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// SAFETY CHECK: Verify monitor still exists before checking
	currentMonitor, err := j.monitorRepo.GetByID(checkCtx, j.monitor.ID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			log.Printf("Monitor %s no longer exists, stopping checks", j.monitor.ID)
			return
		}
		log.Printf("Failed to verify monitor %s existence: %v", j.monitor.ID, err)
		return
	}

	// Update local monitor reference with latest data
	j.monitor = currentMonitor

	// Skip if monitor is inactive
	if !j.monitor.IsActive {
		log.Printf("Monitor %s is inactive, skipping check", j.monitor.ID)
		return
	}

	// Perform the actual HTTP check
	result := j.checker.Check(checkCtx, j.monitor)

	// Create check record
	check := &domain.MonitorCheck{
		MonitorID:      j.monitor.ID,
		StatusCode:     result.StatusCode,
		ResponseTimeMs: result.ResponseTimeMs,
		IsUp:           result.IsUp,
		ErrorMessage:   result.ErrorMessage,
		CheckedAt:      time.Now(),
	}

	// Save check result
	if err := j.monitorRepo.SaveCheck(checkCtx, check); err != nil {
		// Check if it's a foreign key violation (monitor was deleted)
		if isForeignKeyViolation(err) {
			log.Printf("Monitor %s was deleted, stopping checks", j.monitor.ID)
			return
		}
		log.Printf("Failed to save check for monitor %s: %v", j.monitor.ID, err)
		return
	}

	// Handle incidents
	j.handleIncidents(checkCtx, result)

	// Update statistics
	if err := j.metricsService.UpdateStats(checkCtx, j.monitor.ID); err != nil {
		log.Printf("Failed to update stats for monitor %s: %v", j.monitor.ID, err)
	}
}

func (j *MonitorCheckJob) handleIncidents(ctx context.Context, result *domain.CheckResult) {
	openIncident, err := j.incidentRepo.GetOpenIncident(ctx, j.monitor.ID)
	
	if !result.IsUp {
		// Service is DOWN
		if err != nil {
			// No open incident exists, create one
			if _, createErr := j.incidentRepo.Create(ctx, j.monitor.ID, result.ErrorMessage); createErr != nil {
				log.Printf("Failed to create incident for monitor %s: %v", j.monitor.ID, createErr)
			} else {
				log.Printf("Created incident for monitor %s: %s", j.monitor.Name, j.monitor.URL)
			}
		}
		// else: incident already exists, do nothing
	} else {
		// Service is UP
		if openIncident != nil {
			// Resolve the open incident
			if resolveErr := j.incidentRepo.Resolve(ctx, openIncident.ID); resolveErr != nil {
				log.Printf("Failed to resolve incident %d: %v", openIncident.ID, resolveErr)
			} else {
				duration := time.Since(openIncident.StartedAt)
				log.Printf("Resolved incident for monitor %s: %s (downtime: %v)", 
					j.monitor.Name, j.monitor.URL, duration)
			}
		}
	}
}

// isForeignKeyViolation checks if the error is a foreign key constraint violation
func isForeignKeyViolation(err error) bool {
	if err == nil {
		return false
	}
	
	// Check for PostgreSQL foreign key violation error code
	errStr := err.Error()
	return containsAny(errStr, []string{
		"violates foreign key constraint",
		"foreign key constraint",
		"23503", // PostgreSQL error code for foreign key violation
	})
}

func containsAny(str string, substrs []string) bool {
	for _, substr := range substrs {
		if len(str) >= len(substr) {
			for i := 0; i <= len(str)-len(substr); i++ {
				if str[i:i+len(substr)] == substr {
					return true
				}
			}
		}
	}
	return false
}