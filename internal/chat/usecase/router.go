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

// structuredKeywords are signals that the user wants analytics/aggregation.
var structuredKeywords = []string{
	"bao nhiêu", "thống kê", "top", "so sánh", "tỷ lệ", "filter", "lọc", "đếm", "count",
	"how many", "statistics", "compare", "ratio", "percentage", "rank", "ranking",
}

// narrativeKeywords are signals that the user wants summarisation or contextual insight.
var narrativeKeywords = []string{
	"xu hướng", "đánh giá", "tổng quan", "phân tích", "insight", "dự đoán",
	"trend", "overview", "analysis", "analyze", "summary", "summarize", "predict", "sentiment",
}

// ClassifyIntent uses multi-signal scoring: count keyword matches for each intent bucket,
// return the bucket with the most matches.
//
// Default (no matches at all) is NARRATIVE — a query with no analytics keywords is
// most likely a broad contextual search, so we use the lower minScore (0.55) to avoid
// returning 0 results for brand names, product names, or free-form questions.
func ClassifyIntent(query string) QueryIntent {
	q := strings.ToLower(strings.TrimSpace(query))

	structuredScore := 0
	for _, kw := range structuredKeywords {
		if strings.Contains(q, kw) {
			structuredScore++
		}
	}

	narrativeScore := 0
	for _, kw := range narrativeKeywords {
		if strings.Contains(q, kw) {
			narrativeScore++
		}
	}

	if structuredScore > narrativeScore {
		return IntentStructured
	}
	// NARRATIVE wins on tie or when neither matched — safer default for broad queries.
	return IntentNarrative
}
