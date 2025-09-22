package postgres

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

// IsMemberRoom checks if a user is a member of a specific room.
func (r *Repository) IsMemberRoom(ctx context.Context, roomID, userID uuid.UUID) (bool, error) {
	query := `
        SELECT EXISTS(
            SELECT 1 
            FROM room_players
            WHERE room_id = $1 AND user_id = $2
        );`

	var isMember bool
	err := r.db.QueryRowContext(ctx, query, roomID, userID).Scan(&isMember)
	if err != nil {
		// Log the error for debugging, but don't expose it directly to the user.
		return false, fmt.Errorf("failed to check room membership: %w", err)
	}

	return isMember, nil
}
