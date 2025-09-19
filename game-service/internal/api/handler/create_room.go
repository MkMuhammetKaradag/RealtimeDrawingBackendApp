package handler

import (
	"context"
	"fmt"
	"game-service/domain"
	"game-service/internal/api/usecase"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type CreateRoomRequest struct {
	RoomName   string `json:"room_name"`
	MaxPlayers int    `json:"max_players"`
	GameModeID int    `json:"game_mode_id"`
	IsPrivate  bool   `json:"is_private"`
	RoomCode   string `json:"room_code"`
}

type CreateRoomResponse struct {
	Message string `json:"message"`
}

type CreateRoomHandler struct {
	usecase usecase.CreateRoomUseCase
}

func NewCreateRoomHandler(usecase usecase.CreateRoomUseCase) *CreateRoomHandler {
	return &CreateRoomHandler{
		usecase: usecase,
	}
}

func (h *CreateRoomHandler) Handle(fbrCtx *fiber.Ctx, ctx context.Context, req *CreateRoomRequest) (*CreateRoomResponse, int, error) {
	userIDStr := fbrCtx.Get("X-User-ID")

	if userIDStr == "" {
		// Bu hatayı genellikle görmemelisiniz, çünkü AuthGuard'dan geçmiştir.
		// Ancak beklenmedik durumlara karşı kontrol etmek iyi bir pratiktir.
		return nil, fiber.StatusUnauthorized, domain.ErrUnauthorized
	}
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return nil, fiber.StatusBadRequest, fmt.Errorf("Invalid user ID format")

	}
	data := domain.Room{
		RoomName:   req.RoomName,
		MaxPlayers: req.MaxPlayers,
		GameModeID: req.GameModeID,
		IsPrivate:  req.IsPrivate,
		RoomCode:   req.RoomCode,
		CreatorID:  userID,
	}
	status, err := h.usecase.Execute(ctx, data)
	if err != nil {
		return nil, status, err
	}

	return &CreateRoomResponse{Message: "CreateRoom user "}, status, nil
}
