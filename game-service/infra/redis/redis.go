package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

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

// NewRedisManager, yeni Redis manager oluşturur
func NewRedisManager(redisURL string) *RedisManager {
	rdb := redis.NewClient(&redis.Options{
		Addr:     redisURL,
		Password: "", // password yok
		DB:       0,  // default DB
	})

	ctx := context.Background()

	// Bağlantıyı test et
	_, err := rdb.Ping(ctx).Result()
	if err != nil {
		log.Fatalf("Redis bağlantısı kurulamadı: %v", err)
	}

	log.Println("Redis bağlantısı başarılı")

	return &RedisManager{
		client: rdb,
		ctx:    ctx,
	}
}

// PublishToRoom, belirli bir odaya mesaj yayınlar
func (rm *RedisManager) PublishToRoom(roomID string, messageType string, data interface{}) error {
	message := PubSubMessage{
		Type:      messageType,
		RoomID:    roomID,
		Data:      data,
		Timestamp: time.Now(),
	}

	jsonData, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("mesaj JSON'a dönüştürülemedi: %v", err)
	}

	channel := fmt.Sprintf("room:%s", roomID)
	return rm.client.Publish(rm.ctx, channel, jsonData).Err()
}

// PublishGlobal, global mesaj yayınlar (tüm odalar)
func (rm *RedisManager) PublishGlobal(messageType string, data interface{}) error {
	message := PubSubMessage{
		Type:      messageType,
		RoomID:    "*",
		Data:      data,
		Timestamp: time.Now(),
	}

	jsonData, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("mesaj JSON'a dönüştürülemedi: %v", err)
	}

	return rm.client.Publish(rm.ctx, "global", jsonData).Err()
}

// SubscribeToRoom, belirli bir odayı dinler
func (rm *RedisManager) SubscribeToRoom(roomID string, callback func(PubSubMessage)) {
	channel := fmt.Sprintf("room:%s", roomID)
	pubsub := rm.client.Subscribe(rm.ctx, channel)
	defer pubsub.Close()

	log.Printf("Redis kanalı dinleniyor: %s", channel)

	for msg := range pubsub.Channel() {
		var pubSubMsg PubSubMessage
		if err := json.Unmarshal([]byte(msg.Payload), &pubSubMsg); err != nil {
			log.Printf("Redis mesajı parse edilemedi: %v", err)
			continue
		}

		callback(pubSubMsg)
	}
}

// SubscribeGlobal, global mesajları dinler
func (rm *RedisManager) SubscribeGlobal(callback func(PubSubMessage)) {
	pubsub := rm.client.Subscribe(rm.ctx, "global")
	defer pubsub.Close()

	log.Println("Redis global kanal dinleniyor")

	for msg := range pubsub.Channel() {
		var pubSubMsg PubSubMessage
		if err := json.Unmarshal([]byte(msg.Payload), &pubSubMsg); err != nil {
			log.Printf("Redis global mesajı parse edilemedi: %v", err)
			continue
		}

		callback(pubSubMsg)
	}
}

// SetRoomData, oda verilerini Redis'te saklar (cache)
func (rm *RedisManager) SetRoomData(roomID string, data interface{}) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}

	key := fmt.Sprintf("room:data:%s", roomID)
	return rm.client.Set(rm.ctx, key, jsonData, 24*time.Hour).Err() // 24 saat TTL
}

// GetRoomData, oda verilerini Redis'ten alır
func (rm *RedisManager) GetRoomData(roomID string, result interface{}) error {
	key := fmt.Sprintf("room:data:%s", roomID)
	data, err := rm.client.Get(rm.ctx, key).Result()
	if err != nil {
		return err
	}

	return json.Unmarshal([]byte(data), result)
}

// DeleteRoomData, oda verilerini Redis'ten siler
func (rm *RedisManager) DeleteRoomData(roomID string) error {
	key := fmt.Sprintf("room:data:%s", roomID)
	return rm.client.Del(rm.ctx, key).Err()
}

// SetPlayerSession, oyuncu oturum bilgilerini saklar
func (rm *RedisManager) SetPlayerSession(userID, roomID string, data interface{}) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}

	key := fmt.Sprintf("player:session:%s", userID)
	return rm.client.Set(rm.ctx, key, jsonData, 2*time.Hour).Err() // 2 saat TTL
}

// GetPlayerSession, oyuncu oturum bilgilerini alır
func (rm *RedisManager) GetPlayerSession(userID string, result interface{}) error {
	key := fmt.Sprintf("player:session:%s", userID)
	data, err := rm.client.Get(rm.ctx, key).Result()
	if err != nil {
		return err
	}

	return json.Unmarshal([]byte(data), result)
}

// AddPlayerToRoom, oyuncuyu odaya ekler (Redis set)
func (rm *RedisManager) AddPlayerToRoom(roomID, userID string) error {
	key := fmt.Sprintf("room:players:%s", roomID)
	return rm.client.SAdd(rm.ctx, key, userID).Err()
}

// RemovePlayerFromRoom, oyuncuyu odadan çıkarır
func (rm *RedisManager) RemovePlayerFromRoom(roomID, userID string) error {
	key := fmt.Sprintf("room:players:%s", roomID)
	return rm.client.SRem(rm.ctx, key, userID).Err()
}

// GetRoomPlayers, odadaki oyuncuları getirir
func (rm *RedisManager) GetRoomPlayers(roomID string) ([]string, error) {
	key := fmt.Sprintf("room:players:%s", roomID)
	return rm.client.SMembers(rm.ctx, key).Result()
}

// IsPlayerInRoom, oyuncunun odada olup olmadığını kontrol eder
func (rm *RedisManager) IsPlayerInRoom(roomID, userID string) (bool, error) {
	key := fmt.Sprintf("room:players:%s", roomID)
	return rm.client.SIsMember(rm.ctx, key, userID).Result()
}

// Close, Redis bağlantısını kapatır
func (rm *RedisManager) Close() error {
	return rm.client.Close()
}

// Mesaj tipleri
const (
	// Oda yönetimi mesajları
	MSG_ROOM_CREATED     = "room_created"
	MSG_ROOM_DELETED     = "room_deleted"
	MSG_ROOM_UPDATED     = "room_updated"
	MSG_PLAYER_JOINED    = "player_joined"
	MSG_PLAYER_LEFT      = "player_left"
	MSG_PLAYER_KICKED    = "player_kicked"
	MSG_PLAYER_BANNED    = "player_banned"
	MSG_PLAYER_UNBANNED  = "player_unbanned"
	
	// Oyun durumu mesajları
	MSG_GAME_STARTED     = "game_started"
	MSG_GAME_ENDED       = "game_ended"
	MSG_GAME_PAUSED      = "game_paused"
	MSG_GAME_MODE_CHANGED = "game_mode_changed"
	MSG_ROUND_STARTED    = "round_started"
	MSG_ROUND_ENDED      = "round_ended"
	
	// Gerçek zamanlı oyun mesajları (bunlar WebSocket'te kalacak)
	MSG_DRAW_DATA        = "draw_data"
	MSG_CHAT_MESSAGE     = "chat_message"
	MSG_GUESS_MESSAGE    = "guess_message"
	MSG_CORRECT_GUESS    = "correct_guess"
)