package handler

import (
	"auth-service/domain"
	"auth-service/internal/api/usecase"
	"context"

	"github.com/gofiber/fiber/v2"
)

type SignInRequest struct {
	Identifier string `json:"identifier" validate:"required"`
	Password   string `json:"password" validate:"required"`
}

type SignInResponse struct {
	User *domain.User `json:"user"`
}
type SignInHandler struct {
	usecase usecase.SignInUseCase
}

func NewSignInHandler(usecase usecase.SignInUseCase) *SignInHandler {
	return &SignInHandler{
		usecase: usecase,
	}
}

func (h *SignInHandler) Handle(fbrCtx *fiber.Ctx, ctx context.Context, req *SignInRequest) (*SignInResponse,int, error) {
	user,status, err := h.usecase.Execute(fbrCtx, ctx, req.Identifier, req.Password)
	if err != nil {
		return nil, status,err
	}

	return &SignInResponse{User: user},status, nil
}
