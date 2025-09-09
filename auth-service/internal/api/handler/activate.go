package handler

import (
	"auth-service/internal/api/usecase"
	"context"

	"github.com/google/uuid"
)

type ActivateRequest struct {
	ActivationID   uuid.UUID `json:"activation_id" validate:"required"`
	ActivationCode string    `json:"activation_code" validate:"required"`
}

type ActivateResponse struct {
	Message string `json:"message"`
}
type ActivateHandler struct {
	usecase usecase.ActivateUseCase
}

func NewActivateHandler(usecase usecase.ActivateUseCase) *ActivateHandler {
	return &ActivateHandler{
		usecase: usecase,
	}
}

func (h *ActivateHandler) Handle(ctx context.Context, req *ActivateRequest) (*ActivateResponse, error) {
	err := h.usecase.Execute(ctx, req.ActivationID, req.ActivationCode)
	if err != nil {
		return nil, err
	}

	return &ActivateResponse{Message: "user activate"}, nil
}
