package http

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// ChatStream serves the same chat answer over Server-Sent Events. The
// streaming wire format means the front-end can render tokens progressively
// instead of waiting for the full payload; the backend still produces the
// answer in one Generate call (the underlying LLM client gained a real
// stream method in shared-libs v1.0.13 but the chat usecase remains the
// source of truth for citations + persistence, so we tokenise the final
// answer to keep the public contract stable).
//
// Event types emitted (in order):
//   - "meta"     – conversation_id, citations, suggestions, search_meta
//   - "token"    – plain-text answer fragments
//   - "done"     – sentinel so the client can stop reading
func (h *handler) ChatStream(c *gin.Context) {
	ctx := c.Request.Context()

	req, sc, err := h.processChatRequest(c)
	if err != nil {
		h.respondChatError(c, "chat.delivery.http.ChatStream: processChatRequest failed", err)
		return
	}

	out, err := h.uc.Chat(ctx, sc, req.toInput())
	if err != nil {
		h.respondChatError(c, "chat.delivery.http.ChatStream: usecase Chat failed", err)
		return
	}

	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Header().Set("X-Accel-Buffering", "no")

	flusher, ok := c.Writer.(interface{ Flush() })
	if !ok {
		h.l.Errorf(ctx, "chat.delivery.http.ChatStream: writer does not support flushing")
		return
	}

	resp := h.newChatResp(out)
	metaPayload, _ := json.Marshal(map[string]interface{}{
		"conversation_id": resp.ConversationID,
		"citations":       resp.Citations,
		"suggestions":     resp.Suggestions,
		"search_metadata": resp.SearchMetadata,
	})
	writeSSE(c, "meta", string(metaPayload))
	flusher.Flush()

	// Stream the answer in word-sized chunks so the client renders an
	// incremental UX. Cap chunk pacing so the whole stream finishes within
	// ~1s for short answers; long answers still flush quickly because
	// chunk size grows with answer length.
	words := strings.Fields(resp.Answer)
	if len(words) == 0 {
		writeSSE(c, "done", "{}")
		flusher.Flush()
		return
	}
	chunkSize := 1 + len(words)/120
	if chunkSize > 6 {
		chunkSize = 6
	}
	delay := 20 * time.Millisecond
	for start := 0; start < len(words); start += chunkSize {
		end := start + chunkSize
		if end > len(words) {
			end = len(words)
		}
		fragment := strings.Join(words[start:end], " ")
		if end < len(words) {
			fragment += " "
		}
		payload, _ := json.Marshal(map[string]string{"text": fragment})
		writeSSE(c, "token", string(payload))
		flusher.Flush()
		select {
		case <-ctx.Done():
			return
		case <-time.After(delay):
		}
	}
	writeSSE(c, "done", "{}")
	flusher.Flush()
}

func writeSSE(c *gin.Context, event, data string) {
	fmt.Fprintf(c.Writer, "event: %s\ndata: %s\n\n", event, data)
}
