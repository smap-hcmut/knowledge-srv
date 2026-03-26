package usecase

import (
	"encoding/json"
	"fmt"
	"strings"
)

func formatDigestFromPayload(pl map[string]interface{}) string {
	if pl == nil {
		return ""
	}
	domain := str(pl["domain_overlay"])
	platform := str(pl["platform"])
	total := intFrom(pl["total_mentions"])
	start := str(pl["analysis_window_start"])
	end := str(pl["analysis_window_end"])

	var b strings.Builder
	fmt.Fprintf(&b, "Campaign Report: %s\n", domain)
	fmt.Fprintf(&b, "Platform: %s | Total Mentions: %d\n", platform, total)
	fmt.Fprintf(&b, "Analysis Window: %s to %s\n\n", start, end)

	if raw, ok := pl["top_entities"]; ok {
		b.WriteString(formatTopEntities(raw))
	}
	if raw, ok := pl["top_topics"]; ok {
		b.WriteString(formatTopTopics(raw))
	}
	if raw, ok := pl["top_issues"]; ok {
		b.WriteString(formatTopIssues(raw))
	}
	return strings.TrimSpace(b.String())
}

func formatTopEntities(raw interface{}) string {
	arr, ok := raw.([]interface{})
	if !ok || len(arr) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("### Top Brands\n")
	limit := len(arr)
	if limit > 5 {
		limit = 5
	}
	for i := 0; i < limit; i++ {
		m := asMap(arr[i])
		name := str(m["entity_name"])
		mc := intFrom(m["mention_count"])
		ms := num(m["mention_share"])
		fmt.Fprintf(&b, "- %s: %d mentions (%.1f%% share)\n", name, mc, ms*100)
	}
	b.WriteString("\n")
	return b.String()
}

func formatTopTopics(raw interface{}) string {
	arr, ok := raw.([]interface{})
	if !ok || len(arr) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("### Key Discussion Topics\n")
	limit := len(arr)
	if limit > 5 {
		limit = 5
	}
	for i := 0; i < limit; i++ {
		m := asMap(arr[i])
		label := str(m["topic_label"])
		mc := intFrom(m["mention_count"])
		fmt.Fprintf(&b, "- %s: %d mentions", label, mc)
		if q := m["quality_score"]; q != nil {
			fmt.Fprintf(&b, " (quality: %.2f)", num(q))
		}
		b.WriteString("\n")
		if reps, ok := m["representative_texts"].([]interface{}); ok && len(reps) > 0 {
			if s, ok := reps[0].(string); ok {
				fmt.Fprintf(&b, "  Example: %q\n", truncate(s, 120))
			}
		}
	}
	b.WriteString("\n")
	return b.String()
}

func formatTopIssues(raw interface{}) string {
	arr, ok := raw.([]interface{})
	if !ok || len(arr) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("### Critical Issues\n")
	limit := len(arr)
	if limit > 5 {
		limit = 5
	}
	for i := 0; i < limit; i++ {
		m := asMap(arr[i])
		cat := str(m["issue_category"])
		mc := intFrom(m["mention_count"])
		pressure := num(m["issue_pressure_proxy"])
		fmt.Fprintf(&b, "- %s: %d mentions (pressure: %.2f)\n", cat, mc, pressure)
	}
	return b.String()
}

func formatInsightCard(pl map[string]interface{}) string {
	if pl == nil {
		return ""
	}
	title := str(pl["title"])
	insightType := str(pl["insight_type"])
	conf := num(pl["confidence"])
	summary := str(pl["summary"])
	var ev []string
	if raw, ok := pl["evidence_references"]; ok {
		if a, ok := raw.([]interface{}); ok {
			for _, x := range a {
				if s, ok := x.(string); ok {
					ev = append(ev, s)
				}
			}
		}
	}
	var b strings.Builder
	fmt.Fprintf(&b, "### %s — %s (confidence: %.2f)\n", insightType, title, conf)
	b.WriteString(strings.TrimSpace(summary))
	b.WriteString("\n")
	if len(ev) > 0 {
		fmt.Fprintf(&b, "Evidence: %s\n", strings.Join(ev, ", "))
	}
	return b.String()
}

func asMap(v interface{}) map[string]interface{} {
	if m, ok := v.(map[string]interface{}); ok {
		return m
	}
	b, err := json.Marshal(v)
	if err != nil {
		return map[string]interface{}{}
	}
	var out map[string]interface{}
	if err := json.Unmarshal(b, &out); err != nil {
		return map[string]interface{}{}
	}
	return out
}

func str(v interface{}) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return fmt.Sprint(v)
}

func num(v interface{}) float64 {
	switch t := v.(type) {
	case float64:
		return t
	case float32:
		return float64(t)
	case int:
		return float64(t)
	case int64:
		return float64(t)
	case json.Number:
		f, _ := t.Float64()
		return f
	default:
		return 0
	}
}

func intFrom(v interface{}) int {
	switch t := v.(type) {
	case float64:
		return int(t)
	case int:
		return t
	case int64:
		return int(t)
	case json.Number:
		i, _ := t.Int64()
		return int(i)
	default:
		return 0
	}
}

func truncate(s string, n int) string {
	runes := []rune(s)
	if len(runes) <= n {
		return s
	}
	return string(runes[:n]) + "…"
}
