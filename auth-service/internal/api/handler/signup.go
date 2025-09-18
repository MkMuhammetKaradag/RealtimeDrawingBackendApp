package handler

import (
	"auth-service/domain"
	"auth-service/internal/api/usecase"
	"context"
)

type SignUpRequest struct {
	Username string `json:"username" validate:"required"`
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=8"`
}

type SignUpResponse struct {
	Message string `json:"message"`
}
type SignUpHandler struct {
	usecase usecase.SignUpUseCase
}

func NewSignUpHandler(usecase usecase.SignUpUseCase) *SignUpHandler {
	return &SignUpHandler{
		usecase: usecase,
	}
}

func (h *SignUpHandler) Handle(ctx context.Context, req *SignUpRequest) (*SignUpResponse,int, error) {
	status,err := h.usecase.Execute(ctx, &domain.User{
		Username: req.Username,
		Email:    req.Email,
		Password: req.Password,
	})
	if err != nil {
		return nil,status, err
	}

	return &SignUpResponse{Message: " Please check your email"}, status,nil
}
