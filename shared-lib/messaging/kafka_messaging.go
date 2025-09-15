package messaging

import (
	"context"
	"fmt"
	"log"
	"math"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	pb "shared-lib/events" // Oluşturulan protobuf paketi
)

type QuietKafkaLogger struct{}

// Printf implements the kafka.LoggerFunc interface
func (l QuietKafkaLogger) Printf(format string, v ...interface{}) {
	// Timeout ve request time limit mesajlarını filtrele
	msg := strings.ToLower(format)

	// Bu log mesajlarını sessizce atla
	if strings.Contains(msg, "no messages received from kafka") ||
		strings.Contains(msg, "request timed out") ||
		strings.Contains(msg, "request exceeded") ||
		strings.Contains(msg, "allocated time") {
		return
	}

	// Diğer log mesajlarını normal şekilde yazdır
	log.Printf(format, v...)
}

// KafkaClient yapısı
type KafkaClient struct {
	config      KafkaConfig
	producer    *kafka.Writer
	mu          sync.Mutex
	closed      bool
	serviceType pb.ServiceType
}

// NewKafkaClient fonksiyonunu güncelleyin
func NewKafkaClient(config KafkaConfig) (*KafkaClient, error) {
	kc := &KafkaClient{
		config:      config,
		serviceType: config.ServiceType,
	}

	// Topic'leri oluştur
	if err := kc.createTopicsIfNotExists(); err != nil {
		return nil, fmt.Errorf("failed to create topics: %w", err)
	}
	quietLogger := QuietKafkaLogger{}
	// Producer'ı oluştur
	producer := &kafka.Writer{
		Addr:         kafka.TCP(config.Brokers...),
		Topic:        config.Topic,
		Balancer:     &kafka.LeastBytes{},
		WriteTimeout: 10 * time.Second,
		RequiredAcks: kafka.RequireAll,
		BatchSize:    100,
		BatchBytes:   1048576,
		BatchTimeout: 1 * time.Second,
		MaxAttempts:  3,
		//Logger:       kafka.LoggerFunc(log.Printf),
		Logger:      kafka.LoggerFunc(quietLogger.Printf),
		ErrorLogger: kafka.LoggerFunc(log.Printf),
	}

	kc.producer = producer

	log.Printf("Kafka Client initialized for service: %s, main topic: %s", config.ServiceType.String(), config.Topic)
	return kc, nil
}
func (kc *KafkaClient) createTopicsIfNotExists() error {
	conn, err := kafka.Dial("tcp", kc.config.Brokers[0])
	if err != nil {
		return fmt.Errorf("failed to connect to Kafka: %w", err)
	}
	defer conn.Close()

	controller, err := conn.Controller()
	if err != nil {
		return fmt.Errorf("failed to get controller: %w", err)
	}

	controllerConn, err := kafka.Dial("tcp", fmt.Sprintf("%s:%d", controller.Host, controller.Port))
	if err != nil {
		return fmt.Errorf("failed to connect to controller: %w", err)
	}
	defer controllerConn.Close()

	// Oluşturulacak topic'ler
	topics := []kafka.TopicConfig{
		{
			Topic:             kc.config.Topic, // main-events
			NumPartitions:     3,
			ReplicationFactor: 1,
		},
	}

	// Retry topic varsa ekle
	if kc.config.EnableRetry && kc.config.RetryTopic != "" {
		topics = append(topics, kafka.TopicConfig{
			Topic:             kc.config.RetryTopic,
			NumPartitions:     3,
			ReplicationFactor: 1,
		})
	}

	// DLQ topic varsa ekle
	if kc.config.DLQTopic != "" {
		topics = append(topics, kafka.TopicConfig{
			Topic:             kc.config.DLQTopic,
			NumPartitions:     1,
			ReplicationFactor: 1,
		})
	}

	err = controllerConn.CreateTopics(topics...)
	if err != nil {
		// Topic zaten varsa hata vermemeli
		if !strings.Contains(err.Error(), "already exists") {
			return fmt.Errorf("failed to create topics: %w", err)
		}
		log.Println("Some topics already exist, continuing...")
	}

	// Oluşturulan topic'leri logla
	for _, topic := range topics {
		log.Printf("Topic '%s' created or already exists", topic.Topic)
	}

	return nil
}

