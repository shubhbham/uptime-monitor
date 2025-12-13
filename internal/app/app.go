package app

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/shubhbham/uptime-monitor/internal/config"
	"github.com/shubhbham/uptime-monitor/internal/httpclient"
	"github.com/shubhbham/uptime-monitor/internal/middleware"
	"github.com/shubhbham/uptime-monitor/internal/modules/incident"
	"github.com/shubhbham/uptime-monitor/internal/modules/metrics"
	"github.com/shubhbham/uptime-monitor/internal/modules/monitor"
	"github.com/shubhbham/uptime-monitor/internal/scheduler"
	"github.com/shubhbham/uptime-monitor/internal/storage"
	"github.com/shubhbham/uptime-monitor/internal/worker"
)

type App struct {
	Config            *config.Config
	Fiber             *fiber.App
	DB                *storage.PostgresDB
	MonitorRepo       *monitor.Repository
	IncidentRepo      *incident.Repository
	MetricsRepo       *metrics.Repository
	MonitorService    *monitor.Service
	IncidentService   *incident.Service
	MetricsService    *metrics.Service
	MonitorHandler    *monitor.Handler
	IncidentHandler   *incident.Handler
	HTTPClient        *httpclient.Client
	HTTPChecker       *worker.HTTPChecker
	Scheduler         *scheduler.Scheduler
}

func NewApp(cfg *config.Config) (*App, error) {
	db, err := storage.NewPostgresDB(&cfg.Database)
	if err != nil {
		return nil, err
	}

	monitorRepo := monitor.NewRepository(db.Pool)
	incidentRepo := incident.NewRepository(db.Pool)
	metricsRepo := metrics.NewRepository(db.Pool)

	monitorService := monitor.NewService(monitorRepo)
	incidentService := incident.NewService(incidentRepo)
	metricsService := metrics.NewService(metricsRepo, db.Pool)

	monitorHandler := monitor.NewHandler(monitorService)
	incidentHandler := incident.NewHandler(incidentService)

	httpClient := httpclient.NewClient(cfg.Monitor.DefaultTimeout)
	httpChecker := worker.NewHTTPChecker(httpClient)

	sched := scheduler.NewScheduler(httpChecker, monitorRepo, incidentRepo, metricsService)

	app := fiber.New(fiber.Config{
		ErrorHandler:  middleware.ErrorHandler(),
		ReadTimeout:   cfg.Server.ReadTimeout,
		WriteTimeout:  cfg.Server.WriteTimeout,
		CaseSensitive: true,
		StrictRouting: false,
		ServerHeader:  "UptimeMonitor",
		AppName:       "Uptime Monitor API v1.0",
	})

	app.Use(recover.New())
	app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
		AllowMethods: "GET,POST,PUT,DELETE,OPTIONS",
		AllowHeaders: "Origin,Content-Type,Accept,Authorization",
	}))
	app.Use(middleware.RequestLogger())

	return &App{
		Config:          cfg,
		Fiber:           app,
		DB:              db,
		MonitorRepo:     monitorRepo,
		IncidentRepo:    incidentRepo,
		MetricsRepo:     metricsRepo,
		MonitorService:  monitorService,
		IncidentService: incidentService,
		MetricsService:  metricsService,
		MonitorHandler:  monitorHandler,
		IncidentHandler: incidentHandler,
		HTTPClient:      httpClient,
		HTTPChecker:     httpChecker,
		Scheduler:       sched,
	}, nil
}