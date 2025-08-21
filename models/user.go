// models/user.go
package models

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt" // For password hashing
)

// User represents a user in the system.
type User struct {
	ID           string    `json:"id"`
	Username     string    `json:"username"`
	PasswordHash string    `json:"-"` // Don't expose this in JSON
	CreatedAt    time.Time `json:"created_at"`
}

// HashPassword hashes the user's plain-text password using bcrypt.
func (u *User) HashPassword(password string) error {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}
	u.PasswordHash = string(bytes)
	return nil
}

// CheckPasswordHash compares a plain-text password with the stored hash.
func (u *User) CheckPasswordHash(password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password))
	return err == nil
}

// CreateUser inserts a new user into the database.
func CreateUser(user *User) error {
	// Assign a new UUID if one isn't already set (e.g., from an external source)
	if user.ID == "" {
		user.ID = GenerateUUID()
	}
	// Set creation timestamp, good practice to ensure it's recorded
	if user.CreatedAt.IsZero() {
		user.CreatedAt = time.Now()
	}
	stmt, err := DB.Prepare("INSERT INTO users(id, username, password_hash, created_at) VALUES(?, ?, ?, ?)")
	if err != nil {
		return fmt.Errorf("failed to prepare user insert statement: %w", err)
	}
	defer stmt.Close()

	_, err = stmt.Exec(user.ID, user.Username, user.PasswordHash, user.CreatedAt)
	if err != nil {
		// Specific error handling for sqlite3 unique constraint violation
		// (e.g., if username already exists)
		// For sqlite3, the error message often contains "UNIQUE constraint failed"
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			return fmt.Errorf("username '%s' already exists", user.Username)
		}
		return fmt.Errorf("failed to create user: %w", err)
	}
	return nil
}

// AuthenticateUser checks if the provided username and password match a user in the database.
func AuthenticateUser(username, password string) (*User, error) {
	user := &User{}
	row := DB.QueryRow("SELECT id, username, password_hash, created_at FROM users WHERE username = ?", username)
	err := row.Scan(&user.ID, &user.Username, &user.PasswordHash, &user.CreatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("invalid username or password")
		}
		return nil, fmt.Errorf("failed to get user by username: %w", err)
	}

	// Check if the provided password matches the stored hash
	if !user.CheckPasswordHash(password) {
		return nil, fmt.Errorf("invalid username or password")
	}

	return user, nil
}

// GetUserByUsername retrieves a user by their username.
func GetUserByUsername(username string) (*User, error) {
	user := &User{}
	row := DB.QueryRow("SELECT id, username, password_hash, created_at FROM users WHERE username = ?", username)
	err := row.Scan(&user.ID, &user.Username, &user.PasswordHash, &user.CreatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, sql.ErrNoRows
		}
		return nil, fmt.Errorf("failed to get user by username: %w", err)
	}
	return user, nil
}
