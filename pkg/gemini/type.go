package gemini

import pkghttp "knowledge-srv/pkg/http"

// GeminiConfig holds the configuration for the Gemini client
type GeminiConfig struct {
	APIKey string
	Model  string
}

// geminiImpl implements IGemini using the Google Gemini API.
type geminiImpl struct {
	apiKey     string
	model      string
	httpClient pkghttp.IClient
}

// Request defines the request body for Generate Content API
type Request struct {
	Contents []Content `json:"contents"`
}

// Content represents a single content block
type Content struct {
	Parts []Part `json:"parts"`
	Role  string `json:"role,omitempty"`
}

// Part represents a part of the content (text or blob)
type Part struct {
	Text string `json:"text,omitempty"`
}

// Response defines the response body from Generate Content API
type Response struct {
	Candidates    []Candidate   `json:"candidates"`
	UsageMetadata UsageMetadata `json:"usageMetadata"`
}

// Candidate represents a generated candidate
type Candidate struct {
	Content      Content `json:"content"`
	FinishReason string  `json:"finishReason"`
	Index        int     `json:"index"`
}

// UsageMetadata represents token usage
type UsageMetadata struct {
	PromptTokenCount     int `json:"promptTokenCount"`
	CandidatesTokenCount int `json:"candidatesTokenCount"`
	TotalTokenCount      int `json:"totalTokenCount"`
}
