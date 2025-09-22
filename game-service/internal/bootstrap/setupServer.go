package bootstrap

import (
	"game-service/config"
	httpGameHandler "game-service/internal/api/http/handler"
	wsHandler "game-service/internal/api/ws/handler"
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
	gameHandler := wsHandlers["room-connect"].(*wsHandler.WebSocketRoomHandler)
	wsRoute.Get("/game/:room_id", handler.HandleWithFiberWS[wsHandler.WebSocketRoomRequest](gameHandler))

	return app
}
