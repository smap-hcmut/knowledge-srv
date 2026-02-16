package http

import (
	"knowledge-srv/internal/model"
	"knowledge-srv/pkg/scope"

	"github.com/gin-gonic/gin"
)

func (h *handler) processGenerateReportRequest(c *gin.Context) (generateReportReq, model.Scope, error) {
	var req generateReportReq

	ctx := c.Request.Context()
	if err := c.ShouldBindJSON(&req); err != nil {
		h.l.Errorf(ctx, "report.delivery.http.processGenerateReportRequest: ShouldBindJSON failed: %v", err)
		return req, model.Scope{}, err
	}

	sc := scope.GetScopeFromContext(c.Request.Context())
	return req, sc, nil
}

func (h *handler) processGetReportRequest(c *gin.Context) (getReportReq, model.Scope, error) {
	req := getReportReq{
		ReportID: c.Param("report_id"),
	}

	sc := scope.GetScopeFromContext(c.Request.Context())
	return req, sc, nil
}

func (h *handler) processDownloadReportRequest(c *gin.Context) (downloadReportReq, model.Scope, error) {
	req := downloadReportReq{
		ReportID: c.Param("report_id"),
	}

	sc := scope.GetScopeFromContext(c.Request.Context())
	return req, sc, nil
}
