package usecase

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"knowledge-srv/internal/model"
	"knowledge-srv/internal/notebook"
	"knowledge-srv/internal/notebook/repository"
	"knowledge-srv/pkg/maestro"
)

func (uc *implUseCase) EnsureNotebook(ctx context.Context, sc model.Scope, campaignID, periodLabel string) (notebook.NotebookInfo, error) {
	if uc.campaignRepo == nil {
		return notebook.NotebookInfo{}, notebook.ErrInvalidInput
	}
	if campaignID == "" || periodLabel == "" {
		return notebook.NotebookInfo{}, notebook.ErrInvalidInput
	}
	info, err := uc.campaignRepo.GetByCampaignAndPeriod(ctx, campaignID, periodLabel)
	if err == nil {
		return info, nil
	}
	if !errors.Is(err, repository.ErrNotFound) {
		return notebook.NotebookInfo{}, err
	}

	sessionID := uc.getActiveSessionID()
	if sessionID == "" {
		return notebook.NotebookInfo{}, notebook.ErrSessionNotFound
	}

	title := fmt.Sprintf("SMAP | %s | %s", campaignID, periodLabel)
	webhookURL := uc.buildCreateNotebookWebhookURL(campaignID, periodLabel)

	job, err := uc.maestro.CreateNotebook(ctx, sessionID, maestro.CreateNotebookReq{
		Title:         title,
		WebhookURL:    webhookURL,
		WebhookSecret: uc.cfg.WebhookSecret,
	})
	if err != nil {
		return notebook.NotebookInfo{}, err
	}

	jd, err := uc.pollJobUntilDone(ctx, job.JobID)
	if err != nil {
		return notebook.NotebookInfo{}, err
	}

	notebookID, err := extractNotebookIDFromJobResult(jd.Result)
	if err != nil {
		return notebook.NotebookInfo{}, fmt.Errorf("create notebook job: %w", err)
	}

	if err := uc.campaignRepo.Create(ctx, campaignID, periodLabel, notebookID); err != nil {
		return notebook.NotebookInfo{}, err
	}

	return notebook.NotebookInfo{
		ID:          notebookID,
		CampaignID:  campaignID,
		PeriodLabel: periodLabel,
		CreatedAt:   time.Now().UTC().Format(time.RFC3339),
	}, nil
}

func (uc *implUseCase) buildCreateNotebookWebhookURL(campaignID, periodLabel string) string {
	base := uc.cfg.WebhookCallbackURL
	if base == "" {
		return ""
	}
	u, err := url.Parse(base)
	if err != nil {
		return base
	}
	q := u.Query()
	q.Set("action", "create_notebook")
	q.Set("campaign_id", campaignID)
	q.Set("period_label", periodLabel)
	u.RawQuery = q.Encode()
	return u.String()
}

func (uc *implUseCase) getActiveSessionID() string {
	uc.mu.RLock()
	defer uc.mu.RUnlock()
	return uc.activeSession
}

func (uc *implUseCase) pollJobUntilDone(ctx context.Context, jobID string) (maestro.JobData, error) {
	interval := time.Duration(uc.cfg.JobPollIntervalMs) * time.Millisecond
	if interval <= 0 {
		interval = 2 * time.Second
	}
	maxAttempts := uc.cfg.JobPollMaxAttempts
	if maxAttempts <= 0 {
		maxAttempts = 30
	}
	var last maestro.JobData
	for i := 0; i < maxAttempts; i++ {
		j, err := uc.maestro.GetJob(ctx, jobID)
		if err != nil {
			return maestro.JobData{}, err
		}
		last = j
		st := strings.ToLower(strings.TrimSpace(j.Status))
		switch st {
		case "completed", "success", "done":
			return j, nil
		case "failed", "error":
			return j, fmt.Errorf("%w: %v", notebook.ErrJobFailed, j.Error)
		}
		select {
		case <-ctx.Done():
			return maestro.JobData{}, ctx.Err()
		case <-time.After(interval):
		}
	}
	return last, notebook.ErrJobTimeout
}

func extractNotebookIDFromJobResult(result any) (string, error) {
	if result == nil {
		return "", fmt.Errorf("empty job result")
	}
	raw, err := json.Marshal(result)
	if err != nil {
		return "", err
	}
	var m map[string]interface{}
	if err := json.Unmarshal(raw, &m); err != nil {
		return "", err
	}
	for _, k := range []string{"notebookId", "notebook_id", "notebookID"} {
		if v, ok := m[k].(string); ok && v != "" {
			return v, nil
		}
	}
	if nested, ok := m["data"].(map[string]interface{}); ok {
		for _, k := range []string{"notebookId", "notebook_id"} {
			if v, ok := nested[k].(string); ok && v != "" {
				return v, nil
			}
		}
	}
	return "", fmt.Errorf("notebook id not found in job result")
}
