package http

import (
	"knowledge-srv/pkg/response"

	"github.com/gin-gonic/gin"
)

// Search - Handler cho POST /api/v1/search
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
