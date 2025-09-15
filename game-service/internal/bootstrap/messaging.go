package bootstrap

import (
	"context"
	"fmt"
	"game-service/config"
	"game-service/internal/initializer"

	pb "shared-lib/events"
)

type Messaging interface {
	Close() error
	PublishMessage(ctx context.Context, msg *pb.Message) error
}

type MessageHandler interface {
	Handle(ctx context.Context, msg *pb.Message) error
}

func SetupMessaging(handlers map[pb.MessageType]MessageHandler, config config.Config) Messaging {

	messageRouter := func(ctx context.Context, msg *pb.Message) error {
		handler, ok := handlers[msg.Type]
		fmt.Println("messaj geldi-msg.type", msg.Type)
		if !ok {
			return nil
		}
		return handler.Handle(ctx, msg)
	}

	return initializer.InitMessaging(messageRouter)
}
