package http

import (
	"github.com/gin-gonic/gin"

	"knowledge-srv/pkg/response"
)

// Index - Handler cho POST /internal/index/by-file
// @Summary Index batch từ MinIO file
// @Description Internal API cho Analytics Service trigger indexing
// @Tags Indexing (Internal)
// @Accept json
// @Produce json
// @Param body body IndexReq true "Index request"
// @Success 200 {object} IndexResp
// @Failure 400 {object} response.Resp
// @Failure 500 {object} response.Resp
// @Router /internal/index/by-file [post]
func (h *handler) Index(c *gin.Context) {
	ctx := c.Request.Context()

	req, err := h.processIndexReq(c)
	if err != nil {
		h.l.Errorf(ctx, "indexing.delivery.http.Index: processIndexRequest failed: %v", err)
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
