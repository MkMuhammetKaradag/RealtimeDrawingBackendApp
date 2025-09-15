package main

import (
	"game-service/config"
	"game-service/internal/bootstrap"
	_ "game-service/log"

	"go.uber.org/zap"
)

func main() {
	appConfig := config.Read()
	defer zap.L().Sync()
	zap.L().Info("app starting...", zap.String("app name", appConfig.App.Name))

	app := bootstrap.NewApp(appConfig)

	app.Start()
}
