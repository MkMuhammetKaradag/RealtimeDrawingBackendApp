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
type RoomManagerData struct {
	Type    string      `json:"type"`
	Content interface{} `json:"content"`
}

type RoomManager struct {
	RoomID uuid.UUID       `json:"room_id"`
	Type   string          `json:"type"`
	Data   RoomManagerData `json:"data"`
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
	mutex sync.RWMutex
	//roomSubscribers map[uuid.UUID]*redis.PubSub
	//subscriberMutex sync.Mutex
	playerQuit chan struct {
		RoomID uuid.UUID
		UserID uuid.UUID
	}
	inboundMessages chan struct {
		RoomID uuid.UUID
		Msg    RoomManagerData
	}
	repo    Repository
	roomHub *roomHub
	gameHub *GameHub // GameHub'ı buraya ekledi
}

func NewHub(redisClient *redis.Client) *Hub {
	hub := &Hub{
		// Harita yapısını güncelledik
		roomsClients: make(map[uuid.UUID]map[uuid.UUID]*domain.Client),
		redisClient:  redisClient,
		register:     make(chan *domain.Client),
		unregister:   make(chan *domain.Client),
		playerQuit: make(chan struct {
			RoomID uuid.UUID
			UserID uuid.UUID
		}, 20),
		inboundMessages: make(chan struct {
			RoomID uuid.UUID
			Msg    RoomManagerData
		}, 100),
		ctx: context.Background(),
		//roomSubscribers: make(map[uuid.UUID]*redis.PubSub),

	}
	hub.gameHub = NewGameHub(hub)
	hub.roomHub = NewRoomHub(hub.redisClient, hub)
	// go hub.GameHubListener()
	return hub
}

//	func (h *Hub) GameHubListener() {
//		for {
//			select {
//			case quit := <-h.playerQuit:
//				// GameHub'a mesajı ilet
//				h.gameHub.HandlePlayerQuit(quit.RoomID, quit.UserID)
//			}
//		}
//	}
func (h *Hub) GetRoomSettings(roomID uuid.UUID) *GameSettings {
	h.gameHub.mutex.RLock()
	defer h.gameHub.mutex.RUnlock()

	if settings, exists := h.gameHub.roomSettings[roomID]; exists {
		return settings
	}

	return nil
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
			case incoming := <-h.inboundMessages:
				// Gelen mesajları işleme (örneğin, GameHub'a iletme)
				h.gameHub.HandleGameMessage(incoming.RoomID, incoming.Msg)
			case <-ctx.Done():
				// Uygulama kapanınca
				return
			}
		}
	}()
	// go func() {
	// 	for incoming := range h.inboundMessages {
	// 		// **Bu Goroutine, GameHub'ı senkronize olarak çağırır.**
	// 		// Bu, aynı anda sadece bir mesajın GameHub'da işlenmesini garanti etmez,
	// 		// ancak readPump'ın kilitlenmesini engeller.
	// 		h.gameHub.HandleGameMessage(incoming.RoomID, incoming.Msg)
	// 	}
	// }()
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

	// 1. Odaya ait istemci haritasını al. Eğer yoksa oluştur.
	roomClients, ok := h.roomsClients[client.RoomID]
	if !ok {
		// Oda ilk defa oluşturuluyor.
		roomClients = make(map[uuid.UUID]*domain.Client)
		h.roomsClients[client.RoomID] = roomClients
	}

	// Haritaya yeni istemci eklenmeden önceki oyuncu sayısını kontrol et
	// Aynı kullanıcı ID'sine sahip bir istemci var mı kontrol et (Yeniden Bağlantı)
	isReconnection := false
	if existingClient, ok := roomClients[client.ID]; ok {
		log.Printf("User %s is already connected to room %s. Closing old connection.", client.ID, client.RoomID)

		// Önceki bağlantıyı temizle
		close(existingClient.Send)
		close(existingClient.Done)
		delete(roomClients, client.ID)
		isReconnection = true // Yeniden bağlantı olduğunu işaretle
	}

	// Odadaki anlık istemci sayısı
	currentClientCount := len(roomClients)

	// 2. Yeni istemciyi haritaya ekle
	client.Done = make(chan struct{}) // Done kanalını initialize et
	roomClients[client.ID] = client

	// 3. Subscriber (Abone) başlatma mantığı
	// Eğer:
	// a) Bu bir yeniden bağlantı DEĞİLSE (isReconnection == false)
	// b) Ve yeni bağlantıdan önceki sayı SIFIR ise (yani şimdi ODAYA İLK KİŞİ girmişse)
	if !isReconnection && currentClientCount == 0 {
		fmt.Println("Odaya ilk kişi bağlandı. Subscriber başlatılıyor.")
		h.roomHub.StartSubscriber(client.RoomID)
	} else if isReconnection && currentClientCount == 0 {
		fmt.Println("client reconnection")
	}
}

