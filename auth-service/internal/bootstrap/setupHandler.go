package bootstrap

import (
	authHandler "auth-service/internal/api/handler"
	authUsecase "auth-service/internal/api/usecase"
)

func SetupHTTPHandlers(postgresRepository PostgresRepository) map[string]interface{} {

	signUpUseCase := authUsecase.NewSignUpUseCase(postgresRepository)
	signUpHandler := authHandler.NewSignUpHandler(signUpUseCase)

	return map[string]interface{}{
		"signup": signUpHandler,
	}
}
