package usecase

import (
	"knowledge-srv/internal/chat"
	"knowledge-srv/internal/search"
)

// extractCitations - Extract citation objects from search results
func (uc *implUseCase) extractCitations(results []search.SearchResult) []chat.Citation {
	citations := make([]chat.Citation, 0, len(results))
	for _, r := range results {
		content := r.Content
		if len(content) > 200 {
			content = content[:200] + "..."
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

// generateTitle - Auto-generate conversation title from first message
func (uc *implUseCase) generateTitle(message string) string {
	if len(message) <= 50 {
		return message
	}
	return message[:50] + "..."
}

// generateSuggestions - Generate follow-up suggestions based on search results
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

	// Suggest trend analysis
	if len(suggestions) < 3 {
		suggestions = append(suggestions, "Xu hướng theo thời gian?")
	}

	if len(suggestions) > 3 {
		suggestions = suggestions[:3]
	}
	return suggestions
}
