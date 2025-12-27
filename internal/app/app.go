package app

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/shubhbham/uptime-monitor/internal/config"
	"github.com/shubhbham/uptime-monitor/internal/httpclient"
	"github.com/shubhbham/uptime-monitor/internal/middleware"
	"github.com/shubhbham/uptime-monitor/internal/modules/auth"
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

	// Repositories
	MonitorRepo       *monitor.Repository
	IncidentRepo      *incident.Repository
	MetricsRepo       *metrics.Repository
	AuthRepo          *auth.Repository

	// Services
	MonitorService    *monitor.Service
	IncidentService   *incident.Service
	MetricsService    *metrics.Service
	AuthService       *auth.Service

	// Handlers
	MonitorHandler    *monitor.Handler
	IncidentHandler   *incident.Handler
	AuthHandler       *auth.Handler

	// Infrastructure
	HTTPClient        *httpclient.Client
	HTTPChecker       *worker.HTTPChecker
	Scheduler         *scheduler.Scheduler
}

func NewApp(cfg *config.Config) (*App, error) {
	db, err := storage.NewPostgresDB(&cfg.Database)
	if err != nil {
		return nil, err
	}

	// Initialize repositories
	monitorRepo := monitor.NewRepository(db.Pool)
	incidentRepo := incident.NewRepository(db.Pool)
	metricsRepo := metrics.NewRepository(db.Pool)
	authRepo := auth.NewRepository(db.Pool)

	// Initialize services
	monitorService := monitor.NewService(monitorRepo)
	incidentService := incident.NewService(incidentRepo)
	metricsService := metrics.NewService(metricsRepo, db.Pool)
	authService := auth.NewService(authRepo, &cfg.Auth)

	// Initialize handlers
	monitorHandler := monitor.NewHandler(monitorService)
	incidentHandler := incident.NewHandler(incidentService)
	authHandler := auth.NewHandler(authService)

	// Initialize infrastructure
	httpClient := httpclient.NewClient(cfg.Monitor.DefaultTimeout)
	httpChecker := worker.NewHTTPChecker(httpClient)

	sched := scheduler.NewScheduler(httpChecker, monitorRepo, incidentRepo, metricsService)

	// Wire scheduler to monitor service
	monitorService.SetScheduler(sched)

	// Create Fiber app with production-ready configuration
	app := fiber.New(fiber.Config{
		// Error handling
		ErrorHandler:  middleware.ErrorHandler(),

		// Timeouts
		ReadTimeout:   cfg.Server.ReadTimeout,
		WriteTimeout:  cfg.Server.WriteTimeout,
		IdleTimeout:   cfg.Server.ReadTimeout * 2,

		// Production settings
		CaseSensitive:     true,
		StrictRouting:     false,
		ServerHeader:      "UptimeMonitor",
		AppName:           "Uptime Monitor API v1.0",

		// CRITICAL: Proxy awareness for production deployments
		ProxyHeader:       fiber.HeaderXForwardedFor,
		EnableTrustedProxyCheck: true,
		TrustedProxies:    []string{"0.0.0.0/0"}, // Trust all proxies (adjust for production)

		// Body limits
		BodyLimit:         4 * 1024 * 1024, // 4MB

		// Disable startup message in production
		DisableStartupMessage: cfg.Server.Environment == "production",
	})

	// Global middleware
	app.Use(recover.New(recover.Config{
		EnableStackTrace: cfg.Server.Environment == "development",
	}))

	// CORS - Universal public access
	app.Use(cors.New(cors.Config{
		AllowOrigins:     "*",
		AllowMethods:     "GET,POST,PUT,DELETE,OPTIONS,PATCH,HEAD",
		AllowHeaders:     "Origin,Content-Type,Accept,Authorization,X-API-Key,X-Requested-With",
		AllowCredentials: false,
		ExposeHeaders:    "Content-Length,Content-Type",
		MaxAge:           300, // Cache preflight for 5 minutes
	}))

	app.Use(middleware.RequestLogger())

	return &App{
		Config:          cfg,
		Fiber:           app,
		DB:              db,
		MonitorRepo:     monitorRepo,
		IncidentRepo:    incidentRepo,
		MetricsRepo:     metricsRepo,
		AuthRepo:        authRepo,
		MonitorService:  monitorService,
		IncidentService: incidentService,
		MetricsService:  metricsService,
		AuthService:     authService,
		MonitorHandler:  monitorHandler,
		IncidentHandler: incidentHandler,
		AuthHandler:     authHandler,
		HTTPClient:      httpClient,
		HTTPChecker:     httpChecker,
		Scheduler:       sched,
	}, nil
}