func (kc *KafkaClient) Close() error {
	kc.mu.Lock()
	defer kc.mu.Unlock()

	kc.closed = true

	if kc.producer != nil {
		if err := kc.producer.Close(); err != nil {
			return fmt.Errorf("failed to close Kafka producer: %w", err)
		}
	}
	log.Println("Kafka Client closed.")
	return nil
}

func (kc *KafkaClient) isCriticalMessageType(msgType pb.MessageType) bool {
	for _, t := range kc.config.CriticalMessageTypes {
		if t == msgType {
			return true
		}
	}
	return false
}

func (kc *KafkaClient) isAllowedMessageType(svcType pb.ServiceType, msgType pb.MessageType) bool {
	allowed, ok := kc.config.AllowedMessageTypes[svcType]
	if !ok {
		return false
	}
	for _, t := range allowed {
		if t == msgType {
			return true
		}
	}
	return false
}

// PublishMessage fonksiyonu, RabbitMQ'daki PublishMessage'a benzer işlevsellik sunar
func (kc *KafkaClient) PublishMessage(ctx context.Context, msg *pb.Message) error {
	if msg.Id == "" {
		msg.Id = uuid.New().String()
	}
	if msg.Created == nil {
		msg.Created = timestamppb.Now()
	}
	msg.FromService = kc.serviceType

	isCritical := kc.isCriticalMessageType(msg.Type)
	if isCritical {
		msg.Critical = true
	}

	// Birden fazla servise gönderim simülasyonu
	// Kafka'da "routing key" olmadığı için, mesajı ana topice gönderip
	// tüketicilerin kendi filtrelemesini yapmasını bekleriz.
	// Ancak kritik mesajlar için farklı topic'lere yönlendirme yapılabilir.

	// Eğer mesaj belirli servislere gitmeli ise, bu bilgiyi mesajın kendisinde taşırız.
	// Tek bir topic'e gönderip, tüketicinin mesajı alıp almayacağına karar vermesi daha basittir.

	// Eğer hiç ToServices belirtilmemişse veya birden fazla servise gitmesi bekleniyorsa,
	// ana topic'e göndeririz.
	targetTopic := kc.config.Topic // Varsayılan olarak ana topic

	// Eğer mesaj sadece tek bir servise özel ve bu servis kritik bir mesaj bekliyorsa,
	// veya retry/DLQ senaryoları için, farklı topic'ler kullanılabilir.
	// Bu karmaşıklık genelde Kafka'da basit bir topic yapısıyla giderilir.
	// Bu örnekte, basitlik adına tüm mesajları ana topice gönderip tüketicinin filtrelemesini sağlıyoruz.
	// Kritik mesajlar için bir `critical-events` topic'i veya retry için `retry-events` topic'i olabilir.

	messageBytes, err := proto.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal protobuf message: %w", err)
	}

	kafkaMsg := kafka.Message{
		Key:   []byte(msg.Id), // Mesaj anahtarı, aynı anahtara sahip mesajlar aynı bölüme gider
		Value: messageBytes,
		Headers: []kafka.Header{
			{Key: "EventType", Value: []byte(msg.Type.String())},
			{Key: "FromService", Value: []byte(msg.FromService.String())},
			// Diğer header'ları buraya ekleyebilirsiniz
		},
	}

	err = kc.producer.WriteMessages(ctx, kafkaMsg)
	if err != nil {
		if isCritical {
			log.Printf("Failed to publish critical message to Kafka for ID %s. Saving to storage. Error: %v", msg.Id, err)
			kc.saveCriticalMessageToStorage(msg) // Hata durumunda kalıcı depolama
		}
		return fmt.Errorf("failed to write message to Kafka topic %s: %w", targetTopic, err)
	}

	log.Printf("Published message [ID: %s, Type: %s] to topic %s", msg.Id, msg.Type.String(), targetTopic)
	return nil
}

