package consumer

import (
	"knowledge-srv/internal/indexing"
	kafkaDelivery "knowledge-srv/internal/indexing/delivery/kafka"
)

// toIndexInput maps legacy Kafka message DTO to usecase input.
func toIndexInput(m kafkaDelivery.LegacyBatchCompletedMessage) indexing.IndexInput {
	return indexing.IndexInput{
		BatchID:     m.BatchID,
		ProjectID:   m.ProjectID,
		CampaignID:  m.CampaignID,
		FileURL:     m.FileURL,
		RecordCount: m.RecordCount,
	}
}

// toIndexBatchInput maps new Kafka message DTO to usecase input.
func toIndexBatchInput(m kafkaDelivery.BatchCompletedMessage) indexing.IndexBatchInput {
	docs := make([]indexing.InsightMessageInput, len(m.Documents))
	for i, d := range m.Documents {
		aspects := make([]indexing.InsightAspectInput, len(d.NLP.Aspects))
		for j, a := range d.NLP.Aspects {
			aspects[j] = indexing.InsightAspectInput{
				Aspect:   a.Aspect,
				Polarity: a.Polarity,
			}
		}

		entities := make([]indexing.InsightEntityInput, len(d.NLP.Entities))
		for j, e := range d.NLP.Entities {
			entities[j] = indexing.InsightEntityInput{
				Type:  e.Type,
				Value: e.Value,
			}
		}

		docs[i] = indexing.InsightMessageInput{
			Identity: indexing.InsightIdentityInput{
				UapID:        d.Identity.UapID,
				UapType:      d.Identity.UapType,
				UapMediaType: d.Identity.UapMediaType,
				Platform:     d.Identity.Platform,
				PublishedAt:  d.Identity.PublishedAt,
			},
			Content: indexing.InsightContentInput{
				CleanText: d.Content.CleanText,
				Summary:   d.Content.Summary,
			},
			NLP: indexing.InsightNLPInput{
				Sentiment: indexing.InsightSentimentInput{
					Label: d.NLP.Sentiment.Label,
					Score: d.NLP.Sentiment.Score,
				},
				Aspects:  aspects,
				Entities: entities,
			},
			Business: indexing.InsightBusinessInput{
				Impact: indexing.InsightImpactInput{
					Engagement: indexing.InsightEngagementInput{
						Likes:    d.Business.Impact.Engagement.Likes,
						Comments: d.Business.Impact.Engagement.Comments,
						Shares:   d.Business.Impact.Engagement.Shares,
						Views:    d.Business.Impact.Engagement.Views,
					},
					ImpactScore: d.Business.Impact.ImpactScore,
					Priority:    d.Business.Impact.Priority,
				},
			},
			RAG: d.RAG,
		}
	}

	return indexing.IndexBatchInput{
		ProjectID:  m.ProjectID,
		CampaignID: m.CampaignID,
		Documents:  docs,
	}
}

func toIndexInsightInput(m kafkaDelivery.InsightsPublishedMessage) indexing.IndexInsightInput {
	return indexing.IndexInsightInput{
		ProjectID:           m.ProjectID,
		CampaignID:          m.CampaignID,
		RunID:               m.RunID,
		InsightType:         m.InsightType,
		Title:               m.Title,
		Summary:             m.Summary,
		Confidence:          m.Confidence,
		AnalysisWindowStart: m.AnalysisWindowStart,
		AnalysisWindowEnd:   m.AnalysisWindowEnd,
		SupportingMetrics:   m.SupportingMetrics,
		EvidenceReferences:  m.EvidenceReferences,
	}
}

func toIndexDigestInput(m kafkaDelivery.ReportDigestMessage) indexing.IndexDigestInput {
	entities := make([]indexing.TopEntityInput, len(m.TopEntities))
	for i, e := range m.TopEntities {
		entities[i] = indexing.TopEntityInput{
			CanonicalEntityID: e.CanonicalEntityID,
			EntityName:        e.EntityName,
			EntityType:        e.EntityType,
			MentionCount:      e.MentionCount,
			MentionShare:      e.MentionShare,
		}
	}

	topics := make([]indexing.TopTopicInput, len(m.TopTopics))
	for i, t := range m.TopTopics {
		topics[i] = indexing.TopTopicInput{
			TopicKey:            t.TopicKey,
			TopicLabel:          t.TopicLabel,
			MentionCount:        t.MentionCount,
			MentionShare:        t.MentionShare,
			BuzzScoreProxy:      t.BuzzScoreProxy,
			QualityScore:        t.QualityScore,
			RepresentativeTexts: t.RepresentativeTexts,
		}
	}

	issues := make([]indexing.TopIssueInput, len(m.TopIssues))
	for i, iss := range m.TopIssues {
		var severityMix *indexing.SeverityMixInput
		if iss.SeverityMix != nil {
			severityMix = &indexing.SeverityMixInput{
				Low:    iss.SeverityMix.Low,
				Medium: iss.SeverityMix.Medium,
				High:   iss.SeverityMix.High,
			}
		}
		issues[i] = indexing.TopIssueInput{
			IssueCategory:      iss.IssueCategory,
			MentionCount:       iss.MentionCount,
			IssuePressureProxy: iss.IssuePressureProxy,
			SeverityMix:        severityMix,
		}
	}

	return indexing.IndexDigestInput{
		ProjectID:           m.ProjectID,
		CampaignID:          m.CampaignID,
		RunID:               m.RunID,
		AnalysisWindowStart: m.AnalysisWindowStart,
		AnalysisWindowEnd:   m.AnalysisWindowEnd,
		DomainOverlay:       m.DomainOverlay,
		Platform:            m.Platform,
		TotalMentions:       m.TotalMentions,
		TopEntities:         entities,
		TopTopics:           topics,
		TopIssues:           issues,
	}
}
