package bootstrap

import (
	httpHandler "game-service/internal/api/http/handler"
	httpUsecase "game-service/internal/api/http/usecase"
	kafkaHandler "game-service/internal/api/kafka"
	wsHandler "game-service/internal/api/ws/handler"
	wsUsecase "game-service/internal/api/ws/usecase"

	pb "shared-lib/events"
)

func SetupHTTPHandlers(postgresRepository PostgresRepository, sessionManager SessionManager, kafka Messaging) map[string]interface{} {
	createdRoomeUseCase := httpUsecase.NewCreateRoomUseCase(postgresRepository)
	createdRoomeHandler := httpHandler.NewCreateRoomHandler(createdRoomeUseCase)
	return map[string]interface{}{
		"create-room": createdRoomeHandler,
	}
}
func SetupMessageHandlers(postgresRepository PostgresRepository) map[pb.MessageType]MessageHandler {
	createdUserUseCase := httpUsecase.NewCreateUserUseCase(postgresRepository)
	createdUserHandler := kafkaHandler.NewCreatedUserHandler(createdUserUseCase)

	return map[pb.MessageType]MessageHandler{
		pb.MessageType_AUTH_USER_CREATED: createdUserHandler,
	}
}
func SetupWSHandlers(postgresRepository PostgresRepository, wsHub Hub) map[string]interface{} {
	roomManager := wsUsecase.NewRoomManagerUseCase(wsHub, postgresRepository)

	roomManagerHandler := wsHandler.NewWebSocketRoomHandler(roomManager)
	return map[string]interface{}{
		"room-connect": roomManagerHandler,
	}
}
