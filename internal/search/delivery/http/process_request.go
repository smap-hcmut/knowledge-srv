package http

import (
	"knowledge-srv/internal/model"

	"github.com/gin-gonic/gin"
	"github.com/smap-hcmut/shared-libs/go/auth"
)

func (h *handler) processSearchRequest(c *gin.Context) (searchReq, model.Scope, error) {
	var req searchReq

	if err := c.ShouldBindJSON(&req); err != nil {
		return req, model.Scope{}, err
	}

	sc := auth.GetScopeFromContext(c.Request.Context())
	return req, model.ToScope(sc), nil
}
