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
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()

		)`
)

func initDB(db *sql.DB) error {
	if _, err := db.Exec(createUsersTable); err != nil {
		return fmt.Errorf("failed to create users table: %w", err)
	}

	log.Println("Database tables initialized")
	return nil
}
