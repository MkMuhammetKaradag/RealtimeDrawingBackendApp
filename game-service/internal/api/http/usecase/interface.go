package httpUsecase

import (
	"context"

	"github.com/google/uuid"
)

type PostgresRepository interface {
	CreateUser(ctx context.Context, userID uuid.UUID, username, email string) error
	CreateRoom(ctx context.Context, roomName string, creatorID uuid.UUID, maxPlayers int, gameModeID int, isPrivate bool, roomCode string) (uuid.UUID, error)
}