// unregisterClient handles client unregistration (internal).
func (h *Hub) unregisterClient(client *domain.Client) {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	if client.RoomID == uuid.Nil {
		log.Printf("Client %s has no room association", client.ID)
		return
	}

	// İlgili odadan client'ı kaldır
	roomClients, ok := h.roomsClients[client.RoomID]
	if !ok {
		log.Printf("Room %s not found in roomsClients", client.RoomID)
		return
	}

	// Client'ı roomClients'tan sil
	if _, exists := roomClients[client.ID]; !exists {
		log.Printf("Client %s not found in room %s", client.ID, client.RoomID)
		return
	}

	delete(roomClients, client.ID)
	log.Printf("Client %s unregistered from room %s. Remaining: %d",
		client.ID, client.RoomID, len(roomClients))

	// 💡 PlayerQuit sinyalini NON-BLOCKING şekilde gönder
	select {
	case h.playerQuit <- struct {
		RoomID uuid.UUID
		UserID uuid.UUID
	}{RoomID: client.RoomID, UserID: client.ID}:
		log.Printf("PlayerQuit signal sent for user %s in room %s", client.ID, client.RoomID)
	default:
		log.Printf("WARNING: PlayerQuit channel full, signal dropped for user %s", client.ID)
	}

	// Oda boşaldıysa temizle
	if len(roomClients) == 0 {
		log.Printf("Room %s is now empty, cleaning up", client.RoomID)
		h.roomHub.StopSubscriber(client.RoomID)
		delete(h.roomsClients, client.RoomID)
	}

	// Send kanalını güvenli şekilde kapat
	h.closeSendChannel(client)
}
func (h *Hub) closeSendChannel(client *domain.Client) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Recovered from panic while closing channel for client %s: %v", client.ID, r)
		}
	}()

	// Kanalın zaten kapalı olup olmadığını kontrol et
	select {
	case _, ok := <-client.Send:
		if ok {
			// Kanal açık, kapat
			close(client.Send)
			log.Printf("Closed send channel for client %s", client.ID)
		}
	default:
		// Kanal boş ve açık, kapat
		close(client.Send)
		log.Printf("Closed send channel for client %s", client.ID)
	}
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

// readPump, client'tan gelen mesajları okur ve Hub'a iletir.
func (h *Hub) readPump(client *domain.Client) {
	defer func() {
		h.unregister <- client
		client.Conn.Close()
	}()

	for {
		_, payload, err := client.Conn.ReadMessage()
		if err != nil {
			if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
				log.Println("Client connection closed gracefully.")
			} else {
				log.Println("Client read error:", err)
			}
			break
		}

		// Gelen mesajı işle
		var msg RoomManagerData
		if err := json.Unmarshal(payload, &msg); err != nil {
			log.Printf("Failed to unmarshal message: %v", err)
			continue
		}

		switch msg.Type {
		case "get_room_setting":
			// Odanın ayarlarını al
			settings := h.GetRoomSettings(client.RoomID)

			if settings == nil {
				// Ayar bulunamazsa veya GameHub'da henüz oluşturulmamışsa hata gönder
				h.sendErrorToClient(client, "Room settings not found or game not initialized.")
				continue
			}

			// Ayarları istemciye geri gönder
			response := &Message{
				Type:    "room_settings",
				Content: settings, // GameSettings yapısı doğrudan gönderilebilir.
			}

			// İstemciye JSON mesajı gönderme
			if err := h.SendMessageToClient(client, response); err != nil {
				log.Printf("Failed to send room settings to client %s: %v", client.ID, err)
			}

		case "game_started":

			h.inboundMessages <- struct {
				RoomID uuid.UUID
				Msg    RoomManagerData
			}{
				RoomID: client.RoomID,
				Msg:    msg,
			}

		case "player_move":
			// 💡 PlayerID'yi ekleyin
			if contentMap, ok := msg.Content.(map[string]interface{}); ok {
				contentMap["player_id"] = client.ID.String()
			}

			h.inboundMessages <- struct {
				RoomID uuid.UUID
				Msg    RoomManagerData
			}{
				RoomID: client.RoomID,
				Msg:    msg,
			}

		case "game_settings_update":
			// 💡 PlayerID'yi ekleyin
			if contentMap, ok := msg.Content.(map[string]interface{}); ok {
				contentMap["player_id"] = client.ID.String()
			}

			h.inboundMessages <- struct {
				RoomID uuid.UUID
				Msg    RoomManagerData
			}{
				RoomID: client.RoomID,
				Msg:    msg,
			}
		case "canvas_action":
			// 💡 PlayerID'yi ekleyin
			if contentMap, ok := msg.Content.(map[string]interface{}); ok {
				contentMap["player_id"] = client.ID.String()
			}

			h.inboundMessages <- struct {
				RoomID uuid.UUID
				Msg    RoomManagerData
			}{
				RoomID: client.RoomID,
				Msg:    msg,
			}

		}

		// Mesaj işleme mantığı buraya gelecek.
		// Örneğin: h.handleMessage(msg, client)
	}
}
func (h *Hub) GetRoomClients(roomID uuid.UUID) map[uuid.UUID]*domain.Client {
	h.mutex.RLock() // Read Lock kullanıyoruz
	defer h.mutex.RUnlock()

	clients, ok := h.roomsClients[roomID]
	if !ok {
		return nil
	}

	// Haritanın bir kopyasını döndürmek eşzamanlılık açısından en güvenli yoldur,
	// ancak performans kaygısı varsa, sadece ReadLock yeterli olabilir.
	// Basit bir oyun için sadece okuma (RLock) ve orijinal haritayı döndürmek yeterlidir.
	// clients, orijinal haritaya bir referanstır, bu yüzden sadece okuma amaçlı kullanın!
	return clients
}

