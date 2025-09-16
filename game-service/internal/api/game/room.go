package game

import (
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

	// Odadaki oyuncuları tutan harita.
	// Eşzamanlı erişimi yönetmek için sync.Map kullanmak daha güvenli olabilir, ancak
	// burada basit bir map ve mutex kullanıyoruz.
	Players map[string]*Player // Key: Player ID
	mu      sync.RWMutex       // Eşzamanlı okuma/yazma için mutex

	CreatedAt time.Time `json:"createdAt"`
}

// NewRoom, yeni bir Room örneği oluşturur.
func NewRoom(name string, mode, maxPlayers int) *Room {
	return &Room{
		ID:         uuid.NewString(),
		Name:       name,
		Mode:       mode,
		MaxPlayers: maxPlayers,
		Status:     "waiting",
		Players:    make(map[string]*Player),
		CreatedAt:  time.Now(),
	}
}

// AddPlayer, odaya yeni bir oyuncu ekler.
func (r *Room) AddPlayer(player *Player) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.Players[player.ID] = player
	log.Infof("Oyuncu %s odaya katıldı: %s", player.ID, r.ID)
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
