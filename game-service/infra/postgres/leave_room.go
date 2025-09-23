package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"game-service/domain"
	"log"

	"github.com/google/uuid"
)

func (r *Repository) LeaveRoom(ctx context.Context, roomID, userID uuid.UUID) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// 1. Check if the user is in the room and get room details, locking the room row for the transaction.
	var hostID uuid.UUID
	var currentPlayers int
	err = tx.QueryRowContext(ctx,
		`SELECT creator_id, current_players
		 FROM rooms WHERE id = $1 FOR UPDATE`,
		roomID,
	).Scan(&hostID, &currentPlayers)
	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("%w: room not found", domain.ErrNotFound)
		}
		return fmt.Errorf("failed to query room: %w", err)
	}

	// 2. Remove the user from the room.
	res, err := tx.ExecContext(ctx,
		`DELETE FROM room_players WHERE room_id = $1 AND user_id = $2`,
		roomID, userID,
	)
	if err != nil {
		return fmt.Errorf("failed to delete player from room: %w", err)
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("%w: user is not in the room", domain.ErrNotFound)
	}

	// 3. Check for specific conditions based on player count and role.
	if currentPlayers == 1 {
		// If the user is the only player, delete the room.
		_, err = tx.ExecContext(ctx, `DELETE FROM rooms WHERE id = $1`, roomID)
		if err != nil {
			return fmt.Errorf("failed to delete room: %w", err)
		}
	} else {
		// If there are other players, update the player count.
		_, err = tx.ExecContext(ctx,
			`UPDATE rooms SET current_players = current_players - 1 WHERE id = $1`,
			roomID,
		)
		if err != nil {
			return fmt.Errorf("failed to decrement player count: %w", err)
		}

		// If the leaving user was the host, assign a new one.
		if hostID == userID {
			var newHostID uuid.UUID
			err = tx.QueryRowContext(ctx,
				`SELECT user_id FROM room_players WHERE room_id = $1 LIMIT 1`,
				roomID,
			).Scan(&newHostID)
			if err != nil {
				// This case should ideally not be reached if currentPlayers > 1.
				return fmt.Errorf("failed to find a new host: %w", err)
			}

			_, err = tx.ExecContext(ctx,
				`UPDATE rooms SET creator_id = $1 WHERE id = $2`,
				newHostID, roomID,
			)
			if err != nil {
				return fmt.Errorf("failed to update new host: %w", err)
			}
		}
	}

	// 4. Commit the transaction.
	if err = tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	log.Printf("User %s left room %s successfully", userID, roomID)
	return nil
}
