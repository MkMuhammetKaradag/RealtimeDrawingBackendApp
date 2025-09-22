package game

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2/log"
)

// RoomManager, tüm aktif oyun odalarını yönetir
type RoomManager struct {
	Rooms     map[string]*Room  // Key: Room ID
	RoomCodes map[string]string // Key: Room Code, Value: Room ID
	DB        *sql.DB
	mu        sync.RWMutex
}

// GameModes, mevcut oyun modları
var GameModes = map[int]*GameMode{
	1: {ID: 1, Name: "Çizim ve Tahmin", Description: "Her oyuncu bir kelime yazar, diğerleri bu kelimeleri çizmeye çalışır", MinPlayers: 2, MaxPlayers: 8},
	2: {ID: 2, Name: "Ortak Alan", Description: "Tüm oyuncular aynı canvas üzerinde birlikte çizim yapar", MinPlayers: 2, MaxPlayers: 12},
	3: {ID: 3, Name: "Serbest Çizim", Description: "Herkes istediği gibi çizim yapabilir, yarışma yok", MinPlayers: 1, MaxPlayers: 20},
}

// NewRoomManager, yeni bir RoomManager örneği oluşturur
func NewRoomManager(db *sql.DB) *RoomManager {
	rm := &RoomManager{
		Rooms:     make(map[string]*Room),
		RoomCodes: make(map[string]string),
		DB:        db,
	}

	// Aktif odaları veritabanından yükle
	rm.loadActiveRooms()

	// Periyodik temizleme başlat
	go rm.startCleanupRoutine()

	return rm
}

// loadActiveRooms, veritabanından aktif odaları yükler
func (rm *RoomManager) loadActiveRooms() {
	query := `
		SELECT r.id, r.room_name, r.creator_id, r.game_mode_id, r.max_players,
		       r.status, r.is_private, r.room_code, r.created_at, r.started_at
		FROM rooms r 
		WHERE r.status IN ('waiting', 'playing') 
		AND r.created_at > NOW() - INTERVAL '24 hours'`

	rows, err := rm.DB.Query(query)
	if err != nil {
		log.Errorf("Aktif odalar yüklenirken hata: %v", err)
		return
	}
	defer rows.Close()

	for rows.Next() {
		room := &Room{
			Players:       make(map[string]*Player),
			BannedPlayers: make(map[string]bool),
			SharedCanvas:  make([]DrawingData, 0),
		}

		var startedAt sql.NullTime
		var roomCode sql.NullString

		err := rows.Scan(
			&room.ID, &room.Name, &room.CreatorID, &room.GameModeID,
			&room.MaxPlayers, &room.Status, &room.IsPrivate,
			&roomCode, &room.CreatedAt, &startedAt,
		)
		if err != nil {
			log.Errorf("Oda verisi okunurken hata: %v", err)
			continue
		}

		if roomCode.Valid {
			room.RoomCode = roomCode.String
		}
		if startedAt.Valid {
			room.StartedAt = &startedAt.Time
		}

		// Oyun modunu ekle
		if gameMode, exists := GameModes[room.GameModeID]; exists {
			room.GameMode = gameMode
		}

		rm.mu.Lock()
		rm.Rooms[room.ID] = room
		if room.RoomCode != "" {
			rm.RoomCodes[room.RoomCode] = room.ID
		}
		rm.mu.Unlock()

		log.Infof("Aktif oda yüklendi: %s (%s)", room.Name, room.ID)
	}
}

// GetRoom, ID'ye göre bir odayı döndürür
func (rm *RoomManager) GetRoom(roomID string) *Room {
	rm.mu.RLock()
	defer rm.mu.RUnlock()
	return rm.Rooms[roomID]
}

// GetRoomByCode, oda koduna göre bir odayı döndürür
func (rm *RoomManager) GetRoomByCode(roomCode string) *Room {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	if roomID, exists := rm.RoomCodes[roomCode]; exists {
		return rm.Rooms[roomID]
	}
	return nil
}

// GetPublicRooms, herkese açık odaları döndürür
func (rm *RoomManager) GetPublicRooms() []*Room {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	var publicRooms []*Room
	for _, room := range rm.Rooms {
		if !room.IsPrivate && room.Status == "waiting" {
			publicRooms = append(publicRooms, room)
		}
	}
	return publicRooms
}

// JoinPlayerToRoom, oyuncuyu odaya katılım sırasında veritabanına da kaydeder
func (rm *RoomManager) JoinPlayerToRoom(roomID, userID, username string) error {
	room := rm.GetRoom(roomID)
	if room == nil {
		return fmt.Errorf("oda bulunamadı")
	}

	// Veritabanına oyuncu katılımını kaydet
	query := `
		INSERT INTO room_players (room_id, user_id, joined_at, is_online)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (room_id, user_id) 
		DO UPDATE SET is_online = $4, joined_at = $3`

	_, err := rm.DB.Exec(query, roomID, userID, time.Now(), true)
	if err != nil {
		return fmt.Errorf("oyuncu katılımı veritabanına kaydedilemedi: %v", err)
	}

	// Oda bilgilerini güncelle
	return nil
}

