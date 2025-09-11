package usecase

import (
	"context"
	// "fmt"
	"log"

	pb "shared-lib/user-events"

	"github.com/google/uuid"
)

type ActivateUseCase interface {
	Execute(ctx context.Context, activationID uuid.UUID, activationCode string) error
}
type activateUseCase struct {
	postgresRepository PostgresRepository
	kafka              Messaging
}

func NewActivateUseCase(repository PostgresRepository, kafka Messaging) ActivateUseCase {
	return &activateUseCase{
		postgresRepository: repository,
		kafka:              kafka,
	}
}

func (u *activateUseCase) Execute(ctx context.Context, activationID uuid.UUID, activationCode string) error {

	userCreatedData := &pb.UserCreatedData{
		UserId:   "tes-id",
		Username: "test",
		Email:    "test@mail.com",
	}
	message := &pb.Message{
		Type:        pb.MessageType_AUTH_USER_CREATED, // Auth tarafından oluşturulan bir kullanıcı
		FromService: pb.ServiceType_AUTH_SERVICE,
		ToServices:  []pb.ServiceType{pb.ServiceType_USER_SERVICE, pb.ServiceType_CHAT_SERVICE}, // Bu mesaj USER_SERVICE için
		Payload:     &pb.Message_UserCreatedData{UserCreatedData: userCreatedData},
	}
	err := u.kafka.PublishMessage(ctx, message)
	if err != nil {
		log.Printf("Failed to publish UserCreated event for user %v", err)
	} else {
		log.Printf("UserCreated event published ")
	}

	// userUpdatedData := &pb.UserUpdatedData{
	// 	Username: "test-update",
	// }
	// message = &pb.Message{
	// 	Type:        pb.MessageType_USER_UPDATED, // Auth tarafından oluşturulan bir kullanıcı
	// 	FromService: pb.ServiceType_AUTH_SERVICE,
	// 	ToServices:  []pb.ServiceType{pb.ServiceType_CHAT_SERVICE}, // Bu mesaj Chat-service  için
	// 	Payload:     &pb.Message_UserUpdatedData{UserUpdatedData: userUpdatedData},
	// }
	// err = u.kafka.PublishMessage(ctx, message)
	// if err != nil {
	// 	log.Printf("Failed to publish UserUpdated event for user %v", err)
	// } else {
	// 	log.Printf("UserUpdated event published ")
	// }

	// fmt.Println()
	// fmt.Println()
	// fmt.Println()
	// user, err := u.postgresRepository.Activate(ctx, activationID, activationCode)
	// if err != nil {
	// 	return err
	// }
	// fmt.Println("user:", user)

	return nil
}
