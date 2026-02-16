package http

import (
	"knowledge-srv/internal/model"
	"knowledge-srv/pkg/scope"

	"github.com/gin-gonic/gin"
)

func (h *handler) processSearchRequest(c *gin.Context) (searchReq, model.Scope, error) {
	var req searchReq

	if err := c.ShouldBindJSON(&req); err != nil {
		return req, model.Scope{}, err
	}

	sc := scope.GetScopeFromContext(c.Request.Context())
	return req, sc, nil
}
