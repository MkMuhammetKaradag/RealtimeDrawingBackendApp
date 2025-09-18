package usecase

import (
	"context"

	"github.com/gofiber/fiber/v2"
)

type AllLogoutUseCase interface {
	Execute(fbrCtx *fiber.Ctx, ctx context.Context) (int, error)
}
type allLogoutUseCase struct {
	sessionManager SessionManager
}

func NewAllLogoutUseCase(sessionManager SessionManager) AllLogoutUseCase {
	return &allLogoutUseCase{
		sessionManager: sessionManager,
	}
}

func (u *allLogoutUseCase) Execute(fbrCtx *fiber.Ctx, ctx context.Context) (int, error) {

	cookieSessionToken := fbrCtx.Cookies("Session")

	if err := u.sessionManager.DeleteAllUserSessions(ctx, cookieSessionToken); err != nil {
		return fiber.StatusInternalServerError, err
	}

	fbrCtx.Cookie(&fiber.Cookie{
		Name:     "Session",
		Value:    "",
		Path:     "/",   // Cookie yazılırken ne verdiysen aynısı olmalı
		MaxAge:   -1,    // Negatif max-age cookie'yi siler
		Secure:   false, // HTTPS kullanıyorsan true olmalı
		HTTPOnly: true,
		SameSite: "Lax", // Cookie yazılırken ne kullandıysan aynı
	})

	return fiber.StatusOK, nil
}
