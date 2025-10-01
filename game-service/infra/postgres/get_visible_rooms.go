package postgres

import (
	"context"
	"fmt"
	"game-service/domain"
	"log"

	"github.com/google/uuid"
)

const getVisibleRoomsQuery = `
    SELECT
        r.id, r.room_name, r.creator_id, r.max_players, r.current_players, r.status,
        r.game_mode_id, r.is_private, COALESCE(r.room_code, '') AS room_code, gm.mode_name,
		CASE WHEN rp.user_id IS NOT NULL THEN TRUE ELSE FALSE END AS is_user_in_room 
    FROM
        rooms r
    INNER JOIN
        game_modes gm ON r.game_mode_id = gm.id
	LEFT JOIN 
    	room_players rp ON r.id = rp.room_id AND rp.user_id = $1 -- $1
    WHERE
        -- Durumu 'waiting' olan odaları çekiyoruz, böylece sadece oynanabilir odalar listelenir.
        r.status = 'waiting'
        AND (
            -- KOŞUL 1: Gizli olmayan odalar (is_private = FALSE)
            r.is_private = FALSE
            OR
            -- KOŞUL 2: Kullanıcının üyesi olduğu gizli odalar (is_private = TRUE)
            (
                r.is_private = TRUE
                AND EXISTS (
                    SELECT 1 FROM room_players rp
                    WHERE rp.room_id = r.id AND rp.user_id = $1
                )
            )
        )
    ORDER BY
        r.created_at DESC;`

// GetVisibleRooms, kullanıcının görebileceği (gizli olmayan veya üyesi olduğu gizli) tüm odaları döndürür.
func (r *Repository) GetVisibleRooms(ctx context.Context, userID uuid.UUID) ([]domain.Room, error) {
	rows, err := r.db.QueryContext(ctx, getVisibleRoomsQuery, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query visible rooms: %w", err)
	}
	defer rows.Close()

	var rooms []domain.Room
	for rows.Next() {
		var room domain.Room
		// Scan işlemine yeni alanı ekliyoruz.
		err := rows.Scan(
			&room.ID, &room.RoomName, &room.CreatorID, &room.MaxPlayers, &room.CurrentPlayers,
			&room.Status, &room.GameModeID, &room.IsPrivate, &room.RoomCode, &room.ModeName,
			&room.IsUserInRoom, // ⬅️ YENİ ALAN BURAYA EKLENDİ
		)
		if err != nil {
			// Hata mesajını daha anlaşılır yapalım
			return nil, fmt.Errorf("failed to scan room data from DB: %w", err)
		}

		// Gizli olmayan odaların RoomCode'unu temizleme (mevcut mantık)
		if !room.IsPrivate {
			room.RoomCode = ""
		}

		rooms = append(rooms, room)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}

	log.Printf("Found %d visible rooms for user %s", len(rooms), userID)
	return rooms, nil
}
