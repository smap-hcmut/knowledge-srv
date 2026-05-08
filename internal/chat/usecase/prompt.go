package usecase

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"knowledge-srv/internal/chat"
	"knowledge-srv/internal/model"
	"knowledge-srv/internal/search"
)

const systemPrompt = `Bạn là trợ lý phân tích dữ liệu SMAP. Nhiệm vụ:
- Trả lời câu hỏi dựa trên context documents được cung cấp
- Trích dẫn nguồn bằng [1], [2], ... tương ứng với thứ tự documents
- Không suy diễn ngoài dữ liệu; nếu context yếu, nói rõ giới hạn mẫu dữ liệu
- Nếu context không liên quan trực tiếp đến câu hỏi, nói "Không tìm thấy dữ liệu liên quan" thay vì cố bịa
- Phân biệt rõ dữ liệu quan sát được, giả thuyết, và khuyến nghị hành động
- Trả lời bằng tiếng Việt, ngắn gọn, chính xác
- Phân tích sentiment và xu hướng nếu được hỏi`

func (uc *implUseCase) buildPrompt(question string, docs []search.SearchResult, history []model.Message) string {
	var b strings.Builder

	// System prompt
	b.WriteString(systemPrompt)
	b.WriteString("\n\n")

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
		return uc.buildReducedPrompt(question, docs, history)
	}

	return prompt
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
func (uc *implUseCase) buildReducedPrompt(question string, docs []search.SearchResult, history []model.Message) string {
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
	b.WriteString(uc.buildContextBlock(reducedDocs))
	if len(reducedHistory) > 0 {
		b.WriteString(uc.buildHistoryBlock(reducedHistory))
	}
	b.WriteString(fmt.Sprintf("User: %s\nAssistant:", question))

	return b.String()
}
