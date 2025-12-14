package app

import (
	"github.com/gofiber/fiber/v2"
	"github.com/shubhbham/uptime-monitor/internal/modules/monitor"
)

func (a *App) SetupRoutes() {
	api := a.Fiber.Group("/api/v1")

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

	// Monitor routes
	monitor.RegisterRoutes(api, a.MonitorHandler)

	// Incident routes
	api.Get("/incidents", a.IncidentHandler.ListAll)
	api.Get("/incidents/recent", a.IncidentHandler.GetRecent)
	api.Get("/monitors/:monitor_id/incidents", a.IncidentHandler.ListByMonitor)

	// Stats routes
	api.Get("/stats/:monitor_id", func(c *fiber.Ctx) error {
		monitorID := c.Params("monitor_id")
		stats, err := a.MetricsService.GetStats(c.Context(), monitorID)
		if err != nil {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Stats not found",
			})
		}
		return c.JSON(stats)
	})

	api.Get("/stats/:monitor_id/uptime", func(c *fiber.Ctx) error {
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
	api.Get("/scheduler/status", func(c *fiber.Ctx) error {
		scheduledMonitors := a.Scheduler.GetScheduledMonitors()
		return c.JSON(fiber.Map{
			"scheduled_count": len(scheduledMonitors),
			"monitor_ids":     scheduledMonitors,
		})
	})

	api.Get("/scheduler/monitor/:id", func(c *fiber.Ctx) error {
		monitorID := c.Params("id")
		isScheduled := a.Scheduler.IsMonitorScheduled(monitorID)
		return c.JSON(fiber.Map{
			"monitor_id":  monitorID,
			"is_scheduled": isScheduled,
		})
	})

	// Root endpoint
	a.Fiber.Get("/", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"message": "Uptime Monitor API",
			"version": "1.0",
			"endpoints": fiber.Map{
				"health":   "/api/v1/health",
				"monitors": "/api/v1/monitors",
				"incidents": "/api/v1/incidents",
				"stats":    "/api/v1/stats/:monitor_id",
				"scheduler": "/api/v1/scheduler/status",
			},
		})
	})
}