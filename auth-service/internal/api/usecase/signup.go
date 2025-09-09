package usecase

import (
	"auth-service/domain"
	"context"
	"fmt"

	"go.uber.org/zap"
)

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

	id, code, err := u.postgresRepository.SignUp(ctx, user)
	if err != nil {
		return err
	}

	fmt.Printf("id:%v ,    code:%v \n", id, code)
	zap.L().Info("signup created")
	return nil
}
