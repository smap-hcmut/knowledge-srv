package usecase

import (
	"fmt"
	"knowledge-srv/internal/transform"
	"strings"
	"time"
)

// buildPostMarkdown converts a single AnalyticsPostLite into a markdown section.
func buildPostMarkdown(post transform.AnalyticsPostLite, index int) string {
	var sb strings.Builder

	// Header
	createdAt := post.ContentCreatedAt.Format(time.RFC3339)
	sb.WriteString(fmt.Sprintf("### Bài viết #%d\n\n", index+1))

	// Metadata table
	sb.WriteString("| Thuộc tính | Giá trị |\n")
	sb.WriteString("| --- | --- |\n")
	sb.WriteString(fmt.Sprintf("| Nền tảng | %s |\n", post.Platform))
	sb.WriteString(fmt.Sprintf("| Tác giả | %s |\n", post.Author))
	sb.WriteString(fmt.Sprintf("| Ngày đăng | %s |\n", createdAt))
	sb.WriteString(fmt.Sprintf("| Cảm xúc | %s (%.2f) |\n", post.Sentiment, post.SentimentScore))

	if post.RiskLevel != "" {
		sb.WriteString(fmt.Sprintf("| Mức rủi ro | %s (%.2f) |\n", post.RiskLevel, post.RiskScore))
	}

	// Engagement
	sb.WriteString(fmt.Sprintf("| Tương tác | 👍 %d · 💬 %d · 🔄 %d · 👁 %d |\n",
		post.Likes, post.Comments, post.Shares, post.Views))
	sb.WriteString(fmt.Sprintf("| Điểm tương tác | %.2f |\n", post.EngagementScore))
	sb.WriteString("\n")

	// Aspects
	if len(post.Aspects) > 0 {
		sb.WriteString("**Khía cạnh phân tích:**\n\n")
		for _, a := range post.Aspects {
			displayName := a.DisplayName
			if displayName == "" {
				displayName = a.Name
			}
			kw := ""
			if len(a.Keywords) > 0 {
				kw = fmt.Sprintf(" — từ khóa: %s", strings.Join(a.Keywords, ", "))
			}
			sb.WriteString(fmt.Sprintf("- **%s**: %s (%.2f)%s\n", displayName, a.Sentiment, a.SentimentScore, kw))
		}
		sb.WriteString("\n")
	}

	// Keywords
	if len(post.Keywords) > 0 {
		sb.WriteString(fmt.Sprintf("**Từ khóa:** %s\n\n", strings.Join(post.Keywords, ", ")))
	}

	// Content
	sb.WriteString("**Nội dung:**\n\n")
	sb.WriteString("> " + strings.ReplaceAll(post.Content, "\n", "\n> "))
	sb.WriteString("\n\n---\n\n")

	return sb.String()
}

// buildPartHeader generates the summary header for a markdown part.
func buildPartHeader(campaignName, weekLabel string, partNum, postCount int, posts []transform.AnalyticsPostLite) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("# %s — %s — Phần %d\n\n", campaignName, weekLabel, partNum))
	sb.WriteString(fmt.Sprintf("**Tổng số bài viết:** %d\n\n", postCount))

	// Sentiment breakdown
	sentimentCounts := map[string]int{}
	platformCounts := map[string]int{}
	for _, p := range posts {
		sentimentCounts[p.Sentiment]++
		platformCounts[p.Platform]++
	}

	if len(sentimentCounts) > 0 {
		sb.WriteString("**Phân bổ cảm xúc:**\n\n")
		for sentiment, count := range sentimentCounts {
			pct := float64(count) / float64(postCount) * 100
			sb.WriteString(fmt.Sprintf("- %s: %d (%.0f%%)\n", sentiment, count, pct))
		}
		sb.WriteString("\n")
	}

	if len(platformCounts) > 0 {
		sb.WriteString("**Nền tảng:**\n\n")
		for platform, count := range platformCounts {
			sb.WriteString(fmt.Sprintf("- %s: %d bài\n", platform, count))
		}
		sb.WriteString("\n")
	}

	sb.WriteString("---\n\n")
	return sb.String()
}
