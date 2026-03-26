package usecase

import (
	"context"
	"errors"
	"net/url"
	"strings"

	"knowledge-srv/internal/model"
	"knowledge-srv/internal/notebook"
	"knowledge-srv/internal/notebook/repository"
	"knowledge-srv/pkg/maestro"
)

func (uc *implUseCase) SyncPart(ctx context.Context, sc model.Scope, input notebook.SyncPartInput) error {
	if input.CampaignID == "" || len(input.Parts) == 0 {
		return nil
	}
	for i := range input.Parts {
		if err := uc.syncOnePart(ctx, input.CampaignID, input.Parts[i]); err != nil {
			return err
		}
	}
	return nil
}

func (uc *implUseCase) syncOnePart(ctx context.Context, campaignID string, part notebook.MarkdownPart) error {
	if uc.sourceRepo == nil {
		return notebook.ErrInvalidInput
	}
	rec, err := uc.sourceRepo.GetByContentHash(ctx, campaignID, part.ContentHash)
	if err == nil && strings.EqualFold(rec.Status, "SYNCED") {
		return nil
	}
	if err != nil && !errors.Is(err, repository.ErrNotFound) {
		return err
	}

	info, err := uc.EnsureNotebook(ctx, model.Scope{}, campaignID, part.WeekLabel)
	if err != nil {
		return err
	}

	sessionID := uc.getActiveSessionID()
	if sessionID == "" {
		return notebook.ErrSessionNotFound
	}

	webhookURL := uc.buildUploadSourcesWebhookURL(campaignID)

	job, err := uc.maestro.UploadSources(ctx, sessionID, info.ID, maestro.UploadSourcesReq{
		SourceType: "text",
		Sources: []maestro.SourceItem{
			{Title: part.Title, Content: part.Content},
		},
		WebhookURL:    webhookURL,
		WebhookSecret: uc.cfg.WebhookSecret,
	})
	if err != nil {
		return err
	}

	return uc.sourceRepo.CreateUploading(ctx, notebook.SourceUpsertInput{
		CampaignID:   campaignID,
		NotebookID:   info.ID,
		WeekLabel:    part.WeekLabel,
		PartNumber:   part.PartNum,
		Title:        part.Title,
		PostCount:    part.PostCount,
		ContentHash:  part.ContentHash,
		MaestroJobID: job.JobID,
		Status:       "UPLOADING",
	})
}

func (uc *implUseCase) buildUploadSourcesWebhookURL(campaignID string) string {
	base := uc.cfg.WebhookCallbackURL
	if base == "" {
		return ""
	}
	u, err := url.Parse(base)
	if err != nil {
		return base
	}
	q := u.Query()
	q.Set("action", "upload_sources")
	q.Set("campaign_id", campaignID)
	u.RawQuery = q.Encode()
	return u.String()
}

func (uc *implUseCase) RetryFailed(ctx context.Context, sc model.Scope) (notebook.RetryOutput, error) {
	if uc.sourceRepo == nil {
		return notebook.RetryOutput{}, notebook.ErrInvalidInput
	}
	max := uc.cfg.SyncMaxRetries
	if max <= 0 {
		max = 3
	}
	_, err := uc.sourceRepo.ListFailedRetryable(ctx, max)
	if err != nil {
		return notebook.RetryOutput{}, err
	}
	// Content is not persisted on notebook_sources; full retry needs re-running transform+sync from consumer.
	return notebook.RetryOutput{}, nil
}
