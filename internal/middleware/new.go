package middleware

import (
	"os"
	"knowledge-srv/config"

	"github.com/gin-gonic/gin"
	"github.com/smap-hcmut/shared-libs/go/auth"
	"github.com/smap-hcmut/shared-libs/go/log"
)

// Encrypter interface for password/key encryption
type Encrypter interface {
	Decrypt(encrypted string) (string, error)
}

// Middleware wraps shared-libs auth.Middleware and holds service-specific dependencies
type Middleware struct {
	authMiddleware *auth.Middleware
	l              log.Logger
	cookieConfig   config.CookieConfig
	internalKey    string
	encrypter      Encrypter
}

// New creates a new middleware instance
func New(logger log.Logger, jwtManager auth.Manager, cookieConfig config.CookieConfig, internalKey string, encrypter Encrypter) Middleware {
	// Create shared-libs auth middleware
	authMiddleware := auth.NewMiddleware(auth.MiddlewareConfig{
		Manager:                 jwtManager,
		CookieName:              cookieConfig.Name,
		AllowBearerInProduction: os.Getenv("ENVIRONMENT_NAME") != "production",
		ProductionDomain:        cookieConfig.Domain,
	})

	return Middleware{
		authMiddleware: authMiddleware,
		l:              logger,
		cookieConfig:   cookieConfig,
		internalKey:    internalKey,
		encrypter:      encrypter,
	}
}

// Auth returns the Gin authentication middleware
func (m Middleware) Auth() gin.HandlerFunc {
	return m.authMiddleware.Authenticate()
}
