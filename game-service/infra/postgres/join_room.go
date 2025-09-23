package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"game-service/domain"
	"log"
	"strings"

	"github.com/google/uuid"
)

func (r *Repository) JoinRoom(ctx context.Context, roomID, userID uuid.UUID, roomCode string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// 1. Room bilgilerini çek
	var maxPlayers, currentPlayers int
	var status string
	var isPrivate bool
	var dbRoomCode sql.NullString

	err = tx.QueryRowContext(ctx,
		`SELECT max_players, current_players, status, is_private, room_code 
		 FROM rooms WHERE id = $1 FOR UPDATE`,
		roomID,
	).Scan(&maxPlayers, &currentPlayers, &status, &isPrivate, &dbRoomCode)
	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("%w: room not found", domain.ErrNotFound)
		}
		return fmt.Errorf("failed to query room: %w", err)
	}

	// 2. Oda durumu kontrol et
	if status != "waiting" {
		return fmt.Errorf("%w: room is not joinable", domain.ErrConflict)
	}

	// 3. Kapasite kontrolü
	if currentPlayers >= maxPlayers {
		return fmt.Errorf("%w: room is full", domain.ErrConflict)
	}

	// 4. Eğer private oda ise kod kontrol et
	if isPrivate {
		if !dbRoomCode.Valid || dbRoomCode.String == "" {
			return fmt.Errorf("%w: private room has no code", domain.ErrInternal)
		}
		if roomCode != dbRoomCode.String {
			return fmt.Errorf("%w: invalid room code", domain.ErrForbidden)
		}
	}

	// 5. Kullanıcıyı ekle
	_, err = tx.ExecContext(ctx,
		`INSERT INTO room_players (room_id, user_id) VALUES ($1, $2)`,
		roomID, userID,
	)
	if err != nil {
		if strings.Contains(err.Error(), "unique constraint") {
			return fmt.Errorf("%w: user already in room", domain.ErrConflict)
		}
		return fmt.Errorf("failed to insert player: %w", err)
	}

	// 6. Oda oyuncu sayısını arttır
	_, err = tx.ExecContext(ctx,
		`UPDATE rooms SET current_players = current_players + 1 WHERE id = $1`,
		roomID,
	)
	if err != nil {
		return fmt.Errorf("failed to update room: %w", err)
	}

	// 7. Commit
	if err = tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	log.Printf("User %s joined room %s successfully", userID, roomID)
	return nil
}
