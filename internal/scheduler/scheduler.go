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
	jobs           map[string]cron.EntryID
	mu             sync.RWMutex
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
		jobs:           make(map[string]cron.EntryID),
	}
}

func (s *Scheduler) Start() {
	s.cron.Start()
	log.Println("Scheduler started")
}

func (s *Scheduler) Stop() {
	ctx := s.cron.Stop()
	<-ctx.Done()
	log.Println("Scheduler stopped")
}

func (s *Scheduler) AddMonitor(monitor *domain.Monitor) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.jobs[monitor.ID]; exists {
		return fmt.Errorf("monitor %s already scheduled", monitor.ID)
	}

	job := NewMonitorCheckJob(monitor, s.checker, s.monitorRepo, s.incidentRepo, s.metricsService)

	schedule := fmt.Sprintf("@every %ds", monitor.IntervalSeconds)
	entryID, err := s.cron.AddFunc(schedule, job.Run)
	if err != nil {
		return fmt.Errorf("failed to schedule monitor %s: %w", monitor.ID, err)
	}

	s.jobs[monitor.ID] = entryID
	log.Printf("Scheduled monitor: %s (%s) - checking every %d seconds", monitor.Name, monitor.URL, monitor.IntervalSeconds)

	return nil
}

func (s *Scheduler) RemoveMonitor(monitorID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if entryID, exists := s.jobs[monitorID]; exists {
		s.cron.Remove(entryID)
		delete(s.jobs, monitorID)
		log.Printf("Removed monitor from scheduler: %s", monitorID)
	}
}

func (s *Scheduler) UpdateMonitor(monitor *domain.Monitor) error {
	s.RemoveMonitor(monitor.ID)
	return s.AddMonitor(monitor)
}

func (s *Scheduler) LoadMonitors(ctx context.Context, fetcher MonitorFetcher) error {
	monitors, err := fetcher.GetActiveMonitors(ctx)
	if err != nil {
		return fmt.Errorf("failed to fetch active monitors: %w", err)
	}

	for _, monitor := range monitors {
		if err := s.AddMonitor(monitor); err != nil {
			log.Printf("Failed to schedule monitor %s: %v", monitor.ID, err)
		}
	}

	log.Printf("Loaded %d active monitors", len(monitors))
	return nil
}