package hub

import (
	"context"
	"fmt"
	"log"
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
	StartRound(game *Game) error
	EndRound(game *Game, reason string) bool
	// Sadece bildirim gönderme gibi genel yardımcı metotlar arayüze eklenebilir.
	SendPreparationNotifications(game *Game)
}

// Player, oyundaki bir oyuncuyu temsil eder.
type Player struct {
	UserID      uuid.UUID
	Username    string
	Score       int
	IsConnected bool
	// Oyuncuya özgü diğer veriler
}

type GameSettings struct {
	ModeName            string `json:"mode_name"`
	ModeID              string `json:"mode_id"`
	TotalRounds         int    `json:"total_rounds"`
	RoundDuration       int    `json:"round_duration"` // saniye cinsinden
	PreparationDuration int    `json:"preparation_duration"`
	MaxPlayers          int    `json:"max_players"`
	MinPlayers          int    `json:"min_players"`
}

// Game, bir oyunun mevcut durumunu tutar.
type Game struct {
	RoomID              uuid.UUID   `json:"room_id"`
	ModeName            string      `json:"mode_name"`
	ModeID              string      `json:"mode_id"`
	State               string      `json:"state"`
	Players             []*Player   `json:"players"`
	TurnCount           int         `json:"turn_count"`
	TotalRounds         int         `json:"total_rounds"`
	RoundDuration       int         `json:"round_duration"`
	ActivePlayer        uuid.UUID   `json:"active_player"`
	LastMoveTime        time.Time   `json:"last_move_time"`
	PreparationDuration int         `json:"preparation_duration"` // 🎯 YENİ
	ModeData            interface{} `json:"mode_data"`
	CurrentDrawerIndex  int         `json:"current_drawer_index"`
	Mutex               sync.RWMutex
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
	gameEngines     map[string]IGameEngine
	roomSettings    map[uuid.UUID]*GameSettings
	roundTimers     map[uuid.UUID]context.CancelFunc
	timerWaitGroups map[uuid.UUID]*sync.WaitGroup
	roundEndSignal  chan RoundEndSignal

	mutex sync.RWMutex
}
type RoundEndSignal struct {
	RoomID uuid.UUID
	Reason string
}

func NewGameHub(hub *Hub) *GameHub {
	gameHub := &GameHub{
		hub:             hub,
		activeGames:     make(map[uuid.UUID]*Game),
		roomSettings:    make(map[uuid.UUID]*GameSettings),
		gameEngines:     make(map[string]IGameEngine),
		roundTimers:     make(map[uuid.UUID]context.CancelFunc),
		timerWaitGroups: make(map[uuid.UUID]*sync.WaitGroup),
		roundEndSignal:  make(chan RoundEndSignal, 5),
	}

	// gameHub.gameEngines["Çizim ve Tahmin"] = NewDrawingGameEngine(gameHub)
	gameHub.gameEngines["1"] = NewDrawingGameEngine(gameHub)
	gameHub.gameEngines["2"] = NewCollaborativeArtEngine(gameHub)
	// gameHub.gameEngines["Ortak Alan"] = NewDrawingGameEngine(gameHub)
	// gameHub.gameEngines["serbest çizim"] = NewDrawingGameEngine(gameHub)
	go gameHub.RunListener()
	return gameHub
}

