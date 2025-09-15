package handler

import (
	"auth-service/internal/api/usecase"
	"context"

	"github.com/gofiber/fiber/v2"
)

type LogoutRequest struct {
}

type LogoutResponse struct {
	Message string `json:"message"`
}

type LogoutHandler struct {
	usecase usecase.LogoutUseCase
}

func NewLogoutHandler(usecase usecase.LogoutUseCase) *LogoutHandler {
	return &LogoutHandler{
		usecase: usecase,
	}
}

func (h *LogoutHandler) Handle(fbrCtx *fiber.Ctx, ctx context.Context, req *LogoutRequest) (*LogoutResponse, error) {
	err := h.usecase.Execute(fbrCtx, ctx)
	if err != nil {
		return nil, err
	}

	return &LogoutResponse{Message: "logout user "}, nil
}
