package gateway

import (
	"fmt"
	"gateway-service/internal/config"
	"gateway-service/internal/middleware"
	"gateway-service/internal/routes"
	"gateway-service/utils"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/requestid"

	"github.com/gofiber/fiber/v2/middleware/recover"
)

type GatewayServer struct {
	rateLimiter *middleware.RateLimiter
}

func NewGatewayServer() (*GatewayServer, error) {
	// Session manager'ı oluştur
	rateLimitConfig := config.NewDefaultRateLimitConfig()
	fmt.Printf("Rate limit config: %+v\n", rateLimitConfig)
	rateLimiter := middleware.NewRateLimiter(rateLimitConfig)
	return &GatewayServer{
		rateLimiter: rateLimiter,
	}, nil
}

// Start, Gateway server'ı başlatır
// port: Server'ın dinleyeceği port numarası
func (gs *GatewayServer) Start(port string) error {
	// Fiber uygulamasını oluştur
	app := fiber.New()

	// Global middleware'leri ekle
	app.Use(requestid.New())
	app.Use(logger.New())  // Loglama middleware'i
	app.Use(recover.New()) // Panic recovery middleware'i
	app.Use(cors.New())    // CORS middleware'i

	// HTTP route'larını kaydet
	// routes.RegisterHTTPRoutes(app, gs.rateLimiter)
	routes.RegisterRoutes(app, gs.rateLimiter)
	routes.RegisterWebSocketRoutes(app, gs.rateLimiter)

	// Ana sayfa route'u
	app.Get("/", func(c *fiber.Ctx) error {
		return c.SendString("API Gateway çalışıyor!")
	})

	// Sağlık kontrolü route'u
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "UP"})
	})

	// 404 handler - bulunamayan route'lar için
	app.Use(func(c *fiber.Ctx) error {
		return c.Status(404).JSON(fiber.Map{"error": "Route not found"})
	})

	// Graceful shutdown için goroutine başlat
	go utils.GracefulShutdown(app)

	// Server'ı başlat
	return app.Listen(":" + port)
}
