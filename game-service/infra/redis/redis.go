package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// RedisManager, Redis Pub/Sub işlemlerini yönetir
type RedisManager struct {
	client *redis.Client
	ctx    context.Context
}

// PubSubMessage, Redis üzerinden gönderilecek mesaj yapısı
type PubSubMessage struct {
	Type      string      `json:"type"`
	RoomID    string      `json:"roomId"`
	Data      interface{} `json:"data"`
	Timestamp time.Time   `json:"timestamp"`
}

type RoomManager struct {
	RoomID uuid.UUID `json:"room_id"`
	Type   string    `json:"type"`
	Data   struct {
		Type    string      `json:"type"`
		Content interface{} `json:"content"`
	} `json:"data"`
}

// NewRedisManager, yeni Redis manager oluşturur
func NewRedisManager(redisAddr string, password string, db int) *RedisManager {
	rdb := redis.NewClient(&redis.Options{
		Addr:     redisAddr,
		Password: password, // Redis şifresi (varsayılan: boş)
		DB:       db,       // Redis veritabanı numarası (varsayılan: 0)
	})

	ctx := context.Background()

	// Bağlantıyı test et
	_, err := rdb.Ping(ctx).Result()
	if err != nil {
		log.Fatalf("Redis bağlantısı kurulamadı: %v", err)
	}


	return &RedisManager{
		client: rdb,
		ctx:    ctx,
	}
}

// Close, Redis bağlantısını kapatır
func (rm *RedisManager) Close() error {
	return rm.client.Close()
}

func (rm *RedisManager) PublishMessage(ctx context.Context, roomID uuid.UUID, msgType string, dataContent interface{}) {
	// Mesaj içeriğini oluştur
	msg := RoomManager{
		RoomID: roomID,
		Type:   "room_manager", // Mesaj tipi (örneğin: "room_manager")
		Data: struct {
			Type    string      `json:"type"`
			Content interface{} `json:"content"`
		}{
			Type:    msgType,     // Olay tipi (player_joined, player_left)
			Content: dataContent, // Olay ile ilgili veri
		},
	}

	// Mesajı JSON formatına dönüştür
	payload, err := json.Marshal(msg)
	if err != nil {
		log.Printf("Failed to marshal Redis message: %v", err)
		return
	}

	// Odaya özel kanalı belirle
	channel := fmt.Sprintf("room:%s", roomID.String())

	// Mesajı Redis'e yayınla
	err = rm.client.Publish(ctx, channel, payload).Err()
	if err != nil {
		log.Printf("Failed to publish message to Redis channel %s: %v", channel, err)
	}
}

// Mesaj tipleri
const (
	// Oda yönetimi mesajları
	MSG_ROOM_CREATED    = "room_created"
	MSG_ROOM_DELETED    = "room_deleted"
	MSG_ROOM_UPDATED    = "room_updated"
	MSG_PLAYER_JOINED   = "player_joined"
	MSG_PLAYER_LEFT     = "player_left"
	MSG_PLAYER_KICKED   = "player_kicked"
	MSG_PLAYER_BANNED   = "player_banned"
	MSG_PLAYER_UNBANNED = "player_unbanned"

	// Oyun durumu mesajları
	MSG_GAME_STARTED      = "game_started"
	MSG_GAME_ENDED        = "game_ended"
	MSG_GAME_PAUSED       = "game_paused"
	MSG_GAME_MODE_CHANGED = "game_mode_changed"
	MSG_ROUND_STARTED     = "round_started"
	MSG_ROUND_ENDED       = "round_ended"

	// Gerçek zamanlı oyun mesajları (bunlar WebSocket'te kalacak)
	MSG_DRAW_DATA     = "draw_data"
	MSG_CHAT_MESSAGE  = "chat_message"
	MSG_GUESS_MESSAGE = "guess_message"
	MSG_CORRECT_GUESS = "correct_guess"
)
