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
	Activate(ctx context.Context, activationID uuid.UUID, activationCode string) (*domain.User, error)
	SignIn(ctx context.Context, identifier, password string) (*domain.User, error)

	Close() error
}

func InitDatabase(config config.Config) PostgresRepository {
	return initializer.InitDatabase(config)
}
