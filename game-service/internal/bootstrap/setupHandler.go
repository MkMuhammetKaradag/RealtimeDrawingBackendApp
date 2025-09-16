package bootstrap

import (
	"game-service/internal/api/game"
	kafkaHandler "game-service/internal/api/kafka"
	"game-service/internal/api/usecase"
	wsHandler "game-service/internal/api/ws"
	pb "shared-lib/events"
)

func SetupHTTPHandlers(postgresRepository PostgresRepository, sessionManager SessionManager, kafka Messaging) map[string]interface{} {

	return map[string]interface{}{}
}
func SetupMessageHandlers(postgresRepository PostgresRepository) map[pb.MessageType]MessageHandler {
	createdUserUseCase := usecase.NewCreateUserUseCase(postgresRepository)
	createdUserHandler := kafkaHandler.NewCreatedUserHandler(createdUserUseCase)

	return map[pb.MessageType]MessageHandler{
		pb.MessageType_AUTH_USER_CREATED: createdUserHandler,
	}
}
func SetupWSHandlers(postgresRepository PostgresRepository, sessionManager SessionManager) map[string]interface{} {
	roomManager := game.NewRoomManager()

	gameWebSocketHandler := wsHandler.NewWebSocketHandler(roomManager)
	return map[string]interface{}{
		"game": gameWebSocketHandler,
	}
}
