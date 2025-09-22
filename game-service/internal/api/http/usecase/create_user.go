package httpUsecase

import (
	"context"

	"github.com/google/uuid"
)
type CreateUserUseCase interface {
	Execute(ctx context.Context, userID uuid.UUID, userName, email string) error
}
type createUserUseCase struct {
	repository PostgresRepository
}

func NewCreateUserUseCase(repository PostgresRepository) CreateUserUseCase {
	return &createUserUseCase{
		repository: repository,
	}
}

func (u *createUserUseCase) Execute(ctx context.Context, userID uuid.UUID, userName, email string) error {
	err := u.repository.CreateUser(ctx, userID, userName, email)
	if err != nil {
		return err

	}

	return nil
}
