package monitor

import (
	"github.com/gofiber/fiber/v2"
)

func RegisterRoutes(router fiber.Router, handler *Handler) {
	monitors := router.Group("/monitors")
	
	monitors.Post("/", handler.Create)
	monitors.Get("/", handler.List)
	monitors.Get("/:id", handler.Get)
	monitors.Put("/:id", handler.Update)
	monitors.Delete("/:id", handler.Delete)
	monitors.Get("/:id/checks", handler.GetChecks)
}