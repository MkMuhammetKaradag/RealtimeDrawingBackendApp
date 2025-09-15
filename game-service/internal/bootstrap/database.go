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
}

func InitDatabase(config config.Config) PostgresRepository {
	return initializer.InitDatabase(config)
}
