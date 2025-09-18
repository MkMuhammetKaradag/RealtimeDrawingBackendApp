package usecase

import (
	"auth-service/domain"
	"context"
	"time"

	pb "shared-lib/events"

	"github.com/google/uuid"
)

type PostgresRepository interface {
	SignUp(ctx context.Context, auth *domain.User) (uuid.UUID, string, error)
	Activate(ctx context.Context, activationID uuid.UUID, activationCode string) (*domain.User, error)
	SignIn(ctx context.Context, identifier, password string) (*domain.User, error)
}
type SessionManager interface {
	CreateSession(ctx context.Context, token string, data *domain.SessionData, duration time.Duration) error
	DeleteSession(ctx context.Context, token string) error
	DeleteAllUserSessions(ctx context.Context, token string) error
	GetSession(ctx context.Context, token string) (*domain.SessionData, error)
	UpdateSession(ctx context.Context, oldToken, newToken string, data *domain.SessionData, duration time.Duration) error
	GetTokenTTL(ctx context.Context, token string) (time.Duration, error)
}
type Messaging interface {
	Close() error
	PublishMessage(ctx context.Context, msg *pb.Message) error
}
