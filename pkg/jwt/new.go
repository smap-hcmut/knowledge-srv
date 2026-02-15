package jwt

import (
	"fmt"
	"time"
)

// Config holds JWT manager configuration
type Config struct {
	SecretKey string
	Issuer    string
	Audience  []string
	TTL       time.Duration
}

// New creates a new JWT manager with HS256 symmetric key
func New(cfg Config) (*Manager, error) {
	// Validate secret key length (minimum 32 characters)
	if len(cfg.SecretKey) < 32 {
		return nil, fmt.Errorf("secret key must be at least 32 characters long, got %d", len(cfg.SecretKey))
	}

	// Create manager with secret key
	manager := &Manager{
		secretKey: []byte(cfg.SecretKey),
		issuer:    cfg.Issuer,
		audience:  cfg.Audience,
		ttl:       cfg.TTL,
	}

	return manager, nil
}
