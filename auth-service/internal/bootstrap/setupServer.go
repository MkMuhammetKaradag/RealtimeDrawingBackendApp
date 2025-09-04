package bootstrap

import (
	"auth-service/config"
	"auth-service/internal/server"

	"time"

	"github.com/gofiber/fiber/v2"
)

type Config struct {
	Port         string
	IdleTimeout  time.Duration
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
}

func SetupServer(config config.Config, httpHandlers map[string]interface{}) *fiber.App {

	serverConfig := server.Config{
		Port:         config.Server.Port,
		IdleTimeout:  5 * time.Second,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	app := server.NewFiberApp(serverConfig)

	app.Get("/hello", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"message": "Hello from Auth Service!",
		})
	})

	return app
}
