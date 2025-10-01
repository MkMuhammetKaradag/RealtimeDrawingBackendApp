package wsUsecase

import (
	"context"
	"game-service/domain"
	"game-service/internal/api/ws/hub"

	"github.com/gofiber/contrib/websocket"
	"github.com/google/uuid"
)

type ChannelWebSocketListenUseCase interface {
	Execute(c *websocket.Conn, ctx context.Context, roomID uuid.UUID)
}

type PostgresRepository interface {
	IsMemberRoom(ctx context.Context, roomID, userID uuid.UUID) (bool, error)
}
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
