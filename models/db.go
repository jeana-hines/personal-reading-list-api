// models/db.go
package models

import (
	"database/sql"
	"log"

	_ "github.com/mattn/go-sqlite3" // Import the SQLite driver
)

// DB holds the database connection pool
var DB *sql.DB

// InitDB initializes the database connection
func InitDB(dataSourceName string) {
	var err error
	DB, err = sql.Open("sqlite3", dataSourceName) // "sqlite3" is the driver name
	if err != nil {
		log.Fatalf("Error opening database: %v", err)
	}

	// Ping the database to verify the connection
	if err = DB.Ping(); err != nil {
		log.Fatalf("Error connecting to database: %v", err)
	}

	log.Println("Database connection established successfully.")

	// You can call a function here to set up your tables
	createTables()
}

// createTables sets up the necessary tables in the database
func createTables() {
	// SQL to create Users table
	usersTableSQL := `
	CREATE TABLE IF NOT EXISTS users (
		id TEXT PRIMARY KEY,
		username TEXT UNIQUE NOT NULL,
		password_hash TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);`

	// SQL to create Articles table
	articlesTableSQL := `
	CREATE TABLE IF NOT EXISTS articles (
		id TEXT PRIMARY KEY,
		user_id TEXT NOT NULL,
		url TEXT NOT NULL,
		title TEXT NOT NULL,
		summary TEXT,
		tags TEXT, -- Storing as comma-separated string for simplicity initially
		status TEXT NOT NULL DEFAULT 'unread', -- 'read' or 'unread'
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (user_id) REFERENCES users(id)
	);`

	// SQL to create Revoked Tokens table
	revokeTokensTableSQL := `
    CREATE TABLE IF NOT EXISTS revoked_tokens (
        token TEXT PRIMARY KEY,
        expires_at TIMESTAMP
    );
    `
	// Execute table creation queries
	_, err := DB.Exec(usersTableSQL)
	if err != nil {
		log.Fatalf("Error creating users table: %v", err)
	}

	_, err = DB.Exec(articlesTableSQL)
	if err != nil {
		log.Fatalf("Error creating articles table: %v", err)
	}

	_, err = DB.Exec(revokeTokensTableSQL)
	if err != nil {
		log.Fatalf("Error creating revoked_tokens table: %v", err)
	}

	log.Println("Tables created or already exist.")
}

// CloseDB closes the database connection
func CloseDB() {
	if DB != nil {
		err := DB.Close()
		if err != nil {
			log.Printf("Error closing database: %v", err)
		} else {
			log.Println("Database connection closed.")
		}
	}
}
