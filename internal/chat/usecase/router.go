package usecase

import (
	"strings"
)

// QueryIntent represents the intent of a user's query.
type QueryIntent string

const (
	IntentNarrative  QueryIntent = "NARRATIVE"
	IntentStructured QueryIntent = "STRUCTURED"
)

// QueryRoute defines the routing decision for a query.
type QueryRoute struct {
	Intent      QueryIntent
	UseNotebook bool
	UseQdrant   bool
}

// ClassifyIntent is rule-based: structured keywords first, then narrative.
// Default (no match) is STRUCTURED per architecture (safe fallback to Qdrant).
func ClassifyIntent(query string) QueryIntent {
	q := strings.ToLower(strings.TrimSpace(query))

	structuredKeywords := []string{
		"bao nhiêu", "thống kê", "top", "so sánh", "tỷ lệ", "filter", "lọc", "đếm", "count",
	}
	for _, keyword := range structuredKeywords {
		if strings.Contains(q, keyword) {
			return IntentStructured
		}
	}

	narrativeKeywords := []string{
		"xu hướng", "đánh giá", "tổng quan", "phân tích", "insight", "dự đoán",
	}
	for _, keyword := range narrativeKeywords {
		if strings.Contains(q, keyword) {
			return IntentNarrative
		}
	}

	return IntentStructured
}

// RouteQuery sends NARRATIVE to NotebookLM only when enabled and campaign has synced sources; otherwise Qdrant.
func RouteQuery(query string, notebookEnabled, notebookAvailable bool) QueryRoute {
	intent := ClassifyIntent(query)

	if intent == IntentNarrative && notebookEnabled && notebookAvailable {
		return QueryRoute{
			Intent:      IntentNarrative,
			UseNotebook: true,
			UseQdrant:   false,
		}
	}

	return QueryRoute{
		Intent:      intent,
		UseNotebook: false,
		UseQdrant:   true,
	}
}
