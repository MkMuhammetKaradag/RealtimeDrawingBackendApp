package initializer

import (
	"log"
	"time"

	pb "shared-lib/events"
	"shared-lib/messaging"
)

func InitMessaging() *messaging.KafkaClient {
	kafkaBrokers := []string{"localhost:9092"}

	config := messaging.KafkaConfig{
		Brokers:              kafkaBrokers,
		Topic:                "main-events", // Ana olay topic'i
		RetryTopic:           "main-events-retry",
		DLQTopic:             "main-events-dlq",
		ServiceType:          pb.ServiceType_AUTH_SERVICE,
		EnableRetry:          true,
		MaxRetries:           3,
		ConnectionTimeout:    10 * time.Second,
		CriticalMessageTypes: []pb.MessageType{pb.MessageType_AUTH_USER_CREATED},
	}

	kafkaClient, err := messaging.NewKafkaClient(config)
	if err != nil {
		log.Fatalf("kafka bağlantısı kurulamadı: %v", err)
	}
	log.Printf("Kafka Client initialized for service: %s, main topic: %s", config.ServiceType.String(), config.Topic)
	return kafkaClient
}
