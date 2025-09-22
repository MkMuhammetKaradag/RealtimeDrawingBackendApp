package bootstrap

import (
	"context"
	"game-service/domain"
	"game-service/internal/initializer"
)

type Hub interface {
	Run(ctx context.Context)
	RegisterClient(client *domain.Client)
	UnregisterClient(client *domain.Client)
}

func InitWebsocket(ctx context.Context, redisRepo SessionManager) Hub {
	client := redisRepo.GetRedisClient()
	return initializer.InitWebsocket(ctx, client)
}
