// hub/drawing_game_engine.go
package hub

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"

	"github.com/google/uuid"
)

// DrawingGameEngine, "Çizim ve Tahmin" oyununun mantığını uygular.
type DrawingGameEngine struct {
	gameHub *GameHub
}

type DrawArtData struct {
	CurrentWord    string              // Çizilen kelime (temamız)
	RoundHistory   map[int]RoundRecord // Tur Numarası -> O turdaki TÜM vuruşlar
	CurrentStrokes []DrawingStroke     // Mevcut turda yapılan vuruşlar
	GuessedPlayers map[uuid.UUID]bool
}
type RoundRecord struct {
	Word       string
	DrawerID   uuid.UUID       // Bu turda kimin çizdiği
	AllStrokes []DrawingStroke // Bu turdaki tüm vuruşlar (zaten saklıyor olabilirsiniz)
}
type DrawingStroke struct {
	PlayerID uuid.UUID // Bu vuruşu yapan oyuncu
	Data     string    // Vuruşa ait çizim verisi (genellikle JSON formatında)
	// Canvas verisinin ne olduğu (örneğin renk, fırça boyutu, koordinatlar)
	// client tarafında belirlenip string olarak buraya gelir.
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
	artData := &DrawArtData{
		CurrentWord:    "",
		RoundHistory:   make(map[int]RoundRecord), // Geçmişi saklamak için map oluştur
		CurrentStrokes: []DrawingStroke{},
		GuessedPlayers: make(map[uuid.UUID]bool),
	}
	game.ModeData = artData

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
	case "draw", "canvas_action":
		// Sadece sırası gelen oyuncu çizebilir
		if game.ActivePlayer != playerID {
			return fmt.Errorf("it is not your turn to draw")
		}
		fmt.Println("data:", data)
		// Canvas verisini güncelle
		jsonData, err := json.Marshal(data)
		if err != nil {
			return fmt.Errorf("failed to marshal drawing data: %v", err)
		}
		drawingData, _ := game.ModeData.(*DrawArtData)
		if drawingData == nil {
			// Loglama eklemek isteyebilirsiniz: log.Printf("HATA: ModeData DrawArtData değil veya nil.")
			return fmt.Errorf("oyun modu verisi eksik veya yanlış tipte")
		}
		drawingData.CurrentStrokes = append(drawingData.CurrentStrokes, DrawingStroke{
			PlayerID: playerID,
			Data:     string(jsonData),
		})
		log.Printf("Drawing updated for room %s by player %s", game.RoomID, playerID)
		dge.gameHub.hub.BroadcastToOthers(game.RoomID, playerID, &Message{
			Type: "canvas_update",
			Content: map[string]interface{}{
				"drawer_id": playerID,
				"data":      drawingData.CurrentStrokes[len(drawingData.CurrentStrokes)-1].Data,
			},
		})
	case "guess":
		// Herkes tahmin edebilir
		guessText, ok := data["text"].(string)
		if !ok {
			return fmt.Errorf("guess data missing 'text' field")
		}
		fmt.Println("Received guess:", guessText, "from player:", playerID, "in room mode datra:", game.ModeData)
		drawingData, _ := game.ModeData.(*DrawArtData)
		if !ok || drawingData == nil {
			return fmt.Errorf("oyun modu verisi eksik veya yanlış tipte")
		}

		// Tahmin çizerin kendisinden geldiyse (çizer çizdiği kelimeyi tahmin edemez)
		if playerID == game.ActivePlayer {
			// Çizici, tahmin göndermiş. Bunu bir hata olarak ele alabilir veya yok sayabilirsiniz.
			return fmt.Errorf("drawer cannot guess the word")
		}

		// Kelime doğru tahmin edildi mi?
		if guessText == drawingData.CurrentWord {
			// Oyuncu zaten bilmiş mi kontrol et
			if _, alreadyGuessed := drawingData.GuessedPlayers[playerID]; !alreadyGuessed {

				// 🎯 KISIM 1: Oyuncuyu bilmişler listesine ekle (Tekrar puan almayı engeller)
				drawingData.GuessedPlayers[playerID] = true

				log.Printf("Player %s guessed the word correctly in room %s!", playerID, game.RoomID)

				// 🎯 KISIM 2: Skor ekleme mantığı: Hem Tahminci hem de Çizer puan kazanır
				guesserScore := 10       // Tahminci puanı
				drawerScorePerGuess := 5 // Çizerin her başarılı tahminden aldığı puan

				for _, p := range game.Players {
					if p.UserID == playerID {
						// Tahminci puanı
						p.Score += guesserScore
					} else if p.UserID == game.ActivePlayer {
						// Çizer puanı (Her doğru tahminde bir kez alır)
						p.Score += drawerScorePerGuess
					}
				}

				// Tur Bitiş Kontrolü
				isRoundOver, _ := dge.CheckRoundStatus(game)
				if isRoundOver {
					// Tur bittiği için zamanlayıcıyı durdur ve turu bitir
					go dge.gameHub.handleRoundEnd(game.RoomID, "all_guessed")
				}
			}
		}
	}

	// Oyun durumu güncellendi, bu durumu yayınlaması için GameHub'ı bilgilendir
	// dge.gameHub.hub.BroadcastMessage(game.RoomID, &Message{
	// 	Type:    "game_state_update",
	// 	Content: game,
	// })

	return nil
}

