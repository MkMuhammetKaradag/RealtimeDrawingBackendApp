package httpUsecase

import (
	"context"
	"errors"
	"game-service/domain"
	"net/http"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type GetVisibleRoomsUseCase interface {
	Execute(ctx context.Context, userID uuid.UUID) (int, []domain.Room, error)
}

type getVisibleRoomsUseCase struct {
	repository PostgresRepository
}

func NewGetVisibleRoomsUseCase(repository PostgresRepository) GetVisibleRoomsUseCase {
	return &getVisibleRoomsUseCase{
		repository: repository,
	}
}

func (u *getVisibleRoomsUseCase) Execute(ctx context.Context, userID uuid.UUID) (int, []domain.Room, error) {
	rooms, err := u.repository.GetVisibleRooms(ctx, userID)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrInvalidInput):
			return http.StatusBadRequest, nil, err

		case errors.Is(err, domain.ErrConflict):
			return http.StatusConflict, nil, err

		default:
			return http.StatusInternalServerError, nil, err
		}
	}

	return fiber.StatusCreated, rooms, nil
}
