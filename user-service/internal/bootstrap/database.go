package bootstrap

import (
	"user-service/config"
	"user-service/internal/initializer"
)

type PostgresRepository interface {
	Close() error
}

func InitDatabase(config config.Config) PostgresRepository {
	return initializer.InitDatabase(config)
}
