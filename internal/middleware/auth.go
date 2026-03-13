package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/smap-hcmut/shared-libs/go/auth"
	"github.com/smap-hcmut/shared-libs/go/response"
	"github.com/smap-hcmut/shared-libs/go/scope"
)

func (m Middleware) Auth() gin.HandlerFunc {
	return func(c *gin.Context) {
		var tokenString string
		var err error

		// Priority 1: Try to read token from Authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader != "" {
			// Support both "Bearer <token>" and plain token
			if len(authHeader) > 7 && authHeader[:7] == "Bearer " {
				tokenString = authHeader[7:]
			} else {
				tokenString = authHeader
			}
		}

		// Priority 2: If no token in header (or it's just "Bearer "), try cookie
		if tokenString == "" || tokenString == "Bearer " {
			tokenString, err = c.Cookie(m.cookieConfig.Name)
			if err != nil || tokenString == "" {
				response.Unauthorized(c)
				c.Abort()
				return
			}
		}

		// Verify JWT token
		payload, err := m.jwtManager.Verify(tokenString)
		if err != nil {
			response.Unauthorized(c)
			c.Abort()
			return
		}

		// Set payload and scope in context for downstream handlers
		ctx := c.Request.Context()
		ctx = auth.SetPayloadToContext(ctx, payload)
		authScope := auth.NewScope(payload)
		// Convert auth.Scope to scope.Scope
		sc := scope.Scope{
			UserID:   authScope.UserID,
			Username: authScope.Username,
			Role:     authScope.Role,
			JTI:      payload.Id, // Use JWT ID from payload
		}
		ctx = scope.SetScopeToContext(ctx, sc)
		c.Request = c.Request.WithContext(ctx)

		c.Next()
	}
}

// InternalAuth validates the internal key from the Authorization header (Bearer <key> or raw key).
// If internalKey is empty, all requests are rejected with 401.
func (m Middleware) InternalAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			response.Unauthorized(c)
			c.Abort()
			return
		}
		// Support both "Bearer <key>" and raw key
		tokenString := authHeader
		if len(authHeader) > 7 && authHeader[:7] == "Bearer " {
			tokenString = authHeader[7:]
		}
		if m.internalKey == "" || tokenString != m.internalKey {
			response.Unauthorized(c)
			c.Abort()
			return
		}
		c.Next()
	}
}