func (gh *GameHub) GetActiveGame(roomID uuid.UUID) *Game {
	gh.mutex.RLock()
	defer gh.mutex.RUnlock()
	return gh.activeGames[roomID]
}
func (g *GameHub) RunListener() {
	for {
		select {
		case quit := <-g.hub.playerQuit: // (Önceki işlevsellik)
			log.Printf("RUN_LISTENER: Player quit received - Room: %s, User: %s\n", quit.RoomID, quit.UserID)
			g.HandlePlayerQuit(quit.RoomID, quit.UserID)

		// 💡 YENİ CASE: Tur bitiş sinyalini işle
		case endSignal := <-g.roundEndSignal:
			// Sinyal geldiğinde, GÜVENLİ bir şekilde handleRoundEnd'i çağır.
			// Bu zaten bir Gorutin içinde olduğu için kilitlenme riski düşüktür.
			log.Printf("RUN_LISTENER: Round end signal received for room %s. Reason: %s", endSignal.RoomID, endSignal.Reason)
			g.handleRoundEnd(endSignal.RoomID, endSignal.Reason)
		}
	}
}
func (g *GameHub) HandlePlayerQuit(roomID uuid.UUID, userID uuid.UUID) {
	log.Printf("HandlePlayerQuit called for room %s, user %s", roomID, userID)

	g.mutex.Lock()

	game, exists := g.activeGames[roomID]
	if !exists || game.State != GameStateInProgress {
		g.mutex.Unlock()
		log.Printf("No active game found for room %s", roomID)
		return
	}

	// 🔍 Oyuncuyu bul
	var targetPlayer *Player
	for _, p := range game.Players {
		if p.UserID == userID {
			targetPlayer = p
			break
		}
	}

	if targetPlayer == nil {
		g.mutex.Unlock()
		log.Printf("Player %s not found in game", userID)
		return
	}

	// Oyun ayarlarını kontrol et
	settings, settingsExist := g.roomSettings[roomID]
	if !settingsExist {
		g.mutex.Unlock()
		log.Printf("WARNING: Room settings not found for room %s", roomID)
		return
	}

	// Ayrılan oyuncu aktif çizen miydi?
	wasActiveDrawer := game.ActivePlayer == userID

	g.mutex.Unlock()

	// 🆕 Reconnect için grace period (30 saniye) başlat
	go g.handleDisconnectWithGracePeriod(roomID, userID, wasActiveDrawer, settings.MinPlayers)
	// // Oyuncu listesinden çıkar
	// newPlayers := make([]*Player, 0, len(game.Players)-1)
	// playerFound := false
	// var removedPlayer *Player

	// for _, p := range game.Players {
	// 	if p.UserID == userID {
	// 		playerFound = true
	// 		removedPlayer = p
	// 		continue
	// 	}
	// 	newPlayers = append(newPlayers, p)
	// }

	// if !playerFound {
	// 	g.mutex.Unlock()
	// 	log.Printf("Player %s not found in game", userID)
	// 	return
	// }

	// // Oyuncu listesini güncelle
	// game.Players = newPlayers
	// remainingPlayerCount := len(game.Players)

	// log.Printf("Player %s removed. Remaining players: %d", userID, remainingPlayerCount)

	// // Oyun ayarlarını kontrol et
	// settings, settingsExist := g.roomSettings[roomID]
	// if !settingsExist {
	// 	g.mutex.Unlock()
	// 	log.Printf("WARNING: Room settings not found for room %s", roomID)
	// 	return
	// }

	// // Ayrılan oyuncu aktif çizen miydi?
	// wasActiveDrawer := game.ActivePlayer == userID

	// // 🔥 ÖNEMLİ: Mutex'i burada unlock et - çünkü aşağıdaki işlemler lock gerektiriyor
	// g.mutex.Unlock()

	// // Kalan oyuncu sayısı minimum sayısından az mı?
	// if remainingPlayerCount < settings.MinPlayers {
	// 	log.Printf("Insufficient players (%d < %d). Ending game for room %s",
	// 		remainingPlayerCount, settings.MinPlayers, roomID)

	// 	// Zamanlayıcıyı durdur
	// 	g.stopRoundTimer(roomID)

	// 	// Oyunu bitir
	// 	g.handleEndGame(roomID, RoomManagerData{
	// 		Type: "end_game",
	// 		Content: map[string]interface{}{
	// 			"room_id": roomID,
	// 			"reason":  "insufficient_players",
	// 			"message": fmt.Sprintf("Oyun sonlandırıldı. Minimum %d oyuncu gerekli.", settings.MinPlayers),
	// 		},
	// 	})
	// 	return
	// }

	// // Oyunculara ayrılma bildirimini gönder
	// g.hub.BroadcastMessage(roomID, &Message{
	// 	Type: "player_left",
	// 	Content: map[string]interface{}{
	// 		"room_id":       roomID,
	// 		"user_id":       userID,
	// 		"username":      removedPlayer.Username,
	// 		"remaining":     remainingPlayerCount,
	// 		"active_drawer": wasActiveDrawer,
	// 	},
	// })

	// // Eğer ayrılan oyuncu çizen ise, turu bitir
	// if wasActiveDrawer {
	// 	log.Printf("Active drawer %s left. Ending round for room %s", userID, roomID)

	// 	// 💡 NON-BLOCKING şekilde roundEndSignal gönder
	// 	select {
	// 	case g.roundEndSignal <- RoundEndSignal{
	// 		RoomID: roomID,
	// 		Reason: "drawer_left",
	// 	}:
	// 		log.Printf("Round end signal sent for room %s", roomID)
	// 	default:
	// 		log.Printf("WARNING: roundEndSignal channel full, calling handleRoundEnd directly")
	// 		g.handleRoundEnd(roomID, "drawer_left")
	// 	}
	// }
}

