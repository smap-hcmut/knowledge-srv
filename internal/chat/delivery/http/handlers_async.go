package http

import (
	"net/http"

	"knowledge-srv/internal/chat"
	"knowledge-srv/internal/model"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"github.com/smap-hcmut/shared-libs/go/auth"
)

type AsyncChatHandler struct {
	chatUC chat.UseCase
	logger *logrus.Logger
}

// NewAsyncChatHandler creates a new handler for async chat endpoints.
func NewAsyncChatHandler(chatUC chat.UseCase, logger *logrus.Logger) *AsyncChatHandler {
	return &AsyncChatHandler{
		chatUC: chatUC,
		logger: logger,
	}
}

// GetChatJobStatus handles GET requests to check the status of an async chat job.
func (h *AsyncChatHandler) GetChatJobStatus(c *gin.Context) {
	jobID := c.Param("chat_job_id")
	if jobID == "" {
		h.logger.Warn("Missing chat_job_id in request")
		c.JSON(http.StatusBadRequest, gin.H{"error": "chat_job_id is required"})
		return
	}

	sc := auth.GetScopeFromContext(c.Request.Context())
	status, err := h.chatUC.GetChatJobStatus(c.Request.Context(), model.ToScope(sc), jobID)
	if err != nil {
		h.logger.Errorf("Failed to get chat job status: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve job status"})
		return
	}

	switch status.State {
	case chat.JobPending, chat.JobProcessing:
		c.JSON(http.StatusAccepted, gin.H{
			"status":  status.State,
			"message": "Job is still processing",
		})
	case chat.JobCompleted:
		c.JSON(http.StatusOK, gin.H{
			"status":  status.State,
			"answer":  status.Answer,
			"backend": status.Backend,
		})
	case chat.JobExpired:
		c.JSON(http.StatusGone, gin.H{
			"status":  status.State,
			"message": "Job has expired",
		})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  status.State,
			"message": "Unknown job status",
		})
	}
}
