package handler

import (
	"context"
	"fmt"
	"game-service/domain"
	httpUsecase "game-service/internal/api/http/usecase"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type UpdateRoomGameModeRequest struct {
	RoomID     uuid.UUID `params:"room_id"`
	GameModeID int       `json:"game_mode_id"`
}

type UpdateRoomGameModeResponse struct {
	Message string `json:"message"`
}

type UpdateRoomGameModeHandler struct {
	usecase httpUsecase.UpdateRoomGameModeUseCase
}

func NewUpdateRoomGameModeHandler(usecase httpUsecase.UpdateRoomGameModeUseCase) *UpdateRoomGameModeHandler {
	return &UpdateRoomGameModeHandler{
		usecase: usecase,
	}
}

func (h *UpdateRoomGameModeHandler) Handle(fbrCtx *fiber.Ctx, ctx context.Context, req *UpdateRoomGameModeRequest) (*UpdateRoomGameModeResponse, int, error) {
	userIDStr := fbrCtx.Get("X-User-ID")

	if userIDStr == "" {

		return nil, fiber.StatusUnauthorized, domain.ErrUnauthorized
	}
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return nil, fiber.StatusBadRequest, fmt.Errorf("Invalid user ID format")

	}

	status, err := h.usecase.Execute(ctx, req.RoomID, userID, req.GameModeID)
	if err != nil {
		return nil, status, err
	}

	return &UpdateRoomGameModeResponse{Message: "UpdateRoomGameMode "}, status, nil
}
