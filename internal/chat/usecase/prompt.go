package usecase

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"knowledge-srv/internal/chat"
	"knowledge-srv/internal/model"
	"knowledge-srv/internal/search"
	analyticspkg "knowledge-srv/pkg/analytics"
)

const systemPrompt = `Bạn là trợ lý phân tích dữ liệu SMAP. Nhiệm vụ:
- Trả lời câu hỏi dựa trên context documents được cung cấp
- Dùng Analytics Snapshot để trả lời số liệu/tỷ trọng/so sánh nền tảng; dùng documents để minh họa định tính
- Trích dẫn nguồn bằng [1], [2], ... tương ứng với thứ tự documents
- Không suy diễn ngoài dữ liệu; nếu context yếu, nói rõ giới hạn mẫu dữ liệu
- Nếu context không liên quan trực tiếp đến câu hỏi, nói "Không tìm thấy dữ liệu liên quan" thay vì cố bịa
- Phân biệt rõ dữ liệu quan sát được, giả thuyết, và khuyến nghị hành động
- Trả lời bằng tiếng Việt, ngắn gọn, chính xác
- Phân tích sentiment và xu hướng nếu được hỏi`

func (uc *implUseCase) buildPrompt(question string, docs []search.SearchResult, history []model.Message, snapshot *analyticspkg.Snapshot) string {
	var b strings.Builder

	// System prompt
	b.WriteString(systemPrompt)
	b.WriteString("\n\n")

	if snapshot != nil && snapshot.HasData() {
		b.WriteString(uc.buildAnalyticsContextBlock(*snapshot))
	}

	// Context block
	contextBlock := uc.buildContextBlock(docs)
	b.WriteString(contextBlock)

	// History block (if multi-turn)
	if len(history) > 0 {
		historyBlock := uc.buildHistoryBlock(history)
		b.WriteString(historyBlock)
	}

	// Current question
	b.WriteString(fmt.Sprintf("User: %s\nAssistant:", question))

	// Token window management
	prompt := b.String()
	estimatedTokens := utf8.RuneCountInString(prompt) / 2 // Vietnamese ~2 runes per token
	if estimatedTokens > chat.MaxTokenWindow {
		return uc.buildReducedPrompt(question, docs, history, snapshot)
	}

	return prompt
}

func (uc *implUseCase) buildAnalyticsContextBlock(snapshot analyticspkg.Snapshot) string {
	var b strings.Builder
	b.WriteString("Analytics Snapshot (dashboard-grade, đã qua quality gate):\n")
	b.WriteString(fmt.Sprintf("- Total mentions: %s\n", metricFormatted(snapshot.KPIs.Metrics, "Total Mentions", totalDocsFromSnapshot(snapshot))))
	if sentimentScore, ok := metricFormattedIfPresent(snapshot.KPIs.Metrics, "Sentiment Score"); ok {
		b.WriteString(fmt.Sprintf("- Sentiment score: %s\n", sentimentScore))
	}
	engagementFallback := totalEngagement(sortedPlatformStats(snapshot.Platforms.Stats))
	if engagementFallback == 0 {
		engagementFallback = totalPostEngagement(snapshot.Posts.Posts)
	}
	b.WriteString(fmt.Sprintf("- Engagement: %s\n", metricFormatted(snapshot.KPIs.Metrics, "Engagement", engagementFallback)))
	if len(snapshot.Errors) > 0 || !snapshot.HasCoreAnalytics() {
		b.WriteString("- Snapshot note: partial analytics response; avoid treating missing metrics as zero.\n")
	}
	if snapshot.Sentiment.Total > 0 {
		b.WriteString(fmt.Sprintf("- Sentiment share: positive %.1f%%, neutral %.1f%%, negative %.1f%%\n",
			sentimentShare(snapshot.Sentiment.Donut, "positive"),
			sentimentShare(snapshot.Sentiment.Donut, "neutral"),
			sentimentShare(snapshot.Sentiment.Donut, "negative"),
		))
	}
	platforms := sortedPlatformStats(snapshot.Platforms.Stats)
	if len(platforms) > 0 {
		b.WriteString("- Platform breakdown:\n")
		for _, p := range platforms {
			b.WriteString(fmt.Sprintf("  - %s: mentions=%s, sentiment=%.1f%%, engagement=%s, reach=%s\n",
				platformLabel(p), formatInt(p.Mentions), p.Sentiment, formatInt(p.EngagementRaw), formatInt(p.Reach)))
		}
	}
	drivers := topKeywords(snapshot.Keywords.Keywords, 8)
	if len(drivers) > 0 {
		b.WriteString(fmt.Sprintf("- Top topics: %s\n", strings.Join(drivers, ", ")))
	}
	examples := samplePosts(snapshot.Posts.Posts, "", 5)
	if len(examples) > 0 {
		b.WriteString("- High-engagement examples from analytics API:\n")
		for _, post := range examples {
			source := "source unavailable"
			if strings.TrimSpace(post.URL) != "" {
				source = post.URL
			}
			b.WriteString(fmt.Sprintf("  - %s · %s · %s · engagement=%s · source=%s · %s\n",
				platformLabelName(post.Platform), post.Sentiment, authorLabel(post), formatInt(post.Engagement), source, trimRunes(post.Content, 150)))
		}
	}
	b.WriteString("\n")
	return b.String()
}

