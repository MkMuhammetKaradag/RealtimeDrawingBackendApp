package bootstrap

import (
	"game-service/config"
	"game-service/internal/initializer"
)

type PostgresRepository interface {
	Close() error
}

func InitDatabase(config config.Config) PostgresRepository {
	return initializer.InitDatabase(config)
}