// RemovePlayerFromRoom, oyuncuyu odadan çıkarırken veritabanını da günceller
func (rm *RoomManager) RemovePlayerFromRoom(roomID, userID string) error {
	room := rm.GetRoom(roomID)
	if room == nil {
		return fmt.Errorf("oda bulunamadı")
	}

	// Veritabanında oyuncuyu offline yap
	_, err := rm.DB.Exec(
		"UPDATE room_players SET is_online = false WHERE room_id = $1 AND user_id = $2",
		roomID, userID)
	if err != nil {
		log.Errorf("Oyuncu offline durumu güncellenemedi: %v", err)
	}

	// Oda bilgilerini güncelle
	return nil
}

// GetGameModes, mevcut oyun modlarını döndürür
func (rm *RoomManager) GetGameModes() []*GameMode {
	modes := make([]*GameMode, 0, len(GameModes))
	for _, mode := range GameModes {
		modes = append(modes, mode)
	}
	return modes
}

// GetRandomWord, rastgele bir kelime döndürür (Çizim ve Tahmin modu için)
func (rm *RoomManager) GetRandomWord(difficulty int) (*Word, error) {
	query := `
		SELECT id, word, difficulty, category 
		FROM words 
		WHERE difficulty = $1 AND language_code = 'tr'
		ORDER BY RANDOM() 
		LIMIT 1`

	var word Word
	err := rm.DB.QueryRow(query, difficulty).Scan(
		&word.ID, &word.Word, &word.Difficulty, &word.Category)
	if err != nil {
		return nil, fmt.Errorf("kelime alınamadı: %v", err)
	}

	return &word, nil
}

// SaveGameSession, oyun oturumunu veritabanına kaydeder
func (rm *RoomManager) SaveGameSession(session *GameSession) error {
	query := `
		INSERT INTO game_sessions 
		(id, room_id, current_round, total_rounds, round_duration, 
		 current_drawer_id, current_word_id, round_start_time, round_end_time, session_status)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT (id) 
		DO UPDATE SET 
			current_round = $3, 
			current_drawer_id = $6, 
			current_word_id = $7,
			round_start_time = $8,
			round_end_time = $9,
			session_status = $10`

	var currentWordID *int
	if session.CurrentWord != nil {
		currentWordID = &session.CurrentWord.ID
	}

	_, err := rm.DB.Exec(query,
		session.ID, session.RoomID, session.CurrentRound, session.TotalRounds,
		session.RoundDuration, session.CurrentDrawerID, currentWordID,
		session.RoundStartTime, session.RoundEndTime, session.Status)

	return err
}

// SaveDrawingData, çizim verilerini veritabanına kaydeder
func (rm *RoomManager) SaveDrawingData(sessionID, userID string, drawingData DrawingData) error {
	jsonData, err := json.Marshal(drawingData)
	if err != nil {
		return fmt.Errorf("çizim verisi JSON'a dönüştürülemedi: %v", err)
	}

	query := `
		INSERT INTO drawing_data (session_id, user_id, drawing_json, round_number)
		VALUES ($1, $2, $3, $4)`

	_, err = rm.DB.Exec(query, sessionID, userID, jsonData, drawingData.RoundNumber)
	return err
}

// startCleanupRoutine, periyodik temizleme işlemlerini başlatır
func (rm *RoomManager) startCleanupRoutine() {
	ticker := time.NewTicker(5 * time.Minute) // Her 5 dakikada bir
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			rm.cleanupInactiveRooms()
		}
	}
}

// cleanupInactiveRooms, boş ve eski odaları temizler
func (rm *RoomManager) cleanupInactiveRooms() {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	now := time.Now()
	var toDelete []string

	for roomID, room := range rm.Rooms {
		room.mu.RLock()
		shouldDelete := false

		// 1 saatten fazla boş olan odaları sil
		if len(room.Players) == 0 && now.Sub(room.CreatedAt) > time.Hour {
			shouldDelete = true
		}

		// 24 saatten fazla aktif olmayan odaları sil
		if room.Status == "finished" && now.Sub(room.CreatedAt) > 24*time.Hour {
			shouldDelete = true
		}

		room.mu.RUnlock()

		if shouldDelete {
			toDelete = append(toDelete, roomID)
		}
	}

	// Silme işlemlerini gerçekleştir
	for _, roomID := range toDelete {
		if room, exists := rm.Rooms[roomID]; exists {
			// Veritabanından sil
			rm.DB.Exec("DELETE FROM rooms WHERE id = $1", roomID)

			// Oda kodunu temizle
			if room.RoomCode != "" {
				delete(rm.RoomCodes, room.RoomCode)
			}

			delete(rm.Rooms, roomID)
			log.Infof("İnaktif oda temizlendi: %s", roomID)
		}
	}
}
