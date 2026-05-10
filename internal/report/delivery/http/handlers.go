package http

import (
	"context"
	"errors"
	"knowledge-srv/internal/report"

	"github.com/gin-gonic/gin"
	"github.com/smap-hcmut/shared-libs/go/response"
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
// @Router /reports/generate [post]
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
		h.logUsecaseError(ctx, "report.delivery.http.GenerateReport: usecase Generate failed", err)
		response.Error(c, h.mapError(err), h.discord)
		return
	}

	response.OK(c, h.newGenerateReportResp(o))
}

// @Summary List reports
// @Description Return generated reports for a campaign
// @Tags Report
// @Produce json
// @Param campaign_id query string true "Campaign ID"
// @Param page query int false "Page"
// @Param page_size query int false "Page size"
// @Success 200 {object} listReportsResp
// @Failure 400 {object} response.Resp
// @Failure 500 {object} response.Resp
// @Router /reports [get]
func (h *handler) ListReports(c *gin.Context) {
	ctx := c.Request.Context()

	req, sc, err := h.processListReportsRequest(c)
	if err != nil {
		h.l.Errorf(ctx, "report.delivery.http.ListReports: processListReportsRequest failed: %v", err)
		response.Error(c, err, h.discord)
		return
	}

	o, err := h.uc.ListReports(ctx, sc, req.toInput())
	if err != nil {
		h.logUsecaseError(ctx, "report.delivery.http.ListReports: usecase ListReports failed", err)
		response.Error(c, h.mapError(err), h.discord)
		return
	}

	response.OK(c, h.newListReportsResp(o))
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
// @Router /reports/{report_id} [get]
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
		h.logUsecaseError(ctx, "report.delivery.http.GetReport: usecase GetReport failed", err)
		response.Error(c, h.mapError(err), h.discord)
		return
	}

	response.OK(c, h.newReportResp(o))
}

// @Summary Get report process
// @Description Return a report generation process status for UI polling
// @Tags Report
// @Produce json
// @Param report_id path string true "Report ID"
// @Success 200 {object} reportProcessResp
// @Failure 404 {object} response.Resp
// @Failure 500 {object} response.Resp
// @Router /reports/{report_id}/process [get]
func (h *handler) GetReportProcess(c *gin.Context) {
	ctx := c.Request.Context()

	req, sc, err := h.processGetReportProcessRequest(c)
	if err != nil {
		h.l.Errorf(ctx, "report.delivery.http.GetReportProcess: processGetReportProcessRequest failed: %v", err)
		response.Error(c, err, h.discord)
		return
	}

	o, err := h.uc.GetReportProcess(ctx, sc, req.toInput())
	if err != nil {
		h.logUsecaseError(ctx, "report.delivery.http.GetReportProcess: usecase GetReportProcess failed", err)
		response.Error(c, h.mapError(err), h.discord)
		return
	}

	response.OK(c, h.newReportProcessResp(o))
}

// @Summary List report evidence posts
// @Description Return indexed evidence posts used to review a report
// @Tags Report
// @Produce json
// @Param report_id path string true "Report ID"
// @Success 200 {object} listReportPostsResp
// @Failure 404 {object} response.Resp
// @Failure 500 {object} response.Resp
// @Router /reports/{report_id}/posts [get]
func (h *handler) ListReportPosts(c *gin.Context) {
	ctx := c.Request.Context()

	req, sc, err := h.processListReportPostsRequest(c)
	if err != nil {
		h.l.Errorf(ctx, "report.delivery.http.ListReportPosts: processListReportPostsRequest failed: %v", err)
		response.Error(c, err, h.discord)
		return
	}

	o, err := h.uc.ListReportPosts(ctx, sc, req.toInput())
	if err != nil {
		h.logUsecaseError(ctx, "report.delivery.http.ListReportPosts: usecase ListReportPosts failed", err)
		response.Error(c, h.mapError(err), h.discord)
		return
	}

	response.OK(c, h.newListReportPostsResp(o))
}

