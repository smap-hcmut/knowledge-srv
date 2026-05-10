package usecase

import (
	"sort"
	"strings"

	"knowledge-srv/internal/contentquality"
	"knowledge-srv/internal/report"
	"knowledge-srv/internal/search"
)

func buildReportRetrievalQuery(reportType string, filters report.ReportFilters) string {
	parts := []string{buildAggregateQuery(reportType)}

	if prompt := meaningfulReportPrompt(filters.Prompt); prompt != "" {
		parts = append(parts, prompt)
	}
	if len(filters.Sections) > 0 {
		parts = append(parts, strings.Join(filters.Sections, " "))
	}

	return strings.Join(uniqueNonEmpty(parts), " ")
}

func meaningfulReportPrompt(prompt string) string {
	prompt = strings.TrimSpace(prompt)
	if prompt == "" {
		return ""
	}
	if !isGenericReportInstruction(prompt) {
		return prompt
	}

	cleaned := strings.ToLower(prompt)
	for _, phrase := range []string{
		"tạo report", "tao report", "tạo báo cáo", "tao bao cao",
		"generate report", "export report", "lập report", "lap report",
		"làm report", "lam report", "report giúp", "report giup",
		"báo cáo giúp", "bao cao giup", "giúp tớ", "giup to",
		"giúp tôi", "giup toi", "đi", "di",
	} {
		cleaned = strings.ReplaceAll(cleaned, phrase, " ")
	}
	cleaned = strings.Join(strings.Fields(cleaned), " ")
	if len([]rune(cleaned)) < 15 {
		return ""
	}
	return cleaned
}

func isGenericReportInstruction(prompt string) bool {
	lower := strings.ToLower(strings.Join(strings.Fields(prompt), " "))
	if lower == "" {
		return false
	}

	patterns := []string{
		"tạo report", "tao report", "tạo báo cáo", "tao bao cao",
		"generate report", "export report", "report giúp", "report giup",
		"báo cáo giúp", "bao cao giup", "lập report", "lap report",
		"làm report", "lam report",
	}
	for _, pattern := range patterns {
		if strings.Contains(lower, pattern) {
			return true
		}
	}

	words := strings.Fields(lower)
	if len(words) <= 8 && (strings.Contains(lower, "report") || strings.Contains(lower, "báo cáo") || strings.Contains(lower, "bao cao")) {
		return true
	}
	return false
}

func sanitizeReportSearchOutput(output search.SearchOutput) search.SearchOutput {
	filtered := make([]search.SearchResult, 0, len(output.Results))
	for _, result := range output.Results {
		if strings.TrimSpace(result.Content) == "" {
			continue
		}
		if contentquality.IsLowValueMarketingContent(result.Content) {
			continue
		}
		filtered = append(filtered, result)
	}

	output.Results = filtered
	output.TotalFound = len(filtered)
	output.NoRelevantContext = len(filtered) == 0
	output.Aggregations = buildReportAggregations(filtered)
	return output
}

func buildReportAggregations(results []search.SearchResult) search.Aggregations {
	total := len(results)
	if total == 0 {
		return search.Aggregations{}
	}

	sentimentCounts := make(map[string]int)
	platformCounts := make(map[string]int)
	aspectData := make(map[string]struct {
		displayName string
		count       int
		totalScore  float64
	})

	for _, result := range results {
		if result.OverallSentiment != "" {
			sentimentCounts[result.OverallSentiment]++
		}
		if result.Platform != "" {
			platformCounts[result.Platform]++
		}
		for _, aspect := range result.Aspects {
			if aspect.Aspect == "" {
				continue
			}
			item := aspectData[aspect.Aspect]
			item.displayName = aspect.AspectDisplayName
			item.count++
			item.totalScore += aspect.SentimentScore
			aspectData[aspect.Aspect] = item
		}
	}

	bySentiment := make([]search.SentimentAgg, 0, len(sentimentCounts))
	for sentiment, count := range sentimentCounts {
		bySentiment = append(bySentiment, search.SentimentAgg{
			Sentiment:  sentiment,
			Count:      count,
			Percentage: float64(count) / float64(total) * 100,
		})
	}
	sort.SliceStable(bySentiment, func(i, j int) bool {
		if bySentiment[i].Count == bySentiment[j].Count {
			return bySentiment[i].Sentiment < bySentiment[j].Sentiment
		}
		return bySentiment[i].Count > bySentiment[j].Count
	})

	byPlatform := make([]search.PlatformAgg, 0, len(platformCounts))
	for platform, count := range platformCounts {
		byPlatform = append(byPlatform, search.PlatformAgg{
			Platform:   platform,
			Count:      count,
			Percentage: float64(count) / float64(total) * 100,
		})
	}
	sort.SliceStable(byPlatform, func(i, j int) bool {
		if byPlatform[i].Count == byPlatform[j].Count {
			return byPlatform[i].Platform < byPlatform[j].Platform
		}
		return byPlatform[i].Count > byPlatform[j].Count
	})

	byAspect := make([]search.AspectAgg, 0, len(aspectData))
	for name, item := range aspectData {
		avg := 0.0
		if item.count > 0 {
			avg = item.totalScore / float64(item.count)
		}
		byAspect = append(byAspect, search.AspectAgg{
			Aspect:            name,
			AspectDisplayName: item.displayName,
			Count:             item.count,
			AvgSentimentScore: avg,
		})
	}
	sort.SliceStable(byAspect, func(i, j int) bool {
		if byAspect[i].Count == byAspect[j].Count {
			return byAspect[i].Aspect < byAspect[j].Aspect
		}
		return byAspect[i].Count > byAspect[j].Count
	})

	return search.Aggregations{
		BySentiment: bySentiment,
		ByAspect:    byAspect,
		ByPlatform:  byPlatform,
	}
}

func uniqueNonEmpty(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		normalized := strings.Join(strings.Fields(value), " ")
		if normalized == "" {
			continue
		}
		key := strings.ToLower(normalized)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, normalized)
	}
	return out
}
