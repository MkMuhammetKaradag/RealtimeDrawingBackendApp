package hub

import (
	"fmt"
	"log"
	"math/rand"

	// "sync" // Mutex'i Game struct'ı üzerinden kullanacağız

	"github.com/google/uuid"
)

type DrawingStroke struct {
	PlayerID uuid.UUID // Bu vuruşu yapan oyuncu
	Data     string    // Vuruşa ait çizim verisi (genellikle JSON formatında)
	// Canvas verisinin ne olduğu (örneğin renk, fırça boyutu, koordinatlar)
	// client tarafında belirlenip string olarak buraya gelir.
}

// CollaborativeArtData, "Ortak Sanat Projesi" modunun özel verilerini tutar.
type CollaborativeArtData struct {
	CurrentWord string // Çizilen kelime (temamız)
	// Tüm turların verisini saklayacak, Raporlama için anahtar yapı
	RoundHistory   map[int][]DrawingStroke // Tur Numarası -> O turdaki TÜM vuruşlar
	CurrentStrokes []DrawingStroke         // Mevcut turda yapılan vuruşlar
}

// CollaborativeArtEngine, "Ortak Sanat Projesi" oyununun mantığını uygular.
type CollaborativeArtEngine struct {
	gameHub  *GameHub
	wordList []string
}

func NewCollaborativeArtEngine(gameHub *GameHub) *CollaborativeArtEngine {
	return &CollaborativeArtEngine{
		gameHub:  gameHub,
		wordList: defaultWordList, // Varsayılan kelime listesini kullan
	}
}

// InitGame, yeni bir "Ortak Sanat Projesi" oyunu başlattığında ilk ayarları yapar.
func (cae *CollaborativeArtEngine) InitGame(game *Game, players []*Player) error {
	game.Players = players
	game.State = GameStateInProgress
	game.TurnCount = 1
	game.CurrentDrawerIndex = 0 // Bu modda önemi yok, ama yapısal tutarlılık için kalabilir

	// Puanlama sıfırlanır (zorunlu değil ama temizlik için iyi)
	for _, p := range players {
		p.Score = 0
	}

	// Özel verileri oluştur
	artData := &CollaborativeArtData{
		CurrentWord:    "",
		RoundHistory:   make(map[int][]DrawingStroke), // Geçmişi saklamak için map oluştur
		CurrentStrokes: []DrawingStroke{},
	}
	game.ModeData = artData

	log.Printf("Initialized Collaborative Art game for room %s.", game.RoomID)
	return nil
}

// ProcessMove, bir oyuncunun çizim vuruşunu işler. Tahmin artık yok.
func (cae *CollaborativeArtEngine) ProcessMove(game *Game, playerID uuid.UUID, moveData interface{}) error {
	game.Mutex.Lock()
	defer game.Mutex.Unlock()

	if game.State != GameStateInProgress {
		return fmt.Errorf("game is not in progress")
	}

	data, ok := moveData.(map[string]interface{})
	if !ok {
		return fmt.Errorf("invalid move data format")
	}

	actionType, ok := data["type"].(string)
	if !ok || actionType != "draw_stroke" {
		return fmt.Errorf("invalid move type or missing 'type' field")
	}

	// 🔑 ANA DEĞİŞİKLİK: Çizim vuruşunu oyuncu ID'si ile birlikte sakla
	if strokeData, ok := data["stroke"].(string); ok {
		artData, _ := game.ModeData.(*CollaborativeArtData)

		newStroke := DrawingStroke{
			PlayerID: playerID,
			Data:     strokeData,
		}
		// Vuruşu mevcut tur listesine ekle
		artData.CurrentStrokes = append(artData.CurrentStrokes, newStroke)

		log.Printf("Stroke added for room %s by player %s. Total strokes: %d",
			game.RoomID, playerID, len(artData.CurrentStrokes))

		// Yeni vuruşu anında tüm oyunculara yayınla
		cae.gameHub.hub.BroadcastMessage(game.RoomID, &Message{
			Type:    "new_stroke", // Hafif mesaj tipi
			Content: newStroke,    // Çizim vuruşu objesini gönder
		})
	}

	return nil
}

// EndRound, turu sonlandırır, veriyi kaydeder ve bir sonraki tur için hazırlar.
func (cae *CollaborativeArtEngine) EndRound(game *Game, reason string) bool {
	// game.Mutex.Lock()
	// defer game.Mutex.Unlock()

	artData, _ := game.ModeData.(*CollaborativeArtData)

	// 🎯 Geçmişi Sakla
	// Mevcut turdaki tüm vuruşları (CurrentStrokes) o tur numarasıyla (TurnCount) geçmişe kaydet.
	artData.RoundHistory[game.TurnCount] = artData.CurrentStrokes
	log.Printf("Round %d finished and strokes saved to history. Reason: %s", game.TurnCount, reason)

	// Tur Sayısını Artır
	game.TurnCount++

	// Mevcut tur verilerini sıfırla
	artData.CurrentStrokes = []DrawingStroke{}
	// Yeni kelimeyi belirle (StartRound'da belirlenecek ama EndRound'dan hemen sonraki tur için sırayı koru)
	game.ActivePlayer = cae.getNextDrawer(game) // Sıradaki tur için çizer/tema belirleyiciyi koru

	// Oyun Bitiş Kontrolü
	if game.TurnCount > game.TotalRounds {
		game.State = GameStateOver
		// Oyun sonu raporunu yayınla (Yeni metot)
		cae.SendFinalArtReport(game)
		return false // Oyun Bitti
	}

	// Oyun Devam Ediyor
	return true // Yeni tura geçilmesi gerekiyor
}

