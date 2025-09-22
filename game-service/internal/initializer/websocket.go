package initializer

import (
	"context"
	gameHub "game-service/internal/api/ws/hub"

	"github.com/redis/go-redis/v9"
)

func InitWebsocket(ctx context.Context, client *redis.Client) *gameHub.Hub {

	hub := gameHub.NewHub(client)
	go hub.Run(ctx)
	//go hub.StartCleanupJob(ctx)
	return hub
}
