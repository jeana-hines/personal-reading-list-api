// models/utils.go
package models

import (
	"fmt"
	"log"
	"time" // Added for the fallback, if you still want it

	"github.com/google/uuid"
)

// GenerateUUID generates a new UUID string.
func GenerateUUID() string {
	id, err := uuid.NewRandom() // Generates a Version 4 UUID
	if err != nil {
		log.Printf("Error generating UUID: %v. This is a serious issue and should be handled. Returning fallback.", err)
		// In a real production app, you might want to panic or return an error here
		// instead of a non-unique fallback if UUID generation is critical.
		// For development, this fallback is okay.
		return fmt.Sprintf("ERROR-UUID-%d", time.Now().UnixNano()) // More unique fallback
	}
	return id.String()
}
