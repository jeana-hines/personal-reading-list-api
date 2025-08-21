package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/jeana-hines/personal-reading-list-api/config"
	"github.com/jeana-hines/personal-reading-list-api/models" // Import your models package
)

// Define a struct for JWT claims
type Claims struct {
	UserID string `json:"user_id"`
	jwt.RegisteredClaims
}

// generateJWT creates a new JWT for a given user ID
func generateJWT(userID string) (string, error) {
	// Set the token expiration time to, for example, 24 hours
	expirationTime := time.Now().Add(24 * time.Hour)

	// Create the JWT claims, which includes the user ID and expiration time
	claims := &Claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	// Declare the token with the specified claims and signing method
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// Sign the token with our secret key
	tokenString, err := token.SignedString(config.JwtSecret)
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}

	return tokenString, nil
}

// Define a struct for the user registration request body
// This is what the client sends in the JSON payload
type RegisterUserRequest struct {
	Username string `json:"username" example:"testuser@example.com"`
	Password string `json:"password" example:"verysecurepassword"`
}

// Regex pattern for validating usernames
var emailRegex = regexp.MustCompile("^[a-zA-Z0-9.!#$%&'*+/=?^_`{|}~-]+@[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(?:\\.[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$")

// @Summary Register a new user
// @Description Creates a new user account with a unique email address and hashed password.
// @ID register-user
// @Accept json
// @Produce json
// @Param user body RegisterUserRequest true "User registration details"
// @Success 201 {object} models.User "User registered successfully"
// @Failure 400 {object} ErrorResponse "Invalid request payload or missing fields"
// @Failure 409 {object} ErrorResponse "Username already exists"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /auth/register [post]
func RegisterUser(w http.ResponseWriter, r *http.Request) {
	var req RegisterUserRequest
	// Decode the JSON request body into our struct
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		// Respond with a 400 Bad Request if the JSON is malformed
		log.Printf("Error decoding request body: %v", err)
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	// Basic validation (add more comprehensive validation later if needed)
	if req.Username == "" || req.Password == "" {
		http.Error(w, "Username and password are required", http.StatusBadRequest)
		return
	}

	// Validate the username format (email format)
	if !emailRegex.MatchString(req.Username) {
		http.Error(w, "Username must be a valid email address", http.StatusBadRequest)
		return
	}
	// Check if the username already exists
	_, err = models.GetUserByUsername(req.Username)
	if err == nil {
		http.Error(w, fmt.Sprintf("Username '%s' already exists", req.Username), http.StatusConflict) // 409 Conflict
		return
	}

	if err != sql.ErrNoRows {
		log.Printf("Error checking username existence: %v", err)
		http.Error(w, "Failed to check username", http.StatusInternalServerError)
		return
	}
	// Create a new User model instance
	user := &models.User{
		Username: req.Username,
	}

	// Hash the password
	err = user.HashPassword(req.Password)
	if err != nil {
		log.Printf("Error hashing password for user %s: %v", req.Username, err)
		http.Error(w, "Failed to process password", http.StatusInternalServerError)
		return
	}

	// Save the user to the database
	err = models.CreateUser(user)
	if err != nil {
		// Check if the error indicates a duplicate username
		if err.Error() == fmt.Sprintf("username '%s' already exists", req.Username) {
			http.Error(w, err.Error(), http.StatusConflict) // 409 Conflict
			return
		}
		log.Printf("Error creating user %s in database: %v", req.Username, err)
		http.Error(w, "Failed to register user", http.StatusInternalServerError)
		return
	}

	// Respond with success (201 Created) and the created user object (excluding password hash)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(user) // Encode the user struct directly to JSON
}

// @Summary Login a user
// @Description Authenticates a user with username and password.
// @ID login-user
// @Accept json
// @Produce json
// @Param user body LoginUserRequest true "User login details"
// @Success 200 {object} object{token:string} "User logged in successfully"
// @Failure 400 {object} ErrorResponse "Invalid request payload or missing fields"
// @Failure 401 {object} ErrorResponse "Invalid username or password"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /auth/login [post]
func LoginUser(w http.ResponseWriter, r *http.Request) {
	var req LoginUserRequest
	// Decode the JSON request body into our struct
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		// Respond with a 400 Bad Request if the JSON is malformed
		log.Printf("Error decoding request body: %v", err)
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	// Basic validation (add more comprehensive validation later if needed)
	if req.Username == "" || req.Password == "" {
		http.Error(w, "Username and password are required", http.StatusBadRequest)
		return
	}

	// Validate the username format (email format)
	if !emailRegex.MatchString(req.Username) {
		http.Error(w, "Username must be a valid email address", http.StatusBadRequest)
		return
	}

	// Authenticate the user
	user, err := models.AuthenticateUser(req.Username, req.Password)
	if err != nil {
		log.Printf("Authentication failed for user %s: %v", req.Username, err)
		http.Error(w, "Invalid username or password", http.StatusUnauthorized) // 401 Unauthorized
		return
	}

	// 1. Generate a JWT
	tokenString, err := generateJWT(user.ID)
	if err != nil {
		log.Printf("Error generating JWT for user %s: %v", user.Username, err)
		http.Error(w, "Failed to generate token", http.StatusInternalServerError)
		return
	}

	// 2. Send the token back to the client
	response := struct {
		Token string `json:"token"`
	}{
		Token: tokenString,
	}

	// Respond with success (200 OK) and the authenticated user object (excluding password hash)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response) // Encode the user struct directly to JSON
}

// @Summary Logout a user
// @Description Logs out a user by invalidating their JWT.
// @ID logout-user
// @Accept json
// @Produce json
// @Success 200 {string} string "User logged out successfully"
// @Failure 401 {object} ErrorResponse "Unauthorized - Invalid token format or claims"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /auth/logout [post]
func LogoutUser(w http.ResponseWriter, r *http.Request) {
	// Invalidate the JWT token
	// Get the Authorization header from the request
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		http.Error(w, "Authorization header is required", http.StatusUnauthorized)
		return
	}

	// Check if the header starts with "Bearer "
	if !strings.HasPrefix(authHeader, "Bearer ") {
		http.Error(w, "Invalid token format", http.StatusUnauthorized)
		return
	}

	// Extract the token string
	tokenString := strings.TrimPrefix(authHeader, "Bearer ")

	token, _, err := new(jwt.Parser).ParseUnverified(tokenString, jwt.MapClaims{})
	if err != nil {
		http.Error(w, "Invalid token format", http.StatusUnauthorized)
		return
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		http.Error(w, "Invalid token claims", http.StatusUnauthorized)
		return
	}

	expiresAt, ok := claims["exp"].(float64)
	if !ok {
		http.Error(w, "Token expiration time not found", http.StatusUnauthorized)
		return
	}

	expirationTime := time.Unix(int64(expiresAt), 0)

	// Call the function with the models package prefix
	err = models.RevokeToken(tokenString, expirationTime)
	if err != nil {
		http.Error(w, "Failed to revoke token", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Logged out successfully"))
}

// LoginUserRequest represents the request body for user login
type LoginUserRequest struct {
	Username string `json:"username" example:"testuser@example.com"`
	Password string `json:"password" example:"verysecurepassword"`
}

// ErrorResponse is a generic error response structure for Swagger documentation
type ErrorResponse struct {
	Message string `json:"message" example:"An error occurred"`
}