func (g *GameHub) handleDisconnectWithGracePeriod(roomID, userID uuid.UUID, wasActiveDrawer bool, minPlayers int) {
	gracePeriod := 30 * time.Second
	log.Printf("Starting grace period (%v) for player %s in room %s", gracePeriod, userID, roomID)

	// Grace period süresi boyunca bekle
	timer := time.NewTimer(gracePeriod)
	defer timer.Stop()

	<-timer.C

	// Süre doldu, oyuncu hala bağlanmadı mı kontrol et
	g.mutex.Lock()
	game, exists := g.activeGames[roomID]
	if !exists || game.State != GameStateInProgress {
		g.mutex.Unlock()
		log.Printf("Game no longer active for room %s", roomID)
		return
	}

	// 🔍 Oyuncu hala oyunda mı ve bağlı değil mi?
	playerStillDisconnected := false
	var removedPlayer *Player
	newPlayers := make([]*Player, 0, len(game.Players)-1)

	for _, p := range game.Players {
		if p.UserID == userID {
			// Hub'dan kontrol et - oyuncu yeniden bağlandı mı?
			if g.hub.IsClientConnected(roomID, userID) {
				log.Printf("Player %s reconnected within grace period, keeping in game", userID)
				g.mutex.Unlock()
				return
			}
			playerStillDisconnected = true
			removedPlayer = p
			continue
		}
		newPlayers = append(newPlayers, p)
	}

	if !playerStillDisconnected {
		g.mutex.Unlock()
		log.Printf("Player %s not found or already removed from game", userID)
		return
	}

	// Oyuncuyu listeden çıkar
	game.Players = newPlayers
	remainingPlayerCount := len(game.Players)

	log.Printf("Grace period expired. Player %s removed. Remaining players: %d", userID, remainingPlayerCount)

	g.mutex.Unlock()

	// Kalan oyuncu sayısı yetersiz mi?
	if remainingPlayerCount < minPlayers {
		log.Printf("Insufficient players (%d < %d). Ending game for room %s",
			remainingPlayerCount, minPlayers, roomID)

		g.stopRoundTimer(roomID)
		g.handleEndGame(roomID, RoomManagerData{
			Type: "end_game",
			Content: map[string]interface{}{
				"room_id": roomID,
				"reason":  "insufficient_players",
				"message": fmt.Sprintf("Oyun sonlandırıldı. Minimum %d oyuncu gerekli.", minPlayers),
			},
		})
		return
	}

	// Oyunculara ayrılma bildirimini gönder
	g.hub.BroadcastMessage(roomID, &Message{
		Type: "player_left",
		Content: map[string]interface{}{
			"room_id":       roomID,
			"user_id":       userID,
			"username":      removedPlayer.Username,
			"remaining":     remainingPlayerCount,
			"active_drawer": wasActiveDrawer,
		},
	})

	if wasActiveDrawer {
		log.Printf("Active drawer %s left. Ending round for room %s", userID, roomID)

		// 💡 NON-BLOCKING şekilde roundEndSignal gönder
		select {
		case g.roundEndSignal <- RoundEndSignal{
			RoomID: roomID,
			Reason: "drawer_left",
		}:
			log.Printf("Round end signal sent for room %s", roomID)
		default:
			log.Printf("WARNING: roundEndSignal channel full, calling handleRoundEnd directly")
			g.handleRoundEnd(roomID, "drawer_left")
		}
	}
}
func (g *GameHub) startRoundTimer(roomID uuid.UUID, duration time.Duration) {
	// Önceki zamanlayıcı varsa durdur ve bekle (Bloklama burada oluyor)
	g.stopRoundTimer(roomID)

	// Yeni zamanlayıcı için hazırlık
	wg := &sync.WaitGroup{}
	wg.Add(1)

	// 💡 DEĞİŞİKLİK: context.WithCancel yerine context.WithTimeout kullanın!
	ctx, cancel := context.WithTimeout(context.Background(), duration)

	// Haritalara kaydet
	g.mutex.Lock()
	g.roundTimers[roomID] = cancel
	g.timerWaitGroups[roomID] = wg
	g.mutex.Unlock()
	log.Printf("START_TIMER: Goroutine started for room %s, duration: %v", roomID, duration)
	go func() {
		defer wg.Done() // Gorutin işini bitirdiğinde sayacı azalt.

		select {
		case <-ctx.Done():
			// Context, ya süre dolduğu için (Timeout) ya da manuel iptal (Cancel)
			// nedeniyle kapandı.
			log.Printf("TIMER_GOROUTINE: Context closed for room %s. Error: %v", roomID, ctx.Err())
			// 💡 Kontrol: Süre mi doldu, yoksa iptal mi edildi?
			if ctx.Err() == context.DeadlineExceeded {
				g.roundEndSignal <- struct {
					RoomID uuid.UUID
					Reason string
				}{
					RoomID: roomID,
					Reason: "time_expired",
				}
			}
			// (context.Canceled ise, manuel bitmiştir ve bu zaten EndRound'ı tetiklemiştir.)
			return
		}
	}()
}

