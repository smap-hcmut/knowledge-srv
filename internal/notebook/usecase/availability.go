package usecase

import (
	"context"
)

// HasSyncedForCampaign implements notebook.UseCase.
func (uc *implUseCase) HasSyncedForCampaign(ctx context.Context, campaignID string) (bool, error) {
	if uc.sourceRepo == nil || campaignID == "" {
		return false, nil
	}
	return uc.sourceRepo.HasSyncedForCampaign(ctx, campaignID)
}

// ApplyChatFallback implements notebook.UseCase (Qdrant timeout path).
func (uc *implUseCase) ApplyChatFallback(ctx context.Context, jobID, answer string) error {
	if uc.chatJobRepo == nil {
		return nil
	}
	return uc.chatJobRepo.UpdateJobStatus(ctx, jobID, "COMPLETED", nil, &answer, true)
}
