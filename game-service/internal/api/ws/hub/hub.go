package hub

import (
	"context"
	"encoding/json"
	"fmt"
	"game-service/domain"
	"log"
	"sync"
	"time"

	"github.com/fasthttp/websocket"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

type Message struct {
	Type    string      `json:"type"`
	Content interface{} `json:"content"`
}
type RoomManager struct {
	RoomID uuid.UUID `json:"room_id"`
	Type   string    `json:"type"`
	Data   struct {
		Type    string      `json:"type"`
		Content interface{} `json:"content"`
	} `json:"data"`
}

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 512
)

// Hub yapısı
type Hub struct {
	// roomsClients artık odadaki istemcileri ID bazında izleyecek
	roomsClients map[uuid.UUID]map[uuid.UUID]*domain.Client

	redisClient *redis.Client
	register    chan *domain.Client
	unregister  chan *domain.Client
	ctx         context.Context

	// Eşzamanlılık koruması
	mutex           sync.RWMutex
	roomSubscribers map[uuid.UUID]*redis.PubSub
	subscriberMutex sync.Mutex

	repo Repository
	//roomHub *RoomManagerHub
}

func NewHub(redisClient *redis.Client) *Hub {
	hub := &Hub{
		// Harita yapısını güncelledik
		roomsClients:    make(map[uuid.UUID]map[uuid.UUID]*domain.Client),
		redisClient:     redisClient,
		register:        make(chan *domain.Client),
		unregister:      make(chan *domain.Client),
		ctx:             context.Background(),
		roomSubscribers: make(map[uuid.UUID]*redis.PubSub),
		//repo:         repo, //

	}
	//hub.roomHub = NewRoomManagerHub(redisClient, hub)
	return hub
}

func (h *Hub) Run(ctx context.Context) {
	// Ana hub döngüsü, olayları dinler.
	// Bu, tüm senkronizasyon ve kayıt/kayıt silme mantığının kalbidir.
	go func() {
		for {
			select {
			case client := <-h.register:
				// `registerClient` yeni client'ı kaydeder ve eskiyi kapatır
				h.registerClient(client)
				// Her client için okuma ve yazma goroutine'lerini başlatırız.
				go h.readPump(client)
				go h.writePump(client)
			case client := <-h.unregister:
				// `unregisterClient` client'ı haritadan siler.
				h.unregisterClient(client)
			case <-ctx.Done():
				// Uygulama kapanınca
				return
			}
		}
	}()
	//go h.roomHub.Run(ctx)
}

// RegisterClient, client'ı ana hub'ın register kanalına gönderir.
func (h *Hub) RegisterClient(client *domain.Client) {
	h.register <- client
}

// UnregisterClient, client'ı ana hub'ın unregister kanalına gönderir.
func (h *Hub) UnregisterClient(client *domain.Client) {
	// Bu fonksiyon, bir client'ın bağlantısı kesildiğinde veya bir hata olduğunda çağrılmalıdır.
	// `readPump` içinden çağrılacaktır.
	h.unregister <- client
}

// registerClient handles client registration (internal). Bu fonksiyon
// doğrudan bir kanala yazılmaz, sadece Run döngüsü içinden çağrılır.
func (h *Hub) registerClient(client *domain.Client) {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	// 1. Odaya ait istemci haritasını al
	if _, ok := h.roomsClients[client.RoomID]; !ok {
		h.roomsClients[client.RoomID] = make(map[uuid.UUID]*domain.Client)
		h.startRoomSubscriber(client.RoomID)
	}
	roomClients := h.roomsClients[client.RoomID]

	// 2. Aynı kullanıcı ID'sine sahip bir istemci var mı kontrol et
	if existingClient, ok := roomClients[client.ID]; ok {
		log.Printf("User %s is already connected to room %s. Closing old connection.", client.ID, client.RoomID)

		// Eğer mevcut bir bağlantı varsa, kanalını kapat ve haritadan sil.
		close(existingClient.Send)
		close(existingClient.Done)
		delete(roomClients, client.ID)
	}

	// 3. Yeni istemciyi haritaya ekle
	client.Done = make(chan struct{}) // Done kanalını initialize et
	roomClients[client.ID] = client
}

// unregisterClient handles client unregistration (internal).
func (h *Hub) unregisterClient(client *domain.Client) {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	if roomClients, ok := h.roomsClients[client.RoomID]; ok {
		if _, ok := roomClients[client.ID]; ok {
			delete(roomClients, client.ID)
			if len(roomClients) == 0 {
				delete(h.roomsClients, client.RoomID)
				h.stopRoomSubscriber(client.RoomID)
			}
		}
	}

	// Sadece kanal açık değilse kapatmaya çalış
	select {
	case <-client.Send:
	default:
		close(client.Send)
	}
}

