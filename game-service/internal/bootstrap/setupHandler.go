package bootstrap

import (
	httpHandler "game-service/internal/api/http/handler"
	httpUsecase "game-service/internal/api/http/usecase"
	kafkaHandler "game-service/internal/api/kafka"
	wsHandler "game-service/internal/api/ws/handler"
	wsUsecase "game-service/internal/api/ws/usecase"

	pb "shared-lib/events"
)

func SetupHTTPHandlers(postgresRepository PostgresRepository, sessionManager SessionManager, kafka Messaging, roomRedisManager RoomRedisManager) map[string]interface{} {
	createdRoomeUseCase := httpUsecase.NewCreateRoomUseCase(postgresRepository)
	createdRoomeHandler := httpHandler.NewCreateRoomHandler(createdRoomeUseCase)

	joinRoomeUseCase := httpUsecase.NewJoinRoomUseCase(postgresRepository, roomRedisManager)
	joinRoomeHandler := httpHandler.NewJoinRoomHandler(joinRoomeUseCase)

	leaveRoomeUseCase := httpUsecase.NewLeaveRoomUseCase(postgresRepository, roomRedisManager)
	leaveRoomeHandler := httpHandler.NewLeaveRoomHandler(leaveRoomeUseCase)

	updateRoomeGameModeUseCase := httpUsecase.NewUpdateRoomGamemodeUseCase(postgresRepository, roomRedisManager)
	updateRoomeGameModeHandler := httpHandler.NewUpdateRoomGameModeHandler(updateRoomeGameModeUseCase)
	return map[string]interface{}{
		"create-room":           createdRoomeHandler,
		"join-room":             joinRoomeHandler,
		"leave-room":            leaveRoomeHandler,
		"update-room-game-mode": updateRoomeGameModeHandler,
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