// CheckRoundStatus, turun bitip bitmediğini kontrol eder.
func (dge *DrawingGameEngine) CheckRoundStatus(game *Game) (bool, error) {
	// Tüm oyuncular doğru tahmin ettiyse veya süre bittiyse true döner.
	drawingData, _ := game.ModeData.(*DrawArtData)
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
	drawingData, ok := game.ModeData.(*DrawArtData)
	if !ok {
		return fmt.Errorf("mode data is not of expected type CollaborativeArtData")
	}
	// 💡 Kelime seçimi burada yapılır: drawingData.CurrentWord = dge.selectRandomWord()
	selectedWord := dge.selectRandomWord()
	drawingData.CurrentWord = selectedWord         // Örnek olarak
	drawingData.CurrentStrokes = []DrawingStroke{} // Çizimleri sıfırla
	drawingData.GuessedPlayers = make(map[uuid.UUID]bool)
	currentRoundNum := game.TurnCount
	drawingData.RoundHistory[currentRoundNum] = RoundRecord{
		Word: selectedWord,
		// ActivePlayer'ın doğru ayarlandığından emin olun!
		// game.ActivePlayer, bu turu çizecek kişinin ID'si olmalı.
		DrawerID: game.ActivePlayer,
		// AllStrokes şimdilik boş kalabilir, Stroke'lar EndRound'da eklenecektir.
		AllStrokes: []DrawingStroke{},
	}
	// 3. Bildirimleri Gönder
	for _, p := range game.Players {
		// Çizer (Drawer) için özel mesaj
		if p.UserID == game.ActivePlayer {
			dge.gameHub.hub.SendMessageToUser(game.RoomID, p.UserID, &Message{
				Type: "round_start_drawer",
				Content: map[string]interface{}{
					"drawer_id": game.ActivePlayer,
					"word":      selectedWord, // 💡 KELİMEYİ SADECE ÇİZERE GÖNDER
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

var defaultWordList = []string{
	"Köpek", "Araba", "Bilgisayar", "Güneş", "Ayakkabı",
	"Uçak", "Kütüphane", "Kahve", "Telefon", "Gözlük",
	"Bisiklet", "Gitar", "Elma", "Yıldız", "Saat",
}

func (dge *DrawingGameEngine) selectRandomWord() string {
	// Kelime listesi boşsa varsayılan bir değer dön
	if len(defaultWordList) == 0 {
		return "Resim"
	}

	// rand paketi zaten Go'nun standardıdır.
	// 'math/rand' yerine 'crypto/rand' daha güvenli olsa da, oyun için 'math/rand' yeterlidir.
	// Go 1.20 ve sonrası için bu şekilde kullanmak güvenlidir.
	randomIndex := rand.Intn(len(defaultWordList))

	return defaultWordList[randomIndex]
}

// 💡 Yeni Metot: Tur Bitince Yapılacaklar
func (dge *DrawingGameEngine) EndRound(game *Game, reason string) bool {
	fmt.Printf("EndRound called. Reason: %s. Current Round: %d\n", reason, game.TurnCount)

	// Biten turun numarası:
	endedRoundNum := game.TurnCount

	artData, ok := game.ModeData.(*DrawArtData)
	if !ok {
		// Hata yönetimi burada olmalı
		return false
	}

	// 1. **StartRound'da oluşturulan** RoundRecord kaydını al
	// NOT: StartRound'da kelime ve ActivePlayer ayarlanmış olmalı.
	record, exists := artData.RoundHistory[endedRoundNum]
	if !exists {
		// Eğer StartRound doğru çalışmadıysa (hiç olmamalı)
		fmt.Println("HATA: Biten tur için RoundRecord bulunamadı!")
		// ... hata işlemesi ...
		return false
	}

	// 2. O anki (biten) turun CurrentStrokes verisini kayda ekle
	record.AllStrokes = artData.CurrentStrokes

	// 3. Güncellenmiş kaydı geri yaz (map'lerde gerekli)
	artData.RoundHistory[endedRoundNum] = record

	// --- ÖNEMLİ DÜZELTME SONU ---

	// 4. Tur Sayısını Artırma
	game.TurnCount++

	// 5. Sıradaki Çizeri Ayarlama
	game.ActivePlayer = dge.getNextDrawer(game) // sonraki çizeri belirle

	// 6. Current verileri sıfırla (Yeni tur için)
	artData.CurrentStrokes = []DrawingStroke{}
	// artData.CurrentWord = "" // StartRound'da yeniden ayarlanacağı için bu zorunlu değil

	// 7. OYUN BİTİŞ KONTROLÜ
	if game.TurnCount > game.TotalRounds {
		game.State = GameStateOver
		return false // Oyun Bitti
	}

	// 8. OYUN DEVAM EDİYOR
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

// 🎯 YENİ METOT: Hazırlık Bildirimleri Gönder
func (dge *DrawingGameEngine) SendPreparationNotifications(game *Game) {
	fmt.Println("SendPreparationNotifications called for room", game.RoomID)

	// Yeni tur için çizeri belirle (henüz StartRound çağrılmadı)
	//_, _ := game.ModeData.(*DrawingGameData)

	// Sıradaki çizeri hesapla
	nextDrawer := game.ActivePlayer

	// Tüm oyunculara hazırlık bildirimi gönder
	for _, p := range game.Players {
		if p.UserID == nextDrawer {
			// Çizen olacak oyuncuya özel mesaj
			dge.gameHub.hub.SendMessageToUser(game.RoomID, p.UserID, &Message{
				Type: "round_preparation",
				Content: map[string]interface{}{
					"role":                 "drawer",
					"drawer_id":            nextDrawer,
					"preparation_duration": game.PreparationDuration,
					"round_number":         game.TurnCount + 1,
					"total_rounds":         game.TotalRounds,
					"message":              fmt.Sprintf("%d saniye içinde çizim başlayacak. Hazır ol!", game.PreparationDuration),
				},
			})
		} else {
			// Tahmin edecek oyunculara mesaj
			dge.gameHub.hub.SendMessageToUser(game.RoomID, p.UserID, &Message{
				Type: "round_preparation",
				Content: map[string]interface{}{
					"role":                 "guesser",
					"drawer_id":            nextDrawer,
					"preparation_duration": game.PreparationDuration,
					"round_number":         game.TurnCount + 1,
					"total_rounds":         game.TotalRounds,
					"message":              fmt.Sprintf("%d saniye içinde yeni tur başlayacak!", game.PreparationDuration),
				},
			})
		}
	}

	log.Printf("Preparation notifications sent for room %s. Next drawer: %s, Duration: %ds",
		game.RoomID, nextDrawer, game.PreparationDuration)
}
func (dge *DrawingGameEngine) SendFinalArtReport(game *Game) {
	artData, _ := game.ModeData.(*DrawArtData)

	// Nihai rapor yapısı:
	// { "round_1": { "word": "Kedi", "player_id_1": [stroke1, stroke2, ...], ... }, ... }
	finalReport := make(map[string]interface{})

	// Geçmişteki her tur için döngü
	for roundNum, record := range artData.RoundHistory {

		// Bu turdaki kelimeyi tahmin edebilmek için ek bir map tutulmalı
		// Şu anki yapımızda CurrentWord'ü sadece StartRound'da belirliyoruz.
		// Tur kelimesini de RoundHistory'e dahil etmeliyiz, ama şimdilik varsayalım:
		//word := dge.selectRandomWord() // Gerçekte tur kelimesini bir yerde saklamanız GEREKİR.

		// // Oyuncu ID'sine göre vuruşları grupla
		// playerStrokes := make(map[uuid.UUID][]DrawingStroke)
		// for _, stroke := range allStrokes {
		// 	playerStrokes[stroke.PlayerID] = append(playerStrokes[stroke.PlayerID], stroke)
		// }

		// Rapor objesini hazırla
		roundReport := map[string]interface{}{
			"word":    record.Word,
			"actions": record.AllStrokes,
		}

		finalReport[fmt.Sprintf("round_%d", roundNum)] = roundReport
	}

	// Oyun sonu raporunu yayınla
	dge.gameHub.hub.BroadcastMessage(game.RoomID, &Message{
		Type: "game_over",
		Content: map[string]interface{}{
			"rounds": finalReport,
		},
	})

	log.Printf("Final art report published for room %s.", game.RoomID)
}
