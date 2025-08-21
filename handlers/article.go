package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	// Import your custom packages
	"github.com/jeana-hines/personal-reading-list-api/models"   // Import your models package
	"github.com/jeana-hines/personal-reading-list-api/services" // Import your services package
)

// Define a struct for the article submission request body
// This is what the client sends in the JSON payload
type ArticleSubmissionRequest struct {
	URL string `json:"url" example:"https://example.com/article"`
}

// @Summary Submit a new article
// @Description Submits a new article to the reading list.
// @ID submit-article
// @Accept json
// @Produce json
// @Param article body ArticleSubmissionRequest true "Article submission details"
// @Success 201 {object} models.Article "Article submitted successfully"
// @Failure 400 {object} ErrorResponse "Invalid request payload or missing fields"
// @Failure 401 {object} ErrorResponse "Unauthorized: User ID not found"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /articles [post]
func SubmitArticle(w http.ResponseWriter, r *http.Request) {
	// Get the user ID from the context (set by AuthMiddleware)
	userID, ok := r.Context().Value(UserIDKey).(string)
	if !ok {
		log.Println("Unauthorized: User ID not found in context")
		http.Error(w, "Unauthorized: User ID not found", http.StatusUnauthorized)
		return
	}
	var req ArticleSubmissionRequest
	// Decode the JSON request body into our struct
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		// Respond with a 400 Bad Request if the JSON is malformed
		log.Printf("Error decoding request body: %v", err)
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	// Basic validation (add more comprehensive validation later if needed)
	if req.URL == "" {
		http.Error(w, "URL is required", http.StatusBadRequest)
		return
	}

	// Create a new Article model instance
	article := &models.Article{
		UserID: userID,
		URL:    req.URL,
		Status: "processing", // Default status for new articles
	}

	// Save the article to the database
	err = article.Save()
	if err != nil {
		log.Printf("Error creating article in database: %v", err)
		http.Error(w, "Failed to submit article", http.StatusInternalServerError)
		return
	}
	go services.ProcessNewArticle(article)
	// Respond with success (201 Created) and the created article object
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(article) // Encode the article struct directly to JSON
}

// @Summary Delete an article by ID
// @Description Deletes an article by its ID.
// @ID delete-article-by-id
// @Produce json
// @Param id path string true "Article ID"
// @Success 204 "Article deleted successfully"
// @Failure 400 {object} ErrorResponse "Invalid request payload or missing fields"
// @Failure 401 {object} ErrorResponse "Unauthorized: User ID not found"
// @Failure 403 {object} ErrorResponse "Forbidden: Article not owned by user"
// @Failure 404 {object} ErrorResponse "Article not found"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /articles/{id} [delete]
func DeleteArticle(w http.ResponseWriter, r *http.Request) {
	// Get the user ID from the context (set by AuthMiddleware)
	userID, ok := r.Context().Value(UserIDKey).(string)
	if !ok || userID == "" {
		log.Println("Unauthorized: User ID not found in context")
		http.Error(w, "Unauthorized: User ID not found", http.StatusUnauthorized)
		return
	}

	// Get Article ID from URL path parameter
	articleID := chi.URLParam(r, "id")
	if articleID == "" {
		http.Error(w, "Article ID is required", http.StatusBadRequest)
		return
	}

	// Call the model function to delete the article
	err := models.DeleteArticle(articleID, userID)
	if err != nil {
		log.Printf("Error deleting article with ID %s for user %s: %v", articleID, userID, err)
		if strings.Contains(err.Error(), "not found or not owned") {
			http.Error(w, "Article not found or not owned by user", http.StatusNotFound)
		} else {
			http.Error(w, "Failed to delete article", http.StatusInternalServerError)
		}
		return
	}

	w.WriteHeader(http.StatusNoContent) // 204 No Content for successful deletion
}

