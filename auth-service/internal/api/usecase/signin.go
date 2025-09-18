package usecase

import (
	"auth-service/domain"
	"context"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type signInUseCase struct {
	postgresRepository PostgresRepository
	sesionManager      SessionManager
}
type SignInUseCase interface {
	Execute(fbrCtx *fiber.Ctx, ctx context.Context, identifier, password string) (*domain.User, error)
}

func NewSignInUseCase(repository PostgresRepository, sesionManager SessionManager) SignInUseCase {
	return &signInUseCase{
		postgresRepository: repository,
		sesionManager:      sesionManager,
	}
}

func (u *signInUseCase) Execute(fbrCtx *fiber.Ctx, ctx context.Context, identifier, password string) (*domain.User, error) {
	user, err := u.postgresRepository.SignIn(ctx, identifier, password)
	if err != nil {
		return nil, err
	}
	sessionToken := uuid.New().String()
	device := fbrCtx.Get("User-Agent")
	ip := fbrCtx.IP()

	userData := &domain.SessionData{
		UserID:    user.ID,
		Device:    device,
		Username:  "boş",
		Ip:        ip,
		CreatedAt: time.Now(),
	}
	if err := u.sesionManager.CreateSession(ctx, sessionToken, userData, 24*time.Hour); err != nil {
		return nil, err
	}
	fbrCtx.Cookie(&fiber.Cookie{
		Name:     "Session",
		Value:    sessionToken,
		Path:     "/",
		MaxAge:   60 * 60 * 24,
		HTTPOnly: true,
		Secure:   false,
		SameSite: "Lax",
	})

	return user, nil
}
