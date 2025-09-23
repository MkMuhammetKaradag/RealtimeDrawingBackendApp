package handler

import (
	"context"
	"fmt"
	"game-service/domain"
	httpUsecase "game-service/internal/api/http/usecase"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type LeaveRoomRequest struct {
	RoomID uuid.UUID `params:"room_id"`
}

type LeaveRoomResponse struct {
	Message string `json:"message"`
}

type LeaveRoomHandler struct {
	usecase httpUsecase.LeaveRoomUseCase
}

func NewLeaveRoomHandler(usecase httpUsecase.LeaveRoomUseCase) *LeaveRoomHandler {
	return &LeaveRoomHandler{
		usecase: usecase,
	}
}

func (h *LeaveRoomHandler) Handle(fbrCtx *fiber.Ctx, ctx context.Context, req *LeaveRoomRequest) (*LeaveRoomResponse, int, error) {
	userIDStr := fbrCtx.Get("X-User-ID")

	if userIDStr == "" {

		return nil, fiber.StatusUnauthorized, domain.ErrUnauthorized
	}
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return nil, fiber.StatusBadRequest, fmt.Errorf("Invalid user ID format")

	}

	status, err := h.usecase.Execute(ctx, req.RoomID, userID)
	if err != nil {
		return nil, status, err
	}

	return &LeaveRoomResponse{Message: "LeaveRoom user "}, status, nil
}
