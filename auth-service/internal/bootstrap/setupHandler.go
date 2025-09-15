package bootstrap

import (
	authHandler "auth-service/internal/api/handler"
	authUsecase "auth-service/internal/api/usecase"
	pb "shared-lib/events"
)

func SetupHTTPHandlers(postgresRepository PostgresRepository, sessionManager SessionManager, kafka Messaging) map[string]interface{} {

	signUpUseCase := authUsecase.NewSignUpUseCase(postgresRepository)
	signUpHandler := authHandler.NewSignUpHandler(signUpUseCase)

	activateUseCase := authUsecase.NewActivateUseCase(postgresRepository, kafka)
	activateHandler := authHandler.NewActivateHandler(activateUseCase)

	signInUseCase := authUsecase.NewSignInUseCase(postgresRepository, sessionManager)
	signInHandler := authHandler.NewSignInHandler(signInUseCase)

	logoutUseCase := authUsecase.NewLogoutUseCase(sessionManager)
	logoutHandler := authHandler.NewLogoutHandler(logoutUseCase)

	allLogoutUseCase := authUsecase.NewAllLogoutUseCase(sessionManager)
	allLogoutHandler := authHandler.NewAllLogoutHandler(allLogoutUseCase)

	return map[string]interface{}{
		"signup":     signUpHandler,
		"activate":   activateHandler,
		"signin":     signInHandler,
		"logout":     logoutHandler,
		"all-logout": allLogoutHandler,
	}
}
func SetupMessageHandlers() map[pb.MessageType]MessageHandler {
	return map[pb.MessageType]MessageHandler{}
}
