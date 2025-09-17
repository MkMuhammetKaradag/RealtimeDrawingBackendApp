package server

import (
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/requestid"
)

type Config struct {
	Port         string
	IdleTimeout  time.Duration
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
}

func NewFiberApp(cfg Config) *fiber.App {
	app := fiber.New(fiber.Config{
		IdleTimeout:  cfg.IdleTimeout,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
		Concurrency:  256 * 1024,
	})
	app.Use(cors.New(cors.Config{
		// Frontend'inin tam adresini buraya yaz.
		// Bu, * yerine geçen ve kimlik bilgilerini göndermeye izin veren adrestir.
		AllowOrigins:     "http://localhost:5173",
		AllowHeaders:     "Origin, Content-Type, Accept, Authorization",
		AllowMethods:     "GET,POST,PUT,DELETE,OPTIONS",
		AllowCredentials: true,
	}))
	app.Use(requestid.New())

	// basic health endpoint
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "UP"})
	})
	return app
}

func Start(app *fiber.App, port string) error {
	return app.Listen(fmt.Sprintf("0.0.0.0:%s", port))
}
