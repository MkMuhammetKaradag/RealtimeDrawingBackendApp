package bootstrap

import (
	"context"
	"game-service/config"
	"game-service/internal/initializer"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

type SessionManager interface {
	GetRedisClient() *redis.Client
}

func InitSessionRedis(config config.Config) SessionManager {
	return initializer.InitSessionRedis(config)
}
func InitRoomRedis(config config.Config) RoomRedisManager {
	return initializer.InitRoomRedis(config)
}

type RoomRedisManager interface {
	PublishMessage(ctx context.Context, roomID uuid.UUID, msgType string, dataContent interface{})
}
