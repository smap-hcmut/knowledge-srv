package usecase

import (
	"knowledge-srv/internal/search"
	"strings"

	pb "github.com/qdrant/go-client/qdrant"
)

// buildSearchFilter builds a Qdrant filter from domain filters.
func (uc *implUseCase) buildSearchFilter(filters search.SearchFilters) *pb.Filter {
	must := []*pb.Condition{}

	// Note: project_id filtering is implicit — we query proj_{project_id} collections directly.
	// No need for a project_id condition in the filter.

	// 1. Filter by Platform
	if len(filters.Platforms) > 0 {
		platforms := expandFilterKeywordVariants(filters.Platforms)
		must = append(must, &pb.Condition{
			ConditionOneOf: &pb.Condition_Field{
				Field: &pb.FieldCondition{
					Key: "platform",
					Match: &pb.Match{
						MatchValue: &pb.Match_Keywords{
							Keywords: &pb.RepeatedStrings{Strings: platforms},
						},
					},
				},
			},
		})
	}

	// 3. Filter by Sentiment — payload_mapper only emits sentiment_label, so
	// the legacy overall_sentiment branch was dead weight in every search.
	if len(filters.Sentiments) > 0 {
		sentiments := expandFilterKeywordVariants(filters.Sentiments)
		must = append(must, &pb.Condition{
			ConditionOneOf: &pb.Condition_Field{
				Field: &pb.FieldCondition{
					Key: "sentiment_label",
					Match: &pb.Match{
						MatchValue: &pb.Match_Keywords{
							Keywords: &pb.RepeatedStrings{Strings: sentiments},
						},
					},
				},
			},
		})
	}

	// 4. Filter by Date Range
	if filters.DateFrom != nil || filters.DateTo != nil {
		rng := &pb.Range{}
		if filters.DateFrom != nil {
			val := float64(*filters.DateFrom)
			rng.Gte = &val
		}
		if filters.DateTo != nil {
			val := float64(*filters.DateTo)
			rng.Lte = &val
		}
		must = append(must, &pb.Condition{
			ConditionOneOf: &pb.Condition_Field{
				Field: &pb.FieldCondition{
					Key:   "content_created_at",
					Range: rng,
				},
			},
		})
	}

	// 5. Filter by Aspects (Nested)
	if len(filters.Aspects) > 0 {
		for _, aspect := range expandFilterKeywordVariants(filters.Aspects) {
			nestedFilter := &pb.Filter{
				Must: []*pb.Condition{
					{
						ConditionOneOf: &pb.Condition_Field{
							Field: &pb.FieldCondition{
								Key: "aspect",
								Match: &pb.Match{
									MatchValue: &pb.Match_Keyword{Keyword: aspect},
								},
							},
						},
					},
				},
			}
			must = append(must, &pb.Condition{
				ConditionOneOf: &pb.Condition_Nested{
					Nested: &pb.NestedCondition{
						Key:    "aspects",
						Filter: nestedFilter,
					},
				},
			})
		}
	}

	// 6. Filter by Risk Levels
	if len(filters.RiskLevels) > 0 {
		riskLevels := expandFilterKeywordVariants(filters.RiskLevels)
		must = append(must, &pb.Condition{
			ConditionOneOf: &pb.Condition_Field{
				Field: &pb.FieldCondition{
					Key: "risk_level",
					Match: &pb.Match{
						MatchValue: &pb.Match_Keywords{
							Keywords: &pb.RepeatedStrings{Strings: riskLevels},
						},
					},
				},
			},
		})
	}

	// 7. Filter by Min Engagement
	if filters.MinEngagement != nil {
		val := *filters.MinEngagement
		must = append(must, &pb.Condition{
			ConditionOneOf: &pb.Condition_Field{
				Field: &pb.FieldCondition{
					Key:   "engagement_score",
					Range: &pb.Range{Gte: &val},
				},
			},
		})
	}

	// Construct final filter
	return &pb.Filter{Must: must}
}

func expandFilterKeywordVariants(values []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(values)*3)

	add := func(value string) {
		value = strings.TrimSpace(value)
		if value == "" {
			return
		}
		if _, ok := seen[value]; ok {
			return
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}

	for _, value := range values {
		add(value)
		add(strings.ToUpper(value))
		add(strings.ToLower(value))
	}

	return out
}
