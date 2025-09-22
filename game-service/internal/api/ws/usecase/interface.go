package wsUsecase

import (
	"context"
	"game-service/domain"

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
}
