package incident

import (
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) ListByMonitor(c *fiber.Ctx) error {
	monitorID := c.Params("monitor_id")
	if monitorID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Monitor ID is required",
		})
	}

	limit := c.QueryInt("limit", 50)
	if limit > 200 {
		limit = 200
	}

	incidents, err := h.service.ListIncidentsByMonitor(c.Context(), monitorID, limit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch incidents",
		})
	}

	return c.JSON(incidents)
}

func (h *Handler) ListAll(c *fiber.Ctx) error {
	limit := c.QueryInt("limit", 50)
	if limit > 200 {
		limit = 200
	}

	incidents, err := h.service.ListAllIncidents(c.Context(), limit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch incidents",
		})
	}

	return c.JSON(incidents)
}

func (h *Handler) GetRecent(c *fiber.Ctx) error {
	hoursStr := c.Query("hours", "24")
	hours, err := strconv.Atoi(hoursStr)
	if err != nil || hours < 1 || hours > 720 {
		hours = 24
	}

	since := time.Now().Add(-time.Duration(hours) * time.Hour)

	incidents, err := h.service.GetRecentIncidents(c.Context(), since)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch recent incidents",
		})
	}

	return c.JSON(incidents)
}