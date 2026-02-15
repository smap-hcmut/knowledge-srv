package jwt

import (
	"fmt"
	"time"

	"identity-srv/pkg/scope"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// Manager handles JWT token generation and verification
type Manager struct {
	secretKey []byte
	issuer    string
	audience  []string
	ttl       time.Duration
}

// Claims represents JWT claims structure
type Claims struct {
	Email  string   `json:"email"`
	Role   string   `json:"role"`
	Groups []string `json:"groups,omitempty"`
	jwt.RegisteredClaims
}

// GenerateToken generates a new JWT token with HS256 algorithm
func (m *Manager) GenerateToken(userID, email, role string, groups []string) (string, error) {
	now := time.Now()
	expiresAt := now.Add(m.ttl)

	// Generate unique JTI (JWT ID) for token tracking and revocation
	jti := uuid.New().String()

	claims := Claims{
		Email:  email,
		Role:   role,
		Groups: groups,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    m.issuer,
			Subject:   userID,
			Audience:  m.audience,
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(now),
			ID:        jti,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// Sign token with secret key
	tokenString, err := token.SignedString(m.secretKey)
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}

	return tokenString, nil
}

// VerifyToken verifies and parses a JWT token
func (m *Manager) VerifyToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		// Verify signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}

		return m.secretKey, nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	if !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	claims, ok := token.Claims.(*Claims)
	if !ok {
		return nil, fmt.Errorf("invalid claims type")
	}

	return claims, nil
}

// SetConfig sets the issuer, audience, and TTL for the manager
func (m *Manager) SetConfig(issuer string, audience []string, ttl time.Duration) {
	m.issuer = issuer
	m.audience = audience
	m.ttl = ttl
}

// Verify implements scope.Manager interface — verifies HS256 token and returns scope.Payload.
func (m *Manager) Verify(token string) (scope.Payload, error) {
	claims, err := m.VerifyToken(token)
	if err != nil {
		return scope.Payload{}, err
	}

	var expiresAt int64
	if claims.ExpiresAt != nil {
		expiresAt = claims.ExpiresAt.Unix()
	}
	var issuedAt int64
	if claims.IssuedAt != nil {
		issuedAt = claims.IssuedAt.Unix()
	}

	p := scope.Payload{
		UserID:   claims.Subject,
		Username: claims.Email,
		Role:     claims.Role,
	}
	p.Subject = claims.Subject
	p.ExpiresAt = expiresAt
	p.IssuedAt = issuedAt
	p.Id = claims.ID
	p.Issuer = claims.Issuer
	return p, nil
}

// CreateToken implements scope.Manager interface.
func (m *Manager) CreateToken(_ scope.Payload) (string, error) {
	return "", fmt.Errorf("not implemented: use Manager.GenerateToken directly")
}
