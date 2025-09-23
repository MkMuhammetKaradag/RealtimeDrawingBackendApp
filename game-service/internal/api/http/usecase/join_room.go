package httpUsecase

import (
	"context"
	"errors"
	"fmt"
	"game-service/domain"
	"net/http"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type JoinRoomUseCase interface {
	Execute(ctx context.Context, roomID, UserID uuid.UUID, roomCode string) (int, error)
}

type joinRoomUseCase struct {
	repository    PostgresRepository
	roomRedisRepo RoomRedisRepository
}

func NewJoinRoomUseCase(repository PostgresRepository, roomRedisRepo RoomRedisRepository) JoinRoomUseCase {
	return &joinRoomUseCase{
		repository:    repository,
		roomRedisRepo: roomRedisRepo,
	}
}

func (u *joinRoomUseCase) Execute(ctx context.Context, roomID, userID uuid.UUID, roomCode string) (int, error) {
	err := u.repository.JoinRoom(ctx, roomID, userID, roomCode)
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
	go u.roomRedisRepo.PublishMessage(ctx, roomID, "player_joined", map[string]string{"user_id": userID.String()})
	fmt.Println(roomID)
	return fiber.StatusCreated, nil
}
