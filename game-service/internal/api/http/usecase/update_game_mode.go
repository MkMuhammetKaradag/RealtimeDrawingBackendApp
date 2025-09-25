package httpUsecase

import (
	"context"
	"net/http"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type UpdateRoomGameModeUseCase interface {
	Execute(ctx context.Context, roomID, UserID uuid.UUID, gameModeID int) (int, error)
}

type updateRoomGamemodeUseCase struct {
	repository    PostgresRepository
	roomRedisRepo RoomRedisRepository
}

func NewUpdateRoomGamemodeUseCase(repository PostgresRepository, roomRedisRepo RoomRedisRepository) UpdateRoomGameModeUseCase {
	return &updateRoomGamemodeUseCase{
		repository:    repository,
		roomRedisRepo: roomRedisRepo,
	}
}

func (u *updateRoomGamemodeUseCase) Execute(ctx context.Context, roomID, userID uuid.UUID, gameModeID int) (int, error) {

	err := u.repository.UpdateRoomGameMode(ctx, roomID, userID, gameModeID)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	go u.roomRedisRepo.PublishMessage(ctx, roomID,
		"game_mode_change",
		map[string]string{
			"user_id":   userID.String(),
			"mode_id":   strconv.Itoa(gameModeID),
			"mode_name": "null-1"})

	return fiber.StatusCreated, nil
}
