package middleware

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	authModule "github.com/shubhbham/uptime-monitor/internal/modules/auth"
)

// AuthMiddleware creates a hybrid authentication middleware
// Supports both Clerk JWT (Bearer token) and API Key (X-API-Key header)
func AuthMiddleware(authService *authModule.Service) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Try Bearer token first (Clerk JWT)
		authHeader := c.Get("Authorization")
		if authHeader != "" && strings.HasPrefix(authHeader, "Bearer ") {
			token := strings.TrimPrefix(authHeader, "Bearer ")
			authCtx, err := authService.VerifyClerkToken(c.Context(), token)
			if err != nil {
				return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
					"error": "Invalid or expired token",
				})
			}

			// Store auth context in locals
			c.Locals("auth", authCtx)
			return c.Next()
		}

		// Try API Key (X-API-Key header)
		apiKey := c.Get("X-API-Key")
		if apiKey != "" {
			authCtx, err := authService.VerifyAPIKey(c.Context(), apiKey)
			if err != nil {
				return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
					"error": "Invalid API key",
				})
			}

			// Store auth context in locals
			c.Locals("auth", authCtx)
			return c.Next()
		}

		// No authentication provided
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Authentication required. Provide either 'Authorization: Bearer <token>' or 'X-API-Key: <key>' header",
		})
	}
}

// OptionalAuthMiddleware allows requests to pass through with or without auth
// If auth is provided, it validates it and stores in context
func OptionalAuthMiddleware(authService *authModule.Service) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Try Bearer token first
		authHeader := c.Get("Authorization")
		if authHeader != "" && strings.HasPrefix(authHeader, "Bearer ") {
			token := strings.TrimPrefix(authHeader, "Bearer ")
			authCtx, err := authService.VerifyClerkToken(c.Context(), token)
			if err == nil {
				c.Locals("auth", authCtx)
				return c.Next()
			}
		}

		// Try API Key
		apiKey := c.Get("X-API-Key")
		if apiKey != "" {
			authCtx, err := authService.VerifyAPIKey(c.Context(), apiKey)
			if err == nil {
				c.Locals("auth", authCtx)
				return c.Next()
			}
		}

		// No valid auth, but continue anyway (for public endpoints)
		return c.Next()
	}
}