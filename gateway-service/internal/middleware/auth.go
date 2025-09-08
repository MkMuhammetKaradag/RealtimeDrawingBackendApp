package middleware

import (
	"gateway-service/internal/config"
	"strings"

	"github.com/gofiber/fiber/v2"
)

func AuthGuard() fiber.Handler {
	return func(c *fiber.Ctx) error {
		serviceName, _ := c.Locals("service_name").(string)
		path := c.Path()

		if isProtected(serviceName, strings.TrimPrefix(path, "/"+serviceName)) {
			// Require Authorization header or Session cookie
			if c.Get("Authorization") == "" && c.Cookies("Session") == "" {
				return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
			}
		}
		return c.Next()
	}
}

func isProtected(serviceName, path string) bool {
	protectedList, ok := config.ProtectedRoutes[serviceName]
	if !ok {
		return false
	}
	for _, p := range protectedList {
		if strings.HasPrefix(path, p) {
			return true
		}
	}
	return false
}
