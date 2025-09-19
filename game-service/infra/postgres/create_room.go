package postgres

import (
	"context"
	"fmt"
	"game-service/domain"
	"log"
	"strings"

	"github.com/google/uuid"
)

func (r *Repository) CreateRoom(ctx context.Context, roomName string, creatorID uuid.UUID, maxPlayers int, gameModeID int, isPrivate bool, roomCode string) (uuid.UUID, error) {
	// 1. Gelen veriyi doğrula
	// Not: Bu kontrol, daha çok iş mantığı katmanında (service layer) yapılmalıdır.
	// Ancak burada da gösterilebilir.
	// `game_modes` tablosundan min/max değerlerini çekerek daha dinamik bir kontrol yapabilirsiniz.
	if maxPlayers < 2 || maxPlayers > 12 {
		return uuid.Nil, fmt.Errorf("%w: invalid number of players", domain.ErrInvalidInput)
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// 2. Yeni odayı ekle ve ID'yi al
	roomQuery := `
		INSERT INTO rooms (room_name, creator_id, max_players, current_players, status, game_mode_id, is_private, room_code)
		VALUES ($1, $2, $3, 1, 'waiting', $4, $5, $6)
		RETURNING id
	`
	var roomID uuid.UUID
	err = tx.QueryRowContext(ctx, roomQuery, roomName, creatorID, maxPlayers, gameModeID, isPrivate, roomCode).Scan(&roomID)
	if err != nil {
		// PostgreSQL hata kodlarını veya mesajını kontrol et
		if strings.Contains(err.Error(), "unique constraint") {
			return uuid.Nil, fmt.Errorf("%w: room with this name or code already exists", domain.ErrConflict)
		}
		// Diğer beklenmedik veritabanı hataları
		return uuid.Nil, fmt.Errorf("failed to create room: %w", err)
	}

	// 3. Oluşturan kullanıcıyı odaya ekle
	playerQuery := `
		INSERT INTO room_players (room_id, user_id)
		VALUES ($1, $2)
	`
	_, err = tx.ExecContext(ctx, playerQuery, roomID, creatorID)
	if err != nil {
		// Bu hatayı da kontrol edebilirsiniz, ancak normalde bu işlem başarılı olur.
		return uuid.Nil, fmt.Errorf("failed to add creator to room: %w", err)
	}

	if err = tx.Commit(); err != nil {
		return uuid.Nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	log.Printf("Room '%s' created successfully with ID %s", roomName, roomID)
	return roomID, nil
}
