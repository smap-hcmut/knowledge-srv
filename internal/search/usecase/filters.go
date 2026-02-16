package usecase

import (
	"knowledge-srv/internal/search"

	pb "github.com/qdrant/go-client/qdrant"
)

// buildSearchFilter - Build Qdrant filter from domain filters
func (uc *implUseCase) buildSearchFilter(projectIDs []string, filters search.SearchFilters) *pb.Filter {
	must := []*pb.Condition{}

	// 1. Filter by Project IDs (must match one of them)
	if len(projectIDs) > 0 {
		must = append(must, &pb.Condition{
			ConditionOneOf: &pb.Condition_Field{
				Field: &pb.FieldCondition{
					Key: "project_id",
					Match: &pb.Match{
						MatchValue: &pb.Match_Keywords{
							Keywords: &pb.RepeatedStrings{Strings: projectIDs},
						},
					},
				},
			},
		})
	}

	// 2. Filter by Platform
	if len(filters.Platforms) > 0 {
		must = append(must, &pb.Condition{
			ConditionOneOf: &pb.Condition_Field{
				Field: &pb.FieldCondition{
					Key: "platform",
					Match: &pb.Match{
						MatchValue: &pb.Match_Keywords{
							Keywords: &pb.RepeatedStrings{Strings: filters.Platforms},
						},
					},
				},
			},
		})
	}

	// 3. Filter by Sentiment
	if len(filters.Sentiments) > 0 {
		must = append(must, &pb.Condition{
			ConditionOneOf: &pb.Condition_Field{
				Field: &pb.FieldCondition{
					Key: "overall_sentiment",
					Match: &pb.Match{
						MatchValue: &pb.Match_Keywords{
							Keywords: &pb.RepeatedStrings{Strings: filters.Sentiments},
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
		for _, aspect := range filters.Aspects {
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
		must = append(must, &pb.Condition{
			ConditionOneOf: &pb.Condition_Field{
				Field: &pb.FieldCondition{
					Key: "risk_level",
					Match: &pb.Match{
						MatchValue: &pb.Match_Keywords{
							Keywords: &pb.RepeatedStrings{Strings: filters.RiskLevels},
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
