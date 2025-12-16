package app

import (
	"github.com/gofiber/fiber/v2"
	"github.com/shubhbham/uptime-monitor/internal/middleware"
	"github.com/shubhbham/uptime-monitor/internal/modules/monitor"
)

func (a *App) SetupRoutes() {
	// Public routes (no auth required)
	a.Fiber.Get("/", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"message": "Uptime Monitor API",
			"version": "1.0",
			"docs":    "https://github.com/yourusername/uptime-monitor",
			"endpoints": fiber.Map{
				"health": "/api/v1/health",
				"auth":   "/api/v1/auth/*",
				"api":    "/api/v1/*",
			},
		})
	})

	api := a.Fiber.Group("/api/v1")

	// Public health check (no auth)
	api.Get("/health", func(c *fiber.Ctx) error {
		if err := a.DB.Health(c.Context()); err != nil {
			return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
				"status": "unhealthy",
				"error":  err.Error(),
			})
		}

		return c.JSON(fiber.Map{
			"status":  "healthy",
			"version": "1.0",
		})
	})

	// Authentication middleware for all protected routes
	authMiddleware := middleware.AuthMiddleware(a.AuthService)

	// Auth routes (authenticated users can manage their API keys and account)
	authGroup := api.Group("/auth", authMiddleware)
	{
		// User info
		authGroup.Get("/me", a.AuthHandler.GetCurrentUser)
		
		// API Key management
		authGroup.Post("/api-keys", a.AuthHandler.CreateAPIKey)
		authGroup.Get("/api-keys", a.AuthHandler.ListAPIKeys)
		authGroup.Delete("/api-keys/:id", a.AuthHandler.DeleteAPIKey)
		authGroup.Put("/api-keys/:id/status", a.AuthHandler.UpdateAPIKeyStatus)
		
		// Account management
		authGroup.Delete("/account", a.AuthHandler.DeleteAccount)
		authGroup.Post("/account/deactivate", a.AuthHandler.DeactivateAccount)
	}

	// Protected routes (require authentication)
	protected := api.Group("", authMiddleware)
	{
		// Monitor routes
		monitor.RegisterRoutes(protected, a.MonitorHandler)

		// Incident routes (scoped to user's monitors)
		protected.Get("/incidents", func(c *fiber.Ctx) error {
			// This will return all incidents, but they're already filtered
			// by monitors which are user-scoped
			return a.IncidentHandler.ListAll(c)
		})
		protected.Get("/incidents/recent", a.IncidentHandler.GetRecent)
		protected.Get("/monitors/:monitor_id/incidents", a.IncidentHandler.ListByMonitor)

		// Stats routes (scoped to user's monitors)
		protected.Get("/stats/:monitor_id", func(c *fiber.Ctx) error {
			monitorID := c.Params("monitor_id")
			stats, err := a.MetricsService.GetStats(c.Context(), monitorID)
			if err != nil {
				return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
					"error": "Stats not found",
				})
			}
			return c.JSON(stats)
		})

		protected.Get("/stats/:monitor_id/uptime", func(c *fiber.Ctx) error {
			monitorID := c.Params("monitor_id")
			uptime, err := a.MetricsService.GetUptimeStats(c.Context(), monitorID)
			if err != nil {
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"error": "Failed to calculate uptime",
				})
			}
			return c.JSON(uptime)
		})

		// Debug/Admin routes
		protected.Get("/scheduler/status", func(c *fiber.Ctx) error {
			scheduledMonitors := a.Scheduler.GetScheduledMonitors()
			return c.JSON(fiber.Map{
				"scheduled_count": len(scheduledMonitors),
				"monitor_ids":     scheduledMonitors,
			})
		})

		protected.Get("/scheduler/monitor/:id", func(c *fiber.Ctx) error {
			monitorID := c.Params("id")
			isScheduled := a.Scheduler.IsMonitorScheduled(monitorID)
			return c.JSON(fiber.Map{
				"monitor_id":   monitorID,
				"is_scheduled": isScheduled,
			})
		})
	}
}