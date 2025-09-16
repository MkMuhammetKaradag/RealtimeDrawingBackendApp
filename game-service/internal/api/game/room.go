package game

import (
	"fmt"
	"sync"
	"time"

	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2/log"
	"github.com/google/uuid"
)

// Player, WebSocket bağlantısı olan bir oyuncuyu temsil eder.
type Player struct {
	ID     string          // Kullanıcının UUID'si
	Conn   *websocket.Conn // Oyuncunun WebSocket bağlantısı
	Online bool            // Oyuncunun bağlantısı hala aktif mi?
}

// Room, tek bir oyun odasının verilerini tutar.
type Room struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Mode       int    `json:"mode"`
	MaxPlayers int    `json:"maxPlayers"`
	Status     string `json:"status"` // 'waiting', 'active', 'finished'

	CreatorID string `json:"creatorId"` // Odanın sahibinin ID'si

	// Banlanmış oyuncuları tutan map
	BannedPlayers map[string]bool // Key: Player ID, Value: true

	// Odadaki oyuncuları tutan harita.
	// Eşzamanlı erişimi yönetmek için sync.Map kullanmak daha güvenli olabilir, ancak
	// burada basit bir map ve mutex kullanıyoruz.
	Players map[string]*Player // Key: Player ID
	mu      sync.RWMutex       // Eşzamanlı okuma/yazma için mutex

	CreatedAt time.Time `json:"createdAt"`
}

// NewRoom, yeni bir Room örneği oluşturur.
func NewRoom(name string, mode, maxPlayers int, creatorID string) *Room {
	return &Room{
		ID:            uuid.NewString(),
		Name:          name,
		Mode:          mode,
		MaxPlayers:    maxPlayers,
		CreatorID:     creatorID,
		Status:        "waiting",
		Players:       make(map[string]*Player),
		BannedPlayers: make(map[string]bool),
		CreatedAt:     time.Now(),
	}
}

// AddPlayer, odaya yeni bir oyuncu ekler.
func (r *Room) AddPlayer(player *Player) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.Players[player.ID] = player
	log.Infof("Oyuncu %s odaya katıldı: %s", player.ID, r.ID)
}

// BanPlayer, bir oyuncuyu odadan atar ve ban listesine ekler.
func (r *Room) BanPlayer(playerID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	player, exists := r.Players[playerID]
	if exists && player.Conn != nil {
		player.Conn.Close()
	}
	delete(r.Players, playerID)

	r.BannedPlayers[playerID] = true // Oyuncuyu ban listesine ekle
	log.Infof("Oyuncu %s odadan atıldı ve banlandı: %s", playerID, r.ID)

	return nil
}

// IsBanned, bir oyuncunun banlı olup olmadığını kontrol eder.
func (r *Room) IsBanned(playerID string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, exists := r.BannedPlayers[playerID]
	return exists
}

// UnbanPlayer, bir oyuncuyu ban listesinden kaldırır.
func (r *Room) UnbanPlayer(playerID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.BannedPlayers[playerID]; !exists {
		return fmt.Errorf("oyuncu %s zaten banlı değil", playerID)
	}

	delete(r.BannedPlayers, playerID) // Ban listesinden sil
	log.Infof("Oyuncu %s banı kaldırıldı: %s", playerID, r.ID)

	return nil
}

// RemovePlayer, odadan bir oyuncuyu çıkarır.
func (r *Room) RemovePlayer(playerID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.Players, playerID)
	log.Infof("Oyuncu %s odadan ayrıldı: %s", playerID, r.ID)
}

// BroadcastMessage, odadaki tüm oyunculara mesaj gönderir.
func (r *Room) BroadcastMessage(messageType int, message []byte) {
	r.mu.RLock()         // Okuma kilidi al
	defer r.mu.RUnlock() // Okuma kilidini bırak

	for _, player := range r.Players {
		if player.Online {
			if err := player.Conn.WriteMessage(messageType, message); err != nil {
				log.Infof("Oyuncuya mesaj gönderilemedi %s: %v", player.ID, err)
				// Hata durumunda bağlantıyı kesme mantığı burada olabilir.
			}
		}
	}
}
