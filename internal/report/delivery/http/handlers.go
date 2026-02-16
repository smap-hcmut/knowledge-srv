package http

import (
	"knowledge-srv/pkg/response"

	"github.com/gin-gonic/gin"
)

// @Summary Generate a new report
// @Description Create an async report from campaign data using Map-Reduce summarization
// @Tags Report
// @Accept json
// @Produce json
// @Param body body generateReportReq true "Report generation request"
// @Success 200 {object} generateReportResp
// @Failure 400 {object} response.Resp
// @Failure 500 {object} response.Resp
// @Router /api/v1/reports/generate [post]
func (h *handler) GenerateReport(c *gin.Context) {
	ctx := c.Request.Context()

	req, sc, err := h.processGenerateReportRequest(c)
	if err != nil {
		h.l.Errorf(ctx, "report.delivery.http.GenerateReport: processGenerateReportRequest failed: %v", err)
		response.Error(c, err, h.discord)
		return
	}

	o, err := h.uc.Generate(ctx, sc, req.toInput())
	if err != nil {
		h.l.Errorf(ctx, "report.delivery.http.GenerateReport: usecase Generate failed: %v", err)
		response.Error(c, h.mapError(err), h.discord)
		return
	}

	response.OK(c, h.newGenerateReportResp(o))
}

// @Summary Get report status and metadata
// @Description Return the current status and metadata of a report
// @Tags Report
// @Produce json
// @Param report_id path string true "Report ID"
// @Success 200 {object} reportResp
// @Failure 400 {object} response.Resp
// @Failure 404 {object} response.Resp
// @Failure 500 {object} response.Resp
// @Router /api/v1/reports/{report_id} [get]
func (h *handler) GetReport(c *gin.Context) {
	ctx := c.Request.Context()

	req, sc, err := h.processGetReportRequest(c)
	if err != nil {
		h.l.Errorf(ctx, "report.delivery.http.GetReport: processGetReportRequest failed: %v", err)
		response.Error(c, err, h.discord)
		return
	}

	o, err := h.uc.GetReport(ctx, sc, req.toInput())
	if err != nil {
		h.l.Errorf(ctx, "report.delivery.http.GetReport: usecase GetReport failed: %v", err)
		response.Error(c, h.mapError(err), h.discord)
		return
	}

	response.OK(c, h.newReportResp(o))
}

// @Summary Download report file
// @Description Generate a presigned download URL for a completed report
// @Tags Report
// @Produce json
// @Param report_id path string true "Report ID"
// @Success 200 {object} downloadResp
// @Failure 400 {object} response.Resp
// @Failure 404 {object} response.Resp
// @Failure 500 {object} response.Resp
// @Router /api/v1/reports/{report_id}/download [get]
func (h *handler) DownloadReport(c *gin.Context) {
	ctx := c.Request.Context()

	req, sc, err := h.processDownloadReportRequest(c)
	if err != nil {
		h.l.Errorf(ctx, "report.delivery.http.DownloadReport: processDownloadReportRequest failed: %v", err)
		response.Error(c, err, h.discord)
		return
	}

	o, err := h.uc.DownloadReport(ctx, sc, req.toInput())
	if err != nil {
		h.l.Errorf(ctx, "report.delivery.http.DownloadReport: usecase DownloadReport failed: %v", err)
		response.Error(c, h.mapError(err), h.discord)
		return
	}

	response.OK(c, h.newDownloadResp(o))
}