// @Summary Get an article by ID
// @Description Retrieves an article by its ID.
// @ID get-article-by-id
// @Produce json
// @Param id path string true "Article ID"
// @Success 200 {object} models.Article "Article retrieved successfully"
// @Failure 401 {object} ErrorResponse "Unauthorized: User ID not found"
// @Failure 404 {object} ErrorResponse "Article not found"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /articles/{id} [get]
// GetArticleByID retrieves an article by its ID and user ID
func ReturnArticle(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(UserIDKey).(string)
	if !ok || userID == "" {
		log.Println("Unauthorized: User ID not found in context")
		http.Error(w, "Unauthorized: User ID not found", http.StatusUnauthorized)
		return
	}
	//
	// Get Article ID from URL path parameter
	articleID := chi.URLParam(r, "id")
	if articleID == "" {
		http.Error(w, "Article ID is required", http.StatusBadRequest)
		return
	}

	// Fetch the article from the database
	article, err := models.GetArticleByID(articleID, userID)
	if err != nil {
		log.Printf("Error fetching article with ID %s: %v", userID, err)
		http.Error(w, "Failed to fetch article", http.StatusInternalServerError)
		return
	}
	// If the article is not found, return a 404 Not Found
	if article == nil {
		http.Error(w, "Article not found", http.StatusNotFound)
		return
	}
	// Respond with the article data
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(article) // Encode the article struct directly to JSON

}

