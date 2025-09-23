package initializer

import (
	"fmt"
	"game-service/config"
	"game-service/infra/redis"
)

func InitRoomRedis(appConfig config.Config) *redis.RedisManager {
	address := fmt.Sprintf("%s:%s", appConfig.SessionRedis.Host, appConfig.SessionRedis.Port)

	redisManager := redis.NewRedisManager(address, appConfig.SessionRedis.Password, appConfig.SessionRedis.DB)

	return redisManager
}
