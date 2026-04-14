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

// ClassifyIntent is rule-based: structured keywords first, then narrative.
// Default (no match) is STRUCTURED per architecture (safe fallback to Qdrant).
func ClassifyIntent(query string) QueryIntent {
	q := strings.ToLower(strings.TrimSpace(query))

	structuredKeywords := []string{
		"bao nhiêu", "thống kê", "top", "so sánh", "tỷ lệ", "filter", "lọc", "đếm", "count",
		"how many", "statistics", "compare", "ratio", "percentage", "rank", "ranking",
	}
	for _, keyword := range structuredKeywords {
		if strings.Contains(q, keyword) {
			return IntentStructured
		}
	}

	narrativeKeywords := []string{
		"xu hướng", "đánh giá", "tổng quan", "phân tích", "insight", "dự đoán",
		"trend", "overview", "analysis", "analyze", "summary", "summarize", "predict", "sentiment",
	}
	for _, keyword := range narrativeKeywords {
		if strings.Contains(q, keyword) {
			return IntentNarrative
		}
	}

	return IntentStructured
}
