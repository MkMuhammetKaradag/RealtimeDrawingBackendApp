package initializer

import (
	"user-service/config"
	"user-service/infra/session"
	"fmt"
	"log"
)

func InitSessionRedis(appConfig config.Config) *session.SessionManager {
	address := fmt.Sprintf("%s:%s", appConfig.SessionRedis.Host, appConfig.SessionRedis.Port)

	sessionManager, err := session.NewSessionManager(address, appConfig.SessionRedis.Password, appConfig.SessionRedis.DB)
	if err != nil {
		log.Fatalf("Redis connection failed: %v", err)
	}
	return sessionManager
}