// package hub
func (g *GameHub) stopRoundTimer(roomID uuid.UUID) {
	var cancelFunc context.CancelFunc
	var wg *sync.WaitGroup
	var cancelExists, wgExists bool

	// 1. Lock/Unlock ve Haritalardan Kaldırma
	g.mutex.Lock()
	cancelFunc, cancelExists = g.roundTimers[roomID]
	wg, wgExists = g.timerWaitGroups[roomID]

	delete(g.roundTimers, roomID)
	delete(g.timerWaitGroups, roomID)
	g.mutex.Unlock() // Kilit serbest bırakıldı.

	// 2. İptal Sinyalini Gönder
	if cancelExists {
		cancelFunc()
		log.Printf("STOP_TIMER: Cancel signal sent for room %s.", roomID)
	}

	// 3. Goroutine'in Bitmesini GÜVENLİ BİR ŞEKİLDE Bekle (Zaman Aşımı)
	if wgExists {
		done := make(chan struct{})

		// Goroutine'i bekleyen AYRI bir Goroutine başlat
		go func() {
			// ÖNEMLİ: Eğer timer Goroutine'de birden fazla wg.Done() varsa
			// veya hiç yoksa, bu kilitlenir. Ancak timeout bunu çözer.
			wg.Wait()
			close(done)
		}()

		// Zaman aşımı ile bekle
		select {
		case <-done:
			// Başarılı: Goroutine sonlandı.
			log.Printf("STOP_TIMER: Old timer for %s safely stopped.", roomID)
		case <-time.After(500 * time.Millisecond): // 0.5 saniye yeterli olmalı
			// Zaman Aşımı: 0.5 saniye içinde sonlanmadı, ancak devam et.
			log.Printf("STOP_TIMER: WARNING: Old timer for %s did not terminate in 500ms. Continuing.", roomID)
		}
	}
}

