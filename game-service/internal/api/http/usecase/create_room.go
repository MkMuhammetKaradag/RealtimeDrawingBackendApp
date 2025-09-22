package httpUsecase

import (
	"context"
	"errors"
	"fmt"
	"game-service/domain"
	"net/http"

	"github.com/gofiber/fiber/v2"
)

type CreateRoomUseCase interface {
	Execute(ctx context.Context, data domain.Room) (int, error)
}

type createRoomUseCase struct {
	repository PostgresRepository
}

func NewCreateRoomUseCase(repository PostgresRepository) CreateRoomUseCase {
	return &createRoomUseCase{
		repository: repository,
	}
}

func (u *createRoomUseCase) Execute(ctx context.Context, data domain.Room) (int, error) {
	roomID, err := u.repository.CreateRoom(ctx, data.RoomName, data.CreatorID, data.MaxPlayers, data.GameModeID, data.IsPrivate, data.RoomCode)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrInvalidInput):
			return http.StatusBadRequest, err

		case errors.Is(err, domain.ErrConflict):
			return http.StatusConflict, err

		default:
			return http.StatusInternalServerError, err
		}
	}
	fmt.Println(roomID)
	return fiber.StatusCreated, nil
}
