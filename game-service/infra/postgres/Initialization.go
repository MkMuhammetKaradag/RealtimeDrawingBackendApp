package postgres

import (
	"database/sql"
	"fmt"
	"log"
)

const (
	createUsersTable = `
		CREATE TABLE IF NOT EXISTS users (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			username VARCHAR(50) NOT NULL UNIQUE,
			email VARCHAR(100) NOT NULL UNIQUE,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
			games_played INT DEFAULT 0,
			games_won INT DEFAULT 0,
			total_score INT DEFAULT 0
		);`

	createGameModesTable = `
		CREATE TABLE IF NOT EXISTS game_modes (
			id SERIAL PRIMARY KEY,
			mode_name VARCHAR(50) UNIQUE NOT NULL,
			description TEXT,
			min_players INT DEFAULT 2,
			max_players INT DEFAULT 10
		);`

	insertGameModes = `
		INSERT INTO game_modes (mode_name, description, min_players, max_players) VALUES
		('Çizim ve Tahmin', 'Her oyuncu bir kelime yazar, diğerleri bu kelimeleri çizmeye çalışır', 2, 8),
		('Ortak Alan', 'Tüm oyuncular aynı canvas üzerinde birlikte çizim yapar', 2, 12),
		('Serbest Çizim', 'Herkes istediği gibi çizim yapabilir, yarışma yok', 1, 20)
		ON CONFLICT (mode_name) DO NOTHING;`

	createRoomsTable = `
		CREATE TABLE IF NOT EXISTS rooms (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			room_name VARCHAR(100) NOT NULL,
			creator_id UUID REFERENCES users(id) NOT NULL,
			max_players INT NOT NULL,
			current_players INT DEFAULT 0,
			status VARCHAR(20) NOT NULL DEFAULT 'waiting', -- 'waiting', 'playing', 'finished'
			game_mode_id INT REFERENCES game_modes(id) NOT NULL,
			is_private BOOLEAN DEFAULT FALSE,
			room_code VARCHAR(10), -- Özel odalar için kod
			created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
			started_at TIMESTAMP WITH TIME ZONE,
			finished_at TIMESTAMP WITH TIME ZONE
		);`

	createRoomPlayersTable = `
		CREATE TABLE IF NOT EXISTS room_players (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			room_id UUID REFERENCES rooms(id) ON DELETE CASCADE NOT NULL,
			user_id UUID REFERENCES users(id) NOT NULL,
			joined_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
			is_banned BOOLEAN DEFAULT FALSE,
			score INT DEFAULT 0,
			is_online BOOLEAN DEFAULT TRUE,
			UNIQUE(room_id, user_id)
		);`

	createWordsTable = `
		CREATE TABLE IF NOT EXISTS words (
			id SERIAL PRIMARY KEY,
			word VARCHAR(100) UNIQUE NOT NULL,
			language_code VARCHAR(10) NOT NULL DEFAULT 'tr',
			difficulty INT DEFAULT 1, -- 1: Kolay, 2: Orta, 3: Zor
			category VARCHAR(50) -- Hayvan, Nesne, Eylem vs.
		);`

	createGameSessionsTable = `
		CREATE TABLE IF NOT EXISTS game_sessions (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			room_id UUID REFERENCES rooms(id) NOT NULL,
			current_round INT DEFAULT 1,
			total_rounds INT NOT NULL,
			round_duration INT DEFAULT 120, -- Saniye cinsinden
			current_drawer_id UUID REFERENCES users(id),
			current_word_id INT REFERENCES words(id),
			round_start_time TIMESTAMP WITH TIME ZONE,
			round_end_time TIMESTAMP WITH TIME ZONE,
			session_status VARCHAR(20) DEFAULT 'preparing' -- 'preparing', 'active', 'paused', 'finished'
		);`

	createGameActionsTable = `
		CREATE TABLE IF NOT EXISTS game_actions (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			session_id UUID REFERENCES game_sessions(id) ON DELETE CASCADE NOT NULL,
			user_id UUID REFERENCES users(id) NOT NULL,
			action_type VARCHAR(20) NOT NULL, -- 'draw', 'guess', 'chat'
			action_data JSONB, -- Çizim koordinatları, tahmin metni vs.
			created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
			round_number INT NOT NULL
		);`

	createDrawingDataTable = `
		CREATE TABLE IF NOT EXISTS drawing_data (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			session_id UUID REFERENCES game_sessions(id) ON DELETE CASCADE NOT NULL,
			user_id UUID REFERENCES users(id) NOT NULL,
			drawing_json JSONB NOT NULL, -- Çizim verilerinin JSON formatı
			round_number INT NOT NULL,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
		);`

	// Performans için indeksler
	createIndexes = `
		CREATE UNIQUE INDEX IF NOT EXISTS idx_users_username ON users(username);
		CREATE UNIQUE INDEX IF NOT EXISTS idx_users_email ON users(email);
		CREATE INDEX IF NOT EXISTS idx_words_language_code ON words(language_code);
		CREATE INDEX IF NOT EXISTS idx_words_difficulty ON words(difficulty);
		CREATE INDEX IF NOT EXISTS idx_rooms_status ON rooms(status);
		CREATE INDEX IF NOT EXISTS idx_rooms_game_mode ON rooms(game_mode_id);
		CREATE INDEX IF NOT EXISTS idx_room_players_room_id ON room_players(room_id);
		CREATE INDEX IF NOT EXISTS idx_room_players_user_id ON room_players(user_id);
		CREATE INDEX IF NOT EXISTS idx_game_sessions_room_id ON game_sessions(room_id);
		CREATE INDEX IF NOT EXISTS idx_game_actions_session_id ON game_actions(session_id);
		CREATE INDEX IF NOT EXISTS idx_game_actions_type ON game_actions(action_type);
		CREATE INDEX IF NOT EXISTS idx_drawing_data_session_id ON drawing_data(session_id);`

	// Bazı örnek kelimeler ekle
	insertSampleWords = `
		INSERT INTO words (word, language_code, difficulty, category) VALUES
		('kedi', 'tr', 1, 'Hayvan'),
		('köpek', 'tr', 1, 'Hayvan'),
		('araba', 'tr', 1, 'Araç'),
		('ev', 'tr', 1, 'Yapı'),
		('ağaç', 'tr', 1, 'Doğa'),
		('uçak', 'tr', 2, 'Araç'),
		('bilgisayar', 'tr', 2, 'Teknoloji'),
		('telefon', 'tr', 2, 'Teknoloji'),
		('fırın', 'tr', 2, 'Ev Eşyası'),
		('mikroskop', 'tr', 3, 'Bilim'),
		('teleskop', 'tr', 3, 'Bilim')
		ON CONFLICT (word) DO NOTHING;`
)

