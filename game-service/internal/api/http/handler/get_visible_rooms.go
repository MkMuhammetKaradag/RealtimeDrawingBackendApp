package handler

import (
	"context"
	"fmt"
	"game-service/domain"
	httpUsecase "game-service/internal/api/http/usecase"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type GetVisibleRoomsRequest struct {
}

type GetVisibleRoomsResponse struct {
	Message string        `json:"message"`
	Rooms   []domain.Room `json:"rooms"`
}

type GetVisibleRoomsHandler struct {
	usecase httpUsecase.GetVisibleRoomsUseCase
}

func NewGetVisibleRoomsHandler(usecase httpUsecase.GetVisibleRoomsUseCase) *GetVisibleRoomsHandler {
	return &GetVisibleRoomsHandler{
		usecase: usecase,
	}
}

func (h *GetVisibleRoomsHandler) Handle(fbrCtx *fiber.Ctx, ctx context.Context, req *GetVisibleRoomsRequest) (*GetVisibleRoomsResponse, int, error) {
	userIDStr := fbrCtx.Get("X-User-ID")

	if userIDStr == "" {

		return nil, fiber.StatusUnauthorized, domain.ErrUnauthorized
	}
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return nil, fiber.StatusBadRequest, fmt.Errorf("Invalid user ID format")

	}

	status, rooms, err := h.usecase.Execute(ctx, userID)
	if err != nil {
		return nil, status, err
	}

	return &GetVisibleRoomsResponse{Message: " GetVisibleRooms user ", Rooms: rooms}, status, nil
}
