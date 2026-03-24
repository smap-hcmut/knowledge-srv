package maestro

import (
	"time"

	pkghttp "github.com/smap-hcmut/shared-libs/go/httpclient"
)

// MaestroConfig holds configuration for the Maestro client.
type MaestroConfig struct {
	BaseURL    string
	APIKey     string
	HTTPClient pkghttp.Client // Optional; uses default if nil
}

// maestroImpl implements IMaestro using the Maestro HTTP API.
type maestroImpl struct {
	baseURL    string
	apiKey     string
	httpClient pkghttp.Client
}

// CreateSessionReq is the request body for POST /notebooklm/sessions.
type CreateSessionReq struct {
	Env           string `json:"env"` // "LOCAL" | "BROWSERBASE"
	BrowserConfig any    `json:"browserConfig,omitempty"`
}

// CreateNotebookReq is the request body for POST /notebooklm/notebooks.
type CreateNotebookReq struct {
	Title         string `json:"title"`
	WebhookURL    string `json:"webhookUrl,omitempty"`
	WebhookSecret string `json:"webhookSecret,omitempty"`
}

// UploadSourcesReq is the request body for POST /notebooklm/notebooks/{id}/sources.
type UploadSourcesReq struct {
	SourceType    string       `json:"sourceType"` // "text" | "url" | "file"
	Sources       []SourceItem `json:"sources"`
	WebhookURL    string       `json:"webhookUrl,omitempty"`
	WebhookSecret string       `json:"webhookSecret,omitempty"`
}

// SourceItem represents a single source to upload.
type SourceItem struct {
	Title   string `json:"title,omitempty"`
	Content string `json:"content,omitempty"`
	URL     string `json:"url,omitempty"`
}

// ChatNotebookReq is the request body for POST /notebooklm/notebooks/{id}/chat.
type ChatNotebookReq struct {
	Prompt        string `json:"prompt"`
	WebhookURL    string `json:"webhookUrl,omitempty"`
	WebhookSecret string `json:"webhookSecret,omitempty"`
}

// PipelineReq is the request body for POST /notebooklm/pipelines.
type PipelineReq struct {
	Steps         []PipelineStep `json:"steps"`
	WebhookURL    string         `json:"webhookUrl,omitempty"`
	WebhookSecret string         `json:"webhookSecret,omitempty"`
}

// PipelineStep represents a single step in a pipeline.
type PipelineStep struct {
	Action string `json:"action"`
	Input  any    `json:"input"`
}

// SuccessResponse is the generic wrapper returned by Maestro API.
type SuccessResponse[T any] struct {
	ErrorCode int    `json:"errorCode"`
	Message   string `json:"message"`
	Data      T      `json:"data"`
	RequestID string `json:"requestId,omitempty"`
}

// SessionData represents a Maestro browser session.
type SessionData struct {
	SessionID      string    `json:"sessionId"`
	Plugin         string    `json:"plugin,omitempty"`
	Status         string    `json:"status"` // "ready" | "busy" | "tearingDown"
	CreatedAt      time.Time `json:"createdAt,omitempty"`
	LastActivityAt time.Time `json:"lastActivityAt,omitempty"`
}

// JobEnqueued is the response when an async job is created.
type JobEnqueued struct {
	JobID     string    `json:"jobId"`
	Status    string    `json:"status"`
	Action    string    `json:"action,omitempty"`
	PollURL   string    `json:"pollUrl,omitempty"`
	CreatedAt time.Time `json:"createdAt,omitempty"`
}

// JobData represents the full status of a Maestro job.
type JobData struct {
	JobID     string    `json:"jobId"`
	Status    string    `json:"status"` // "queued" | "processing" | "completed" | "failed"
	Action    string    `json:"action"`
	Result    any       `json:"result,omitempty"`
	Error     any       `json:"error,omitempty"`
	CreatedAt time.Time `json:"createdAt,omitempty"`
}

// NotebookData represents a NotebookLM notebook.
type NotebookData struct {
	NotebookID string `json:"notebookId"`
	Title      string `json:"title"`
}

// PipelineData is the response when a pipeline is submitted.
type PipelineData struct {
	PipelineID string `json:"pipelineId"`
	JobID      string `json:"jobId"`
	Status     string `json:"status"`
	PollURL    string `json:"pollUrl,omitempty"`
}

// WebhookPayload is the payload sent by Maestro via webhook callback.
type WebhookPayload struct {
	JobID  string `json:"jobId"`
	Action string `json:"action"`
	Status string `json:"status"`
	Result any    `json:"result,omitempty"`
	Error  any    `json:"error,omitempty"`
}
