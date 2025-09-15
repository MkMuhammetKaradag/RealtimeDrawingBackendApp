package initializer

import (
	"context"
	"log"
	"time"

	pb "shared-lib/events"
	"shared-lib/messaging"
)

func InitMessaging(handlers func(context.Context, *pb.Message) error) *messaging.KafkaClient {
	kafkaBrokers := []string{"localhost:9092"}

	config := messaging.KafkaConfig{
		Brokers:           kafkaBrokers,
		Topic:             "main-events",
		RetryTopic:        "retry-events",
		DLQTopic:          "dlq-events",
		ServiceType:       pb.ServiceType_GAME_SERVICE,
		EnableRetry:       true,
		MaxRetries:        3,
		ConnectionTimeout: 10 * time.Second,
		AllowedMessageTypes: map[pb.ServiceType][]pb.MessageType{
			pb.ServiceType_GAME_SERVICE: {
				pb.MessageType_AUTH_USER_CREATED,
			},
			pb.ServiceType_RETRY_SERVICE: {
				pb.MessageType_AUTH_USER_CREATED,
			},
		},
		CriticalMessageTypes: []pb.MessageType{pb.MessageType_AUTH_USER_CREATED},
	}

	kafkaClient, err := messaging.NewKafkaClient(config)
	if err != nil {
		log.Fatalf("kafka bağlantısı kurulamadı: %v", err)
	}
	log.Printf("Kafka Client initialized for service: %s, main topic: %s", config.ServiceType.String(), config.Topic)

	ctx, cancel := context.WithCancel(context.Background())
	// Consumer'ları başlat
	// Ana Consumer
	go func() {
		log.Println("Starting Kafka consumer for main-events...")
		groupID := config.ServiceType.String() + "-main-group"
		topic := config.Topic
		if err := kafkaClient.ConsumeMessages(ctx, handlers, &topic, &groupID); err != nil {
			log.Printf("Main consumer error: %v", err)
			cancel()
		}
	}()

	// Retry Consumer
	go func() {
		log.Println("Starting Kafka consumer for retry-events...")
		// Aynı KafkaClient'ı kullanarak, sadece farklı bir topic ve grup adı veriyoruz.
		groupID := config.ServiceType.String() + "-retry-group"
		topic := config.RetryTopic
		if err := kafkaClient.ConsumeMessages(ctx, handlers, &topic, &groupID); err != nil {
			log.Printf("Retry consumer error: %v", err)
			cancel()
		}
	}()

	// DLQ Consumer
	go func() {
		log.Println("Starting DLQ recovery consumer...")
		if err := kafkaClient.ConsumeDLQWithRecovery(ctx, handlers); err != nil {
			log.Printf("DLQ consumer error: %v", err)
		}
	}()

	return kafkaClient
}
