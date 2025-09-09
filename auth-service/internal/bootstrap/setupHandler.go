package bootstrap

import (
	authHandler "auth-service/internal/api/handler"
	authUsecase "auth-service/internal/api/usecase"
)

func SetupHTTPHandlers(postgresRepository PostgresRepository, sessionManager SessionManager) map[string]interface{} {

	signUpUseCase := authUsecase.NewSignUpUseCase(postgresRepository)
	signUpHandler := authHandler.NewSignUpHandler(signUpUseCase)

	activateUseCase := authUsecase.NewActivateUseCase(postgresRepository)
	activateHandler := authHandler.NewActivateHandler(activateUseCase)

	signInUseCase := authUsecase.NewSignInUseCase(postgresRepository, sessionManager)
	signInHandler := authHandler.NewSignInHandler(signInUseCase)

	return map[string]interface{}{
		"signup":   signUpHandler,
		"activate": activateHandler,
		"signin":   signInHandler,
	}
}
