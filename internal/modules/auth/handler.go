package auth

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

// CreateAPIKey creates a new API key
func (h *Handler) CreateAPIKey(c *fiber.Ctx) error {
	authCtx := c.Locals("auth").(*domain.AuthContext)

	var req domain.CreateAPIKeyRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if req.Name == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "API key name is required",
		})
	}

	apiKey, err := h.service.CreateAPIKey(c.Context(), authCtx.UserID, &req)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message": "⚠️ API key created successfully. Save this key securely - it won't be shown again!",
		"api_key": apiKey,
	})
}

// ListAPIKeys lists all API keys for the authenticated user
func (h *Handler) ListAPIKeys(c *fiber.Ctx) error {
	authCtx := c.Locals("auth").(*domain.AuthContext)

	apiKeys, err := h.service.ListAPIKeys(c.Context(), authCtx.UserID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch API keys",
		})
	}

	return c.JSON(fiber.Map{
		"api_keys": apiKeys,
		"count":    len(apiKeys),
	})
}

// DeleteAPIKey deletes an API key
func (h *Handler) DeleteAPIKey(c *fiber.Ctx) error {
	authCtx := c.Locals("auth").(*domain.AuthContext)
	keyID := c.Params("id")

	if keyID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "API key ID is required",
		})
	}

	if err := h.service.DeleteAPIKey(c.Context(), keyID, authCtx.UserID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(fiber.StatusNoContent).Send(nil)
}

// UpdateAPIKeyStatus updates the status of an API key
func (h *Handler) UpdateAPIKeyStatus(c *fiber.Ctx) error {
	authCtx := c.Locals("auth").(*domain.AuthContext)
	keyID := c.Params("id")

	if keyID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "API key ID is required",
		})
	}

	var req struct {
		IsActive bool `json:"is_active"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if err := h.service.UpdateAPIKeyStatus(c.Context(), keyID, authCtx.UserID, req.IsActive); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	status := "activated"
	if !req.IsActive {
		status = "deactivated"
	}

	return c.JSON(fiber.Map{
		"message": "API key " + status + " successfully",
	})
}

// GetCurrentUser returns the current authenticated user info
func (h *Handler) GetCurrentUser(c *fiber.Ctx) error {
	authCtx := c.Locals("auth").(*domain.AuthContext)
	return c.JSON(authCtx)
}

// DeleteAccount deletes the user account and all associated data
func (h *Handler) DeleteAccount(c *fiber.Ctx) error {
	authCtx := c.Locals("auth").(*domain.AuthContext)

	// Confirmation check
	var req struct {
		Confirm string `json:"confirm"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if req.Confirm != "DELETE" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Please confirm deletion by sending {\"confirm\": \"DELETE\"}",
		})
	}

	if err := h.service.DeleteUser(c.Context(), authCtx.UserID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to delete account",
		})
	}

	return c.JSON(fiber.Map{
		"message": "Account and all associated data deleted successfully",
	})
}

// DeactivateAccount deactivates the user account (soft delete)
func (h *Handler) DeactivateAccount(c *fiber.Ctx) error {
	authCtx := c.Locals("auth").(*domain.AuthContext)

	if err := h.service.DeactivateUser(c.Context(), authCtx.UserID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to deactivate account",
		})
	}

	return c.JSON(fiber.Map{
		"message": "Account deactivated successfully. All monitors and API keys have been disabled.",
	})
}