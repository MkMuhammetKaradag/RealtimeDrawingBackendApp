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

type LeaveRoomUseCase interface {
	Execute(ctx context.Context, roomID, UserID uuid.UUID) (int, error)
}

type leaveRoomUseCase struct {
	repository    PostgresRepository
	roomRedisRepo RoomRedisRepository
}

func NewLeaveRoomUseCase(repository PostgresRepository, roomRedisRepo RoomRedisRepository) LeaveRoomUseCase {
	return &leaveRoomUseCase{
		repository:    repository,
		roomRedisRepo: roomRedisRepo,
	}
}

func (u *leaveRoomUseCase) Execute(ctx context.Context, roomID, userID uuid.UUID) (int, error) {
	err := u.repository.LeaveRoom(ctx, roomID, userID)
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
	go u.roomRedisRepo.PublishMessage(ctx, roomID, "player_left", map[string]string{"user_id": userID.String()})

	return fiber.StatusOK, nil
}
