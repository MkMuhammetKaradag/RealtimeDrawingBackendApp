package httpUsecase

import (
	"context"
	"game-service/domain"

	"github.com/google/uuid"
)

type PostgresRepository interface {
	CreateUser(ctx context.Context, userID uuid.UUID, username, email string) error
	CreateRoom(ctx context.Context, roomName string, creatorID uuid.UUID, maxPlayers int, gameModeID int, isPrivate bool, roomCode string) (uuid.UUID, error)
	JoinRoom(ctx context.Context, roomID, userID uuid.UUID, roomCode string) error
	LeaveRoom(ctx context.Context, roomID, userID uuid.UUID) error
	UpdateRoomGameMode(ctx context.Context, roomID uuid.UUID, userID uuid.UUID, newGameModeID int) error
	GetVisibleRooms(ctx context.Context, userID uuid.UUID) ([]domain.Room, error)
}
type RoomRedisRepository interface {
	PublishMessage(ctx context.Context, roomID uuid.UUID, msgType string, dataContent interface{})
}
