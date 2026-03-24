package http

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"

	"knowledge-srv/internal/model"
	"knowledge-srv/internal/notebook"

	"github.com/gin-gonic/gin"
	"github.com/smap-hcmut/shared-libs/go/log"
)

// NotebookHandler handles HTTP requests for the notebook domain.
type NotebookHandler struct {
	l             log.Logger
	notebookUC    notebook.UseCase
	webhookSecret string
}

// New creates a new NotebookHandler instance.
func New(l log.Logger, notebookUC notebook.UseCase, webhookSecret string) *NotebookHandler {
	return &NotebookHandler{
		l:             l,
		notebookUC:    notebookUC,
		webhookSecret: webhookSecret,
	}
}

// RegisterRoutes registers the notebook endpoints to the Gin router.
func (h *NotebookHandler) RegisterRoutes(r *gin.RouterGroup) {
	// Webhook endpoint (must be publicly accessible by Maestro)
	r.POST("/notebook/callback", h.HandleMaestroWebhook)
}

// HandleMaestroWebhook verifies and processes callbacks from Maestro.
func (h *NotebookHandler) HandleMaestroWebhook(c *gin.Context) {
	// 1. Verify Signature
	signature := c.GetHeader("X-Maestro-Signature")
	if signature == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing signature"})
		return
	}

	bodyBytes, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}

	if !h.verifySignature(bodyBytes, signature) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid signature"})
		return
	}

	// 2. Parse Payload
	var payload notebook.WebhookPayload
	if err := json.Unmarshal(bodyBytes, &payload); err != nil {
		h.l.Errorf(c.Request.Context(), "Failed to unmarshal maestro webhook payload: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload format"})
		return
	}

	// Inject internal chat_job_id from query params if present, so handleChatWebhook can use it
	chatJobID := c.Query("chat_job_id")
	if chatJobID != "" {
		if payload.Data == nil {
			payload.Data = make(map[string]interface{})
		}
		payload.Data["chat_job_id"] = chatJobID
	}

	// System scope since this is a background callback
	sc := model.Scope{}

	if err := h.notebookUC.HandleWebhook(c.Request.Context(), sc, payload); err != nil {
		h.l.Errorf(c.Request.Context(), "Failed to handle maestro webhook: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to process webhook"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (h *NotebookHandler) verifySignature(payload []byte, signature string) bool {
	mac := hmac.New(sha256.New, []byte(h.webhookSecret))
	mac.Write(payload)
	expectedMAC := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(signature), []byte(expectedMAC))
}
