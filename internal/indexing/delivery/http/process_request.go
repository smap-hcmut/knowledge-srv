package http

import (
	"errors"

	"github.com/gin-gonic/gin"

	"knowledge-srv/pkg/scope"
)

func (h *handler) processIndexReq(c *gin.Context) (IndexReq, error) {
	var req IndexReq

	if err := c.ShouldBindJSON(&req); err != nil {
		h.l.Errorf(c, "indexing.delivery.http.processIndexReq: ShouldBindJSON failed: %v", err)
		return req, err
	}

	if err := req.validate(); err != nil {
		h.l.Errorf(c, "indexing.delivery.http.processIndexReq: validate failed: %v", err)
		return req, err
	}

	sc := scope.GetScopeFromContext(c.Request.Context())
	if sc.UserID == "" {
		h.l.Errorf(c, "indexing.delivery.http.processIndexReq: GetScopeFromContext failed: scope not found")
		return req, errors.New("scope not found")
	}

	return req, nil
}
