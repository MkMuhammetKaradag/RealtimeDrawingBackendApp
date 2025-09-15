package bootstrap

import (
	"auth-service/config"
	"auth-service/domain"
	"auth-service/internal/initializer"
	"context"
	"time"
	//"github.com/redis/go-redis/v9"
)

type SessionManager interface {
	//GetRedisClient() *redis.Client
	CreateSession(ctx context.Context, token string, data *domain.SessionData, duration time.Duration) error
	DeleteSession(ctx context.Context, token string) error
	DeleteAllUserSessions(ctx context.Context, token string) error
}

func InitSessionRedis(config config.Config) SessionManager {
	return initializer.InitSessionRedis(config)
}
