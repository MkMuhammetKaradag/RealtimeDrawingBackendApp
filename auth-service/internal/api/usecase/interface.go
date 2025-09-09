package usecase

import (
	"auth-service/domain"
	"context"
	"time"

	"github.com/google/uuid"
)

type PostgresRepository interface {
	SignUp(ctx context.Context, auth *domain.User) (uuid.UUID, string, error)
	Activate(ctx context.Context, activationID uuid.UUID, activationCode string) (*domain.User, error)
	SignIn(ctx context.Context, identifier, password string) (*domain.User, error)
}
type SessionManager interface {
	CreateSession(ctx context.Context, userID, token string, userData map[string]string, duration time.Duration) error
}
