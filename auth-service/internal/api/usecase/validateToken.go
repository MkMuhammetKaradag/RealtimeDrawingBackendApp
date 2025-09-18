package usecase

import (
	"auth-service/domain"
	"context"
	"errors"
	"time"

	"github.com/gofiber/fiber/v2"
)

type validateTokenUseCase struct {
	sessionManager SessionManager
}

type ValidateTokenUseCase interface {
	Execute(ctx context.Context, token string) (*domain.SessionData, time.Duration, int, error)
}

func NewValidateTokenUseCase(sessionManager SessionManager) ValidateTokenUseCase {
	return &validateTokenUseCase{
		sessionManager: sessionManager,
	}
}

func (u *validateTokenUseCase) Execute(ctx context.Context, token string) (*domain.SessionData, time.Duration, int, error) {
	// 1. Tokenın kalan süresini al
	ttl, err := u.sessionManager.GetTokenTTL(ctx, token)
	if err != nil {
		return nil, 0, fiber.StatusUnauthorized, err
	}
	if ttl <= 0 {
		// Süresi dolmuşsa hata dön
		return nil, 0, fiber.StatusUnauthorized, domain.ErrSessionExpired
	}

	// 2. Token verilerini al
	sessionData, err := u.sessionManager.GetSession(ctx, token)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrUnauthorized):
			return nil, 0, fiber.StatusUnauthorized, err

		default:
			return nil, 0, fiber.StatusInternalServerError, err
		}

	}
	if sessionData == nil {
		// Oturum bulunamazsa hata dön
		return nil, 0, fiber.StatusUnauthorized, domain.ErrSessionNotFound
	}

	return sessionData, ttl, fiber.StatusOK, nil
}