// MessageHandler, Kafka mesajlarını işlemek için kullanılır
type MessageHandler func(context.Context, *pb.Message) error

// ConsumeMessages fonksiyonu, belirtilen topici dinler ve mesajları işler
func (kc *KafkaClient) ConsumeMessages(ctx context.Context, handler MessageHandler, topic *string, groupID *string) error {
	// Consumer grubu, RabbitMQ'daki her servisin kendi kuyruğuna karşılık gelir.
	// Bu sayede her servisin bir kopyası olsa bile, mesajlar grup içinde bir kez işlenir.
	quietLogger := QuietKafkaLogger{}
	consumerGroupID := groupID
	if consumerGroupID == nil {
		id := kc.serviceType.String() + "-group"
		consumerGroupID = &id
	}

	consumerTopic := topic
	if consumerTopic == nil {
		consumerTopic = &kc.config.Topic
	}
	readerConfig := kafka.ReaderConfig{
		Brokers:        kc.config.Brokers,
		GroupID:        *consumerGroupID, // <-- Burayı güncelledik, // Her servisin kendi tüketici grubu
		Topic:          *consumerTopic,
		MinBytes:       10e3, // 10KB
		MaxBytes:       10e6, // 10MB
		MaxWait:        1 * time.Second,
		CommitInterval: 1 * time.Second, // Offsetleri otomatik kaydetme aralığı
		// StartOffset:    kafka.FirstOffset, // Uygulama ilk başladığında nereden başlanacağı
		//Logger:      kafka.LoggerFunc(log.Printf),
		//ErrorLogger: kafka.LoggerFunc(log.Printf),

		Logger:      kafka.LoggerFunc(quietLogger.Printf),
		ErrorLogger: kafka.LoggerFunc(quietLogger.Printf),
	}

	reader := kafka.NewReader(readerConfig)
	defer reader.Close()

	log.Printf("Kafka Consumer started for service %s, topic: %s, group: %s",
		kc.serviceType.String(), kc.config.Topic, readerConfig.GroupID)

	for {
		select {
		case <-ctx.Done():
			log.Println("Consumer context cancelled, shutting down.")
			return nil
		default:
			m, err := reader.FetchMessage(ctx)
			if err != nil {
				if err == context.Canceled {
					log.Println("Context cancelled during FetchMessage.")
					return nil
				}
				if !strings.Contains(err.Error(), "request timed out") &&
					!strings.Contains(err.Error(), "no messages received") {
					log.Printf("Error fetching message: %v", err)
				}
				//log.Printf("Error fetching message: %v", err)
				time.Sleep(time.Second) // Hata durumunda kısa bir bekleme
				continue
			}

			var message pb.Message
			if err := proto.Unmarshal(m.Value, &message); err != nil {
				log.Printf("Failed to unmarshal protobuf message from Kafka: %v", err)
				// Hatalı mesajı atla, ancak offset'i commit etmeliyiz ki tekrar denemesin
				if commitErr := reader.CommitMessages(ctx, m); commitErr != nil {
					log.Printf("Failed to commit offset for bad message: %v", commitErr)
				}
				continue
			}

			// ToServices mantığını Kafka'da uygulama tarafında yönetiyoruz
			if len(message.ToServices) > 0 {
				isForThisService := false
				for _, svc := range message.ToServices {
					if svc == kc.serviceType {
						isForThisService = true
						break
					}
				}
				if !isForThisService {
					log.Printf("Message [ID: %s, Type: %s] is not for this service (%s). Skipping.",
						message.Id, message.Type.String(), kc.serviceType.String())
					// Mesajı işleme, ancak offset'i commit et
					if commitErr := reader.CommitMessages(ctx, m); commitErr != nil {
						log.Printf("Failed to commit offset for skipped message: %v", commitErr)
					}
					continue
				}
			}

			// Mesaj tipinin bu servis için izin verilip verilmediğini kontrol et
			if !kc.isAllowedMessageType(kc.serviceType, message.Type) {
				log.Printf("Message type '%s' is not allowed for service '%s'. Skipping. ID: %s",
					message.Type.String(), kc.serviceType.String(), message.Id)
				if commitErr := reader.CommitMessages(ctx, m); commitErr != nil {
					log.Printf("Failed to commit offset for unauthorized message type: %v", commitErr)
				}
				continue
			}

			log.Printf("Processing message [ID: %s, Type: %s, RetryCount: %d] from topic %s, partition %d, offset %d",
				message.Id, message.Type.String(), message.RetryCount, m.Topic, m.Partition, m.Offset)

			// Handler'ı çağır
			err = handler(ctx, &message)

			if err != nil {
				log.Printf("Message processing failed for ID %s: %v", message.Id, err)

				// Kritik mesajlar veya retry mekanizması
				if kc.isCriticalMessageType(message.Type) || kc.shouldRetry(&message) {
					// Retry logic
					kc.handleRetry(ctx, &message) // Mesajı retry topic'e gönder
				} else {
					log.Printf("Message failed permanently for ID %s, sending to DLQ topic: %s", message.Id, kc.config.DLQTopic)
					kc.sendToDLQ(ctx, &message) // DLQ topic'e gönder
				}
				// Önemli: Mesaj işleme hatasında orijinal mesajın offset'ini commit etmeliyiz
				// çünkü retry/DLQ'ya göndererek işini bitirdik. Aksi takdirde aynı mesaj sürekli geri gelir.
				if commitErr := reader.CommitMessages(ctx, m); commitErr != nil {
					log.Printf("Failed to commit offset after processing error: %v", commitErr)
				}
			} else {
				log.Printf("Message processed successfully. ID: %s", message.Id)
				// Mesaj başarıyla işlendiğinde offset'i commit et
				if commitErr := reader.CommitMessages(ctx, m); commitErr != nil {
					log.Printf("Failed to commit offset: %v", commitErr)
				}
			}
		}
	}
}

