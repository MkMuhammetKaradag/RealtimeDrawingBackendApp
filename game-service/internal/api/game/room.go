package game

import (
	"fmt"
	"sync"
	"time"

	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2/log"
	"github.com/google/uuid"
)

// GameMode, oyun modlarını tanımlar
type GameMode struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	MinPlayers  int    `json:"minPlayers"`
	MaxPlayers  int    `json:"maxPlayers"`
}

// Player, WebSocket bağlantısı olan bir oyuncuyu temsil eder
type Player struct {
	ID       string          `json:"id"`
	Username string          `json:"username"`
	Conn     *websocket.Conn `json:"-"` // JSON'da gösterilmesin
	Online   bool            `json:"online"`
	Score    int             `json:"score"`
	JoinedAt time.Time       `json:"joinedAt"`
	IsBanned bool            `json:"isBanned"`
}

// DrawPoint, çizim noktalarını temsil eder
type DrawPoint struct {
	X     float64 `json:"x"`
	Y     float64 `json:"y"`
	Color string  `json:"color,omitempty"`
	Size  int     `json:"size,omitempty"`
	Type  string  `json:"type"` // "start", "draw", "end"
}

// DrawingData, çizim verilerini tutar
type DrawingData struct {
	PlayerID    string      `json:"playerId"`
	Points      []DrawPoint `json:"points"`
	Timestamp   time.Time   `json:"timestamp"`
	RoundNumber int         `json:"roundNumber"`
}

// Word, çizilecek kelimeleri temsil eder
type Word struct {
	ID         int    `json:"id"`
	Word       string `json:"word"`
	Difficulty int    `json:"difficulty"`
	Category   string `json:"category"`
}

// GameSession, aktif oyun oturumunu yönetir
type GameSession struct {
	ID              string            `json:"id"`
	RoomID          string            `json:"roomId"`
	CurrentRound    int               `json:"currentRound"`
	TotalRounds     int               `json:"totalRounds"`
	RoundDuration   int               `json:"roundDuration"` // Saniye
	CurrentDrawerID string            `json:"currentDrawerId"`
	CurrentWord     *Word             `json:"currentWord,omitempty"`
	RoundStartTime  time.Time         `json:"roundStartTime"`
	RoundEndTime    time.Time         `json:"roundEndTime"`
	Status          string            `json:"status"`         // 'preparing', 'active', 'paused', 'finished'
	PlayerWords     map[string]string `json:"playerWords"`    // Mod 1 için: PlayerID -> Word
	CorrectGuesses  map[string]bool   `json:"correctGuesses"` // Bu roundda doğru tahmin edenler
	RoundScores     map[string]int    `json:"roundScores"`
	mu              sync.RWMutex
}

// Room, oyun odasını temsil eder
type Room struct {
	ID            string             `json:"id"`
	Name          string             `json:"name"`
	CreatorID     string             `json:"creatorId"`
	GameModeID    int                `json:"gameModeId"`
	GameMode      *GameMode          `json:"gameMode,omitempty"`
	MaxPlayers    int                `json:"maxPlayers"`
	Status        string             `json:"status"` // 'waiting', 'playing', 'finished'
	IsPrivate     bool               `json:"isPrivate"`
	RoomCode      string             `json:"roomCode,omitempty"`
	Players       map[string]*Player `json:"players"`
	BannedPlayers map[string]bool    `json:"-"` // JSON'da gösterilmesin
	CreatedAt     time.Time          `json:"createdAt"`
	StartedAt     *time.Time         `json:"startedAt,omitempty"`

	// Oyun oturumu
	GameSession *GameSession `json:"gameSession,omitempty"`

	// Canvas verileri (Ortak Alan modu için)
	SharedCanvas []DrawingData `json:"-"` // Bellek tasarrufu için JSON'da gösterilmesin

	mu sync.RWMutex
}

// NewGameSession, yeni bir oyun oturumu oluşturur
func NewGameSession(roomID string, totalRounds, roundDuration int) *GameSession {
	return &GameSession{
		ID:             uuid.NewString(),
		RoomID:         roomID,
		CurrentRound:   0,
		TotalRounds:    totalRounds,
		RoundDuration:  roundDuration,
		Status:         "preparing",
		PlayerWords:    make(map[string]string),
		CorrectGuesses: make(map[string]bool),
		RoundScores:    make(map[string]int),
	}
}

// NewRoom, yeni bir Room örneği oluşturur
func NewRoom(name string, gameModeID, maxPlayers int, creatorID string, isPrivate bool) *Room {
	room := &Room{
		ID:            uuid.NewString(),
		Name:          name,
		CreatorID:     creatorID,
		GameModeID:    gameModeID,
		MaxPlayers:    maxPlayers,
		Status:        "waiting",
		IsPrivate:     isPrivate,
		Players:       make(map[string]*Player),
		BannedPlayers: make(map[string]bool),
		CreatedAt:     time.Now(),
		SharedCanvas:  make([]DrawingData, 0),
	}

	if isPrivate {
		room.RoomCode = generateRoomCode()
	}

	return room
}

// generateRoomCode, özel odalar için 6 haneli kod üretir
func generateRoomCode() string {
	return fmt.Sprintf("%06d", time.Now().UnixNano()%1000000)
}

