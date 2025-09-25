package hub

import (
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

// GameState, oyunun hangi aşamada olduğunu gösteren sabitler.
const (
	GameStateInProgress = "in_progress"
	GameStateOver       = "over"
)

// IGameEngine, tüm oyun motorları için ortak bir arayüz tanımlar.
type IGameEngine interface {
	InitGame(game *Game, players []*Player) error
	ProcessMove(game *Game, playerID uuid.UUID, moveData interface{}) error
	CheckRoundStatus(game *Game) (bool, error)
	// Diğer oyun mantığı metotları buraya eklenebilir.
}

// Player, oyundaki bir oyuncuyu temsil eder.
type Player struct {
	UserID   uuid.UUID
	Username string
	Score    int
	// Oyuncuya özgü diğer veriler
}

type GameSettings struct {
	ModeName      string `json:"mode_name"`
	ModeID        string `json:"mode_id"`
	TotalRounds   int    `json:"total_rounds"`
	RoundDuration int    `json:"round_duration"` // saniye cinsinden
	MaxPlayers    int    `json:"max_players"`
	MinPlayers    int    `json:"min_players"`
}

// Game, bir oyunun mevcut durumunu tutar.
type Game struct {
	RoomID        uuid.UUID   `json:"room_id"`
	ModeName      string      `json:"mode_name"`
	State         string      `json:"state"`
	Players       []*Player   `json:"players"`
	TurnCount     int         `json:"turn_count"`
	TotalRounds   int         `json:"total_rounds"`
	RoundDuration int         `json:"round_duration"`
	ActivePlayer  uuid.UUID   `json:"active_player"`
	LastMoveTime  time.Time   `json:"last_move_time"`
	ModeData      interface{} `json:"mode_data"`
	Mutex         sync.RWMutex
}

// Game'e özel yapılar
type DrawingGameData struct {
	CurrentWord    string
	CurrentDrawer  uuid.UUID
	GuessedPlayers map[uuid.UUID]bool
	CanvasData     string
}

type CommonAreaGameData struct {
	CanvasData string
}

// GameHub, oyunun iş mantığından sorumludur.
type GameHub struct {
	hub *Hub
	// roomID -> game nesnesi
	activeGames map[uuid.UUID]*Game
	// gameModeName -> IGameEngine arayüzü
	gameEngines  map[string]IGameEngine
	roomSettings map[uuid.UUID]*GameSettings
	mutex        sync.RWMutex
}

func NewGameHub(hub *Hub) *GameHub {
	gameHub := &GameHub{
		hub:          hub,
		activeGames:  make(map[uuid.UUID]*Game),
		roomSettings: make(map[uuid.UUID]*GameSettings),
		gameEngines:  make(map[string]IGameEngine),
	}

	// gameHub.gameEngines["Çizim ve Tahmin"] = NewDrawingGameEngine(gameHub)
	gameHub.gameEngines["1"] = NewDrawingGameEngine(gameHub)
	// gameHub.gameEngines["Ortak Alan"] = NewDrawingGameEngine(gameHub)
	// gameHub.gameEngines["serbest çizim"] = NewDrawingGameEngine(gameHub)

	return gameHub
}

func (g *GameHub) HandleGameMessage(roomID uuid.UUID, msg RoomManagerData) {
	g.mutex.Lock()
	defer g.mutex.Unlock()

	fmt.Println("GameHub'da gelen mesaj:", msg.Type, "RoomID:", roomID)

	switch msg.Type {
	case "game_mode_change":
		g.handleGameModeChange(roomID, msg)
	case "game_settings_update":
		g.handleGameSettingsUpdate(roomID, msg)
	// case "game_started":
	// 	g.handleGameStarted(roomID, msg)
	// case "player_move":
	// 	g.handlePlayerMove(roomID, msg)
	// case "end_game":
	// 	g.handleEndGame(roomID, msg)
	// case "drawing_data":
	// 	g.handleDrawingData(roomID, msg)
	// case "guess_word":
	// 	g.handleGuessWord(roomID, msg)
	// case "next_round":
	// 	g.handleNextRound(roomID, msg)
	default:
		fmt.Printf("GameHub: Bilinmeyen mesaj tipi: %s\n", msg.Type)
	}
}

// handleGameModeChange, oyun modu değişikliğini işler
func (g *GameHub) handleGameModeChange(roomID uuid.UUID, msg RoomManagerData) {
	fmt.Printf("Oyun modu değişikliği - Room: %s\n", roomID)

	modeData, ok := msg.Content.(map[string]interface{})
	if !ok {
		fmt.Println("Game mode verisi parse edilemedi")
		return
	}
	fmt.Printf("Oyun modu değişikliği - data : %s\n", modeData)

	modeID, ok := modeData["mode_id"].(string)
	if !ok {
		fmt.Println("Mode id  bulunamadı")
		return
	}

	// Oyun motoru var mı kontrol et
	if _, exists := g.gameEngines[modeID]; !exists {
		fmt.Printf("Desteklenmeyen oyun modu: %s\n", modeID)
		return
	}

	// Odanın ayarlarını al veya oluştur
	settings, exists := g.roomSettings[roomID]
	if !exists {
		settings = g.getDefaultSettings(modeID)
	} else {
		settings.ModeID = modeID
		// Mode değiştiğinde ayarları yeniden hesapla
		g.calculateGameSettings(roomID, settings)
	}

	g.roomSettings[roomID] = settings

	// Oyun modu değişikliğini odadaki herkese bildir
	response := &Message{
		Type: "game_mode_changed",
		Content: map[string]interface{}{
			"room_id":  roomID,
			"mode_id":  modeID,
			"settings": settings,
		},
	}

	g.hub.BroadcastMessage(roomID, response)
	fmt.Printf("Oyun modu değiştirildi - Room: %s, Mode: %s\n", roomID, modeID)
}

// handleGameSettingsUpdate, oyun ayarları güncellemesini işler
func (g *GameHub) handleGameSettingsUpdate(roomID uuid.UUID, msg RoomManagerData) {
	fmt.Printf("Oyun ayarları güncelleniyor - Room: %s\n", roomID)

	settingsData, ok := msg.Content.(map[string]interface{})
	if !ok {
		fmt.Println("Settings verisi parse edilemedi")
		return
	}

	settings, exists := g.roomSettings[roomID]
	if !exists {
		// Varsayılan ayarları oluştur
		settings = &GameSettings{
			ModeName:      "Çizim ve Tahmin", // default
			TotalRounds:   2,
			RoundDuration: 60,
			MaxPlayers:    8,
			MinPlayers:    2,
		}
	}

	// Ayarları güncelle
	if rounds, ok := settingsData["total_rounds"].(float64); ok {
		settings.TotalRounds = int(rounds)
	}
	if duration, ok := settingsData["round_duration"].(float64); ok {
		settings.RoundDuration = int(duration)
	}
	if maxPlayers, ok := settingsData["max_players"].(float64); ok {
		settings.MaxPlayers = int(maxPlayers)
	}
	if minPlayers, ok := settingsData["min_players"].(float64); ok {
		settings.MinPlayers = int(minPlayers)
	}

	g.roomSettings[roomID] = settings

	// Ayar güncellemesini bildir
	response := &Message{
		Type: "game_settings_updated",
		Content: map[string]interface{}{
			"room_id":  roomID,
			"settings": settings,
		},
	}

	g.hub.BroadcastMessage(roomID, response)
	fmt.Printf("Oyun ayarları güncellendi - Room: %s\n", roomID)
}

// handleGameStarted, oyun başlatıldığında çağrılır
func (g *GameHub) handleGameStarted(roomID uuid.UUID, msg RoomManagerData) {
	fmt.Printf("Oyun başlatılıyor - Room: %s\n", roomID)

	// Odanın ayarlarını kontrol et
	settings, exists := g.roomSettings[roomID]
	if !exists {
		fmt.Printf("Oda ayarları bulunamadı, varsayılan ayarlar kullanılıyor - Room: %s\n", roomID)
		settings = g.getDefaultSettings("Çizim ve Tahmin")
		g.roomSettings[roomID] = settings
	}

	// Odadaki oyuncu sayısını kontrol et
	playerCount := g.hub.GetRoomClientCount(roomID)
	if playerCount < settings.MinPlayers {
		fmt.Printf("Yetersiz oyuncu sayısı - Room: %s, Mevcut: %d, Minimum: %d\n",
			roomID, playerCount, settings.MinPlayers)

		// Yetersiz oyuncu mesajı gönder
		response := &Message{
			Type: "game_start_failed",
			Content: map[string]interface{}{
				"room_id": roomID,
				"reason":  "insufficient_players",
				"message": fmt.Sprintf("Minimum %d oyuncu gerekli", settings.MinPlayers),
			},
		}
		g.hub.BroadcastMessage(roomID, response)
		return
	}

	// Odadaki oyuncuları al (Bu fonksiyonu Hub'a eklemen gerekecek)
	players := g.getRoomPlayers(roomID)

	// Yeni oyun oluştur
	game := &Game{
		RoomID:        roomID,
		ModeName:      settings.ModeName,
		State:         GameStateInProgress,
		Players:       players,
		TurnCount:     0,
		TotalRounds:   settings.TotalRounds,
		RoundDuration: settings.RoundDuration,
		LastMoveTime:  time.Now(),
	}

	// Oyun motorunu al ve oyunu başlat
	engine, exists := g.gameEngines[settings.ModeName]
	if !exists {
		fmt.Printf("Oyun motoru bulunamadı: %s\n", settings.ModeName)
		return
	}

	if err := engine.InitGame(game, players); err != nil {
		fmt.Printf("Oyun başlatılamadı: %v\n", err)
		return
	}

	// Aktif oyunlar listesine ekle
	g.activeGames[roomID] = game

	// Oyun başladı mesajını tüm oyunculara gönder
	response := &Message{
		Type: "game_started",
		Content: map[string]interface{}{
			"room_id":        roomID,
			"mode_name":      settings.ModeName,
			"players":        g.playersToMap(players),
			"total_rounds":   game.TotalRounds,
			"round_duration": game.RoundDuration,
			"current_round":  1,
		},
	}

	g.hub.BroadcastMessage(roomID, response)
	fmt.Printf("Oyun başlatıldı - Room: %s, Mode: %s, Oyuncu Sayısı: %d\n",
		roomID, settings.ModeName, len(players))
}

// getDefaultSettings, oyun moduna göre varsayılan ayarları döner
func (g *GameHub) getDefaultSettings(modeName string) *GameSettings {
	switch modeName {
	case "Çizim ve Tahmin":
		return &GameSettings{
			ModeName:      modeName,
			TotalRounds:   2, // Her oyuncu 2 kez çizer
			RoundDuration: 60,
			MaxPlayers:    8,
			MinPlayers:    2,
		}
	case "Ortak Alan":
		return &GameSettings{
			ModeName:      modeName,
			TotalRounds:   1,
			RoundDuration: 120, //2 dakika
			MaxPlayers:    10,
			MinPlayers:    2,
		}
	default:
		return &GameSettings{
			ModeName:      "Çizim ve Tahmin",
			TotalRounds:   2,
			RoundDuration: 60,
			MaxPlayers:    8,
			MinPlayers:    2,
		}
	}
}

// calculateGameSettings, oda durumuna göre ayarları hesaplar
func (g *GameHub) calculateGameSettings(roomID uuid.UUID, settings *GameSettings) {
	playerCount := g.hub.GetRoomClientCount(roomID)

	// Çizim ve Tahmin modunda her oyuncu çizecekse
	if settings.ModeName == "Çizim ve Tahmin" {
		// Her oyuncunun çizme fırsatı olması için round sayısını ayarla
		if playerCount > 0 {
			settings.TotalRounds = playerCount * 2 // Her oyuncu 2 kez çizer
		}
	}
}

// getRoomPlayers, odadaki oyuncuları Player yapısına dönüştürür
func (g *GameHub) getRoomPlayers(roomID uuid.UUID) []*Player {
	// Bu fonksiyon için Hub'da roomsClients'a erişim sağlayacak bir method gerekli
	// Şimdilik basit bir implementation
	playerCount := g.hub.GetRoomClientCount(roomID)

	var players []*Player
	for i := 0; i < playerCount; i++ {
		// Bu kısım gerçek implementasyonda Hub'dan client bilgileri alınacak
		players = append(players, &Player{
			UserID:   uuid.New(),                   // Geçici
			Username: fmt.Sprintf("Player%d", i+1), // Geçici
			Score:    0,
		})
	}

	return players
}

// playersToMap, Player slice'ını map formatına dönüştürür
func (g *GameHub) playersToMap(players []*Player) []map[string]interface{} {
	var result []map[string]interface{}

	for _, player := range players {
		result = append(result, map[string]interface{}{
			"user_id":  player.UserID,
			"username": player.Username,
			"score":    player.Score,
		})
	}

	return result
}



// IsGameActive, odada aktif oyun olup olmadığını kontrol eder
func (g *GameHub) IsGameActive(roomID uuid.UUID) bool {
	g.mutex.RLock()
	defer g.mutex.RUnlock()

	game, exists := g.activeGames[roomID]
	return exists && game.State == GameStateInProgress
}

// Diğer handler metodları aynı kalacak...
func (g *GameHub) handlePlayerMove(roomID uuid.UUID, msg RoomManagerData) {
	// Önceki implementation aynı
}

func (g *GameHub) handleEndGame(roomID uuid.UUID, msg RoomManagerData) {
	// Oyun bittiğinde roomSettings'i de temizle
	delete(g.activeGames, roomID)
	delete(g.roomSettings, roomID)

	// Diğer işlemler...
}

func (g *GameHub) handleDrawingData(roomID uuid.UUID, msg RoomManagerData) {
	// Önceki implementation aynı
	response := &Message{
		Type:    "drawing_data",
		Content: msg.Content,
	}

	g.hub.BroadcastMessage(roomID, response)
}

func (g *GameHub) handleGuessWord(roomID uuid.UUID, msg RoomManagerData) {
	// Önceki implementation aynı
}

func (g *GameHub) handleNextRound(roomID uuid.UUID, msg RoomManagerData) {
	// Önceki implementation aynı
}
