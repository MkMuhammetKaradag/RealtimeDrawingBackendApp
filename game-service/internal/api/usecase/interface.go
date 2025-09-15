package usecase

import (
	"context"

	"github.com/google/uuid"
)

type PostgresRepository interface {
	CreateUser(ctx context.Context, userID uuid.UUID, username, email string) error
}
