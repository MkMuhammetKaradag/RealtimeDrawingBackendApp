package postgres

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

// IsMemberRoom checks if a user is a member of a specific room.
func (r *Repository) IsMemberAndHostRoom(ctx context.Context, roomID, userID uuid.UUID) (isMember bool, isHost bool, err error) {
	// Sorgu: Hem room_players tablosunda kullanıcının varlığını, hem de rooms
	// tablosunda kullanıcının creator_id olup olmadığını kontrol eder.
	query := `
        SELECT
            EXISTS (SELECT 1 FROM room_players WHERE room_id = $1 AND user_id = $2),
            EXISTS (SELECT 1 FROM rooms WHERE id = $1 AND creator_id = $2);`

	err = r.db.QueryRowContext(ctx, query, roomID, userID).Scan(&isMember, &isHost)
	if err != nil {
		// Hata oluşursa (örn. veritabanı bağlantı sorunu)
		return false, false, fmt.Errorf("oda üyeliği ve yöneticilik kontrolü başarısız: %w", err)
	}

	return isMember, isHost, nil
}
