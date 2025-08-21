package handlers

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/jeana-hines/personal-reading-list-api/config"
)

// ContextKey is a custom type for context keys to avoid collisions
type ContextKey string

const UserIDKey ContextKey = "userID"

// AuthMiddleware is a Chi middleware that validates JWTs and stores user ID in context
func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get the Authorization header from the request
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "Authorization header is required", http.StatusUnauthorized)
			return
		}

		// The header should be in the format "Bearer <token>"
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			http.Error(w, "Invalid Authorization header format", http.StatusUnauthorized)
			return
		}

		tokenString := parts[1]

		// Parse and validate the token
		claims := &Claims{}
		token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
			// Make sure the signing method is what we expect
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return config.JwtSecret, nil
		})

		if err != nil || !token.Valid {
			http.Error(w, "Invalid or expired token", http.StatusUnauthorized)
			return
		}

		// If the token is valid, get the user ID from the claims
		userID := claims.UserID

		// Store the user ID in the request's context
		ctx := context.WithValue(r.Context(), UserIDKey, userID)

		// Call the next handler in the chain with the new context
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
