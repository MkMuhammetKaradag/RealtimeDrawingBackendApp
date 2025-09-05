// Package routes, API Gateway'in route tanımlamalarını içerir
package routes

import (
	"fmt"
	"gateway-service/internal/config"
	"gateway-service/internal/middleware"
	"gateway-service/utils"

	"github.com/gofiber/fiber/v2"
)

func RegisterHTTPRoutes(app *fiber.App, rateLimiter *middleware.RateLimiter) {

	// config.Services içindeki her servis için bir route oluştur
	// prefix: Servis adı (örn: "auth", "chat" vb.)
	for prefix, _ := range config.Services {
		// Her servis için bir proxy handler oluştur
		protectedPaths := config.ProtectedRoutes[prefix]

		for _, protectedPath := range protectedPaths {
			fullPath := fmt.Sprintf("/%s%s", prefix, protectedPath)
			fmt.Println("Registering protected route:", fullPath) // Debug için
		}
		// 🔥 Burada tüm istekler için service_name eklenmeli
		app.Use("/"+prefix+"/*", func(c *fiber.Ctx) error {
			c.Locals("service_name", prefix)
			return c.Next()
		})
		app.Use("/"+prefix+"/*", rateLimiter.Middleware())
		// Örnek: /auth/* istekleri auth servisine yönlendirilir
		app.All("/"+prefix+"/*", utils.BuildProxyHandler(prefix))
	}
}

// RegisterRoutes, tüm HTTP ve WebSocket rotalarını kaydeder.
func RegisterRoutes(app *fiber.App, rateLimiter *middleware.RateLimiter) {

	// HTTP servisleri için rotaları kaydet
	for prefix, _ := range config.Services {
		// Her servise özel bir middleware grubu oluştur
		// middleware.ServiceName ve rateLimiter.Middleware kullanıldı
		serviceGroup := app.Group("/"+prefix, middleware.ServiceName(prefix), rateLimiter.Middleware())
		fmt.Println(serviceGroup)
		// Gelen tüm HTTP isteklerini ilgili servise yönlendir
		// utils.BuildProxyHandler kullanıldı
		serviceGroup.All("/*", utils.BuildProxyHandler(prefix))
	}

	

}
