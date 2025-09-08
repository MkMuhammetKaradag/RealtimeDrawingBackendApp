package postgres

import (
	"errors"

	"github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"
)

func (r *Repository) hashPassword(password string) (string, error) {

	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(hashed), err
}
func (r *Repository) isDuplicateKeyError(err error) bool {
	var pqErr *pq.Error
	if errors.As(err, &pqErr) {
		// PostgreSQL error code for unique_violation
		return pqErr.Code == "23505"
	}

	return false
}
