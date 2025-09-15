package domain

import "time"

type User struct {
	ID                  string    `json:"id"`
	Username            string    `json:"username"`
	Email               string    `json:"email"`
	Password            string    `json:"password"`
	ActivationCode      string    `json:"activationCode"`
	ActivationExpiry    time.Time `json:"activationExpiry"`
	FailedLoginAttempts int       `json:"failedLoginAttempts"`
	AccountLocked       bool      `json:"accountLocked"`
	LockUntil           time.Time `json:"lockUntil"`
	Is2FAEnabled        bool      `json:"is2FAEnabled"`
}