// SendMessageToClient, belirtilen client'a JSON formatında bir mesaj gönderir.
func (h *Hub) SendMessageToClient(client *domain.Client, msg *Message) error {
	messageBytes, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	select {
	case client.Send <- messageBytes:
		return nil
	default:
		// Kanal doluysa veya kapalıysa
		log.Printf("Client %s's send channel is full, dropping message.", client.ID)
		return fmt.Errorf("client send channel is full")
	}
}

// sendErrorToClient, belirtilen client'a bir hata mesajı gönderir.
func (h *Hub) sendErrorToClient(client *domain.Client, errorMessage string) {
	errorMsg := &Message{
		Type:    "error",
		Content: errorMessage,
	}

	// Hata mesajını istemciye gönderme
	if err := h.SendMessageToClient(client, errorMsg); err != nil {
		log.Printf("Failed to send error message to client %s: %v", client.ID, err)
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
func (h *Hub) BroadcastToOthers(roomID uuid.UUID, senderID uuid.UUID, msg *Message) {
	// Okuma kilidi al
	h.mutex.RLock()
	defer h.mutex.RUnlock()

	// Odayı bul
	roomClients, ok := h.roomsClients[roomID]
	if !ok {
		log.Printf("Room %s not found for targeted broadcast.", roomID)
		return
	}

	// JSON mesajını doğru şekilde oluştur
	messageBytes, err := json.Marshal(msg)
	if err != nil {
		log.Printf("Failed to marshal message for broadcast to others: %v", err)
		return
	}

	// Odadaki tüm istemciler üzerinde döngü yap
	for _, client := range roomClients {
		// 💡 KENDİNDEN BAŞKA HERKESE GÖNDERME KONTROLÜ
		if client.ID == senderID {
			// Mesajı gönderen istemciyi atla (kendine gönderme)
			continue
		}

		select {
		case client.Send <- messageBytes:
			// Mesaj başarıyla gönderildi
		default:
			// İstemcinin kanalı doluysa mesajı düşür ve logla
			log.Printf("Client %s's send channel is full, dropping message (sent by %s).", client.ID, senderID)
			// İPUCU: Kritik mesajlar için burada istemcinin bağlantısını kesmeyi düşünebilirsiniz.
		}
	}
}

func (h *Hub) GetRoomClientCount(roomID uuid.UUID) int {

	h.mutex.RLock()
	defer h.mutex.RUnlock()

	if clients, ok := h.roomsClients[roomID]; ok {
		return len(clients)
	}

	return 0
}
func (h *Hub) SendMessageToUser(roomID uuid.UUID, userID uuid.UUID, msg *Message) error {
	h.mutex.RLock() // Haritadan okuma yapacağımız için RLock
	defer h.mutex.RUnlock()
	fmt.Println("SendMessageToUser msg:", msg)
	roomClients, ok := h.roomsClients[roomID]
	if !ok {
		return fmt.Errorf("room %s not found for user %s", roomID, userID)
	}

	client, ok := roomClients[userID]
	if !ok {
		return fmt.Errorf("client %s not found in room %s", userID, roomID)
	}

	// Mesajı JSON'a çevir
	messageBytes, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal message for user %s: %w", userID, err)
	}

	// Client'ın Send kanalına gönder
	select {
	case client.Send <- messageBytes:
		return nil
	default:
		// Kanal doluysa veya kapalıysa
		log.Printf("Client %s's send channel is full, dropping message.", client.ID)
		return fmt.Errorf("client send channel is full")
	}
}
func (h *Hub) IsGameActive(roomID uuid.UUID) bool {

	return h.gameHub.IsGameActive(roomID)
}
func (h *Hub) GetActiveGame(roomID uuid.UUID) *Game {

	return h.gameHub.GetActiveGame(roomID)
}

func (h *Hub) IsPlayerInActiveGame(roomID, userID uuid.UUID) bool {
	h.gameHub.mutex.RLock()
	defer h.gameHub.mutex.RUnlock()

	game, exists := h.gameHub.activeGames[roomID]
	if !exists {
		return false
	}

	for _, player := range game.Players {
		fmt.Println("Checking player:", player.UserID, "against userID:", userID)
		if player.UserID == userID {
			return true
		}
	}
	return false
}
func (h *Hub) IsClientConnected(roomID, userID uuid.UUID) bool {
	h.mutex.RLock()
	defer h.mutex.RUnlock()

	roomClients, ok := h.roomsClients[roomID]
	if !ok {
		return false
	}

	_, exists := roomClients[userID]
	return exists
}
