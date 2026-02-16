package http

import (
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

	return req, nil
}

func (h *handler) processRetryFailedReq(c *gin.Context) (RetryFailedReq, error) {
	var req RetryFailedReq

	ctx := c.Request.Context()
	if err := c.ShouldBindJSON(&req); err != nil {
		h.l.Errorf(ctx, "indexing.delivery.http.processRetryFailedReq: ShouldBindJSON failed: %v", err)
		return req, err
	}

	if req.Limit <= 0 {
		req.Limit = 100 // Default limit
	}

	return req, nil
}

func (h *handler) processReconcileReq(c *gin.Context) (ReconcileReq, error) {
	var req ReconcileReq

	ctx := c.Request.Context()
	if err := c.ShouldBindJSON(&req); err != nil {
		h.l.Errorf(ctx, "indexing.delivery.http.processReconcileReq: ShouldBindJSON failed: %v", err)
		return req, err
	}

	if req.StaleDurationMinutes <= 0 {
		req.StaleDurationMinutes = 30 // Default: 30 minutes
	}
	if req.Limit <= 0 {
		req.Limit = 100
	}

	return req, nil
}