func (g *GameHub) handleRoundEnd(roomID uuid.UUID, reason string) {
	// 1. En üst seviye kilit: GameHub'ı kilitliyoruz.

	log.Printf("HANDLE_ROUND_END: Starting round end process for room %s. Reason: %s", roomID, reason)
	// 2. Zamanlayıcıyı hemen durdur.
	g.stopRoundTimer(roomID)
	g.mutex.Lock()
	defer g.mutex.Unlock()
	fmt.Println("Round timer stopped for room", roomID)

	game, exists := g.activeGames[roomID]
	if !exists {
		return // Oyun zaten bitmiş olabilir
	}
	log.Printf("Round ended for room %s. Reason: %s", roomID, reason)

	engine, ok := g.gameEngines[game.ModeID]
	if !ok {
		log.Printf("Game engine not found for mode: %s", game.ModeID)
		return
	}
	// dge, _ := engine.(*DrawingGameEngine)
	log.Printf("HANDLE_ROUND_END: Attempting to acquire game.Mutex for room %s.", roomID)
	// 3. Oyun durumunu güncellemek için Game kilidini alıyoruz.
	game.Mutex.Lock()

	// dge.EndRound metodu, puanlama ve tur/oyun bitiş kontrolünü yapar.
	// Artık bu metodun içinde kilit yok.
	shouldContinue := engine.EndRound(game, reason)

	// Game kilidini serbest bırak (çok önemli!).
	game.Mutex.Unlock()
	log.Printf("HANDLE_ROUND_END: EndRound finished for room %s. Should continue: %v", roomID, shouldContinue)
	gameSnapshot := *game
	var cleanModeData interface{}
	if game.ModeID == "2" {
		artData, ok := game.ModeData.(*CollaborativeArtData)
		if ok {
			// RoundHistory'si olmayan yeni bir CollaborativeArtData oluştur.
			cleanModeData = &CollaborativeArtData{
				CurrentWord:    artData.CurrentWord,
				CurrentStrokes: artData.CurrentStrokes, // Bu zaten boş olmalı
				RoundHistory:   nil,                    // 🔑 ÖNEMLİ: Geçmişi gönderme!
			}
			gameSnapshot.ModeData = cleanModeData
		}
	}
	// 4. Her tur bittiğinde oyunculara genel bir "tur bitti" mesajı yayınla.
	g.hub.BroadcastMessage(roomID, &Message{
		Type: "round_ended",
		Content: map[string]interface{}{
			"room_id": roomID,
			"reason":  reason,
			"game":    gameSnapshot, // Güncel oyun durumunu gönder
		},
	})

	// 5. Bir sonraki tura geçilecek mi, yoksa oyun mu bitecek kararını ver.
	if shouldContinue {
		// Yeni tur varsa:
		log.Printf("NEXT_ROUND: Starting in background for room %s.", roomID)
		// Yeniden Game kilidini alıyoruz, çünkü StartRound oyun nesnesini değiştirecek.
		// ⚠️ KRİTİK DEĞİŞİKLİK: StartRound ve Timer'ı yeni bir Goroutine'e taşı!
		go func(g *GameHub, engine IGameEngine, game *Game, roomID uuid.UUID) {
			preparationDuration := time.Duration(game.PreparationDuration) * time.Second

			game.Mutex.Lock()
			engine.SendPreparationNotifications(game)
			game.Mutex.Unlock()

			log.Printf("PREPARATION: Waiting %v seconds before starting round for room %s",
				game.PreparationDuration, roomID)

			// 2. HAZIRLIK SÜRESİNİ BEKLE
			time.Sleep(preparationDuration)
			// Yeni turu başlat (Game kilidi GOROUTINE içinde alınmalı!)
			game.Mutex.Lock()

			if err := engine.StartRound(game); err != nil {
				log.Printf("GOROUTINE ERROR: Error starting next round: %v", err)
			}

			game.Mutex.Unlock()

			// Yeni tur zamanlayıcısını başlat.
			duration := time.Duration(game.RoundDuration) * time.Second
			g.startRoundTimer(roomID, duration)

		}(g, engine, game, roomID) // Değişkenleri Goroutine'e geçir.

		log.Printf("NEXT_ROUND: Starting in background for room %s.", roomID)

	} else {
		// 🚨 OYUN BİTTİYSE: Moda özel sonlandırma ve raporlama.
		log.Printf("GAME_OVER: Game finished for room %s. Mode: %s", roomID, game.ModeID)

		gameOverContent := make(map[string]interface{})
		gameOverContent["scores"] = game.Players // Skorları her zaman göndermek kötü değil.
		gameOverContent["data"] = game.ModeData

		// 🎯 KRİTİK DEĞİŞİKLİK: Sadece DrawingGameEngine gibi puanlamalı modlar için kazananı belirle.
		if game.ModeID == "1" {
			// Motoru somut tipine dönüştürmemiz GEREKİYOR, çünkü determineWinner IGameEngine'de yok.
			// Bu, arayüzün zayıflığıdır, ancak puanlamasız modları desteklemek için gerekli bir tavizdir.
			dge, ok := engine.(*DrawingGameEngine)
			if ok {
				gameOverContent["winner"] = dge.determineWinner(game)
				dge.SendFinalArtReport(game)
			}
		} else if game.ModeID == "2" {
			// CollaborativeArtEngine'e özel bir "Oyun Bitti" aksiyonu varsa çağır.
			// Örneğin, önceden tanımladığınız SendFinalArtReport metodu buraya gelir.
			// Not: Bu çağrı zaten EndRound içinde de yapılıyor olabilir, kontrol edin.
			// Eğer SendFinalArtReport çağrısı EndRound içinde yapılmıyorsa:
			// cae, ok := engine.(*CollaborativeArtEngine)
			// if ok { cae.SendFinalArtReport(game) }
			cae, ok := engine.(*CollaborativeArtEngine)
			if ok {
				// Bu, RoundHistory'yi toplayıp özel bir 'game_over_report' mesajı yayınlar.
				cae.SendFinalArtReport(game)
			}
			// Ortak Sanat Projesinde kazanan yerine sadece final rapor bilgisi gönderilir.
			gameOverContent["message"] = "Ortak Sanat Projesi Tamamlandı. Lütfen Raporu kontrol edin."

		}

		// Oyun Bitti mesajını yayınla.
		// g.hub.BroadcastMessage(game.RoomID, &Message{
		// 	Type:    "game_over",
		// 	Content: gameOverContent,
		// })

		// Aktif oyunlardan kaldır.
		delete(g.activeGames, roomID)
		// delete(g.roomSettings, roomID)
	}
}
func (g *GameHub) HandleGameMessage(roomID uuid.UUID, msg RoomManagerData) {
	// g.mutex.Lock()
	// defer g.mutex.Unlock()

	fmt.Println("GameHub'da gelen mesaj:", msg.Type, "RoomID:", roomID)

	switch msg.Type {
	case "game_mode_change":
		g.handleGameModeChange(roomID, msg)
	case "game_settings_update":
		g.handleGameSettingsUpdate(roomID, msg)
	case "game_started":
		g.handleGameStarted(roomID, msg)
	case "player_move":
		g.handlePlayerMove(roomID, msg)
	case "canvas_action":
		g.handlePlayerMove(roomID, msg)

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
		settings.ModeID = modeID
		settings.ModeName = modeData["mode_name"].(string)
	} else {
		settings.ModeID = modeID
		// Mode değiştiğinde ayarları yeniden hesapla
		// g.calculateGameSettings(roomID, settings)
	}

	g.roomSettings[roomID] = settings

	// Oyun modu değişikliğini odadaki herkese bildir
	response := &Message{
		Type: "game_mode_changed",
		Content: map[string]interface{}{
			"room_id":      roomID,
			"game_mode_id": modeID,
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
			ModeName:            "Çizim ve Tahmin", // default
			ModeID:              "1",
			TotalRounds:         2,
			RoundDuration:       60,
			PreparationDuration: 5, // 🎯 Varsayılan 5 saniye
			MaxPlayers:          8,
			MinPlayers:          2,
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
			"max_players":    settings.MaxPlayers,
			"min_players":    settings.MinPlayers,
			"game_mode_id":   settings.ModeID,
			"mode_name":      settings.ModeName,
			"total_rounds":   settings.TotalRounds,
			"round_duration": settings.RoundDuration,
		},
	}

	g.hub.BroadcastMessage(roomID, response)
	fmt.Printf("Oyun ayarları güncellendi - Room: %s\n", roomID)
}

