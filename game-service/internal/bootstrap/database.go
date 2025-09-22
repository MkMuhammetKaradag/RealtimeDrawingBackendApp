package bootstrap

import (
	"context"
	"game-service/config"
	"game-service/internal/initializer"

	"github.com/google/uuid"
)

type PostgresRepository interface {
	Close() error
	CreateUser(ctx context.Context, userID uuid.UUID, username, email string) error
	CreateRoom(ctx context.Context, roomName string, creatorID uuid.UUID, maxPlayers int, gameModeID int, isPrivate bool, roomCode string) (uuid.UUID, error)
	IsMemberRoom(ctx context.Context, roomID, userID uuid.UUID) (bool, error)
}

func InitDatabase(config config.Config) PostgresRepository {
	return initializer.InitDatabase(config)
}
