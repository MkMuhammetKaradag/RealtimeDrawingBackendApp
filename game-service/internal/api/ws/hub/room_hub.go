package hub

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// roomHub, Redis Pub/Sub üzerinden odaların durumunu yönetir.
type roomHub struct {
	redisClient *redis.Client
	hub         *Hub // Hub'a referans, mesajları broadcast etmek için
	gameHub     *GameHub
	subscribers map[uuid.UUID]*redis.PubSub
	mutex       sync.Mutex
}

func NewRoomHub(redisClient *redis.Client, hub *Hub) *roomHub {
	return &roomHub{
		redisClient: redisClient,
		hub:         hub,
		gameHub:     hub.gameHub,
		subscribers: make(map[uuid.UUID]*redis.PubSub),
	}
}

// StartSubscriber, belirli bir oda için Redis aboneliğini başlatır.
func (rm *roomHub) StartSubscriber(roomID uuid.UUID) {
	rm.mutex.Lock()
	defer rm.mutex.Unlock()

	channel := fmt.Sprintf("room:%s", roomID.String())
	if _, ok := rm.subscribers[roomID]; ok {
		log.Printf("Subscriber for room %s already exists.", roomID)
		return
	}

	pubsub := rm.redisClient.Subscribe(context.Background(), channel)
	rm.subscribers[roomID] = pubsub

	go func() {
		defer pubsub.Close()
		log.Printf("Subscribed to Redis channel: %s", channel)

		for msg := range pubsub.Channel() {
			var redisMessage RoomManager
			if err := json.Unmarshal([]byte(msg.Payload), &redisMessage); err != nil {
				log.Printf("Failed to unmarshal Redis message for room %s: %v", roomID, err)
				continue
			}
			rm.handleRedisMessage(roomID, redisMessage.Data)
		}
		log.Printf("Unsubscribed from Redis channel: %s", channel)
	}()
}

// StopSubscriber, belirli bir oda için Redis aboneliğini sonlandırır.
func (rm *roomHub) StopSubscriber(roomID uuid.UUID) {
	rm.mutex.Lock()
	defer rm.mutex.Unlock()

	if pubsub, ok := rm.subscribers[roomID]; ok {
		pubsub.Unsubscribe(context.Background(), fmt.Sprintf("room:%s", roomID.String()))
		delete(rm.subscribers, roomID)
	}
}

// handleRedisMessage, Redis'ten gelen mesajları tipine göre yönlendirir.
func (rm *roomHub) handleRedisMessage(roomID uuid.UUID, data RoomManagerData) {
	
	switch data.Type {
	case "game_mode_change", "game_settings_update":
		rm.gameHub.HandleGameMessage(roomID, data)
	case "player_left":
		rm.handlePlayerLeft(roomID, data)
	case "player_joined":
		rm.handlePlayerLeft(roomID, data)

	default:
		log.Printf("Unknown message content type : %s", data.Type)
	}
}

func (rm *roomHub) handleGameStarted(roomID uuid.UUID, msg RoomManagerData) {
	log.Printf("Game started in room %s.", roomID)

	message := &Message{
		Type:    "game_started",
		Content: msg,
	}
	rm.hub.BroadcastMessage(roomID, message)
}

func (rm *roomHub) handlePlayerLeft(roomID uuid.UUID, msg RoomManagerData) {
	log.Printf("Player left room %s.", roomID)

	message := &Message{
		Type:    "player_left",
		Content: msg.Content,
	}
	rm.hub.BroadcastMessage(roomID, message)
}
func (rm *roomHub) handlePlayerJoin(roomID uuid.UUID, msg RoomManagerData) {
	log.Printf("Player join room %s.", roomID)

	message := &Message{
		Type:    "player_joined",
		Content: msg.Content,
	}
	rm.hub.BroadcastMessage(roomID, message)
}