// Kafka'da manuel retry ve DLQ yönetimi için ayrı writer'lara ihtiyacımız olacak.
// Bu writer'lar NewKafkaClient içinde oluşturulabilir veya gerektiğinde on-demand oluşturulabilir.

// sendToDLQ mesajı DLQ topic'ine gönderir
func (kc *KafkaClient) sendToDLQ(ctx context.Context, msg *pb.Message) {
	if kc.config.DLQTopic == "" {
		log.Println("DLQ topic not configured, cannot send message to DLQ.")
		return
	}

	dlqProducer := &kafka.Writer{
		Addr:         kafka.TCP(kc.config.Brokers...),
		Topic:        kc.config.DLQTopic,
		Balancer:     &kafka.LeastBytes{},
		WriteTimeout: 5 * time.Second,
		RequiredAcks: kafka.RequireNone, // DLQ için daha az sıkı gereksinimler olabilir
	}
	defer dlqProducer.Close()

	messageBytes, err := proto.Marshal(msg)
	if err != nil {
		log.Printf("Failed to marshal DLQ message: %v", err)
		return
	}

	kafkaMsg := kafka.Message{
		Key:   []byte(msg.Id),
		Value: messageBytes,
		Headers: []kafka.Header{
			{Key: "OriginalTopic", Value: []byte(kc.config.Topic)},
			{Key: "FailedTimestamp", Value: []byte(time.Now().Format(time.RFC3339))},
		},
	}

	if err := dlqProducer.WriteMessages(ctx, kafkaMsg); err != nil {
		log.Printf("Failed to send message to DLQ topic %s for ID %s: %v", kc.config.DLQTopic, msg.Id, err)
		// DLQ'ya gönderilemezse kalıcı depolama düşünebilirsiniz
		kc.saveCriticalMessageToStorage(msg)
	} else {
		log.Printf("Message ID %s sent to DLQ topic %s.", msg.Id, kc.config.DLQTopic)
	}
}

