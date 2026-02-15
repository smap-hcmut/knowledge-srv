package jwt

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Config holds JWT manager configuration.
type Config struct {
	SecretKey string
	Issuer    string
	Audience  []string
	TTL       time.Duration
}

// managerImpl implements IManager.
type managerImpl struct {
	secretKey []byte
	issuer    string
	audience  []string
	ttl       time.Duration
}

// Claims represents JWT claims structure.
type Claims struct {
	Email  string   `json:"email"`
	Role   string   `json:"role"`
	Groups []string `json:"groups,omitempty"`
	jwt.RegisteredClaims
}
