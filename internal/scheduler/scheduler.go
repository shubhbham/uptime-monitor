package scheduler

import (
	"context"
	"fmt"
	"log"
	"sync"

	"github.com/robfig/cron/v3"
	"github.com/shubhbham/uptime-monitor/internal/domain"
	"github.com/shubhbham/uptime-monitor/internal/worker"
)

type Scheduler struct {
	cron           *cron.Cron
	checker        *worker.HTTPChecker
	monitorRepo    MonitorRepository
	incidentRepo   IncidentRepository
	metricsService MetricsService
	jobs           map[string]*scheduledJob
	mu             sync.RWMutex
}

type scheduledJob struct {
	entryID   cron.EntryID
	monitorID string
	cancel    context.CancelFunc
}

type MonitorFetcher interface {
	GetActiveMonitors(ctx context.Context) ([]*domain.Monitor, error)
}

func NewScheduler(
	checker *worker.HTTPChecker,
	monitorRepo MonitorRepository,
	incidentRepo IncidentRepository,
	metricsService MetricsService,
) *Scheduler {
	return &Scheduler{
		cron:           cron.New(),
		checker:        checker,
		monitorRepo:    monitorRepo,
		incidentRepo:   incidentRepo,
		metricsService: metricsService,
		jobs:           make(map[string]*scheduledJob),
	}
}

func (s *Scheduler) Start() {
	s.cron.Start()
	log.Println("Scheduler started")
}

func (s *Scheduler) Stop() {
	s.mu.Lock()
	
	// Cancel all running jobs
	for _, job := range s.jobs {
		if job.cancel != nil {
			job.cancel()
		}
	}
	s.mu.Unlock()

	ctx := s.cron.Stop()
	<-ctx.Done()
	log.Println("Scheduler stopped")
}

func (s *Scheduler) AddMonitor(monitor *domain.Monitor) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if already scheduled
	if existingJob, exists := s.jobs[monitor.ID]; exists {
		log.Printf("Monitor %s already scheduled, removing old job", monitor.ID)
		s.cron.Remove(existingJob.entryID)
		if existingJob.cancel != nil {
			existingJob.cancel()
		}
		delete(s.jobs, monitor.ID)
	}

	// Create a cancellable context for this monitor's jobs
	ctx, cancel := context.WithCancel(context.Background())

	job := NewMonitorCheckJob(
		monitor,
		s.checker,
		s.monitorRepo,
		s.incidentRepo,
		s.metricsService,
		ctx,
	)

	schedule := fmt.Sprintf("@every %ds", monitor.IntervalSeconds)
	entryID, err := s.cron.AddFunc(schedule, job.Run)
	if err != nil {
		cancel()
		return fmt.Errorf("failed to schedule monitor %s: %w", monitor.ID, err)
	}

	s.jobs[monitor.ID] = &scheduledJob{
		entryID:   entryID,
		monitorID: monitor.ID,
		cancel:    cancel,
	}

	log.Printf("Scheduled monitor: %s (%s) - checking every %d seconds", monitor.Name, monitor.URL, monitor.IntervalSeconds)

	return nil
}

func (s *Scheduler) RemoveMonitor(monitorID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if job, exists := s.jobs[monitorID]; exists {
		// Remove from cron
		s.cron.Remove(job.entryID)
		
		// Cancel any running jobs
		if job.cancel != nil {
			job.cancel()
		}
		
		delete(s.jobs, monitorID)
		log.Printf("Removed monitor from scheduler: %s", monitorID)
	}
}

func (s *Scheduler) UpdateMonitor(monitor *domain.Monitor) error {
	// Remove existing job
	s.RemoveMonitor(monitor.ID)
	
	// Add new job with updated configuration
	return s.AddMonitor(monitor)
}

func (s *Scheduler) LoadMonitors(ctx context.Context, fetcher MonitorFetcher) error {
	monitors, err := fetcher.GetActiveMonitors(ctx)
	if err != nil {
		return fmt.Errorf("failed to fetch active monitors: %w", err)
	}

	successCount := 0
	for _, monitor := range monitors {
		if err := s.AddMonitor(monitor); err != nil {
			log.Printf("Failed to schedule monitor %s: %v", monitor.ID, err)
		} else {
			successCount++
		}
	}

	log.Printf("Loaded %d out of %d active monitors", successCount, len(monitors))
	return nil
}

func (s *Scheduler) GetScheduledMonitors() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	monitorIDs := make([]string, 0, len(s.jobs))
	for monitorID := range s.jobs {
		monitorIDs = append(monitorIDs, monitorID)
	}
	return monitorIDs
}

func (s *Scheduler) IsMonitorScheduled(monitorID string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	_, exists := s.jobs[monitorID]
	return exists
}