// getNextDrawer, sadece sonraki turda kimin aktif (temayı belirleyen) olduğunu tutmak için kalabilir.
func (cae *CollaborativeArtEngine) getNextDrawer(game *Game) uuid.UUID {
	// ... (DrawingGameEngine'daki gibi kalsın)
	playerCount := len(game.Players)
	if playerCount == 0 {
		return uuid.Nil
	}
	nextIndex := (game.CurrentDrawerIndex + 1) % playerCount
	game.CurrentDrawerIndex = nextIndex
	return game.Players[nextIndex].UserID
}

// SendFinalArtReport, oyun bittiğinde her tur ve her oyuncu için çizilenleri yayınlar.
func (cae *CollaborativeArtEngine) SendFinalArtReport(game *Game) {
	artData, _ := game.ModeData.(*CollaborativeArtData)

	// Nihai rapor yapısı:
	// { "round_1": { "word": "Kedi", "player_id_1": [stroke1, stroke2, ...], ... }, ... }
	finalReport := make(map[string]interface{})

	// Geçmişteki her tur için döngü
	for roundNum, allStrokes := range artData.RoundHistory {

		// Bu turdaki kelimeyi tahmin edebilmek için ek bir map tutulmalı
		// Şu anki yapımızda CurrentWord'ü sadece StartRound'da belirliyoruz.
		// Tur kelimesini de RoundHistory'e dahil etmeliyiz, ama şimdilik varsayalım:
		word := cae.selectRandomWord() // Gerçekte tur kelimesini bir yerde saklamanız GEREKİR.

		// Oyuncu ID'sine göre vuruşları grupla
		playerStrokes := make(map[uuid.UUID][]DrawingStroke)
		for _, stroke := range allStrokes {
			playerStrokes[stroke.PlayerID] = append(playerStrokes[stroke.PlayerID], stroke)
		}

		// Rapor objesini hazırla
		roundReport := map[string]interface{}{
			"word":                 word,
			"player_contributions": playerStrokes,
		}

		finalReport[fmt.Sprintf("round_%d", roundNum)] = roundReport
	}

	// Oyun sonu raporunu yayınla
	cae.gameHub.hub.BroadcastMessage(game.RoomID, &Message{
		Type:    "game_over_report",
		Content: finalReport,
	})

	log.Printf("Final art report published for room %s.", game.RoomID)
}

// selectRandomWord metodu DrawingGameEngine'den aynen alınabilir.
func (cae *CollaborativeArtEngine) selectRandomWord() string {
	// ... (Aynen kullanılır)
	if len(cae.wordList) == 0 {
		return "Resim"
	}
	randomIndex := rand.Intn(len(cae.wordList))
	return cae.wordList[randomIndex]
}
func (dge *CollaborativeArtEngine) SendPreparationNotifications(game *Game) {

	dge.gameHub.hub.BroadcastMessage(game.RoomID, &Message{
		Type: "round_preparation",
		Content: map[string]interface{}{
			"role":                 "drawer",
			"preparation_duration": game.PreparationDuration,
			"round_number":         game.TurnCount + 1,
			"total_rounds":         game.TotalRounds,
			"message":              fmt.Sprintf("%d saniye içinde çizim başlayacak. Hazır ol!", game.PreparationDuration),
		},
	})

	log.Printf("Preparation notifications sent for room %s. Next drawer: %s, Duration: %ds",
		game.RoomID, game.PreparationDuration)
}

// StartRound, yeni bir tur başlatır, kelimeyi seçer ve başlangıç bildirimlerini gönderir.
func (cae *CollaborativeArtEngine) StartRound(game *Game) error {
	// game.Mutex.Lock()
	// defer game.Mutex.Unlock()

	log.Printf("Starting collaborative art round %d in room %s", game.TurnCount, game.RoomID)

	// 1. Verileri Hazırla/Sıfırla
	artData, ok := game.ModeData.(*CollaborativeArtData)
	if !ok {
		return fmt.Errorf("mode data is not of expected type CollaborativeArtData")
	}

	// NOT: CurrentStrokes EndRound'da sıfırlanmıştı, burada sadece kelimeyi güncelliyoruz.

	// 2. Kelime seçimi
	// Bu, bu turda çizilecek temadır.
	artData.CurrentWord = cae.selectRandomWord()
	log.Printf("Selected word for collaborative art: %s", artData.CurrentWord)

	// 3. Tüm oyunculara tur başlangıcını (gizli kelime ile) bildir
	for _, p := range game.Players {
		cae.gameHub.hub.SendMessageToUser(game.RoomID, p.UserID, &Message{
			Type: "round_start_collab",
			Content: map[string]interface{}{
				"word_length": len(artData.CurrentWord), // Kelime uzunluğunu gönder
				"duration":    game.RoundDuration,
				"round":       game.TurnCount,
				"message":     "Yeni tur başladı! Şimdi çizim yapma sırası.",
			},
		})
	}

	// 4. Genel oyun durumu yayınını yap (tur bilgisinin güncellenmesi için)
	cae.gameHub.hub.BroadcastMessage(game.RoomID, &Message{
		Type:    "game_state_update",
		Content: game,
	})

	return nil
}
