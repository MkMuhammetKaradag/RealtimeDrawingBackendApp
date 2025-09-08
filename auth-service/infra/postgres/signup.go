package postgres

import (
	"auth-service/domain"
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
	"time"

	"github.com/google/uuid"
)

func (r *Repository) SignUp(ctx context.Context, user *domain.User) (uuid.UUID, string, error) {

	hashedPassword, err := r.hashPassword(user.Password)
	if err != nil {
		return uuid.Nil, "", fmt.Errorf("hashing error: %w", err)
	}

	if user.ActivationCode == "" {
		code, err := generateRandomCode(6)
		if err != nil {
			return uuid.Nil, "", err
		}
		user.ActivationCode = code
	}

	if user.ActivationExpiry.IsZero() {
		user.ActivationExpiry = time.Now().Add(10 * time.Minute)
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return uuid.Nil, "", fmt.Errorf("transaction error: %w", err)
	}
	defer tx.Rollback()

	query := `
		INSERT INTO users (
			username, email, password, 
			activation_code, activation_expiry,
			failed_login_attempts, account_locked, is_2fa_enabled
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8) RETURNING activation_id`

	var activationID uuid.UUID
	err = tx.QueryRowContext(ctx, query,
		user.Username,
		user.Email,
		hashedPassword,
		user.ActivationCode,
		user.ActivationExpiry,
		0,
		false,
		false,
	).Scan(&activationID)

	if err != nil {
		if r.isDuplicateKeyError(err) {
			return uuid.Nil, "", fmt.Errorf("username or email already exists: %w", err)
		}
		return uuid.Nil, "", fmt.Errorf("insert error: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return uuid.Nil, "", fmt.Errorf("commit error: %w", err)
	}

	return activationID, user.ActivationCode, nil
}
func generateRandomCode(length int) (string, error) {
	if length <= 0 {
		return "", fmt.Errorf("geçersiz uzunluk")
	}

	max := big.NewInt(10)
	code := make([]byte, length)

	for i := range code {
		n, err := rand.Int(rand.Reader, max)
		if err != nil {
			return "", fmt.Errorf("rastgele sayı üretilemedi: %w", err)
		}
		code[i] = byte(n.Int64()) + '0'
	}

	return string(code), nil
}
