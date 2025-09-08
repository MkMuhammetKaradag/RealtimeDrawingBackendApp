package usecase

import (
	"auth-service/domain"
	"context"
	"fmt"

	"github.com/google/uuid"
)

type PostgresRepository interface {
	SignUp(ctx context.Context, auth *domain.User) (uuid.UUID, string, error)
}

type SignUpUseCase interface {
	Execute(ctx context.Context, user *domain.User) error
}
type signUpUseCase struct {
	postgresRepository PostgresRepository
}

type SignUpRequest struct {
	Username string
	Email    string
	Password string
}

func NewSignUpUseCase(repository PostgresRepository) SignUpUseCase {
	return &signUpUseCase{
		postgresRepository: repository,
	}
}

func (u *signUpUseCase) Execute(ctx context.Context, user *domain.User) error {

	fmt.Println("signup-user", user)

	return nil
}