// handleRetry mesajı retry topic'ine gönderir
func (kc *KafkaClient) handleRetry(ctx context.Context, msg *pb.Message) {
	if !kc.config.EnableRetry || kc.config.RetryTopic == "" {
		log.Println("Retry mechanism not enabled or retry topic not configured.")
		return
	}

	msg.RetryCount++
	// Üstel artışla bekleme süresi (exponential backoff)
	// RabbitMQ'daki Expiration header'ı Kafka'da yoktur.
	// Bunun yerine, retry topic'indeki mesajlar belirli bir süre sonra işlenmelidir.
	// Bu, genellikle retry consumer'ın kendisinin bir bekleme süresi uygulayarak veya
	// KIP-266 (Retryable Topics) gibi Kafka özellikleriyle yapılır.
	// Bu örnekte, mesajı tekrar retry topic'e gönderip, retry consumer'ın bir delay ile çalışmasını bekleyeceğiz.
	retryDelay := time.Duration(math.Min(float64(1*time.Second)*math.Pow(2, float64(msg.RetryCount)), float64(30*time.Second)))
	log.Printf("Scheduling retry for message ID %s, retry count: %d, delay: %v", msg.Id, msg.RetryCount, retryDelay)

	retryProducer := &kafka.Writer{
		Addr:         kafka.TCP(kc.config.Brokers...),
		Topic:        kc.config.RetryTopic,
		Balancer:     &kafka.LeastBytes{},
		WriteTimeout: 5 * time.Second,
		RequiredAcks: kafka.RequireNone,
	}
	defer retryProducer.Close()

	// Mesajı yeniden marshal et (güncel RetryCount ile)
	messageBytes, err := proto.Marshal(msg)
	if err != nil {
		log.Printf("Failed to marshal retry message: %v", err)
		kc.saveCriticalMessageToStorage(msg) // Hata durumunda kalıcı depolama
		return
	}

	kafkaMsg := kafka.Message{
		Key:   []byte(msg.Id),
		Value: messageBytes,
		Headers: []kafka.Header{
			{Key: "OriginalTopic", Value: []byte(kc.config.Topic)},
			{Key: "RetryAttempt", Value: []byte(fmt.Sprintf("%d", msg.RetryCount))},
			{Key: "ScheduledDelay", Value: []byte(retryDelay.String())},
		},
	}

	if err := retryProducer.WriteMessages(ctx, kafkaMsg); err != nil {
		log.Printf("Failed to send message to retry topic %s for ID %s: %v", kc.config.RetryTopic, msg.Id, err)
		kc.saveCriticalMessageToStorage(msg) // Hata durumunda kalıcı depolama
	} else {
		log.Printf("Message ID %s sent to retry topic %s. Scheduled for retry with delay %v", msg.Id, kc.config.RetryTopic, retryDelay)
	}
}

// Kritik mesajları kalıcı depolamaya kaydetme fonksiyonu
func (kc *KafkaClient) saveCriticalMessageToStorage(msg *pb.Message) {
	data, err := proto.Marshal(msg) // Protobuf formatında kaydetmek daha iyidir
	if err != nil {
		log.Printf("Failed to marshal critical message for storage: %v", err)
		return
	}
	filename := fmt.Sprintf("critical_messages/%s_%s.pb", msg.Type.String(), msg.Id)

	if err := os.MkdirAll("critical_messages", 0755); err != nil {
		log.Printf("Failed to create directory for critical messages: %v", err)
	}

	if err := os.WriteFile(filename, data, 0644); err != nil {
		log.Printf("Failed to save critical message to storage: %v", err)
	} else {
		log.Printf("Critical message saved to %s", filename)
	}
}

