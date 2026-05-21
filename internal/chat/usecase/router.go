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

var platformKeywords = map[string][]string{
	"TIKTOK":   {"tiktok", "tik tok"},
	"FACEBOOK": {"facebook", "fb"},
	"YOUTUBE":  {"youtube", "you tube", "yt"},
}

// structuredKeywords are signals that the user wants analytics/aggregation.
var structuredKeywords = []string{
	"bao nhiêu", "thống kê", "top", "so sánh", "tỷ lệ", "filter", "lọc", "đếm", "count",
	"đào sâu", "sentiment thấp", "nền tảng", "theo platform",
	"how many", "statistics", "compare", "ratio", "percentage", "rank", "ranking", "platform",
}

// narrativeKeywords are signals that the user wants summarisation or contextual insight.
var narrativeKeywords = []string{
	"xu hướng", "đánh giá", "tổng quan", "phân tích", "insight", "dự đoán",
	"tại sao", "vì sao", "lý do", "nguyên nhân", "trái chiều", "mâu thuẫn", "đối lập",
	"trend", "overview", "analysis", "analyze", "summary", "summarize", "predict", "sentiment",
}

// ClassifyIntent uses multi-signal scoring: count keyword matches for each intent bucket,
// return the bucket with the most matches.
//
// Default (no matches at all) is NARRATIVE — a query with no analytics keywords is
// most likely a broad contextual search, so chat uses a lower minScore to avoid
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

// ShouldUseAnalyticsFirst decides whether a query is better served by the
// dashboard analytics snapshot before vector search or LLM generation. These
// queries need fast, numeric, campaign-scoped answers, and Qdrant can be empty
// even when the dashboard already has enough aggregate data.
func ShouldUseAnalyticsFirst(intent QueryIntent, query string) bool {
	q := strings.ToLower(strings.TrimSpace(query))
	if q == "" {
		return false
	}
	if isAnalyticsReasoningQuestion(q) {
		return true
	}
	if isSummaryQuestion(q) {
		return true
	}
	if intent != IntentStructured {
		return false
	}
	aggregateSignals := []string{
		"bao nhiêu", "thống kê", "so sánh", "tỷ lệ", "lọc", "đếm", "count",
		"statistics", "compare", "ratio", "percentage", "rank", "ranking",
		"nền tảng", "platform", "engagement", "mentions",
	}
	for _, signal := range aggregateSignals {
		if strings.Contains(q, signal) {
			return true
		}
	}
	return false
}

// IsSmallTalkMessage reports whether a message is only a lightweight greeting
// or connectivity check. These messages should not run campaign search because
// attaching random citations to "alo" style prompts makes the assistant look
// noisy and stale.
func IsSmallTalkMessage(query string) bool {
	q := strings.ToLower(strings.TrimSpace(query))
	q = strings.Trim(q, " \t\n\r.,!?;:~`'\"…")
	if q == "" {
		return false
	}
	tokens := strings.Fields(q)
	if len(tokens) == 0 || len(tokens) > 4 {
		return false
	}
	greetings := map[string]struct{}{
		"alo": {}, "alô": {}, "hello": {}, "hi": {}, "hey": {},
		"chào": {}, "xin": {}, "ping": {}, "test": {},
	}
	for _, token := range tokens {
		token = strings.Trim(token, " \t\n\r.,!?;:~`'\"…")
		if _, ok := greetings[token]; !ok {
			return false
		}
	}
	return true
}

// ShouldUseAnalyticsFallback decides whether an empty RAG search may be answered
// from dashboard-grade analytics. It intentionally rejects qualitative/evidence
// questions so generic KPI summaries do not replace actual campaign context.
func ShouldUseAnalyticsFallback(intent QueryIntent, query string) bool {
	q := strings.ToLower(strings.TrimSpace(query))
	if q == "" {
		return false
	}
	if isAnalyticsReasoningQuestion(q) {
		return true
	}

	evidenceSignals := []string{
		"bài viết", "post", "comment", "bình luận", "trích dẫn", "citation",
		"ví dụ", "example", "nổi bật", "đáng chú ý", "vì sao", "lý do",
		"nguyên nhân", "khách hàng nói gì",
	}
	for _, signal := range evidenceSignals {
		if strings.Contains(q, signal) {
			return false
		}
	}

	if isSummaryQuestion(q) {
		return true
	}

	if intent != IntentStructured {
		return false
	}

	aggregateSignals := []string{
		"bao nhiêu", "thống kê", "so sánh", "tỷ lệ", "lọc", "đếm", "count",
		"statistics", "compare", "ratio", "percentage", "rank", "ranking",
	}
	for _, signal := range aggregateSignals {
		if strings.Contains(q, signal) {
			return true
		}
	}
	return false
}

func isSummaryQuestion(q string) bool {
	summarySignals := []string{
		"tóm tắt", "tom tat", "tổng quan", "tong quan", "insight chính", "insight chinh",
		"điểm chính", "diem chinh", "kết luận chính", "ket luan chinh", "campaign summary",
		"main insight", "summary", "overview",
	}
	for _, signal := range summarySignals {
		if strings.Contains(q, signal) {
			return true
		}
	}
	return false
}

func isAnalyticsReasoningQuestion(q string) bool {
	reasonSignals := []string{
		"sentiment thấp", "sentiment thap", "cảm xúc thấp", "cam xuc thap",
		"đào sâu lý do", "dao sau ly do", "lý do sentiment", "ly do sentiment",
		"nguyên nhân sentiment", "nguyen nhan sentiment",
		"trái chiều", "trai chieu", "mâu thuẫn", "mau thuan", "đối lập", "doi lap",
	}
	for _, signal := range reasonSignals {
		if strings.Contains(q, signal) {
			return true
		}
	}
	if (strings.Contains(q, "tại sao") || strings.Contains(q, "vì sao")) &&
		(strings.Contains(q, "sentiment") || strings.Contains(q, "đánh giá") || strings.Contains(q, "campaign") || strings.Contains(q, "chiến dịch")) {
		return true
	}
	return false
}

func InferPlatforms(query string) []string {
	q := strings.ToLower(strings.TrimSpace(query))
	platforms := make([]string, 0, len(platformKeywords))

	for platform, keywords := range platformKeywords {
		for _, keyword := range keywords {
			if strings.Contains(q, keyword) {
				platforms = append(platforms, platform)
				break
			}
		}
	}

	return platforms
}
