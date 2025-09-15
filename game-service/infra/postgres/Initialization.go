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
			description TEXT
		);`

	insertGameModes = `
		INSERT INTO game_modes (mode_name, description) VALUES
		('Çizim ve Tahmin', 'Bir oyuncunun çizim yaptığı ve diğerlerinin kelimeyi tahmin etmeye çalıştığı klasik  mod.'),
		('Ortak Alan', 'Birden fazla oyuncunun aynı kanvas üzerinde çizim yapabildiği ve işbirliği yaptığı mod.')
		ON CONFLICT (mode_name) DO NOTHING;`

	createRoomsTable = `
		CREATE TABLE IF NOT EXISTS rooms (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			room_name VARCHAR(100) NOT NULL,
			creator_id UUID REFERENCES users(id) NOT NULL,
			max_players INT NOT NULL,
			status VARCHAR(20) NOT NULL DEFAULT 'waiting',
			created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
			game_mode_id INT REFERENCES game_modes(id) NOT NULL
		);`

	createWordsTable = `
		CREATE TABLE IF NOT EXISTS words (
			id SERIAL PRIMARY KEY,
			word VARCHAR(100) UNIQUE NOT NULL,
			language_code VARCHAR(10) NOT NULL DEFAULT 'tr',
			difficulty INT DEFAULT 1
		);`

	createGameStatsTable = `
		CREATE TABLE IF NOT EXISTS game_stats (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			game_id UUID UNIQUE NOT NULL,
			winner_id UUID REFERENCES users(id),
			start_time TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
			end_time TIMESTAMP WITH TIME ZONE,
			total_rounds INT NOT NULL,
			winning_score INT
		);`

	// Performans için indeksler
	createIndexes = `
		CREATE UNIQUE INDEX IF NOT EXISTS idx_users_username ON users(username);
		CREATE UNIQUE INDEX IF NOT EXISTS idx_users_email ON users(email);
		CREATE INDEX IF NOT EXISTS idx_words_language_code ON words(language_code);
		CREATE INDEX IF NOT EXISTS idx_rooms_status ON rooms(status);`
)

// initDB, tüm veritabanı tablolarını oluşturur.
func initDB(db *sql.DB) error {
	// 1. Önce bağımlılığı olmayan temel tabloları oluştur
	if _, err := db.Exec(createUsersTable); err != nil {
		return fmt.Errorf("failed to create 'users' table: %w", err)
	}

	if _, err := db.Exec(createGameModesTable); err != nil {
		return fmt.Errorf("failed to create 'game_modes' table: %w", err)
	}

	// 2. Başlangıç verilerini ekle (bu, bağımlı tabloları oluşturmadan önce yapılabilir)
	if _, err := db.Exec(insertGameModes); err != nil {
		return fmt.Errorf("failed to insert game modes: %w", err)
	}

	// 3. Daha sonra, bağımlı tabloları oluştur
	if _, err := db.Exec(createRoomsTable); err != nil {
		return fmt.Errorf("failed to create 'rooms' table: %w", err)
	}

	if _, err := db.Exec(createWordsTable); err != nil {
		return fmt.Errorf("failed to create 'words' table: %w", err)
	}

	if _, err := db.Exec(createGameStatsTable); err != nil {
		return fmt.Errorf("failed to create 'game_stats' table: %w", err)
	}

	// 4. Son olarak indeksleri oluştur
	if _, err := db.Exec(createIndexes); err != nil {
		return fmt.Errorf("failed to create indexes: %w", err)
	}

	log.Println("Database tables initialized successfully")
	return nil
}
