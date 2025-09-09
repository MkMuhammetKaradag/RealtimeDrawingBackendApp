package postgres

import (
	"auth-service/domain"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

func (r *Repository) Activate(ctx context.Context, activationID uuid.UUID, code string) (*domain.User, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("transaction begin failed: %w", err)
	}
	defer tx.Rollback()

	query := `
        UPDATE users
        SET is_active = true, is_email_verified = true
        WHERE activation_id = $1 AND activation_code = $2 AND activation_expiry > $3
        RETURNING id, username, email;`

	var user domain.User
	err = tx.QueryRowContext(ctx, query, activationID, code, time.Now()).Scan(&user.ID, &user.Username, &user.Email)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// This means no row was updated, so the link is invalid or expired.
			return nil, ErrInvalidActivation
		}
		return nil, fmt.Errorf("failed to activate user: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("transaction commit failed: %w", err)
	}

	return &user, nil
}
