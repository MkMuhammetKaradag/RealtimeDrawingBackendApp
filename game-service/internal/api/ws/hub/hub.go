package hub

import (
	"context"
	"game-service/domain"
	"log"
	"sync"
	"time"

	"github.com/fasthttp/websocket"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
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
	mutex sync.RWMutex

	repo Repository
}

func NewHub(redisClient *redis.Client) *Hub {
	hub := &Hub{
		// Harita yapısını güncelledik
		roomsClients: make(map[uuid.UUID]map[uuid.UUID]*domain.Client),
		redisClient:  redisClient,
		register:     make(chan *domain.Client),
		unregister:   make(chan *domain.Client),
		ctx:          context.Background(),
		//repo:         repo, //
	}
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
	}
	roomClients := h.roomsClients[client.RoomID]

	// 2. Aynı kullanıcı ID'sine sahip bir istemci var mı kontrol et
	if existingClient, ok := roomClients[client.ID]; ok {
		log.Printf("User %s is already connected to room %s. Closing old connection.", client.ID, client.RoomID)

		// Eğer mevcut bir bağlantı varsa, kanalını kapat ve haritadan sil.
		close(existingClient.Send)
		delete(roomClients, client.ID)
	}

	// 3. Yeni istemciyi haritaya ekle
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

// readPump, client'tan gelen mesajları okur ve Hub'a iletir.
func (h *Hub) readPump(client *domain.Client) {
	defer func() {
		h.unregister <- client
		client.Conn.Close()
	}()

	for {
		_, _, err := client.Conn.ReadMessage()
		if err != nil {
			if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
				log.Println("Client connection closed gracefully.")
			} else {
				log.Println("Client read error:", err)
			}
			break
		}
		// Mesaj işleme mantığı buraya gelecek.
		// Örneğin: h.handleMessage(msg, client)
	}
}

// writePump, client'ın Send kanalına gelen mesajları yazar.
func (h *Hub) writePump(client *domain.Client) {
	defer func() {
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
			err := client.Conn.WriteMessage(websocket.TextMessage, msg)
			client.WriteLock.Unlock()
			if err != nil {
				log.Println("WebSocket write error:", err)
				return
			}
		// Bağlantının ping/pong mesajlarıyla hayatta kalmasını sağlamak için ping gönder
		case <-time.After(1 * time.Minute):
			client.Conn.WriteMessage(websocket.PingMessage, nil)
		}
	}
}
