package handler

import (
	"context"
	"fmt"
	"game-service/domain"
	httpUsecase "game-service/internal/api/http/usecase"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type JoinRoomRequest struct {
	RoomID   uuid.UUID `params:"room_id"`
	RoomCode string    `json:"room_code"`
}

type JoinRoomResponse struct {
	Message string `json:"message"`
}

type JoinRoomHandler struct {
	usecase httpUsecase.JoinRoomUseCase
}

func NewJoinRoomHandler(usecase httpUsecase.JoinRoomUseCase) *JoinRoomHandler {
	return &JoinRoomHandler{
		usecase: usecase,
	}
}

func (h *JoinRoomHandler) Handle(fbrCtx *fiber.Ctx, ctx context.Context, req *JoinRoomRequest) (*JoinRoomResponse, int, error) {
	userIDStr := fbrCtx.Get("X-User-ID")

	if userIDStr == "" {

		return nil, fiber.StatusUnauthorized, domain.ErrUnauthorized
	}
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return nil, fiber.StatusBadRequest, fmt.Errorf("Invalid user ID format")

	}

	status, err := h.usecase.Execute(ctx, req.RoomID, userID, req.RoomCode)
	if err != nil {
		return nil, status, err
	}

	return &JoinRoomResponse{Message: "JoinRoom user "}, status, nil
}
