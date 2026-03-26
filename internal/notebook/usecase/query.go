package usecase

import (
	"context"
	"fmt"

	"knowledge-srv/internal/model"
	"knowledge-srv/internal/notebook"
	"knowledge-srv/pkg/maestro"
)

// SubmitChatJob submits a chat message to a notebook and returns a job ID asynchronously.
func (uc *implUseCase) SubmitChatJob(ctx context.Context, sc model.Scope, conversationID, campaignID, userMessage string) (string, error) {
	// Ensure the notebook exists for the given campaign
	notebookInfo, err := uc.EnsureNotebook(ctx, sc, campaignID, "2023-Q1")
	if err != nil {
		return "", err
	}

	// Initialize the job in the database with PENDING status
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

	// Submit the query to the Maestro service asynchronously
	maestroJob, err := uc.maestro.ChatNotebook(ctx, activeSession, notebookInfo.ID, maestro.ChatNotebookReq{
		Prompt:        userMessage,
		WebhookURL:    webhookURL, // Webhook will process the completion
		WebhookSecret: uc.cfg.WebhookSecret,
	})
	
	if err != nil {
		// Mark the job as FAILED if Maestro submission fails
		_ = uc.chatJobRepo.UpdateJobStatus(ctx, job.ID, "FAILED", nil, nil, true)
		return "", err
	}

	// Update the database with the PROCESSING status and maestro_job_id
	err = uc.chatJobRepo.UpdateJobStatus(ctx, job.ID, "PROCESSING", &maestroJob.JobID, nil, false)
	if err != nil {
		uc.l.Errorf(ctx, "Failed to update chat job %s with maestro job ID %s: %v", job.ID, maestroJob.JobID, err)
	}

	return job.ID, nil
}

// GetChatJobStatus retrieves the status and result of a chat job.
func (uc *implUseCase) GetChatJobStatus(ctx context.Context, sc model.Scope, jobID string) (notebook.ChatJob, error) {
	return uc.chatJobRepo.GetJobByID(ctx, jobID)
}
