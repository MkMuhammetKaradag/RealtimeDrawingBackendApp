package bootstrap

import (
	"context"
	"game-service/config"
	"game-service/domain"
	"game-service/internal/initializer"

	"github.com/google/uuid"
)

type PostgresRepository interface {
	Close() error
	CreateUser(ctx context.Context, userID uuid.UUID, username, email string) error
	CreateRoom(ctx context.Context, roomName string, creatorID uuid.UUID, maxPlayers int, gameModeID int, isPrivate bool, roomCode string) (uuid.UUID, error)
	IsMemberAndHostRoom(ctx context.Context, roomID, userID uuid.UUID) (bool, bool, error)
	JoinRoom(ctx context.Context, roomID, userID uuid.UUID, roomCode string) error
	LeaveRoom(ctx context.Context, roomID, userID uuid.UUID) error
	UpdateRoomGameMode(ctx context.Context, roomID uuid.UUID, userID uuid.UUID, newGameModeID int) error
	GetVisibleRooms(ctx context.Context, userID uuid.UUID) ([]domain.Room, error)
}

func InitDatabase(config config.Config) PostgresRepository {
	return initializer.InitDatabase(config)
}
