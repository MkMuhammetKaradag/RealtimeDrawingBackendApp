package bootstrap

import (
	"game-service/config"
	httpGameHandler "game-service/internal/api/handler"
	wsHandler "game-service/internal/api/ws"
	"game-service/internal/handler"
	"game-service/internal/server"

	"time"

	"github.com/gofiber/fiber/v2"
)

type Config struct {
	Port         string
	IdleTimeout  time.Duration
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
}

func SetupServer(config config.Config, httpHandlers map[string]interface{}, wsHandlers map[string]interface{}) *fiber.App {

	serverConfig := server.Config{
		Port:         config.Server.Port,
		IdleTimeout:  5 * time.Second,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	app := server.NewFiberApp(serverConfig)

	createRoomHandler := httpHandlers["create-room"].(*httpGameHandler.CreateRoomHandler)
	app.Post("/create-room", handler.HandleWithFiber[httpGameHandler.CreateRoomRequest, httpGameHandler.CreateRoomResponse](createRoomHandler))

	wsRoute := app.Group("/ws")
	gameHandler := wsHandlers["game"].(*wsHandler.WebSocketHandler)
	wsRoute.Get("/game/:id", handler.HandleWithFiberWS[wsHandler.WebSocketListenRequest](gameHandler))

	return app
}