// handleGameStarted, oyun başlatıldığında çağrılır
func (g *GameHub) handleGameStarted(roomID uuid.UUID, msg RoomManagerData) {
	fmt.Printf("Oyun başlatılıyor - Room: %s\n", roomID)

	g.mutex.RLock()
	game, gameExists := g.activeGames[roomID]
	settings, settingsExists := g.roomSettings[roomID]
	g.mutex.RUnlock() // 🛑 Okuma bitti, GameHub kilidini serbest bırak!

	if gameExists && game.State == GameStateInProgress {
		fmt.Printf("Oyun zaten devam ediyor. Yeni oyun başlatma isteği reddedildi - Room: %s\n", roomID)
		// Oyunculara hata mesajı gönder

		g.hub.BroadcastMessage(roomID, &Message{
			Type: "game_start_failed",
			Content: map[string]interface{}{
				"room_id": roomID,
				"reason":  "game_already_in_progress",
				"message": "Bu odada zaten bir oyun devam ediyor.",
			},
		})
		return
	}

	if !settingsExists {
		fmt.Printf("Oda ayarları bulunamadı, varsayılan ayarlar kullanılıyor - Room: %s\n", roomID)
		settings = g.getDefaultSettings("Çizim ve Tahmin")

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
	initialPlayerCount := len(players)

	if settings.TotalRounds < initialPlayerCount {
		settings.TotalRounds = initialPlayerCount
	}

	// Yeni oyun oluştur

	newGame := &Game{
		RoomID:      roomID,
		ModeName:    settings.ModeName,
		ModeID:      settings.ModeID,
		State:       GameStateInProgress,
		Players:     players,
		TurnCount:   0,
		TotalRounds: settings.TotalRounds,
		//InitialPlayerCount: initialPlayerCount,
		PreparationDuration: settings.PreparationDuration,
		RoundDuration:       settings.RoundDuration,
		LastMoveTime:        time.Now(),
	}

	g.mutex.RLock()
	engine, engineExists := g.gameEngines[settings.ModeID]
	g.mutex.RUnlock()
	if !engineExists {
		fmt.Printf("Oyun motoru bulunamadı: %v\n", settings)
		return
	}

	if err := engine.InitGame(newGame, players); err != nil {
		fmt.Printf("Oyun başlatılamadı: %v\n", err)
		return
	}

	// Aktif oyunlar listesine ekle
	g.mutex.Lock()
	g.activeGames[roomID] = newGame
	if !settingsExists {
		g.roomSettings[roomID] = settings // 🛑 Ayar yoksa, onu da kaydet
	}
	g.mutex.Unlock()
	// Oyun başladı mesajını tüm oyunculara gönder
	response := &Message{
		Type: "game_started",
		Content: map[string]interface{}{
			"room_id":              roomID,
			"mode_name":            settings.ModeName,
			"mode_id":              settings.ModeName,
			"players":              g.playersToMap(players),
			"total_rounds":         newGame.TotalRounds,
			"round_duration":       newGame.RoundDuration,
			"initial_player_count": initialPlayerCount,
			"preparation_duration": newGame.PreparationDuration,
			"current_round":        1,
		},
	}
	g.hub.BroadcastMessage(roomID, response) // 💡 İLK MESAJ GİTTİ!

	go func(g *GameHub, engine IGameEngine, game *Game, roomID uuid.UUID) {
		preparationDuration := time.Duration(game.PreparationDuration) * time.Second

		game.Mutex.Lock()
		engine.SendPreparationNotifications(game)
		game.Mutex.Unlock()

		log.Printf("FIRST_ROUND_PREP: Waiting %v seconds before starting first round for room %s",
			game.PreparationDuration, roomID)

		// 2. HAZIRLIK SÜRESİ BEKLE
		time.Sleep(preparationDuration)

		// 4. İLK TURU BAŞLAT
		// game.Mutex'i burada kullanabilirsiniz (StartRound'un iç yapısına bağlı olarak).
		game.Mutex.Lock() // Eğer StartRound game objesini değiştiriyorsa
		if err := engine.StartRound(game); err != nil {
			fmt.Printf("GOROUTINE: İlk tur başlatılamadı: %v\n", err)
		}
		game.Mutex.Unlock() // Eğer StartRound game objesini değiştiriyorsa

		// 5. ZAMANLAYICIYI BAŞLAT
		duration := time.Duration(game.RoundDuration) * time.Second
		// 💡 Bu çağrı RunListener'a sinyal göndereceği için, oyun döngüsü başlar.
		g.startRoundTimer(roomID, duration)

	}(g, engine, newGame, roomID)
	fmt.Printf("Oyun başlatıldı - Room: %s, Mode: %s, Oyuncu Sayısı: %d\n",
		roomID, settings.ModeName, len(players))
}

// getDefaultSettings, oyun moduna göre varsayılan ayarları döner
func (g *GameHub) getDefaultSettings(modeName string) *GameSettings {
	switch modeName {
	case "Çizim ve Tahmin":
		return &GameSettings{
			ModeName:            modeName,
			ModeID:              "1",
			TotalRounds:         2, // Her oyuncu 2 kez çizer
			RoundDuration:       60,
			MaxPlayers:          8,
			PreparationDuration: 5,
			MinPlayers:          2,
		}
	case "Ortak Alan":
		return &GameSettings{
			ModeName:            modeName,
			ModeID:              "2",
			TotalRounds:         1,
			RoundDuration:       120, //2 dakika
			PreparationDuration: 5,
			MaxPlayers:          10,
			MinPlayers:          2,
		}
	default:
		return &GameSettings{
			ModeName:            "Çizim ve Tahmin",
			TotalRounds:         2,
			RoundDuration:       60,
			PreparationDuration: 5,
			MaxPlayers:          8,
			MinPlayers:          2,
		}
	}
}

// calculateGameSettings, oda durumuna göre ayarları hesaplar
// func (g *GameHub) calculateGameSettings(roomID uuid.UUID, settings *GameSettings) {
// 	playerCount := g.hub.GetRoomClientCount(roomID)

// 	// Çizim ve Tahmin modunda her oyuncu çizecekse
// 	if settings.ModeName == "Çizim ve Tahmin" {
// 		// Her oyuncunun çizme fırsatı olması için round sayısını ayarla
// 		if playerCount > 0 {
// 			settings.TotalRounds = playerCount * 2 // Her oyuncu 2 kez çizer
// 		}
// 	}
// }

// getRoomPlayers, odadaki oyuncuları Player yapısına dönüştürür
func (g *GameHub) getRoomPlayers(roomID uuid.UUID) []*Player {

	// Hub'dan odadaki bağlı client'ları al
	roomClients := g.hub.GetRoomClients(roomID)

	if len(roomClients) == 0 {
		return nil
	}

	var players []*Player

	// Hub'dan gelen her Client nesnesini bir Player nesnesine dönüştür
	for _, client := range roomClients {
		// NOT: Client nesnesinde sadece ID var.
		// Eğer Username/Kullanıcı Adı bilgisini client nesnesinde veya veritabanında tutuyorsanız,
		// onu kullanmalısınız.

		// Varsayım: `domain.Client` yapınızda `Username` alanı var.
		// Eğer yoksa, geçici olarak ID'yi veya veritabanından çekilen bilgiyi kullanın.

		username := fmt.Sprintf("User-%s", client.ID.String()[:4]) // Geçici: ID'nin bir kısmını kullan

		// Eğer `domain.Client` yapınızda kullanıcı adı alanı varsa:
		// username := client.Username // Burası domain.Client yapısına bağlı

		players = append(players, &Player{
			UserID:   client.ID,
			Username: username, // Gerçek kullanıcı adını buradan alın
			Score:    0,        // Yeni oyunda skor her zaman 0 başlar
		})
	}

	// Oyun başlama sırasını karıştırmak isterseniz burada karıştırma (shuffle) yapabilirsiniz.

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
	//g.mutex.RLock() // ActiveGames ve GameEngines'a erişmek için
	g.mutex.RLock()
	game, exists := g.activeGames[roomID]

	if !exists {
		g.mutex.RUnlock() // 🛑 Erken çıkışta kilidi bırak!
		log.Printf("PLAYER_MOVE_FAIL: Room %s, No active game found.", roomID)
		return
	}

	// 2. Artık 'game' objesi nil değil. ModeID'yi güvenle okuyabiliriz.
	engine, engineExists := g.gameEngines[game.ModeID]
	g.mutex.RUnlock() // 🛑 RLock bitti, şimdi kilidi bı

	if !engineExists || game.State != GameStateInProgress {
		log.Printf("PLAYER_MOVE_FAIL: Room %s, Game state: %s or engine missing.", roomID, game.State)
		return
	}

	// Mesajın içeriğinden PlayerID'yi al
	moveData, ok := msg.Content.(map[string]interface{})
	if !ok {
		log.Printf("PLAYER_MOVE_FAIL: Invalid content format for room %s", roomID)
		return
	}

	playerIDStr, ok := moveData["player_id"].(string)
	if !ok {
		log.Printf("PLAYER_MOVE_FAIL: Player ID missing in message for room %s", roomID)
		return
	}

	playerID, err := uuid.Parse(playerIDStr)
	if err != nil {
		log.Printf("PLAYER_MOVE_FAIL: Invalid UUID format for room %s", roomID)
		return
	}
	fmt.Printf("Player %s made a move in room %s: %v\n", playerID, roomID, moveData)

	// // 🎯 KRİTİK ADIM: Hareketi oyun motoruna ilet
	if err := engine.ProcessMove(game, playerID, moveData); err != nil {
		log.Printf("PLAYER_MOVE_ERROR: %s, Error: %v", playerID, err)
		// Oyuncuya hata mesajı gönderebilirsiniz.
		// g.hub.SendMessageToUser(roomID, playerID, &Message{...})
		return
	}

	// ProcessMove başarılı oldu, oyun durumu zaten yayınlanmıştır (DrawingGameEngine içinde).
	log.Printf("Player %s's move processed successfully in room %s.", playerID, roomID)
	return

}

func (g *GameHub) handleEndGame(roomID uuid.UUID, msg RoomManagerData) {
	fmt.Println("handleEndGame called for room", roomID)
	// Oyun bittiğinde roomSettings'i de temizle
	g.mutex.Lock()

	game, exists := g.activeGames[roomID]
	if !exists {
		g.mutex.Unlock()
		log.Printf("No active game to end for room %s", roomID)
		return
	}

	// Oyun durumunu güncelle
	game.State = GameStateOver

	// Oyunu aktif oyunlardan ve ayarlardan kaldır
	delete(g.activeGames, roomID)
	//delete(g.roomSettings, roomID)

	g.mutex.Unlock()

	// Zamanlayıcıyı durdur (mutex dışında)
	g.stopRoundTimer(roomID)

	// Oyun bitiş mesajını yayınla
	content := msg.Content.(map[string]interface{})
	reason := content["reason"].(string)
	message := content["message"].(string)

	g.hub.BroadcastMessage(roomID, &Message{
		Type: "game_ended",
		Content: map[string]interface{}{
			"room_id": roomID,
			"reason":  reason,
			"message": message,
			"scores":  g.playersToMap(game.Players),
		},
	})

	log.Printf("Game ended for room %s. Reason: %s", roomID, reason)

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
