package http

import (
	"github.com/gin-gonic/gin"
	"github.com/smap-hcmut/shared-libs/go/middleware"
)

func (h *handler) RegisterRoutes(r *gin.RouterGroup, mw *middleware.Middleware) {
	r.Use(mw.Auth())
	{
		r.GET("/reports", h.ListReports)
		r.POST("/reports/generate", h.GenerateReport)
		r.GET("/reports/:report_id", h.GetReport)
		r.GET("/reports/:report_id/process", h.GetReportProcess)
		r.GET("/reports/:report_id/posts", h.ListReportPosts)
		r.GET("/reports/:report_id/content", h.GetReportContent)
		r.GET("/reports/:report_id/download", h.DownloadReport)
		r.POST("/reports/:report_id/cancel", h.CancelReport)
		r.POST("/reports/:report_id/retry", h.RetryReport)
		r.GET("/reports/posts/:post_id/comments", h.ListPostComments)
	}
}
