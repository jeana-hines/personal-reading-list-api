// config/config.go
package config

// JwtSecret is a secret key for signing JWTs.
// In a real application, this should be loaded from a secure environment variable.
var JwtSecret = []byte("your-highly-secret-and-random-key")