// initDB, tüm veritabanı tablolarını oluşturur.
func initDB(db *sql.DB) error {
	// 1. Temel tabloları oluştur
	tables := []struct {
		name  string
		query string
	}{
		{"users", createUsersTable},
		{"game_modes", createGameModesTable},
		{"rooms", createRoomsTable},
		{"room_players", createRoomPlayersTable},
		{"words", createWordsTable},
		{"game_sessions", createGameSessionsTable},
		{"game_actions", createGameActionsTable},
		{"drawing_data", createDrawingDataTable},
	}

	for _, table := range tables {
		if _, err := db.Exec(table.query); err != nil {
			return fmt.Errorf("failed to create '%s' table: %w", table.name, err)
		}
		log.Printf("Table '%s' created successfully", table.name)
	}

	// 2. Başlangıç verilerini ekle
	if _, err := db.Exec(insertGameModes); err != nil {
		return fmt.Errorf("failed to insert game modes: %w", err)
	}

	if _, err := db.Exec(insertSampleWords); err != nil {
		return fmt.Errorf("failed to insert sample words: %w", err)
	}

	// 3. İndeksleri oluştur
	if _, err := db.Exec(createIndexes); err != nil {
		return fmt.Errorf("failed to create indexes: %w", err)
	}

	log.Println("Database initialized successfully with all tables and indexes")
	return nil
}
