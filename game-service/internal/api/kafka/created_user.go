package handler

import (
	"context"
	"fmt"
	httpUsecase "game-service/internal/api/http/usecase"
	pb "shared-lib/events"

	"github.com/google/uuid"
)

type CreatedUserHandler struct {
	usecase httpUsecase.CreateUserUseCase
}

func NewCreatedUserHandler(createdUserUsecase httpUsecase.CreateUserUseCase) *CreatedUserHandler {
	return &CreatedUserHandler{
		usecase: createdUserUsecase,
	}
}

func (h *CreatedUserHandler) Handle(ctx context.Context, msg *pb.Message) error {

	data := msg.GetUserCreatedData()
	if data == nil {
		return fmt.Errorf("UserCreatedData payload is nil for message ID: %s", msg.Id)
	}
	idUUID, err := uuid.Parse(data.UserId)
	if err != nil {
		return fmt.Errorf("UserCreatedData payload is nil for message ID: %s", msg.Id)
	}
	// ctx := context.Background()
	return h.usecase.Execute(ctx, idUUID, data.Username, data.Email)
}
