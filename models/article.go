// models/article.go
package models

import (
	"database/sql"
	"fmt"
	"strings"
	"time"
)

// Article represents a saved article in the reading list.
type Article struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	URL       string    `json:"url"`
	Title     string    `json:"title"`
	Summary   string    `json:"summary,omitempty"` // omitempty will hide if empty
	Tags      []string  `json:"tags"`
	Status    string    `json:"status"` // "processing", "failed", "read", or "unread"
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Save inserts a new article or updates an existing one if ID exists.
func (a *Article) Save() error {
	// Convert []string tags to a comma-separated string for DB storage
	tagsStr := strings.Join(a.Tags, ",")

	var stmt *sql.Stmt
	var err error
	if a.ID == "" { // Insert new article
		a.ID = GenerateUUID()
		a.CreatedAt = time.Now() // Set creation timestamp for new articles
		// For new articles, UpdatedAt is same as CreatedAt initially
		a.UpdatedAt = a.CreatedAt

		stmt, err = DB.Prepare("INSERT INTO articles(id, user_id, url, title, summary, tags, status, created_at, updated_at) VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?)")
		if err != nil {
			return fmt.Errorf("failed to prepare article insert statement: %w", err)
		}
		defer stmt.Close()
		_, err = stmt.Exec(a.ID, a.UserID, a.URL, a.Title, a.Summary, tagsStr, a.Status, a.CreatedAt, a.UpdatedAt)
		if err != nil {
			return fmt.Errorf("failed to insert article: %w", err)
		}
	} else { // Update existing article
		// For updates, only update UpdatedAt
		a.UpdatedAt = time.Now()
		stmt, err = DB.Prepare("UPDATE articles SET url=?, title=?, summary=?, tags=?, status=?, updated_at=? WHERE id=? AND user_id=?")
		if err != nil {
			return fmt.Errorf("failed to prepare article update statement: %w", err)
		}
		defer stmt.Close()
		_, err = stmt.Exec(a.URL, a.Title, a.Summary, tagsStr, a.Status, a.UpdatedAt, a.ID, a.UserID)
		if err != nil {
			return fmt.Errorf("failed to update article: %w", err)
		}
	}
	return nil
}

// DeleteArticle deletes an article by ID and user ID.
func DeleteArticle(id, userID string) error {
	stmt, err := DB.Prepare("DELETE FROM articles WHERE id=? AND user_id=?")
	if err != nil {
		return fmt.Errorf("failed to prepare article delete statement: %w", err)
	}
	defer stmt.Close()

	result, err := stmt.Exec(id, userID)
	if err != nil {
		return fmt.Errorf("failed to delete article: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("article with ID '%s' not found or not owned by user '%s'", id, userID)
	}

	return nil
}

// UpdateArticleStatus updates the status of an existing article.
func UpdateArticleStatus(id, userID, newStatus string) error {
	// Prepare the statement with a WHERE clause that includes both ID and UserID for security
	stmt, err := DB.Prepare("UPDATE articles SET status=?, updated_at=CURRENT_TIMESTAMP WHERE id=? AND user_id=?")
	if err != nil {
		return fmt.Errorf("failed to prepare article status update statement: %w", err)
	}
	defer stmt.Close()

	result, err := stmt.Exec(newStatus, id, userID)
	if err != nil {
		return fmt.Errorf("failed to update article status: %w", err)
	}

	// Check if a row was actually affected to ensure the article existed and belonged to the user
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rowsAffected == 0 {
		// Handling not-found or unauthorized updates
		return fmt.Errorf("article with ID '%s' not found or not owned by user '%s'", id, userID)
	}

	return nil
}

// UpdateArticleTags updates the tags of an existing article.
func UpdateArticleTags(id, userID string, newTags []string) error {
	// Convert []string tags to a comma-separated string for DB storage
	tagsStr := strings.Join(newTags, ",")

	// Prepare the statement with a WHERE clause that includes both ID and UserID for security
	stmt, err := DB.Prepare("UPDATE articles SET tags=?, updated_at=CURRENT_TIMESTAMP WHERE id=? AND user_id=?")
	if err != nil {
		return fmt.Errorf("failed to prepare article tags update statement: %w", err)
	}
	defer stmt.Close()

	result, err := stmt.Exec(tagsStr, id, userID)
	if err != nil {
		return fmt.Errorf("failed to update article tags: %w", err)
	}

	// Check if a row was actually affected to ensure the article existed and belonged to the user
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("article with ID '%s' not found or not owned by user '%s'", id, userID)
	}

	return nil
}

// GetArticlesByUserID retrieves all articles for a given user, with optional filters.
func GetArticlesByUserID(userID, statusFilter, tagFilter string) ([]Article, error) {
	query := "SELECT id, user_id, url, title, summary, tags, status, created_at, updated_at FROM articles WHERE user_id = ?"
	args := []interface{}{userID}

	if statusFilter != "" {
		query += " AND status = ?"
		args = append(args, statusFilter)
	}
	if tagFilter != "" {
		// Use LIKE for tag filtering, assuming comma-separated tags
		query += " AND tags LIKE ?"
		args = append(args, "%"+tagFilter+"%") // Matches if tagFilter is anywhere in the string
	}

	rows, err := DB.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query articles: %w", err)
	}
	defer rows.Close()

	var articles []Article
	for rows.Next() {
		var a Article
		var tagsStr string // Temporary variable for scanning tags
		err := rows.Scan(
			&a.ID, &a.UserID, &a.URL, &a.Title, &a.Summary,
			&tagsStr, &a.Status, &a.CreatedAt, &a.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan article row: %w", err)
		}
		a.Tags = strings.Split(tagsStr, ",") // Convert back to []string
		articles = append(articles, a)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating article rows: %w", err)
	}

	return articles, nil
}

// GetArticleByID retrieves a single article by its ID and user ID.
func GetArticleByID(id, userID string) (*Article, error) {
	article := &Article{}
	var tagsStr string
	row := DB.QueryRow("SELECT id, user_id, url, title, summary, tags, status, created_at, updated_at FROM articles WHERE id = ? AND user_id = ?", id, userID)
	err := row.Scan(
		&article.ID, &article.UserID, &article.URL, &article.Title, &article.Summary,
		&tagsStr, &article.Status, &article.CreatedAt, &article.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // Article not found
		}
		return nil, fmt.Errorf("failed to get article by ID: %w", err)
	}
	article.Tags = strings.Split(tagsStr, ",")
	return article, nil
}
func GetTagsByUserID(userID string) ([]string, error) {
	query := "SELECT DISTINCT tags FROM articles WHERE user_id = ?"
	rows, err := DB.Query(query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query tags: %w", err)
	}
	defer rows.Close()

	tagSet := make(map[string]struct{}) // Use a map to avoid duplicates
	for rows.Next() {
		var tagsStr string
		if err := rows.Scan(&tagsStr); err != nil {
			return nil, fmt.Errorf("failed to scan tag row: %w", err)
		}
		tags := strings.Split(tagsStr, ",")
		for _, tag := range tags {
			tagSet[tag] = struct{}{} // Add to set
		}
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating tag rows: %w", err)
	}

	var tags []string
	for tag := range tagSet {
		tags = append(tags, tag)
	}

	return tags, nil
}
