package bootstrap

import (
	"auth-service/config"
	"auth-service/domain"
	"auth-service/internal/initializer"
	"context"

	"github.com/google/uuid"
)

type PostgresRepository interface {
	SignUp(ctx context.Context, auth *domain.User) (uuid.UUID, string, error)
}

func InitDatabase(config config.Config) PostgresRepository {
	return initializer.InitDatabase(config)
}