// AddPlayer, odaya yeni bir oyuncu ekler
func (r *Room) AddPlayer(player *Player) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Ban kontrolü
	if r.BannedPlayers[player.ID] {
		return fmt.Errorf("oyuncu %s bu odaya girişi yasaklandı", player.ID)
	}

	// Maksimum oyuncu kontrolü
	if len(r.Players) >= r.MaxPlayers {
		return fmt.Errorf("oda dolu")
	}

	// Oyun başladıktan sonra katılım kontrolü
	if r.Status == "playing" {
		return fmt.Errorf("oyun devam ediyor, katılamazsınız")
	}

	player.JoinedAt = time.Now()
	r.Players[player.ID] = player
	log.Infof("Oyuncu %s odaya katıldı: %s", player.ID, r.ID)

	// Oda dolu ve oyun başlatılabilir durumda mı kontrol et
	if len(r.Players) >= 2 && r.Status == "waiting" {
		// Otomatik başlatma mantığı burada olabilir
	}

	return nil
}

// RemovePlayer, odadan bir oyuncuyu çıkarır
func (r *Room) RemovePlayer(playerID string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if player, exists := r.Players[playerID]; exists {
		if player.Conn != nil {
			player.Conn.Close()
		}
		delete(r.Players, playerID)
		log.Infof("Oyuncu %s odadan ayrıldı: %s", playerID, r.ID)

		// Eğer oda sahibi ayrıldıysa ve başka oyuncular varsa, oda sahipliğini devret
		if r.CreatorID == playerID && len(r.Players) > 0 {
			for newCreatorID := range r.Players {
				r.CreatorID = newCreatorID
				log.Infof("Oda sahipliği %s oyuncusuna devredildi: %s", newCreatorID, r.ID)
				break
			}
		}
	}
}

// BanPlayer, bir oyuncuyu odadan atar ve ban listesine ekler
func (r *Room) BanPlayer(playerID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	player, exists := r.Players[playerID]
	if !exists {
		return fmt.Errorf("oyuncu odada bulunamadı")
	}

	if player.Conn != nil {
		player.Conn.Close()
	}
	delete(r.Players, playerID)
	r.BannedPlayers[playerID] = true

	log.Infof("Oyuncu %s odadan atıldı ve banlandı: %s", playerID, r.ID)
	return nil
}

// UnbanPlayer, bir oyuncuyu ban listesinden kaldırır
func (r *Room) UnbanPlayer(playerID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if !r.BannedPlayers[playerID] {
		return fmt.Errorf("oyuncu zaten banlı değil")
	}

	delete(r.BannedPlayers, playerID)
	log.Infof("Oyuncu %s banı kaldırıldı: %s", playerID, r.ID)
	return nil
}

// IsBanned, bir oyuncunun banlı olup olmadığını kontrol eder
func (r *Room) IsBanned(playerID string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.BannedPlayers[playerID]
}

// BroadcastMessage, odadaki tüm oyunculara mesaj gönderir
func (r *Room) BroadcastMessage(messageType int, message []byte) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, player := range r.Players {
		if player.Online && player.Conn != nil {
			if err := player.Conn.WriteMessage(messageType, message); err != nil {
				log.Errorf("Oyuncuya mesaj gönderilemedi %s: %v", player.ID, err)
				player.Online = false
			}
		}
	}
}

// BroadcastToOthers, belirtilen oyuncu hariç diğer oyunculara mesaj gönderir
func (r *Room) BroadcastToOthers(excludePlayerID string, messageType int, message []byte) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for playerID, player := range r.Players {
		if playerID != excludePlayerID && player.Online && player.Conn != nil {
			if err := player.Conn.WriteMessage(messageType, message); err != nil {
				log.Errorf("Oyuncuya mesaj gönderilemedi %s: %v", player.ID, err)
				player.Online = false
			}
		}
	}
}

// StartGame, oyunu başlatır
func (r *Room) StartGame(totalRounds, roundDuration int) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.Status != "waiting" {
		return fmt.Errorf("oyun zaten başladı veya bitti")
	}

	if len(r.Players) < 2 {
		return fmt.Errorf("oyun başlatmak için en az 2 oyuncu gerekli")
	}

	r.Status = "playing"
	now := time.Now()
	r.StartedAt = &now
	r.GameSession = NewGameSession(r.ID, totalRounds, roundDuration)

	log.Infof("Oyun başlatıldı: %s", r.ID)
	return nil
}

// AddDrawingData, çizim verisini ekler
func (r *Room) AddDrawingData(data DrawingData) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Ortak Alan modu için tüm çizim verilerini sakla
	if r.GameModeID == 2 { // Ortak Alan
		r.SharedCanvas = append(r.SharedCanvas, data)
	}
}

// GetRoomInfo, odanın temel bilgilerini döndürür (WebSocket'te gönderilmek için)
func (r *Room) GetRoomInfo() map[string]interface{} {
	r.mu.RLock()
	defer r.mu.RUnlock()

	playerList := make([]Player, 0, len(r.Players))
	for _, player := range r.Players {
		playerList = append(playerList, *player)
	}

	return map[string]interface{}{
		"id":             r.ID,
		"name":           r.Name,
		"creatorId":      r.CreatorID,
		"gameModeId":     r.GameModeID,
		"maxPlayers":     r.MaxPlayers,
		"currentPlayers": len(r.Players),
		"status":         r.Status,
		"isPrivate":      r.IsPrivate,
		"roomCode":       r.RoomCode,
		"players":        playerList,
		"createdAt":      r.CreatedAt,
		"startedAt":      r.StartedAt,
	}
}
