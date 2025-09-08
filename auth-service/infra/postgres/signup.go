package postgres

import (
	"auth-service/domain"
	"context"

	"github.com/google/uuid"
)

func (r *Repository) SignUp(ctx context.Context, user *domain.User) (uuid.UUID, string, error) {
	return uuid.UUID{}, "", nil
}
