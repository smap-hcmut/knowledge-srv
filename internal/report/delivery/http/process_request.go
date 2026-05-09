package http

import (
	"knowledge-srv/internal/model"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/smap-hcmut/shared-libs/go/auth"
)

func (h *handler) processGenerateReportRequest(c *gin.Context) (generateReportReq, model.Scope, error) {
	var req generateReportReq

	ctx := c.Request.Context()
	if err := c.ShouldBindJSON(&req); err != nil {
		h.l.Errorf(ctx, "report.delivery.http.processGenerateReportRequest: ShouldBindJSON failed: %v", err)
		return req, model.Scope{}, err
	}

	sc := auth.GetScopeFromContext(c.Request.Context())
	return req, model.ToScope(sc), nil
}

func (h *handler) processListReportsRequest(c *gin.Context) (listReportsReq, model.Scope, error) {
	req := listReportsReq{
		CampaignID: c.Query("campaign_id"),
		Status:     c.Query("status"),
		Page:       queryInt(c, "page", 1),
		PageSize:   queryInt(c, "page_size", 20),
	}

	sc := auth.GetScopeFromContext(c.Request.Context())
	return req, model.ToScope(sc), nil
}

func (h *handler) processGetReportRequest(c *gin.Context) (getReportReq, model.Scope, error) {
	req := getReportReq{
		ReportID: c.Param("report_id"),
	}

	sc := auth.GetScopeFromContext(c.Request.Context())
	return req, model.ToScope(sc), nil
}

func (h *handler) processGetReportProcessRequest(c *gin.Context) (getReportProcessReq, model.Scope, error) {
	req := getReportProcessReq{
		ReportID: c.Param("report_id"),
	}

	sc := auth.GetScopeFromContext(c.Request.Context())
	return req, model.ToScope(sc), nil
}

func (h *handler) processListReportPostsRequest(c *gin.Context) (listReportPostsReq, model.Scope, error) {
	req := listReportPostsReq{
		ReportID:  c.Param("report_id"),
		Page:      queryInt(c, "page", 1),
		PageSize:  queryInt(c, "page_size", 20),
		Sentiment: c.Query("sentiment"),
		Platform:  c.Query("platform"),
	}

	sc := auth.GetScopeFromContext(c.Request.Context())
	return req, model.ToScope(sc), nil
}

func (h *handler) processListPostCommentsRequest(c *gin.Context) (listPostCommentsReq, model.Scope, error) {
	req := listPostCommentsReq{
		PostID:   c.Param("post_id"),
		Page:     queryInt(c, "page", 1),
		PageSize: queryInt(c, "page_size", 20),
	}

	sc := auth.GetScopeFromContext(c.Request.Context())
	return req, model.ToScope(sc), nil
}

func (h *handler) processDownloadReportRequest(c *gin.Context) (downloadReportReq, model.Scope, error) {
	req := downloadReportReq{
		ReportID: c.Param("report_id"),
	}

	sc := auth.GetScopeFromContext(c.Request.Context())
	return req, model.ToScope(sc), nil
}

func (h *handler) processCancelReportRequest(c *gin.Context) (cancelReportReq, model.Scope, error) {
	req := cancelReportReq{
		ReportID: c.Param("report_id"),
	}

	sc := auth.GetScopeFromContext(c.Request.Context())
	return req, model.ToScope(sc), nil
}

func (h *handler) processRetryReportRequest(c *gin.Context) (retryReportReq, model.Scope, error) {
	req := retryReportReq{
		ReportID: c.Param("report_id"),
	}

	sc := auth.GetScopeFromContext(c.Request.Context())
	return req, model.ToScope(sc), nil
}

func queryInt(c *gin.Context, key string, fallback int) int {
	raw := c.Query(key)
	if raw == "" {
		return fallback
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	return value
}
