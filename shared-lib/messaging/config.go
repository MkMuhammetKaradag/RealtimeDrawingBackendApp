package messaging

import (
	pb "shared-lib/events" // Oluşturulan protobuf paketi
	"time"
)

// Config yapısını Kafka için uyarlayın
type KafkaConfig struct {
	Brokers              []string
	GroupID              string
	Topic                string
	ClientID             string // İsteğe bağlı, Kafka loglarında görünür
	QueueDurable         bool   // Kafka'da topic'ler varsayılan olarak kalıcıdır
	QueueAutoDelete      bool   // Kafka'da bu kavram yoktur
	EnableRetry          bool
	MaxRetries           int
	RetryTopic           string
	DLQTopic             string
	ConnectionTimeout    time.Duration
	ServiceType          pb.ServiceType                      // Hangi servis olduğumuzu belirtir
	AllowedMessageTypes  map[pb.ServiceType][]pb.MessageType // Servisin hangi mesajları işleyebileceği
	CriticalMessageTypes []pb.MessageType                    // Hangi mesaj tipleri kritik
}

func NewDefaultConfig(kafkaBrokers []string) KafkaConfig {
	if kafkaBrokers == nil || len(kafkaBrokers) == 0 {
		kafkaBrokers = []string{"localhost:9092"}
	}

	return KafkaConfig{
		Brokers:              kafkaBrokers,
		Topic:                "main-events",
		RetryTopic:           "main-events-retry",
		DLQTopic:             "main-events-dlq",
		ServiceType:          pb.ServiceType_UNKNOWN_SERVICE,
		EnableRetry:          true,
		MaxRetries:           3,
		ConnectionTimeout:    10 * time.Second,
		CriticalMessageTypes: []pb.MessageType{},
	}
}
