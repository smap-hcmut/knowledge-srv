package usecase

import (
	"context"

	"knowledge-srv/internal/indexing"
)

// GetStatistics - Lấy thống kê indexing cho monitoring
func (uc *implUseCase) GetStatistics(ctx context.Context, projectID string) (indexing.StatisticOutput, error) {
	stats, err := uc.repo.CountDocumentsByProject(ctx, projectID)
	if err != nil {
		uc.l.Errorf(ctx, "indexing.usecase.GetStatistics: Failed to count documents by project: %v", err)
		return indexing.StatisticOutput{}, err
	}

	return indexing.StatisticOutput{
		ProjectID:      projectID,
		TotalIndexed:   stats.TotalIndexed,
		TotalFailed:    stats.TotalFailed,
		TotalPending:   stats.TotalPending,
		LastIndexedAt:  stats.LastIndexedAt,
		AvgIndexTimeMs: stats.AvgIndexTimeMs,
	}, nil
}
