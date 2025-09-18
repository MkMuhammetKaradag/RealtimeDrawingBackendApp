package usecase

import (
	"auth-service/domain"
	"context"
	"fmt"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
)

type SignUpUseCase interface {
	Execute(ctx context.Context, user *domain.User) (int, error)
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

func (u *signUpUseCase) Execute(ctx context.Context, user *domain.User) (int, error) {

	id, code, err := u.postgresRepository.SignUp(ctx, user)
	if err != nil {
		return fiber.StatusInternalServerError, err
	}

	fmt.Printf("id:%v ,    code:%v \n", id, code)
	zap.L().Info("signup created")
	return fiber.StatusOK, nil
}
