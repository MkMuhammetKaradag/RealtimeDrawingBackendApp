package handler

import (
	"context"
	"fmt"
	"game-service/internal/api/usecase"
	pb "shared-lib/events"

	"github.com/google/uuid"
)

type CreatedUserHandler struct {
	usecase usecase.CreateUserUseCase
}

func NewCreatedUserHandler(createdUserUsecase usecase.CreateUserUseCase) *CreatedUserHandler {
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
