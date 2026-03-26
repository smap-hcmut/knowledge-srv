package http

import (
	"net/http"

	"knowledge-srv/internal/chat"
	"knowledge-srv/internal/model"

	"github.com/gin-gonic/gin"
	"github.com/smap-hcmut/shared-libs/go/auth"
	"github.com/smap-hcmut/shared-libs/go/response"
)

// @Summary Chat with knowledge service
// @Description Send a message and receive an answer with citations
// @Tags Chat
// @Accept json
// @Produce json
// @Param body body chatReq true "Chat request"
// @Success 200 {object} chatResp
// @Failure 400 {object} response.Resp
// @Failure 500 {object} response.Resp
// @Router /api/v1/chat [post]
func (h *handler) Chat(c *gin.Context) {
	ctx := c.Request.Context()

	req, sc, err := h.processChatRequest(c)
	if err != nil {
		h.l.Errorf(ctx, "chat.delivery.http.Chat: processChatRequest failed: %v", err)
		response.Error(c, err, h.discord)
		return
	}

	o, err := h.uc.Chat(ctx, sc, req.toInput())
	if err != nil {
		h.l.Errorf(ctx, "chat.delivery.http.Chat: usecase Chat failed: %v", err)
		response.Error(c, h.mapError(err), h.discord)
		return
	}

	if o.IsAsync {
		c.JSON(http.StatusAccepted, gin.H{
			"status":           "ACCEPTED",
			"conversation_id":  o.ConversationID,
			"chat_job_id":      o.ChatJobID,
			"query_intent":     o.QueryIntent,
			"backend":          o.Backend,
			"poll_jobs_path":   "GET /api/v1/knowledge/chat/jobs/:job_id",
		})
		return
	}

	response.OK(c, h.newChatResp(o))
}

// GetChatJob returns status of an async NotebookLM chat job (or Qdrant fallback result).
func (h *handler) GetChatJob(c *gin.Context) {
	ctx := c.Request.Context()
	jobID := c.Param("job_id")
	if jobID == "" {
		response.Error(c, errJobIDRequired, h.discord)
		return
	}

	sc := model.ToScope(auth.GetScopeFromContext(c.Request.Context()))
	st, err := h.uc.GetChatJobStatus(ctx, sc, jobID)
	if err != nil {
		h.l.Errorf(ctx, "chat.delivery.http.GetChatJob: %v", err)
		response.Error(c, h.mapError(err), h.discord)
		return
	}

	switch st.State {
	case chat.JobPending, chat.JobProcessing:
		c.JSON(http.StatusAccepted, gin.H{
			"status":  st.State,
			"backend": st.Backend,
			"message": "Job is still processing",
		})
	case chat.JobCompleted:
		c.JSON(http.StatusOK, gin.H{
			"status":  st.State,
			"answer":  st.Answer,
			"backend": st.Backend,
		})
	case chat.JobFailed:
		c.JSON(http.StatusOK, gin.H{
			"status":  st.State,
			"backend": st.Backend,
			"message": "Notebook job failed",
		})
	case chat.JobExpired:
		c.JSON(http.StatusGone, gin.H{
			"status":  st.State,
			"message": "Job has expired",
		})
	default:
		c.JSON(http.StatusOK, gin.H{"status": st.State, "backend": st.Backend})
	}
}

// @Summary Get conversation detail
// @Description Return full conversation info and messages
// @Tags Chat
// @Produce json
// @Param conversation_id path string true "Conversation ID"
// @Success 200 {object} conversationResp
// @Failure 400 {object} response.Resp
// @Failure 500 {object} response.Resp
// @Router /api/v1/conversations/{conversation_id} [get]
func (h *handler) GetConversation(c *gin.Context) {
	ctx := c.Request.Context()

	req, sc, err := h.processGetConversationRequest(c)
	if err != nil {
		h.l.Errorf(ctx, "chat.delivery.http.GetConversation: processGetConversationRequest failed: %v", err)
		response.Error(c, err, h.discord)
		return
	}

	o, err := h.uc.GetConversation(ctx, sc, req.toInput())
	if err != nil {
		h.l.Errorf(ctx, "chat.delivery.http.GetConversation: usecase GetConversation failed: %v", err)
		response.Error(c, h.mapError(err), h.discord)
		return
	}

	response.OK(c, h.newConversationResp(o))
}

// @Summary List conversations by campaign
// @Description Paginate conversations for a given campaign
// @Tags Chat
// @Produce json
// @Param campaign_id path string true "Campaign ID"
// @Param limit query int false "Number of records per page (default 20)"
// @Param offset query int false "Number of records to skip (default 0)"
// @Success 200 {array} conversationResp
// @Failure 400 {object} response.Resp
// @Failure 500 {object} response.Resp
// @Router /api/v1/campaigns/{campaign_id}/conversations [get]
func (h *handler) ListConversations(c *gin.Context) {
	ctx := c.Request.Context()

	req, sc, err := h.processListConversationsRequest(c)
	if err != nil {
		h.l.Errorf(ctx, "chat.delivery.http.ListConversations: processListConversationsRequest failed: %v", err)
		response.Error(c, err, h.discord)
		return
	}

	o, err := h.uc.ListConversations(ctx, sc, req.toInput())
	if err != nil {
		h.l.Errorf(ctx, "chat.delivery.http.ListConversations: usecase ListConversations failed: %v", err)
		response.Error(c, h.mapError(err), h.discord)
		return
	}

	response.OK(c, h.newListConversationsResp(o))
}

// @Summary Get smart suggestions
// @Description Return a list of suggested queries for a campaign
// @Tags Chat
// @Produce json
// @Param campaign_id path string true "Campaign ID"
// @Success 200 {object} suggestionsResp
// @Failure 400 {object} response.Resp
// @Failure 500 {object} response.Resp
// @Router /api/v1/campaigns/{campaign_id}/suggestions [get]
func (h *handler) GetSuggestions(c *gin.Context) {
	ctx := c.Request.Context()

	req, sc, err := h.processGetSuggestionsRequest(c)
	if err != nil {
		h.l.Errorf(ctx, "chat.delivery.http.GetSuggestions: processGetSuggestionsRequest failed: %v", err)
		response.Error(c, err, h.discord)
		return
	}

	o, err := h.uc.GetSuggestions(ctx, sc, req.toInput())
	if err != nil {
		h.l.Errorf(ctx, "chat.delivery.http.GetSuggestions: usecase GetSuggestions failed: %v", err)
		response.Error(c, h.mapError(err), h.discord)
		return
	}

	response.OK(c, h.newSuggestionsResp(o))
}
