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
	game.Mutex.Lock()
	defer game.Mutex.Unlock()

	// Oyuncuları oyun nesnesine ekle
	game.Players = players

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
				// Bu oyuncunun skoru güncellendi mesajı yayınlayabilirsin
				// dge.gameHub.hub.BroadcastMessage(...)
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
