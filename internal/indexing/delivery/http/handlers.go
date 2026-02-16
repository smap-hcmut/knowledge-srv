package http

import (
	"github.com/gin-gonic/gin"

	"knowledge-srv/pkg/response"
)

// Index - Handler cho POST /internal/index
// @Summary Index batch từ MinIO file
// @Description Internal API cho Analytics Service trigger indexing
// @Tags Indexing (Internal)
// @Accept json
// @Produce json
// @Param body body IndexReq true "Index request"
// @Success 200 {object} IndexResp
// @Failure 400 {object} response.Resp
// @Failure 500 {object} response.Resp
// @Router /internal/index [post]
func (h *handler) Index(c *gin.Context) {
	ctx := c.Request.Context()

	req, err := h.processIndexReq(c)
	if err != nil {
		h.l.Errorf(ctx, "indexing.delivery.http.Index: processIndexReq failed: %v", err)
		response.Error(c, err, h.discord)
		return
	}

	o, err := h.uc.Index(ctx, req.toInput())
	if err != nil {
		h.l.Errorf(ctx, "indexing.delivery.http.Index: Index failed: %v", err)
		response.Error(c, h.mapError(err), h.discord)
		return
	}

	response.OK(c, h.newIndexResp(o))
}

// RetryFailed - Handler cho POST /internal/index/retry
// @Summary Retry failed indexing records
// @Description Retry indexing for records that previously failed
// @Tags Indexing (Internal)
// @Accept json
// @Produce json
// @Param body body RetryFailedReq true "Retry request"
// @Success 200 {object} RetryFailedResp
// @Failure 400 {object} response.Resp
// @Failure 500 {object} response.Resp
// @Router /internal/index/retry [post]
func (h *handler) RetryFailed(c *gin.Context) {
	ctx := c.Request.Context()

	req, err := h.processRetryFailedReq(c)
	if err != nil {
		h.l.Errorf(ctx, "indexing.delivery.http.RetryFailed: processRetryFailedReq failed: %v", err)
		response.Error(c, err, h.discord)
		return
	}

	o, err := h.uc.RetryFailed(ctx, req.toInput())
	if err != nil {
		h.l.Errorf(ctx, "indexing.delivery.http.RetryFailed: RetryFailed failed: %v", err)
		response.Error(c, h.mapError(err), h.discord)
		return
	}

	response.OK(c, h.newRetryFailedResp(o))
}

// Reconcile - Handler cho POST /internal/index/reconcile
// @Summary Reconcile stale pending records
// @Description Reconcile records that have been in PENDING status for too long
// @Tags Indexing (Internal)
// @Accept json
// @Produce json
// @Param body body ReconcileReq true "Reconcile request"
// @Success 200 {object} ReconcileResp
// @Failure 400 {object} response.Resp
// @Failure 500 {object} response.Resp
// @Router /internal/index/reconcile [post]
func (h *handler) Reconcile(c *gin.Context) {
	ctx := c.Request.Context()

	req, err := h.processReconcileReq(c)
	if err != nil {
		h.l.Errorf(ctx, "indexing.delivery.http.Reconcile: processReconcileReq failed: %v", err)
		response.Error(c, err, h.discord)
		return
	}

	o, err := h.uc.Reconcile(ctx, req.toInput())
	if err != nil {
		h.l.Errorf(ctx, "indexing.delivery.http.Reconcile: Reconcile failed: %v", err)
		response.Error(c, h.mapError(err), h.discord)
		return
	}

	response.OK(c, h.newReconcileResp(o))
}

// GetStatistics - Handler cho GET /internal/index/statistics/:project_id
// @Summary Get indexing statistics for a project
// @Description Get indexing statistics including total indexed, failed, pending records for a project
// @Tags Indexing (Internal)
// @Produce json
// @Param project_id path string true "Project ID"
// @Success 200 {object} StatisticsResp
// @Failure 400 {object} response.Resp
// @Failure 500 {object} response.Resp
// @Router /internal/index/statistics/{project_id} [get]
func (h *handler) GetStatistics(c *gin.Context) {
	ctx := c.Request.Context()

	projectID := c.Param("project_id")
	if projectID == "" {
		h.l.Errorf(ctx, "indexing.delivery.http.GetStatistics: missing project_id parameter")
		response.Error(c, ErrMissingProjectID, h.discord)
		return
	}

	o, err := h.uc.GetStatistics(ctx, projectID)
	if err != nil {
		h.l.Errorf(ctx, "indexing.delivery.http.GetStatistics: GetStatistics failed: %v", err)
		response.Error(c, h.mapError(err), h.discord)
		return
	}

	response.OK(c, h.newStatisticsResp(o))
}
