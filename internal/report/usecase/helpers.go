package usecase

import (
	"fmt"
	"strings"

	"knowledge-srv/internal/search"
)

func emptyAsDash(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "-"
	}
	return value
}

// formatAggregation formats search aggregations into a readable summary.
func formatAggregation(agg search.Aggregations) string {
	var sb strings.Builder

	// Sentiment distribution
	if len(agg.BySentiment) > 0 {
		sb.WriteString("**Phân bố cảm xúc:**\n")
		for _, s := range agg.BySentiment {
			sb.WriteString(fmt.Sprintf("- %s: %d (%.1f%%)\n", s.Sentiment, s.Count, s.Percentage))
		}
		sb.WriteString("\n")
	}

	// Aspect breakdown
	if len(agg.ByAspect) > 0 {
		sb.WriteString("**Phân bố theo khía cạnh:**\n")
		for _, a := range agg.ByAspect {
			sb.WriteString(fmt.Sprintf("- %s: %d mentions, avg sentiment %.2f\n",
				a.Aspect, a.Count, a.AvgSentimentScore))
		}
		sb.WriteString("\n")
	}

	// Platform breakdown
	if len(agg.ByPlatform) > 0 {
		sb.WriteString("**Phân bố theo nền tảng:**\n")
		for _, p := range agg.ByPlatform {
			sb.WriteString(fmt.Sprintf("- %s: %d (%.1f%%)\n", p.Platform, p.Count, p.Percentage))
		}
	}

	if sb.Len() == 0 {
		return "(Không có dữ liệu tổng hợp)"
	}

	return sb.String()
}
