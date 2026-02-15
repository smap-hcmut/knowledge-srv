package http

import (
	"github.com/gin-gonic/gin"

	"knowledge-srv/internal/model"
	"knowledge-srv/pkg/scope"
)

// processIndexByFileRequest - Validate + set scope to context
func (h *handler) processIndexByFileRequest(c *gin.Context) (indexByFileReq, error) {
	var req indexByFileReq

	// Bind JSON
	if err := c.ShouldBindJSON(&req); err != nil {
		return req, err
	}

	// Custom validate
	if err := req.validate(); err != nil {
		return req, err
	}

	// Extract scope from context or create default (service-to-service)
	sc := scope.GetScopeFromContext(c.Request.Context())
	if sc.UserID == "" {
		// Default scope for internal service calls
		sc = h.createDefaultScope()
	}

	// Set scope to context for downstream layers
	ctx := scope.SetScopeToContext(c.Request.Context(), sc)
	c.Request = c.Request.WithContext(ctx)

	return req, nil
}

// createDefaultScope - Create default scope for internal API calls
func (h *handler) createDefaultScope() model.Scope {
	return model.Scope{
		UserID:   "system",
		Username: "analytics-service",
		Role:     "SYSTEM",
	}
}
