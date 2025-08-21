package models

import (
	"time"
)

// RevokedToken represents an entry in the revoked_tokens table.
type RevokedToken struct {
	Token     string
	ExpiresAt time.Time
}

// RevokeToken inserts a token into the revoked_tokens table.
func RevokeToken(token string, expiresAt time.Time) error {
	stmt, err := DB.Prepare("INSERT INTO revoked_tokens(token, expires_at) VALUES(?, ?)")
	if err != nil {
		return err
	}
	defer stmt.Close()

	_, err = stmt.Exec(token, expiresAt)
	return err
}

// IsTokenRevoked checks if a token exists in the revoked_tokens table.
func IsTokenRevoked(token string) (bool, error) {
	var exists bool
	err := DB.QueryRow("SELECT EXISTS(SELECT 1 FROM revoked_tokens WHERE token = ?)", token).Scan(&exists)
	if err != nil {
		return false, err
	}
	return exists, nil
}
