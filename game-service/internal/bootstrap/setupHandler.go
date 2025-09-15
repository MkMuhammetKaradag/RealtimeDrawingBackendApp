package bootstrap

import (
	kafkaHandler "game-service/internal/api/kafka"
	"game-service/internal/api/usecase"
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