// ConsumeDLQWithRecovery fonksiyonu, DLQ topic'ini dinler ve kritik mesajları tekrar işleme sokar
func (kc *KafkaClient) ConsumeDLQWithRecovery(ctx context.Context, handler MessageHandler) error {
	if kc.config.DLQTopic == "" {
		return fmt.Errorf("DLQ topic not configured for recovery consumer")
	}

	readerConfig := kafka.ReaderConfig{
		Brokers: kc.config.Brokers,
		GroupID: kc.serviceType.String() + "-dlq-recovery-group",
		Topic:   kc.config.DLQTopic,
		// DLQ için genellikle FirstOffset (en baştan başla) ayarı tercih edilir
		StartOffset: kafka.FirstOffset,
		// ... diğer ayarlar ...
	}
	reader := kafka.NewReader(readerConfig)
	defer reader.Close()

	log.Printf("Kafka DLQ Recovery Consumer started for service %s, DLQ topic: %s, group: %s",
		kc.serviceType.String(), kc.config.DLQTopic, readerConfig.GroupID)

	for {
		select {
		case <-ctx.Done():
			log.Println("DLQ Recovery Consumer context cancelled, shutting down.")
			return nil
		default:
			m, err := reader.FetchMessage(ctx)
			if err != nil {
				if err == context.Canceled {
					return nil
				}
				log.Printf("Error fetching message from DLQ: %v", err)
				time.Sleep(time.Second)
				continue
			}

			var message pb.Message
			if err := proto.Unmarshal(m.Value, &message); err != nil {
				log.Printf("Failed to unmarshal protobuf message from DLQ: %v", err)
				if commitErr := reader.CommitMessages(ctx, m); commitErr != nil {
					log.Printf("Failed to commit offset for bad DLQ message: %v", commitErr)
				}
				continue
			}

			log.Printf("Received message from DLQ [ID: %s, Type: %s]. Attempting recovery...",
				message.Id, message.Type.String())

			if kc.isCriticalMessageType(message.Type) {
				log.Printf("Critical message from DLQ, resetting retry count and re-publishing: %s", message.Id)
				message.RetryCount = 0  // Sıfırdan denemek için retry sayısını resetle
				message.Critical = true // Kritik olarak işaretle (eğer değilse)

				// Mesajı tekrar ana topice veya orijinal servise özel topice gönder
				// Bu, PublishMessage fonksiyonunun içinde ToServices'a göre doğru topice yönlendirilmesi anlamına gelir.
				// Bu örnekte PublishMessage ana topice gönderiyor, bu da consumer tarafından tekrar işlenmesini sağlar.
				if err := kc.PublishMessage(ctx, &message); err != nil {
					log.Printf("Failed to re-publish critical message from DLQ for ID %s: %v", message.Id, err)
					// Tekrar yayınlanamazsa, DLQ'da kalmalı veya kalıcı depolamaya kaydedilmeli.
					// Bu durumda, offset'i commit etmeyiz ve mesaj tekrar gelir.
					// Ya da Nack + requeue mantığı Kafka'da olmadığı için, manuel olarak DLQ'ya tekrar gönderebiliriz.
					// Bu basit örnekte, hata durumunda mesajın DLQ'da kalmasına izin veriyoruz.
					kc.saveCriticalMessageToStorage(&message) // Ek güvenlik için kaydet
				} else {
					log.Printf("Successfully re-published critical message from DLQ for ID %s. Committing offset.", message.Id)
					if commitErr := reader.CommitMessages(ctx, m); commitErr != nil {
						log.Printf("Failed to commit offset for recovered DLQ message: %v", commitErr)
					}
				}
			} else {
				// Kritik olmayan mesajlar için normal işleme (eğer DLQ handler'ı farklı bir işlem yapacaksa)
				log.Printf("Non-critical message from DLQ, passing to handler: %s", message.Id)
				err := handler(ctx, &message)
				if err != nil {
					log.Printf("Handler failed for non-critical DLQ message ID %s: %v", message.Id, err)
					// Hata durumunda commit etmeyebiliriz
				} else {
					if commitErr := reader.CommitMessages(ctx, m); commitErr != nil {
						log.Printf("Failed to commit offset for handled non-critical DLQ message: %v", commitErr)
					}
				}
			}
		}
	}
}

// shouldRetry fonksiyonu, mesajın tekrar denenip denenmeyeceğini kontrol eder
func (kc *KafkaClient) shouldRetry(msg *pb.Message) bool {
	if !kc.config.EnableRetry {
		return false
	}
	// Mesaj tipinin retryable olup olmadığını burada kontrol edebilirsiniz
	// Veya direkt olarak MaxRetries'ı aşmamışsa denenebilir varsayabilirsiniz
	return int(msg.RetryCount) < kc.config.MaxRetries
}
