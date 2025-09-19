package handler

import (
	"auth-service/domain"
	"auth-service/internal/api/usecase"
	"context"
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
)

type ValidateTokenRequest struct {
}

type ValidateTokenResponse struct {
	UserID string `json:"user_id"`
}
type ValidateTokenHandler struct {
	usecase usecase.ValidateTokenUseCase
}

func NewValidateTokenHandler(usecase usecase.ValidateTokenUseCase) *ValidateTokenHandler {
	return &ValidateTokenHandler{
		usecase: usecase,
	}
}

func (h *ValidateTokenHandler) Handle(fbrCtx *fiber.Ctx, ctx context.Context, req *ValidateTokenRequest) (*ValidateTokenResponse, int, error) {
	// Gateway'den veya direkt client'tan gelen token'ı al
	fmt.Println("buraya istek geldi-")
	token := fbrCtx.Cookies("Session")
	if token == "" {
		return nil, fiber.StatusUnauthorized, domain.ErrSessionNotFound

	}

	sessionData, ttl, status, err := h.usecase.Execute(ctx, token)
	if err != nil {

		return nil, status, err
	}

	// Yanıt başlığına yenileme sinyali ekle
	refreshThreshold := 1 * time.Minute
	fbrCtx.Response().Header.Set("X-User-ID", sessionData.UserID)

	if ttl > 0 && ttl <= refreshThreshold {
		fmt.Println("tll zaman daralmış")
		fbrCtx.Response().Header.Set("x-refresh-needed", "true")
	}

	return &ValidateTokenResponse{UserID: sessionData.UserID}, status, nil
}
