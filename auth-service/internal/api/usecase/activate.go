package usecase

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

type ActivateUseCase interface {
	Execute(ctx context.Context, activationID uuid.UUID, activationCode string) error
}
type activateUseCase struct {
	postgresRepository PostgresRepository
}

func NewActivateUseCase(repository PostgresRepository) ActivateUseCase {
	return &activateUseCase{
		postgresRepository: repository,
	}
}

func (u *activateUseCase) Execute(ctx context.Context, activationID uuid.UUID, activationCode string) error {
	user, err := u.postgresRepository.Activate(ctx, activationID, activationCode)
	if err != nil {
		return err
	}
	fmt.Println("user:", user)

	return nil
}