func (h *Hub) startRoomSubscriber(roomID uuid.UUID) {
	h.subscriberMutex.Lock()
	defer h.subscriberMutex.Unlock()

	channel := fmt.Sprintf("room:%s", roomID.String())
	pubsub := h.redisClient.Subscribe(h.ctx, channel)
	h.roomSubscribers[roomID] = pubsub

	go func() {
		defer pubsub.Close()
		log.Printf("Subscribed to Redis channel: %s", channel)

		for msg := range pubsub.Channel() {
			var roomManagerMsg RoomManager
			if err := json.Unmarshal([]byte(msg.Payload), &roomManagerMsg); err != nil {
				log.Printf("Failed to unmarshal Redis message for room %s: %v", roomID, err)
				continue
			}
			fmt.Println("messaj geldi:", roomManagerMsg)

			// Gelen mesaj türüne göre işlem yap
			switch roomManagerMsg.Data.Type {
			case "player_left":
				// İçerik tipini kontrol et ve kullanıcı ID'sini al
				contentMap, ok := roomManagerMsg.Data.Content.(map[string]interface{})
				if !ok {
					log.Println("Invalid content format for player_left message.")
					continue
				}

				userIDStr, ok := contentMap["user_id"].(string)
				if !ok {
					log.Println("user_id not found in player_left content.")
					continue
				}

				userID, err := uuid.Parse(userIDStr)
				if err != nil {
					log.Printf("Failed to parse user ID: %v", err)
					continue
				}

				// WebSocket bağlantısını kapat
				h.closeClientConnection(userID)

				// Mesajı odaya yayınla
				h.BroadcastMessage(roomID, &Message{
					Type:    roomManagerMsg.Data.Type,
					Content: roomManagerMsg.Data.Content,
				})

			// Diğer durumlarda (add_player, kick_player vb.)
			default:
				// Mesajı odaya yayınla
				h.BroadcastMessage(roomID, &Message{
					Type:    roomManagerMsg.Data.Type,
					Content: roomManagerMsg.Data.Content,
				})
			}
		}
		log.Printf("Unsubscribed from Redis channel: %s", channel)
	}()
}
func (h *Hub) closeClientConnection(userID uuid.UUID) {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	// Tüm odaları dönerek kullanıcıyı bul
	for _, clients := range h.roomsClients {
		if client, ok := clients[userID]; ok {
			log.Printf("Closing WebSocket connection for user %s", userID)

			// Bağlantıyı kapat
			client.Conn.Close()

			// Unregister kanalına gönder, bu sayede readPump/writePump goroutine'leri kapanır
			h.unregister <- client
			return
		}
	}
	log.Printf("User %s not found in any room.", userID)
}

// stopRoomSubscriber, odaya özel Redis aboneliğini sonlandırır.
func (h *Hub) stopRoomSubscriber(roomID uuid.UUID) {
	h.subscriberMutex.Lock()
	defer h.subscriberMutex.Unlock()

	if pubsub, ok := h.roomSubscribers[roomID]; ok {
		pubsub.Unsubscribe(h.ctx, fmt.Sprintf("room:%s", roomID.String()))
		delete(h.roomSubscribers, roomID)
	}
}

// readPump, client'tan gelen mesajları okur ve Hub'a iletir.
func (h *Hub) readPump(client *domain.Client) {
	defer func() {
		h.unregister <- client
		client.Conn.Close()
	}()

	for {
		messageType, payload, err := client.Conn.ReadMessage()
		if err != nil {
			if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
				log.Println("Client connection closed gracefully.")
			} else {
				log.Println("Client read error:", err)
			}
			break
		}

		// Gelen mesajı işle
		var msg Message
		if err := json.Unmarshal(payload, &msg); err != nil {
			log.Printf("Failed to unmarshal message: %v", err)
			continue
		}
		fmt.Println("msg:", msg, "messagetype:", messageType)
		// Mesaj işleme mantığı buraya gelecek.
		// Örneğin: h.handleMessage(msg, client)
	}
}

// writePump, client'ın Send kanalına gelen mesajları yazar.
func (h *Hub) writePump(client *domain.Client) {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		client.Conn.Close()
		h.unregister <- client
	}()

	for {
		select {
		case msg, ok := <-client.Send:
			if !ok {
				// Hub, client'a ait `Send` kanalını kapatmış.
				client.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			// Mesajı yaz
			client.WriteLock.Lock()
			client.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			err := client.Conn.WriteMessage(websocket.TextMessage, msg)
			client.WriteLock.Unlock()
			if err != nil {
				log.Println("WebSocket write error:", err)
				return
			}

		case <-ticker.C:
			client.WriteLock.Lock()
			client.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := client.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				client.WriteLock.Unlock()
				return
			}
			client.WriteLock.Unlock()

		case <-client.Done:
			return

			// case <-time.After(1 * time.Minute):
			// 	client.Conn.WriteMessage(websocket.PingMessage, nil)
		}
	}
}

func (h *Hub) BroadcastMessage(roomID uuid.UUID, msg *Message) {
	h.mutex.RLock()
	defer h.mutex.RUnlock()

	roomClients, ok := h.roomsClients[roomID]
	if !ok {
		log.Printf("Room %s not found for broadcast message.", roomID)
		return
	}

	// JSON mesajını doğru şekilde oluştur
	messageBytes, err := json.Marshal(msg)
	if err != nil {
		log.Printf("Failed to marshal message: %v", err)
		return
	}

	for _, client := range roomClients {
		select {
		case client.Send <- messageBytes:
		default:
			log.Printf("Client %s's send channel is full, dropping message.", client.ID)
		}
	}
}
