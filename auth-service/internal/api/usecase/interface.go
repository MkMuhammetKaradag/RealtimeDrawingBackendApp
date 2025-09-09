package usecase

import (
	"auth-service/domain"
	"context"

	"github.com/google/uuid"
)

type PostgresRepository interface {
	SignUp(ctx context.Context, auth *domain.User) (uuid.UUID, string, error)
	Activate(ctx context.Context, activationID uuid.UUID, activationCode string) (*domain.User, error)
}
