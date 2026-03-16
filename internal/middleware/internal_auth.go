package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/smap-hcmut/shared-libs/go/auth"
)

// InternalAuth validates the raw X-Internal-Key header for internal routes.
func (m Middleware) InternalAuth() gin.HandlerFunc {
	return auth.InternalAuth(auth.InternalAuthConfig{ExpectedKey: m.internalKey})
}
