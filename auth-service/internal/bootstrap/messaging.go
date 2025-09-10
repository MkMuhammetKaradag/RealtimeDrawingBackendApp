package bootstrap

import (
	"auth-service/config"
	"auth-service/internal/initializer"
	"context"

	pb "shared-lib/user-events"
)

type Messaging interface {
	Close() error
	PublishMessage(ctx context.Context, msg *pb.Message) error
}

type MessageHandler interface {
	Handle(msg *pb.Message) error
}

func SetupMessaging(handlers map[pb.MessageType]MessageHandler, config config.Config) Messaging {

	return initializer.InitMessaging()
}
