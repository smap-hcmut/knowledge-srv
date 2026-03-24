package usecase

import (
	"crypto/sha256"
	"fmt"
	"knowledge-srv/internal/transform"
	"sort"
	"strings"
)

// BuildParts groups posts by ISO week, splits by maxPostsPerPart, and returns markdown parts.
func (uc *implUseCase) BuildParts(input transform.TransformInput) ([]transform.MarkdownPart, error) {
	if len(input.Posts) == 0 {
		return nil, nil
	}

	// Group posts by ISO week label
	weekGroups := uc.groupByWeek(input.Posts)

	// Sort weeks chronologically
	weekLabels := make([]string, 0, len(weekGroups))
	for wl := range weekGroups {
		weekLabels = append(weekLabels, wl)
	}
	sort.Strings(weekLabels)

	var parts []transform.MarkdownPart

	for _, weekLabel := range weekLabels {
		posts := weekGroups[weekLabel]
		chunks := uc.splitChunks(posts)

		for i, chunk := range chunks {
			partNum := i + 1
			title := uc.buildTitle(input.CampaignName, weekLabel, partNum, len(chunks))

			content := uc.buildPartContent(input.CampaignName, weekLabel, partNum, chunk)
			hash := fmt.Sprintf("%x", sha256.Sum256([]byte(content)))

			parts = append(parts, transform.MarkdownPart{
				Title:       title,
				Content:     content,
				WeekLabel:   weekLabel,
				PartNum:     partNum,
				PostCount:   len(chunk),
				ContentHash: hash,
			})
		}
	}

	return parts, nil
}

// groupByWeek groups posts by their ISO week label (e.g. "2026-W12").
func (uc *implUseCase) groupByWeek(posts []transform.AnalyticsPostLite) map[string][]transform.AnalyticsPostLite {
	groups := make(map[string][]transform.AnalyticsPostLite)
	for _, p := range posts {
		year, week := p.ContentCreatedAt.ISOWeek()
		weekLabel := fmt.Sprintf("%d-W%02d", year, week)
		groups[weekLabel] = append(groups[weekLabel], p)
	}
	return groups
}

// splitChunks splits posts into chunks of maxPostsPerPart.
func (uc *implUseCase) splitChunks(posts []transform.AnalyticsPostLite) [][]transform.AnalyticsPostLite {
	if len(posts) <= uc.maxPostsPerPart {
		return [][]transform.AnalyticsPostLite{posts}
	}

	var chunks [][]transform.AnalyticsPostLite
	for i := 0; i < len(posts); i += uc.maxPostsPerPart {
		end := i + uc.maxPostsPerPart
		if end > len(posts) {
			end = len(posts)
		}
		chunks = append(chunks, posts[i:end])
	}
	return chunks
}

// buildTitle creates the title for a markdown part.
func (uc *implUseCase) buildTitle(campaignName, weekLabel string, partNum, totalParts int) string {
	name := campaignName
	if name == "" {
		name = "Campaign"
	}
	if totalParts <= 1 {
		return fmt.Sprintf("SMAP | %s | %s", name, weekLabel)
	}
	return fmt.Sprintf("SMAP | %s | %s | Part %d", name, weekLabel, partNum)
}

// buildPartContent builds the full markdown document for a part.
func (uc *implUseCase) buildPartContent(campaignName, weekLabel string, partNum int, posts []transform.AnalyticsPostLite) string {
	var sb strings.Builder

	// Header with summary stats
	sb.WriteString(buildPartHeader(campaignName, weekLabel, partNum, len(posts), posts))

	// Individual post sections
	for i, post := range posts {
		sb.WriteString(buildPostMarkdown(post, i))
	}

	return sb.String()
}
