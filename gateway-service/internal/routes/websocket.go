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
		// Her servise özel bir middleware grubu oluştur
		// middleware.ServiceName ve rateLimiter.Middleware kullanıldı
		serviceGroup := app.Group("/"+prefix, middleware.ServiceName(prefix), rateLimiter.Middleware())
		// Gelen tüm HTTP isteklerini ilgili servise yönlendir
		// utils.BuildProxyHandler kullanıldı
		// serviceGroup.All("/*", utils.BuildWebSocketProxy(prefix))

		serviceGroup.All("/*", func(c *fiber.Ctx) error {
			// Gelen isteğin yolunu locals'a kaydet.
			// Bu değer, daha sonra WebSocket handler'ında kullanılacak.
			c.Locals("ws_path", c.Params("*"))
			c.Locals("ws_header", c.GetReqHeaders())
			// Buradan BuildWebSocketProxy'e geçiş yapıyoruz.
			return websocket.New(utils.BuildWebSocketProxy(prefix))(c)
		})

	}

}
