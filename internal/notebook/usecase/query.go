package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"knowledge-srv/internal/model"
	"knowledge-srv/internal/notebook"
	"knowledge-srv/pkg/maestro"
)

// SubmitChatJob submits a chat message to a notebook and returns a job ID asynchronously.
func (uc *implUseCase) SubmitChatJob(ctx context.Context, sc model.Scope, conversationID, campaignID, userMessage string) (string, error) {
	y, w := time.Now().UTC().ISOWeek()
	periodLabel := fmt.Sprintf("%d-W%02d", y, w)

	notebookInfo, err := uc.EnsureNotebook(ctx, sc, campaignID, periodLabel)
	if err != nil {
		return "", err
	}

	job := notebook.ChatJob{
		ConversationID: conversationID,
		CampaignID:     campaignID,
		UserMessage:    userMessage,
		Status:         "PENDING",
	}

	job, err = uc.chatJobRepo.CreateJob(ctx, job)
	if err != nil {
		return "", err
	}

	uc.mu.RLock()
	activeSession := uc.activeSession
	uc.mu.RUnlock()

	webhookURL := uc.cfg.WebhookCallbackURL
	if len(webhookURL) > 0 {
		webhookURL = fmt.Sprintf("%s?chat_job_id=%s", webhookURL, job.ID)
	}

	maestroJob, err := uc.maestro.ChatNotebook(ctx, activeSession, notebookInfo.ID, maestro.ChatNotebookReq{
		Prompt:        userMessage,
		WebhookURL:    webhookURL,
		WebhookSecret: uc.cfg.WebhookSecret,
	})

	if err != nil {
		_ = uc.chatJobRepo.UpdateJobStatus(ctx, job.ID, "FAILED", nil, nil, true)
		return "", err
	}

	err = uc.chatJobRepo.UpdateJobStatus(ctx, job.ID, "PROCESSING", &maestroJob.JobID, nil, false)
	if err != nil {
		uc.l.Errorf(ctx, "Failed to update chat job %s with maestro job ID %s: %v", job.ID, maestroJob.JobID, err)
	}

	return job.ID, nil
}

// GetChatJobStatus loads the job and, when still processing, polls Maestro once to refresh status.
func (uc *implUseCase) GetChatJobStatus(ctx context.Context, sc model.Scope, jobID string) (notebook.ChatJob, error) {
	job, err := uc.chatJobRepo.GetJobByID(ctx, jobID)
	if err != nil {
		return notebook.ChatJob{}, err
	}

	switch strings.ToUpper(strings.TrimSpace(job.Status)) {
	case "COMPLETED", "FAILED", "EXPIRED":
		return job, nil
	}

	if job.MaestroJobID == "" || uc.maestro == nil {
		return job, nil
	}

	mj, err := uc.maestro.GetJob(ctx, job.MaestroJobID)
	if err != nil {
		uc.l.Warnf(ctx, "notebook.GetChatJobStatus: maestro GetJob %s: %v", job.MaestroJobID, err)
		return job, nil
	}

	st := strings.ToLower(strings.TrimSpace(mj.Status))
	switch st {
	case "completed", "success", "done":
		answer := extractAnswerFromMaestroResult(mj.Result)
		mid := job.MaestroJobID
		_ = uc.chatJobRepo.UpdateJobStatus(ctx, jobID, "COMPLETED", &mid, &answer, false)
		job.Status = "COMPLETED"
		job.NotebookAnswer = answer
	case "failed", "error":
		mid := job.MaestroJobID
		_ = uc.chatJobRepo.UpdateJobStatus(ctx, jobID, "FAILED", &mid, nil, false)
		job.Status = "FAILED"
	}

	return job, nil
}

func extractAnswerFromMaestroResult(result any) string {
	if result == nil {
		return ""
	}
	raw, err := json.Marshal(result)
	if err != nil {
		return fmt.Sprint(result)
	}
	var m map[string]interface{}
	if err := json.Unmarshal(raw, &m); err != nil {
		return fmt.Sprint(result)
	}
	for _, k := range []string{"answer", "text", "response", "content"} {
		if v, ok := m[k].(string); ok && strings.TrimSpace(v) != "" {
			return v
		}
	}
	if nested, ok := m["data"].(map[string]interface{}); ok {
		for _, k := range []string{"answer", "text", "response"} {
			if v, ok := nested[k].(string); ok && strings.TrimSpace(v) != "" {
				return v
			}
		}
	}
	return strings.TrimSpace(fmt.Sprint(result))
}
