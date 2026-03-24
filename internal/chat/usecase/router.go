package usecase

import (
	"strings"
)

// QueryIntent represents the intent of a user's query.
type QueryIntent string

const (
	IntentNarrative  QueryIntent = "NARRATIVE"  // Analytical or overview queries
	IntentStructured QueryIntent = "STRUCTURED" // Statistical or comparative queries
	IntentDefault    QueryIntent = "DEFAULT"    // Fallback intent
)

// QueryRoute defines the routing decision for a query.
type QueryRoute struct {
	Intent        QueryIntent // Classified intent
	UseNotebook   bool        // Whether to use Notebook for processing
	UseQdrant     bool        // Whether to use Qdrant for processing
	QdrantFilters []string    // Filters to apply for Qdrant queries
}

// ClassifyIntent determines the intent of a query based on its content.
func ClassifyIntent(query string) QueryIntent {
	query = strings.ToLower(query)

	// Keywords for narrative intent
	narrativeKeywords := []string{"xu hướng", "đánh giá", "tổng quan", "phân tích", "dự đoán"}
	for _, keyword := range narrativeKeywords {
		if strings.Contains(query, keyword) {
			return IntentNarrative
		}
	}

	// Keywords for structured intent
	structuredKeywords := []string{"bao nhiêu", "thống kê", "top", "so sánh", "tỷ lệ"}
	for _, keyword := range structuredKeywords {
		if strings.Contains(query, keyword) {
			return IntentStructured
		}
	}

	// Default intent
	return IntentDefault
}

// RouteQuery determines the routing logic for a query based on its intent.
func RouteQuery(query string, notebookEnabled bool) QueryRoute {
	intent := ClassifyIntent(query)

	switch intent {
	case IntentNarrative:
		if notebookEnabled {
			return QueryRoute{
				Intent:      IntentNarrative,
				UseNotebook: true,
				UseQdrant:   false,
			}
		}
		// Fallback to Qdrant if Notebook is disabled
		return QueryRoute{
			Intent:      IntentNarrative,
			UseNotebook: false,
			UseQdrant:   true,
		}

	case IntentStructured:
		return QueryRoute{
			Intent:      IntentStructured,
			UseNotebook: false,
			UseQdrant:   true,
		}

	default:
		// Default fallback to Qdrant
		return QueryRoute{
			Intent:      IntentDefault,
			UseNotebook: false,
			UseQdrant:   true,
		}
	}
}
