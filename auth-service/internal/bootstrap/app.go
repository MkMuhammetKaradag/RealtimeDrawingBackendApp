package bootstrap

import (
	"auth-service/config"
	"auth-service/pkg/graceful"
	"context"
	"time"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
)

type App struct {
	config         config.Config
	postgresRepo   PostgresRepository
	sessionManager SessionManager
	fiberApp       *fiber.App
	httpHandlers   map[string]interface{}
}

func NewApp(config config.Config) *App {
	app := &App{
		config: config,
	}
	app.initDependencies()
	return app
}

func (a *App) initDependencies() {
	a.postgresRepo = InitDatabase(a.config)
	a.sessionManager = InitSessionRedis(a.config)
	a.httpHandlers = SetupHTTPHandlers(a.postgresRepo, a.sessionManager)
	a.fiberApp = SetupServer(a.config, a.httpHandlers)
}

func (a *App) Start() {
	go func() {
		port := a.config.Server.Port
		if err := a.fiberApp.Listen(":" + port); err != nil {
			zap.L().Error("Failed to start server", zap.Error(err))
		}
	}()

	zap.L().Info("Server started on port", zap.String("port", a.config.Server.Port))

	defer func() {
		if err := a.postgresRepo.Close(); err != nil {
			zap.L().Error("Failed to close database", zap.Error(err))
		}
	}()

	graceful.WaitForShutdown(a.fiberApp, 5*time.Second, context.Background())
}
