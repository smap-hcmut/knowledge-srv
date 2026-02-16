package usecase

import (
	"fmt"
	"strings"

	"knowledge-srv/internal/chat"
	"knowledge-srv/internal/model"
	"knowledge-srv/internal/search"
)

const systemPrompt = `Bạn là trợ lý phân tích dữ liệu SMAP. Nhiệm vụ:
- Trả lời câu hỏi dựa trên context documents được cung cấp
- Trích dẫn nguồn bằng [1], [2], ... tương ứng với thứ tự documents
- Nếu không có context phù hợp, nói rõ "Không tìm thấy dữ liệu liên quan"
- Trả lời bằng tiếng Việt, ngắn gọn, chính xác
- Phân tích sentiment và xu hướng nếu được hỏi`

// buildPrompt - Build complete LLM prompt with token management
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
	estimatedTokens := len(prompt) / 4 // ~4 chars ≈ 1 token
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
		b.WriteString(fmt.Sprintf("[%d] \"%s\" (Platform: %s, Sentiment: %s, Score: %.2f)\n",
			i+1, content, doc.Platform, doc.OverallSentiment, doc.Score))
	}
	b.WriteString("\n")
	return b.String()
}

// buildHistoryBlock - Format conversation history
func (uc *implUseCase) buildHistoryBlock(msgs []model.Message) string {
	var b strings.Builder
	b.WriteString("Conversation History:\n")
	for _, msg := range msgs {
		role := msg.Role
		if len(role) > 0 {
			role = strings.ToUpper(role[:1]) + role[1:]
		}
		b.WriteString(fmt.Sprintf("%s: %s\n", role, msg.Content))
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
