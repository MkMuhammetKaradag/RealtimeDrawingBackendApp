package main

import (
	"user-service/config"
	"user-service/internal/bootstrap"
	_ "user-service/log"

	"go.uber.org/zap"
)

func main() {
	appConfig := config.Read()
	defer zap.L().Sync()
	zap.L().Info("app starting...", zap.String("app name", appConfig.App.Name))

	app := bootstrap.NewApp(appConfig)

	app.Start()
}
