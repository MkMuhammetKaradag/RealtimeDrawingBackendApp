package bootstrap

import (
	"game-service/config"
	"game-service/internal/initializer"

	"github.com/redis/go-redis/v9"
)

type SessionManager interface {
	GetRedisClient() *redis.Client
}

func InitSessionRedis(config config.Config) SessionManager {
	return initializer.InitSessionRedis(config)
}
