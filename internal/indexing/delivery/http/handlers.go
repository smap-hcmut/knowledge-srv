package http

import (
	"github.com/gin-gonic/gin"

	"knowledge-srv/pkg/response"
)

// IndexByFile - Handler cho POST /internal/index/by-file
// @Summary Index batch từ MinIO file
// @Description Internal API cho Analytics Service trigger indexing
// @Tags Indexing (Internal)
// @Accept json
// @Produce json
// @Param body body indexByFileReq true "Index request"
// @Success 200 {object} indexByFileResp
// @Failure 400 {object} response.Resp
// @Failure 500 {object} response.Resp
// @Router /internal/index/by-file [post]
func (h *handler) IndexByFile(c *gin.Context) {
	ctx := c.Request.Context()

	// Process request
	req, err := h.processIndexByFileRequest(c)
	if err != nil {
		h.l.Errorf(ctx, "processIndexByFileRequest failed: %v", err)
		response.Error(c, err, h.discord)
		return
	}

	// Convert to UseCase input
	input := req.toInput()

	// Call UseCase (scope already in context)
	output, err := h.uc.Index(ctx, input)
	if err != nil {
		h.l.Errorf(ctx, "Index failed: %v", err)
		response.Error(c, h.mapError(err), h.discord)
		return
	}

	// Return response
	resp := h.newIndexByFileResp(output)
	response.OK(c, resp)
}
