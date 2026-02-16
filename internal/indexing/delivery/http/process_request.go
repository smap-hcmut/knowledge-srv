package http

import (
	"errors"

	"knowledge-srv/pkg/scope"

	"github.com/gin-gonic/gin"
)

func (h *handler) processIndexReq(c *gin.Context) (IndexReq, error) {
	var req IndexReq

	ctx := c.Request.Context()
	if err := c.ShouldBindJSON(&req); err != nil {
		h.l.Errorf(ctx, "indexing.delivery.http.processIndexReq: ShouldBindJSON failed: %v", err)
		return req, err
	}

	if err := req.validate(); err != nil {
		h.l.Errorf(ctx, "indexing.delivery.http.processIndexReq: validate failed: %v", err)
		return req, err
	}

	sc := scope.GetScopeFromContext(ctx)
	if sc.UserID == "" {
		h.l.Errorf(ctx, "indexing.delivery.http.processIndexReq: GetScopeFromContext failed: scope not found")
		return req, errors.New("scope not found")
	}

	return req, nil
}
