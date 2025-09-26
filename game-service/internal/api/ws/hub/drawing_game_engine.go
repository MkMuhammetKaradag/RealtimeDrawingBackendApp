// hub/drawing_game_engine.go
package hub

import (
	"fmt"
	"log"

	"github.com/google/uuid"
)

// DrawingGameEngine, "Çizim ve Tahmin" oyununun mantığını uygular.
type DrawingGameEngine struct {
	gameHub *GameHub
}

func NewDrawingGameEngine(gameHub *GameHub) *DrawingGameEngine {
	return &DrawingGameEngine{gameHub: gameHub}
}

// InitGame, yeni bir "Çizim ve Tahmin" oyunu başlattığında ilk ayarları yapar.
func (dge *DrawingGameEngine) InitGame(game *Game, players []*Player) error {
	// game.Mutex.Lock()
	// defer game.Mutex.Unlock()

	game.Players = players
	game.State = GameStateInProgress // Durumu başlat
	game.TurnCount = 1               // İlk turu 1 olarak ayarla
	game.CurrentDrawerIndex = 0
	// Oyuncular arasında sırayla dönmek için başlangıç çizerini ayarla
	if len(players) > 0 {
		game.ActivePlayer = players[0].UserID
	}

	// Bu modun özel verilerini oluştur
	drawingData := &DrawingGameData{
		CurrentWord:    "", // Kelimeyi daha sonra belirleyeceğiz
		CurrentDrawer:  game.ActivePlayer,
		GuessedPlayers: make(map[uuid.UUID]bool),
		CanvasData:     "{}", // Boş başlangıç canvas'ı
	}
	game.ModeData = drawingData

	log.Printf("Initialized Drawing & Guessing game for room %s. First drawer: %s", game.RoomID, game.ActivePlayer)
	return nil
}

// ProcessMove, bir oyuncunun hamlesini (çizim veya tahmin) işler.
func (dge *DrawingGameEngine) ProcessMove(game *Game, playerID uuid.UUID, moveData interface{}) error {
	game.Mutex.Lock()
	defer game.Mutex.Unlock()

	// Oyunun devam edip etmediğini kontrol et
	if game.State != GameStateInProgress {
		return fmt.Errorf("game is not in progress")
	}

	// Gelen veriyi map'e dönüştür
	data, ok := moveData.(map[string]interface{})
	if !ok {
		return fmt.Errorf("invalid move data format")
	}

	actionType, ok := data["type"].(string)
	if !ok {
		return fmt.Errorf("move data missing 'type' field")
	}

	switch actionType {
	case "draw":
		// Sadece sırası gelen oyuncu çizebilir
		if game.ActivePlayer != playerID {
			return fmt.Errorf("it is not your turn to draw")
		}
		// Canvas verisini güncelle
		if canvasData, ok := data["canvas"].(string); ok {
			drawingData, _ := game.ModeData.(*DrawingGameData)
			drawingData.CanvasData = canvasData
			log.Printf("Drawing updated for room %s by player %s", game.RoomID, playerID)
		}
	case "guess":
		// Herkes tahmin edebilir
		guessText, ok := data["text"].(string)
		if !ok {
			return fmt.Errorf("guess data missing 'text' field")
		}

		drawingData, _ := game.ModeData.(*DrawingGameData)
		if guessText == drawingData.CurrentWord {
			// Doğru tahmin! Oyuncuya puan ekle.
			log.Printf("Player %s guessed the word correctly in room %s!", playerID, game.RoomID)

			// Bu oyuncu zaten bilmişse bir şey yapma
			if _, guessed := drawingData.GuessedPlayers[playerID]; !guessed {
				drawingData.GuessedPlayers[playerID] = true

				// Skor ekleme mantığı
				for _, p := range game.Players {
					if p.UserID == playerID {
						p.Score += 10 // Örnek puan
						break
					}
				}
				isRoundOver, _ := dge.CheckRoundStatus(game)
				if isRoundOver {
					// Tur bittiği için zamanlayıcıyı durdur ve turu bitir
					// Burayı `GameHub`'a taşımalıyız
					go dge.gameHub.handleRoundEnd(game.RoomID, "all_guessed")
				}
			}
		}
	}

	// Oyun durumu güncellendi, bu durumu yayınlaması için GameHub'ı bilgilendir
	dge.gameHub.hub.BroadcastMessage(game.RoomID, &Message{
		Type:    "game_state_update",
		Content: game,
	})

	return nil
}

// CheckRoundStatus, turun bitip bitmediğini kontrol eder.
func (dge *DrawingGameEngine) CheckRoundStatus(game *Game) (bool, error) {
	// Tüm oyuncular doğru tahmin ettiyse veya süre bittiyse true döner.
	drawingData, _ := game.ModeData.(*DrawingGameData)
	return len(drawingData.GuessedPlayers) == len(game.Players)-1, nil
}

