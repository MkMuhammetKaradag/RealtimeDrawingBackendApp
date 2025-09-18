package usecase

import (
	"auth-service/domain"
	"context"
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type refreshTokenUseCase struct {
	sessionManager SessionManager
}

type RefreshTokenUseCase interface {
	Execute(fbrCtx *fiber.Ctx, ctx context.Context) (string, int, error)
}

func NewRefreshTokenUseCase(sessionManager SessionManager) RefreshTokenUseCase {
	return &refreshTokenUseCase{
		sessionManager: sessionManager,
	}
}

func (u *refreshTokenUseCase) Execute(fbrCtx *fiber.Ctx, ctx context.Context) (string, int, error) {
	// 1. Eski token'ı al
	oldToken := fbrCtx.Cookies("Session")
	if oldToken == "" {
		return "", fiber.StatusUnauthorized, fmt.Errorf("no session token found")
	}

	// 2. Oturum verilerini al ve kontrol et
	sessionData, err := u.sessionManager.GetSession(ctx, oldToken)
	if err != nil || sessionData == nil {
		return "", fiber.StatusUnauthorized, fmt.Errorf("session expired or invalid, please sign in again")
	}

	// 3. Cihaz ve IP kontrolü
	device := fbrCtx.Get("User-Agent")
	ip := fbrCtx.IP()

	if sessionData.Device != device || sessionData.Ip != ip {
		return "", fiber.StatusUnauthorized, fmt.Errorf("device or IP mismatch, please sign in again")
	}

	// 4. Maksimum oturum süresi kontrolü (1 hafta)
	maxSessionDuration := 7 * 24 * time.Hour
	if time.Since(sessionData.CreatedAt) > maxSessionDuration {
		// Oturumu sil ve hata döndür
		_ = u.sessionManager.DeleteSession(ctx, oldToken) // Hata yönetimi burada daha detaylı olabilir
		return "", fiber.StatusUnauthorized, fmt.Errorf("maximum session duration exceeded, please sign in again")
	}

	// 5. Yeni token oluştur ve oturumu güncelle
	newToken := uuid.New().String()
	newSessionData := &domain.SessionData{
		UserID:    sessionData.UserID,
		Username:  sessionData.Username,
		Device:    device,
		Ip:        ip,
		CreatedAt: sessionData.CreatedAt, // Oluşturulma zamanı değişmez
	}
	// Oturumu 24 saat daha uzat
	if err := u.sessionManager.UpdateSession(ctx, oldToken, newToken, newSessionData, 2*time.Minute); err != nil {
		return "", fiber.StatusInternalServerError, fmt.Errorf("failed to refresh session")
	}

	// 6. Yeni token'ı cookie olarak ayarla
	fbrCtx.Cookie(&fiber.Cookie{
		Name:     "Session",
		Value:    newToken,
		Path:     "/",
		MaxAge:   60 * 60 * 24, // 24 saat
		HTTPOnly: true,
		Secure:   false,
		SameSite: "Lax",
	})

	return newToken, fiber.StatusOK, nil
}