// buildContextBlock - Format search results as numbered context
func (uc *implUseCase) buildContextBlock(docs []search.SearchResult) string {
	if len(docs) == 0 {
		return "Context: Không có documents liên quan.\n\n"
	}

	var b strings.Builder
	b.WriteString("Context:\n")
	for i, doc := range docs {
		content := doc.Content
		if len(content) > chat.MaxDocContentLen {
			content = content[:chat.MaxDocContentLen] + "..."
		}
		b.WriteString(fmt.Sprintf("[%d] \"%s\" (Platform: %s, Sentiment: %s, Score: %.2f, Risk: %s, Engagement: %.2f",
			i+1, content, doc.Platform, doc.OverallSentiment, doc.Score, doc.RiskLevel, doc.EngagementScore))
		if len(doc.Keywords) > 0 {
			b.WriteString(fmt.Sprintf(", Keywords: %s", joinLimited(doc.Keywords, 8)))
		}
		if len(doc.Aspects) > 0 {
			b.WriteString(fmt.Sprintf(", Aspects: %s", formatAspects(doc.Aspects, 5)))
		}
		b.WriteString(")\n")
	}
	b.WriteString("\n")
	return b.String()
}

func joinLimited(values []string, limit int) string {
	if len(values) == 0 {
		return ""
	}
	if len(values) > limit {
		values = values[:limit]
	}
	return strings.Join(values, ", ")
}

func formatAspects(aspects []search.AspectResult, limit int) string {
	if len(aspects) == 0 {
		return ""
	}
	if len(aspects) > limit {
		aspects = aspects[:limit]
	}
	parts := make([]string, 0, len(aspects))
	for _, aspect := range aspects {
		name := aspect.Aspect
		if aspect.AspectDisplayName != "" {
			name = aspect.AspectDisplayName
		}
		if aspect.Sentiment != "" {
			parts = append(parts, fmt.Sprintf("%s/%s", name, aspect.Sentiment))
		} else {
			parts = append(parts, name)
		}
	}
	return strings.Join(parts, ", ")
}

// buildHistoryBlock - Format conversation history with per-message truncation
func (uc *implUseCase) buildHistoryBlock(msgs []model.Message) string {
	const maxMsgLen = 500
	var b strings.Builder
	b.WriteString("Conversation History:\n")
	for _, msg := range msgs {
		role := msg.Role
		if len(role) > 0 {
			role = strings.ToUpper(role[:1]) + role[1:]
		}
		content := msg.Content
		if utf8.RuneCountInString(content) > maxMsgLen {
			// Truncate to maxMsgLen runes
			runeCount := 0
			for i := range content {
				runeCount++
				if runeCount == maxMsgLen {
					content = content[:i] + "..."
					break
				}
			}
		}
		b.WriteString(fmt.Sprintf("%s: %s\n", role, content))
	}
	b.WriteString("\n")
	return b.String()
}

// buildReducedPrompt - Rebuild prompt with fewer docs and history to fit token window
func (uc *implUseCase) buildReducedPrompt(question string, docs []search.SearchResult, history []model.Message, snapshot *analyticspkg.Snapshot) string {
	// Reduce: fewer docs (max 5), fewer history (last 10)
	reducedDocs := docs
	if len(reducedDocs) > 5 {
		reducedDocs = reducedDocs[:5]
	}
	reducedHistory := history
	if len(reducedHistory) > 10 {
		reducedHistory = reducedHistory[len(reducedHistory)-10:]
	}

	var b strings.Builder
	b.WriteString(systemPrompt)
	b.WriteString("\n\n")
	if snapshot != nil && snapshot.HasData() {
		b.WriteString(uc.buildAnalyticsContextBlock(*snapshot))
	}
	b.WriteString(uc.buildContextBlock(reducedDocs))
	if len(reducedHistory) > 0 {
		b.WriteString(uc.buildHistoryBlock(reducedHistory))
	}
	b.WriteString(fmt.Sprintf("User: %s\nAssistant:", question))

	return b.String()
}