// @Summary Get all articles for a user
// @Description Retrieves all articles associated with a user.
// @ID get-articles-by-user
// @Produce json
// @Param status query string false "Filter by article status (e.g., read, unread)"
// @Param tag query string false "Filter by article tag"
// @Success 200 {array} models.Article "List of articles"
// @Failure 401 {object} ErrorResponse "Unauthorized: User ID not found"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /articles [get]
func GetArticlesByUserID(w http.ResponseWriter, r *http.Request) {
	// This function returns all articles for a user from sqllite3 database
	userID, ok := r.Context().Value(UserIDKey).(string)
	if !ok || userID == "" {
		log.Println("Unauthorized: User ID not found in context")
		http.Error(w, "Unauthorized: User ID not found", http.StatusUnauthorized)
		return
	}

	statusFilter := r.URL.Query().Get("status") // Optional status filter
	tagFilter := r.URL.Query().Get("tag")       // Optional tag filter

	articles, err := models.GetArticlesByUserID(userID, statusFilter, tagFilter)
	if err != nil {
		log.Printf("Error fetching articles for user %s: %v", userID, err)
		http.Error(w, "Failed to fetch articles", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(articles) // Encode the articles directly to JSON
}

// @Summary Get all tags for a user
// @Description Retrieves all unique tags associated with articles for a user.
// @ID get-tags-by-user
// @Produce json
// @Success 200 {array} string "List of tags"
// @Failure 401 {object} ErrorResponse "Unauthorized: User ID not found"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /tags [get]
// GetTagsByUserID retrieves all unique tags for a user
func GetTagsByUserID(w http.ResponseWriter, r *http.Request) {
	// This function returns all tags for a user from sqllite3 database
	userID, ok := r.Context().Value(UserIDKey).(string)
	if !ok || userID == "" {
		log.Println("Unauthorized: User ID not found in context")
		http.Error(w, "Unauthorized: User ID not found", http.StatusUnauthorized)
		return
	}

	tags, err := models.GetTagsByUserID(userID)
	if err != nil {
		log.Printf("Error fetching tags for user %s: %v", userID, err)
		http.Error(w, "Failed to fetch tags for user", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tags) // Encode the tags directly to JSON
}

// UpdateArticleStatusRequest defines the payload for updating an article's status.
type UpdateArticleStatusRequest struct {
	Status string `json:"status" example:"read" enums:"read,unread,processing, failed"`
}

// @Summary Update an article's status
// @Description Updates the status of an existing article.
// @ID update-article-status
// @Accept json
// @Produce json
// @Param id path string true "Article ID"
// @Param status body UpdateArticleStatusRequest true "New status for the article"
// @Success 200 {object} MessageResponse "Status updated successfully"
// @Failure 400 {object} ErrorResponse "Invalid request payload or missing fields"
// @Failure 401 {object} ErrorResponse "Unauthorized: User ID not found"
// @Failure 403 {object} ErrorResponse "Forbidden: Article not owned by user"
// @Failure 404 {object} ErrorResponse "Article not found"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /articles/{id}/status [put]
// UpdateArticleStatus updates the status of an existing article
func UpdateArticleStatus(w http.ResponseWriter, r *http.Request) {
	// Get the user ID from the context (set by AuthMiddleware)
	userID, ok := r.Context().Value(UserIDKey).(string)
	if !ok {
		log.Println("Unauthorized: User ID not found in context")
		http.Error(w, "Unauthorized: User ID not found", http.StatusUnauthorized)
		return
	}

	// Get Article ID from URL path parameter
	articleID := chi.URLParam(r, "id")
	if articleID == "" {
		http.Error(w, "Article ID is required", http.StatusBadRequest)
		return
	}

	var req UpdateArticleStatusRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	if req.Status != "read" && req.Status != "unread" {
		http.Error(w, "Status must be 'processing', 'read' or 'unread'", http.StatusBadRequest)
		return
	}

	// Call the new model function to update the status
	err = models.UpdateArticleStatus(articleID, userID, req.Status)
	if err != nil {
		log.Printf("Error updating article status for user %s, article %s: %v", userID, articleID, err)
		// Check for the "not found" error from the model and return 404
		if strings.Contains(err.Error(), "not found or not owned") {
			http.Error(w, "Article not found or not owned by user", http.StatusNotFound)
		} else {
			http.Error(w, "Failed to update article status", http.StatusInternalServerError)
		}
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(MessageResponse{Message: "Status updated successfully"})

}

// UpdateArticleTagRequest defines the payload for updating an article's tags.
type UpdateArticleTagRequest struct {
	Tags []string `json:"tags" example:"[\"tag1\",\"tag2\"]"`
}

// @Summary Update an article's tags
// @Description Updates the tags of an existing article.
// @ID update-article-tags
// @Accept json
// @Produce json
// @Param id path string true "Article ID"
// @Param tags body UpdateArticleTagRequest true "New tags for the article"
// @Success 200 {object} MessageResponse "Tags updated successfully"
// @Failure 400 {object} ErrorResponse "Invalid request payload or missing fields"
// @Failure 401 {object} ErrorResponse "Unauthorized: User ID not found"
// @Failure 403 {object} ErrorResponse "Forbidden: Article not owned by user"
// @Failure 404 {object} ErrorResponse "Article not found"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /articles/{id}/tags [put]
func UpdateArticleTags(w http.ResponseWriter, r *http.Request) {
	// Get the user ID from the context (set by AuthMiddleware)
	userID, ok := r.Context().Value(UserIDKey).(string)
	if !ok {
		log.Println("Unauthorized: User ID not found in context")
		http.Error(w, "Unauthorized: User ID not found", http.StatusUnauthorized)
		return
	}

	// Get Article ID from URL path parameter
	articleID := chi.URLParam(r, "id")
	if articleID == "" {
		http.Error(w, "Article ID is required", http.StatusBadRequest)
		return
	}

	var req UpdateArticleTagRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	if len(req.Tags) == 0 {
		http.Error(w, "Tags cannot be empty", http.StatusBadRequest)
		return
	}

	// Call the new model function to update the tags
	err = models.UpdateArticleTags(articleID, userID, req.Tags)
	if err != nil {
		log.Printf("Error updating article tags for user %s, article %s: %v", userID, articleID, err)
		if strings.Contains(err.Error(), "not found or not owned") {
			http.Error(w, "Article not found or not owned by user", http.StatusNotFound)
		} else {
			http.Error(w, "Failed to update article tags", http.StatusInternalServerError)
		}
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(MessageResponse{Message: "Tags updated successfully"})
}

// A simple struct for a success message response
type MessageResponse struct {
	Message string `json:"message" example:"Success message"`
}
