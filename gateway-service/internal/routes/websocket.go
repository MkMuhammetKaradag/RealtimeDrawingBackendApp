package routes

import (
	"gateway-service/internal/config"
	"gateway-service/internal/middleware"
	"gateway-service/utils"

	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
)

func RegisterWebSocketRoutes(app *fiber.App, rateLimiter *middleware.RateLimiter) {

	for prefix, _ := range config.WebSocketServices {
		// WS grup: service name, rate limit, auth guard
		serviceGroup := app.Group("/"+prefix, middleware.ServiceName(prefix), rateLimiter.Middleware(), middleware.AuthGuard())

		serviceGroup.All("/*", func(c *fiber.Ctx) error {
			c.Locals("ws_path", c.Params("*"))
			c.Locals("ws_header", c.GetReqHeaders())
			return websocket.New(utils.BuildWebSocketProxy(prefix))(c)
		})

	}

}