// @Summary List indexed comments for report evidence post
// @Description Return comments for an evidence post when available
// @Tags Report
// @Produce json
// @Param post_id path string true "Post ID"
// @Success 200 {object} listPostCommentsResp
// @Failure 500 {object} response.Resp
// @Router /reports/posts/{post_id}/comments [get]
func (h *handler) ListPostComments(c *gin.Context) {
	ctx := c.Request.Context()

	req, sc, err := h.processListPostCommentsRequest(c)
	if err != nil {
		h.l.Errorf(ctx, "report.delivery.http.ListPostComments: processListPostCommentsRequest failed: %v", err)
		response.Error(c, err, h.discord)
		return
	}

	o, err := h.uc.ListPostComments(ctx, sc, req.toInput())
	if err != nil {
		h.logUsecaseError(ctx, "report.delivery.http.ListPostComments: usecase ListPostComments failed", err)
		response.Error(c, h.mapError(err), h.discord)
		return
	}

	response.OK(c, h.newListPostCommentsResp(o))
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
// @Router /reports/{report_id}/download [get]
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
		h.logUsecaseError(ctx, "report.delivery.http.DownloadReport: usecase DownloadReport failed", err)
		response.Error(c, h.mapError(err), h.discord)
		return
	}

	response.OK(c, h.newDownloadResp(o))
}

// @Summary Get report markdown content
// @Description Return completed report artifact content for in-app rendering
// @Tags Report
// @Produce json
// @Param report_id path string true "Report ID"
// @Success 200 {object} reportContentResp
// @Failure 400 {object} response.Resp
// @Failure 404 {object} response.Resp
// @Failure 500 {object} response.Resp
// @Router /reports/{report_id}/content [get]
func (h *handler) GetReportContent(c *gin.Context) {
	ctx := c.Request.Context()

	req, sc, err := h.processDownloadReportRequest(c)
	if err != nil {
		h.l.Errorf(ctx, "report.delivery.http.GetReportContent: processDownloadReportRequest failed: %v", err)
		response.Error(c, err, h.discord)
		return
	}

	o, err := h.uc.GetReportContent(ctx, sc, report.GetReportContentInput{ReportID: req.ReportID})
	if err != nil {
		h.logUsecaseError(ctx, "report.delivery.http.GetReportContent: usecase GetReportContent failed", err)
		response.Error(c, h.mapError(err), h.discord)
		return
	}

	response.OK(c, h.newReportContentResp(o))
}

// @Summary Cancel report generation
// @Description Mark a processing report as cancelled
// @Tags Report
// @Produce json
// @Param report_id path string true "Report ID"
// @Success 200 {object} cancelResp
// @Failure 404 {object} response.Resp
// @Failure 500 {object} response.Resp
// @Router /reports/{report_id}/cancel [post]
func (h *handler) CancelReport(c *gin.Context) {
	ctx := c.Request.Context()

	req, sc, err := h.processCancelReportRequest(c)
	if err != nil {
		h.l.Errorf(ctx, "report.delivery.http.CancelReport: processCancelReportRequest failed: %v", err)
		response.Error(c, err, h.discord)
		return
	}

	o, err := h.uc.CancelReport(ctx, sc, req.toInput())
	if err != nil {
		h.logUsecaseError(ctx, "report.delivery.http.CancelReport: usecase CancelReport failed", err)
		response.Error(c, h.mapError(err), h.discord)
		return
	}

	response.OK(c, h.newCancelResp(o))
}

// @Summary Retry report generation
// @Description Retry a failed or cancelled report with the same stored parameters
// @Tags Report
// @Produce json
// @Param report_id path string true "Report ID"
// @Success 200 {object} retryResp
// @Failure 404 {object} response.Resp
// @Failure 500 {object} response.Resp
// @Router /reports/{report_id}/retry [post]
func (h *handler) RetryReport(c *gin.Context) {
	ctx := c.Request.Context()

	req, sc, err := h.processRetryReportRequest(c)
	if err != nil {
		h.l.Errorf(ctx, "report.delivery.http.RetryReport: processRetryReportRequest failed: %v", err)
		response.Error(c, err, h.discord)
		return
	}

	o, err := h.uc.RetryReport(ctx, sc, req.toInput())
	if err != nil {
		h.logUsecaseError(ctx, "report.delivery.http.RetryReport: usecase RetryReport failed", err)
		response.Error(c, h.mapError(err), h.discord)
		return
	}

	response.OK(c, h.newRetryResp(o))
}

func (h *handler) logUsecaseError(ctx context.Context, msg string, err error) {
	if errors.Is(err, report.ErrReportForbidden) ||
		errors.Is(err, report.ErrReportNotFound) ||
		errors.Is(err, report.ErrReportNotCompleted) ||
		errors.Is(err, report.ErrCampaignRequired) ||
		errors.Is(err, report.ErrInvalidReportType) ||
		errors.Is(err, report.ErrDuplicateProcessing) {
		h.l.Warnf(ctx, "%s: %v", msg, err)
		return
	}
	h.l.Errorf(ctx, "%s: %v", msg, err)
}
