package usecase

import (
	"context"
	"fmt"

	"github.com/gofiber/fiber/v2"
)

type LogoutUseCase interface {
	Execute(fbrCtx *fiber.Ctx, ctx context.Context) error
}

type logoutUseCase struct {
	sessionManager SessionManager
}

func NewLogoutUseCase(sessionMng SessionManager) LogoutUseCase {
	return &logoutUseCase{
		sessionManager: sessionMng,
	}
}

func (u *logoutUseCase) Execute(fbrCtx *fiber.Ctx, ctx context.Context) error {
	cookieSessionToken := fbrCtx.Cookies("Session")

	// // session, err := u.sessionManager.GetSession(ctx, cookieSessionToken)
	// // if err != nil {
	// // 	return fmt.Errorf("failed to get session: %w", err)
	// // }
	fmt.Println("user session:", cookieSessionToken)

	// // Oturumu sil
	// if err := u.sessionManager.DeleteSession(ctx, cookieSessionToken); err != nil {
	// 	return fmt.Errorf("failed to delete session: %w", err)
	// }

	// // Cookie'yi temizle
	// // Bu işlem, kullanıcının oturumunu sonlandırır ve tarayıcıdaki cookie'yi kaldırır

	// fbrCtx.Cookie(&fiber.Cookie{
	// 	Name:     "session_token",
	// 	Value:    "",
	// 	Path:     "/",   // Cookie yazılırken ne verdiysen aynısı olmalı
	// 	MaxAge:   -1,    // Negatif max-age cookie'yi siler
	// 	Secure:   false, // HTTPS kullanıyorsan true olmalı
	// 	HTTPOnly: true,
	// 	SameSite: "Lax", // Cookie yazılırken ne kullandıysan aynı
	// })

	return nil
}
