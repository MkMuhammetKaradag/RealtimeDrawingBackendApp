package handler

import (
	"auth-service/internal/api/usecase"
	"context"
	"fmt"

	"github.com/gofiber/fiber/v2"
)

type RefreshTokenRequest struct {
}

type RefreshTokenResponse struct {
	Message string `json:"message"`
}

type RefreshTokenHandler struct {
	usecase usecase.RefreshTokenUseCase
}

func NewRefreshTokenHandler(usecase usecase.RefreshTokenUseCase) *RefreshTokenHandler {
	return &RefreshTokenHandler{
		usecase: usecase,
	}
}

func (h *RefreshTokenHandler) Handle(fbrCtx *fiber.Ctx, ctx context.Context, req *RefreshTokenRequest) (*RefreshTokenResponse, int,error) {
	token,status, err := h.usecase.Execute(fbrCtx, ctx)
	if err != nil {
		return nil,status, err
	}
	fmt.Println("token", token)

	return &RefreshTokenResponse{Message: "RefreshToken user "},status, nil
}
