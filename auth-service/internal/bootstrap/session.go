package bootstrap

import (
	"auth-service/config"
	"auth-service/internal/initializer"
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

type SessionManager interface {
	GetRedisClient() *redis.Client
	CreateSession(ctx context.Context, userID, token string, userData map[string]string, duration time.Duration) error
}

func InitSessionRedis(config config.Config) SessionManager {
	return initializer.InitSessionRedis(config)
}
