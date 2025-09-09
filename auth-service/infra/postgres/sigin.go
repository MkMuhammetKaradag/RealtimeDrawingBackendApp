package postgres

import (
	"auth-service/domain"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"golang.org/x/crypto/bcrypt"
)

const maxLoginAttempts = 5 // You can set this value based on your security policy.

func (r *Repository) SignIn(ctx context.Context, identifier, password string) (*domain.User, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("transaction begin failed: %w", err)
	}
	defer tx.Rollback()

	// 1. Fetch user data and check for account status in a single query.
	const query = `
        SELECT id, username, email, password, failed_login_attempts, account_locked, lock_until
        FROM users
        WHERE (username = $1 OR email = $1) AND is_active = true`

	var user domain.User
	var hashedPassword string
	var failedAttempts int
	var accountLocked bool
	var lockUntil sql.NullTime

	err = tx.QueryRowContext(ctx, query, identifier).Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&hashedPassword,
		&failedAttempts,
		&accountLocked,
		&lockUntil,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrInvalidCredentials
		}
		return nil, fmt.Errorf("query error: %w", err)
	}

	// 2. Check for a locked account and if the lock has expired.
	if accountLocked && lockUntil.Valid && lockUntil.Time.After(time.Now()) {
		return nil, ErrAccountLocked
	}

	// 3. Compare passwords and handle the result.
	if err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password)); err != nil {
		// Password comparison failed.
		newAttempts := failedAttempts + 1
		var lockUntilTime sql.NullTime

		// If attempts exceed the limit, lock the account.
		if newAttempts >= maxLoginAttempts {
			lockUntilTime.Time = time.Now().Add(1 * time.Hour) // Lock for 1 hour.
			lockUntilTime.Valid = true
			accountLocked = true
		}

		// Update failed attempts and lock status in a single query.
		updateQuery := `
            UPDATE users SET failed_login_attempts = $1, account_locked = $2, lock_until = $3 WHERE id = $4`
		if _, updateErr := tx.ExecContext(ctx, updateQuery, newAttempts, accountLocked, lockUntilTime, user.ID); updateErr != nil {
			// Log this failure but don't return it to the user.
			fmt.Printf("Failed to update login attempts for user %s: %v\n", user.Email, updateErr)
		}
		return nil, ErrInvalidCredentials
	}

	// 4. If the password is correct, reset attempts and update last login time.
	updateQuery := `
        UPDATE users SET failed_login_attempts = 0, account_locked = false, last_login = NOW(), lock_until = NULL WHERE id = $1`
	if _, updateErr := tx.ExecContext(ctx, updateQuery, user.ID); updateErr != nil {
		fmt.Printf("Failed to update last login for user %s: %v\n", user.Email, updateErr)
	}

	// 5. Commit the transaction and return the user.
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("transaction commit failed: %w", err)
	}

	return &user, nil
}
