package bootstrap

import (
	"context"
	"game-service/domain"
	"game-service/internal/api/ws/hub"
	"game-service/internal/initializer"

	"github.com/google/uuid"
)

type Hub interface {
	Run(ctx context.Context)
	RegisterClient(client *domain.Client)
	UnregisterClient(client *domain.Client)
	GetRoomClientCount(roomID uuid.UUID) int
	IsGameActive(roomID uuid.UUID) bool
	GetActiveGame(roomID uuid.UUID) *hub.Game
	IsPlayerInActiveGame(roomID, userID uuid.UUID) bool
	BroadcastMessage(roomID uuid.UUID, msg *hub.Message)
}

func InitWebsocket(ctx context.Context, redisRepo SessionManager) Hub {
	client := redisRepo.GetRedisClient()
	return initializer.InitWebsocket(ctx, client)
}
