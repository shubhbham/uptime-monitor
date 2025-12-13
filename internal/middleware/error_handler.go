package middleware

import (
	"log"

	"github.com/gofiber/fiber/v2"
)

func ErrorHandler() fiber.ErrorHandler {
	return func(c *fiber.Ctx, err error) error {
		code := fiber.StatusInternalServerError

		if e, ok := err.(*fiber.Error); ok {
			code = e.Code
		}

		log.Printf("Error: %v | Path: %s | Method: %s", err, c.Path(), c.Method())

		return c.Status(code).JSON(fiber.Map{
			"error":   err.Error(),
			"code":    code,
			"path":    c.Path(),
			"method":  c.Method(),
		})
	}
}