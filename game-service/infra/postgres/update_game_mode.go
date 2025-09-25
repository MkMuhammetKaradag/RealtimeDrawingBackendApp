package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"game-service/domain"
	"log"
	"strings"

	"github.com/google/uuid"
)

func (r *Repository) UpdateRoomGameMode(ctx context.Context, roomID uuid.UUID, userID uuid.UUID, newGameModeID int) error {
	// 1. Oda yaratıcısının kim olduğunu ve odanın varlığını kontrol et.
	// Aynı zamanda yeni game_mode_id'nin var olup olmadığını kontrol etmek de iyi bir fikirdir.

	checkQuery := `
		SELECT 
			creator_id
		FROM 
			rooms
		WHERE 
			id = $1
	`
	var creatorID uuid.UUID
	err := r.db.QueryRowContext(ctx, checkQuery, roomID).Scan(&creatorID)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// Oda bulunamadı hatası
			return fmt.Errorf("%w: room not found with ID %s", domain.ErrNotFound, roomID)
		}
		// Diğer beklenmedik veritabanı hataları
		return fmt.Errorf("failed to check room creator: %w", err)
	}

	// 2. Kullanıcının oda yaratıcısı olup olmadığını kontrol et.
	if creatorID != userID {
		// Yetkilendirme hatası
		return fmt.Errorf("%w: only the room creator can change the game mode", domain.ErrForbidden)
	}

	// 3. Oyun modunu güncelleme sorgusunu hazırla.
	updateQuery := `
		UPDATE 
			rooms
		SET 
			game_mode_id = $1
		WHERE 
			id = $2
	`

	// 4. Güncelleme işlemini gerçekleştir.
	result, err := r.db.ExecContext(ctx, updateQuery, newGameModeID, roomID)
	if err != nil {
		// Örneğin, FOREIGN KEY hatası (game_mode_id'nin game_modes tablosunda olmaması) kontrol edilebilir.
		// PostgreSQL hata mesajını kontrol etmek zor olduğundan, bu kontrolü yapan ayrı bir sorgu daha güvenilirdir.
		if strings.Contains(err.Error(), "foreign key constraint") {
			return fmt.Errorf("%w: game mode ID %d does not exist", domain.ErrInvalidInput, newGameModeID)
		}
		return fmt.Errorf("failed to update room game mode: %w", err)
	}

	// 5. Kaç satırın etkilendiğini kontrol et (güncellemenin başarılı olup olmadığını doğrulamak için).
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %w", err)
	}

	if rowsAffected == 0 {
		// Bu kısma normalde düşülmemeli (çünkü varlık kontrolü daha önce yapıldı),
		// ancak yine de bir güvenlik kontrolü olarak tutulabilir.
		return fmt.Errorf("%w: room not found or no change was needed", domain.ErrNotFound)
	}

	log.Printf("Room %s game mode updated to %d by creator %s", roomID, newGameModeID, userID)
	return nil
}
