package usecase

import (
	"knowledge-srv/internal/chat"
	"knowledge-srv/internal/search"
)

func (uc *implUseCase) extractCitations(results []search.SearchResult) []chat.Citation {
	citations := make([]chat.Citation, 0, len(results))
	for _, r := range results {
		content := r.Content
		if runes := []rune(content); len(runes) > 200 {
			content = string(runes[:200]) + "..."
		}
		citations = append(citations, chat.Citation{
			ID:             r.ID,
			Content:        content,
			RelevanceScore: r.Score,
			Platform:       r.Platform,
			Sentiment:      r.OverallSentiment,
		})
	}
	return citations
}

func (uc *implUseCase) generateTitle(message string) string {
	runes := []rune(message)
	if len(runes) <= 50 {
		return message
	}
	return string(runes[:50]) + "..."
}

func (uc *implUseCase) generateSuggestions(query string, output search.SearchOutput) []string {
	suggestions := make([]string, 0, 3)

	// Suggest aspect deep-dive if negative aspects found
	for _, a := range output.Aggregations.ByAspect {
		if a.AvgSentimentScore < -0.3 {
			suggestions = append(suggestions, "Chi tiết về "+a.AspectDisplayName+" thì sao?")
		}
		if len(suggestions) >= 3 {
			break
		}
	}

	// Suggest comparison if multiple platforms
	if len(output.Aggregations.ByPlatform) > 1 && len(suggestions) < 3 {
		suggestions = append(suggestions, "So sánh giữa các nền tảng?")
	}

	// Prefer grounded follow-ups before broader analytical questions.
	if len(output.Results) > 0 && len(suggestions) < 3 {
		suggestions = append(suggestions, "Khách hàng đang phản hồi gì nổi bật?")
	}

	// Suggest trend analysis only when there is enough context to infer change.
	if len(output.Results) >= 5 && len(suggestions) < 3 {
		suggestions = append(suggestions, "Xu hướng theo thời gian?")
	}

	if len(output.Results) > 0 && len(suggestions) < 3 {
		suggestions = append(suggestions, "Các phản hồi tích cực nổi bật là gì?")
	}

	if len(suggestions) > 3 {
		suggestions = suggestions[:3]
	}
	return suggestions
}
