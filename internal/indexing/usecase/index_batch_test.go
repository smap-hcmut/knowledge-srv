package usecase

import (
	"testing"

	"knowledge-srv/internal/indexing"
)

func TestIsIndexableInsightAcceptsDomainGenericRelevantDocument(t *testing.T) {
	doc := indexing.InsightMessageInput{
		Business: indexing.InsightBusinessInput{
			RelevanceScore: indexing.MinBusinessRelevanceScore,
		},
	}

	if !isIndexableInsight(doc, "Kotex campaign discussion has a clear creative hook") {
		t.Fatal("expected relevant non-logistics document to be indexable")
	}
}

func TestIsIndexableInsightRejectsShortOrIrrelevantDocument(t *testing.T) {
	doc := indexing.InsightMessageInput{
		Business: indexing.InsightBusinessInput{
			RelevanceScore: indexing.MinBusinessRelevanceScore,
		},
	}

	if isIndexableInsight(doc, "short") {
		t.Fatal("expected short document to be skipped")
	}

	doc.Business.RelevanceScore = 0.10
	if isIndexableInsight(doc, "Long enough but unrelated generic discussion text") {
		t.Fatal("expected low-relevance document to be skipped")
	}
}
