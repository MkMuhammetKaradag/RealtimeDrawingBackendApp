package handler

import (
	"auth-service/internal/api/usecase"
	"context"

	"github.com/gofiber/fiber/v2"
)

type AllLogoutRequest struct {
}

type AllLogoutResponse struct {
	Message string `json:"message"`
}

type AllLogoutHandler struct {
	usecase usecase.AllLogoutUseCase
}

func NewAllLogoutHandler(usecase usecase.AllLogoutUseCase) *AllLogoutHandler {
	return &AllLogoutHandler{
		usecase: usecase,
	}
}

func (h *AllLogoutHandler) Handle(fbrCtx *fiber.Ctx, ctx context.Context, req *AllLogoutRequest) (*AllLogoutResponse, error) {
	err := h.usecase.Execute(fbrCtx, ctx)
	if err != nil {
		return nil, err
	}

	return &AllLogoutResponse{Message: "all logout user "}, nil
}
