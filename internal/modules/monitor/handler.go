package monitor

import (
	"github.com/gofiber/fiber/v2"
	"github.com/shubhbham/uptime-monitor/internal/domain"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) Create(c *fiber.Ctx) error {
	// Get authenticated user from context
	authCtx := c.Locals("auth").(*domain.AuthContext)

	var req domain.CreateMonitorRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	monitor, err := h.service.CreateMonitor(c.Context(), authCtx.UserID, &req)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(fiber.StatusCreated).JSON(monitor)
}

func (h *Handler) Get(c *fiber.Ctx) error {
	authCtx := c.Locals("auth").(*domain.AuthContext)
	id := c.Params("id")
	
	if id == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Monitor ID is required",
		})
	}

	monitor, err := h.service.GetMonitor(c.Context(), id, authCtx.UserID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Monitor not found",
		})
	}

	return c.JSON(monitor)
}

func (h *Handler) List(c *fiber.Ctx) error {
	authCtx := c.Locals("auth").(*domain.AuthContext)

	monitors, err := h.service.ListMonitors(c.Context(), authCtx.UserID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch monitors",
		})
	}

	return c.JSON(monitors)
}

func (h *Handler) Update(c *fiber.Ctx) error {
	authCtx := c.Locals("auth").(*domain.AuthContext)
	id := c.Params("id")
	
	if id == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Monitor ID is required",
		})
	}

	var req domain.UpdateMonitorRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	monitor, err := h.service.UpdateMonitor(c.Context(), id, authCtx.UserID, &req)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(monitor)
}

func (h *Handler) Delete(c *fiber.Ctx) error {
	authCtx := c.Locals("auth").(*domain.AuthContext)
	id := c.Params("id")
	
	if id == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Monitor ID is required",
		})
	}

	if err := h.service.DeleteMonitor(c.Context(), id, authCtx.UserID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(fiber.StatusNoContent).Send(nil)
}

func (h *Handler) GetChecks(c *fiber.Ctx) error {
	authCtx := c.Locals("auth").(*domain.AuthContext)
	id := c.Params("id")
	
	if id == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Monitor ID is required",
		})
	}

	// Verify the monitor belongs to the user
	_, err := h.service.GetMonitor(c.Context(), id, authCtx.UserID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Monitor not found",
		})
	}

	limit := c.QueryInt("limit", 50)
	if limit > 200 {
		limit = 200
	}

	checks, err := h.service.GetRecentChecks(c.Context(), id, limit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch checks",
		})
	}

	return c.JSON(checks)
}