// 💡 Yeni Metot: Tur Başlatma ve Rol Bildirimlerini Yönetir
func (dge *DrawingGameEngine) StartRound(game *Game) error {
	fmt.Println("StartRound called for room", game.RoomID)
	// game.Mutex.Lock()
	// defer game.Mutex.Unlock()
	fmt.Println("Starting round", game.TurnCount, "in room", game.RoomID)

	// 1. Oyuncular arasında sırayı ilerlet (veya ilk oyuncuyu seç)
	// Bu mantığı burada uygulamanız veya `Game` nesnesine bir `CurrentDrawerIndex` eklemeniz gerekir.
	// Şimdilik sadece `InitGame`'deki gibi ilk oyuncuyu çizer kabul edelim.
	// game.ActivePlayer = dge.getNextDrawer(game) // Gerçek uygulamada bu gerekecek

	// 2. Tur verilerini sıfırla/hazırla
	drawingData, _ := game.ModeData.(*DrawingGameData)
	drawingData.CurrentDrawer = game.ActivePlayer
	drawingData.GuessedPlayers = make(map[uuid.UUID]bool)
	drawingData.CanvasData = "{}"
	fmt.Println("Current drawer is:", drawingData.CurrentDrawer)
	// 💡 Kelime seçimi burada yapılır: drawingData.CurrentWord = dge.selectRandomWord()
	drawingData.CurrentWord = "Kedi"                                  // Örnek olarak
	fmt.Println("Selected word for drawer:", drawingData.CurrentWord) // Konsola yazdır

	// 3. Bildirimleri Gönder
	for _, p := range game.Players {
		// Çizer (Drawer) için özel mesaj
		if p.UserID == game.ActivePlayer {
			dge.gameHub.hub.SendMessageToUser(game.RoomID, p.UserID, &Message{
				Type: "round_start_drawer",
				Content: map[string]interface{}{
					"drawer_id": game.ActivePlayer,
					"word":      drawingData.CurrentWord, // 💡 KELİMEYİ SADECE ÇİZERE GÖNDER
					"duration":  game.RoundDuration,
				},
			})
		} else {
			// Diğer tahmin edenler (Guessers) için mesaj
			dge.gameHub.hub.SendMessageToUser(game.RoomID, p.UserID, &Message{
				Type: "round_start_guesser",
				Content: map[string]interface{}{
					"drawer_id": game.ActivePlayer,
					"hint":      "____", // İpucu gönderebilirsin
					"duration":  game.RoundDuration,
				},
			})
		}
	}

	return nil
}

// 💡 Yeni Metot: Tur Bitince Yapılacaklar
func (dge *DrawingGameEngine) EndRound(game *Game, reason string) bool {
	// Kilitler (Mutex) zaten GameHub tarafından tutuluyor olmalıdır.

	fmt.Printf("EndRound called. Reason: %s. Current Round: %d\n", reason, game.TurnCount)
	// game.Mutex.Lock()
	// defer game.Mutex.Unlock()
	// 1. Puanlama ve Durum Güncellemeleri buraya gelir.
	// Örn: dge.calculateScores(game, reason)

	// 2. Tur Sayısını Artırma
	game.TurnCount++

	// 3. Sıradaki Çizeri Ayarlama
	game.ActivePlayer = dge.getNextDrawer(game) // sonraki çizeri belirle

	// 4. OYUN BİTİŞ KONTROLÜ
	if game.TurnCount > game.TotalRounds {
		game.State = GameStateOver
		return false // Oyun Bitti
	}

	// 5. OYUN DEVAM EDİYOR
	return true // Yeni tura geçilmesi gerekiyor
}

// getNextDrawer, sıradaki çizerin ID'sini döndürür ve indeksi günceller.
func (dge *DrawingGameEngine) getNextDrawer(game *Game) uuid.UUID {
	// NOT: Bu metot EndRound içinden kilitli olarak çağrılacağı için burada kilit koymuyoruz.

	playerCount := len(game.Players)
	if playerCount == 0 {
		return uuid.Nil
	}

	// İndeksi bir sonraki oyuncuya ilerlet
	nextIndex := (game.CurrentDrawerIndex + 1) % playerCount
	game.CurrentDrawerIndex = nextIndex

	// Yeni ActivePlayer'ı döndür
	return game.Players[nextIndex].UserID
}

// determineWinner, oyunu kazanan oyuncuları döndürür (beraberlik için birden fazla olabilir).
func (dge *DrawingGameEngine) determineWinner(game *Game) []*Player {
	// NOT: Bu metot EndRound içinden kilitli olarak çağrılacağı için burada kilit koymuyoruz.

	if len(game.Players) == 0 {
		return nil
	}

	var winners []*Player
	maxScore := -1

	// 1. En yüksek skoru bul
	for _, player := range game.Players {
		if player.Score > maxScore {
			maxScore = player.Score
		}
	}

	// 2. En yüksek skora sahip tüm oyuncuları topla
	for _, player := range game.Players {
		if player.Score == maxScore {
			winners = append(winners, player)
		}
	}

	return winners
}
