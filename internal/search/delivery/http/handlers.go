package http

import (
	"knowledge-srv/pkg/response"

	"github.com/gin-gonic/gin"
)

// Search - Search for analytics posts with filters
// @Summary Search analytics posts
// @Description Search for analytics posts by query with optional filters (sentiments, aspects, platforms, dates, risk levels)
// @Tags Search
// @Accept json
// @Produce json
// @Param body body searchReq true "Search request"
// @Success 200 {object} searchResp
// @Failure 400 {object} response.Resp
// @Failure 500 {object} response.Resp
// @Router /api/v1/search [post]
func (h *handler) Search(c *gin.Context) {
	ctx := c.Request.Context()

	// 1. Process request
	req, sc, err := h.processSearchRequest(c)
	if err != nil {
		h.l.Errorf(ctx, "search.delivery.http.Search: processSearchRequest failed: %v", err)
		response.Error(c, err, h.discord)
		return
	}

	// 2. Convert to UseCase input
	input := req.toInput()

	// 3. Call UseCase
	output, err := h.uc.Search(ctx, sc, input)
	if err != nil {
		h.l.Errorf(ctx, "search.delivery.http.Search: usecase Search failed: %v", err)
		response.Error(c, h.mapError(err), h.discord)
		return
	}

	// 4. Return response
	resp := h.newSearchResp(output)
	response.OK(c, resp)
}
