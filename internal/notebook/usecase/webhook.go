package usecase

import (
	"context"
	"fmt"

	"knowledge-srv/internal/model"
	"knowledge-srv/internal/notebook"
)

// HandleWebhook processes webhook callbacks from Maestro.
func (uc *implUseCase) HandleWebhook(ctx context.Context, sc model.Scope, payload notebook.WebhookPayload) error {
	switch payload.Action {
	case "upload_sources":
		// Phase 2: Update notebook_sources
		uc.l.Infof(ctx, "Received upload_sources webhook: %v", payload.Data)
		return nil
	case "create_notebook":
		// Phase 2: Update notebook_campaigns
		uc.l.Infof(ctx, "Received create_notebook webhook: %v", payload.Data)
		return nil
	case "chat":
		// Phase 3: Update notebook_chat_jobs
		return uc.handleChatWebhook(ctx, payload)
	default:
		return fmt.Errorf("unknown webhook action: %s", payload.Action)
	}
}

func (uc *implUseCase) handleChatWebhook(ctx context.Context, payload notebook.WebhookPayload) error {
	// Extract internal job ID from payload if we appended it, or handle it via maestro_job_id
	// Maestro sends its own job_id in Data["job_id"]
	internalJobID, ok := payload.Data["chat_job_id"].(string)
	if !ok {
		// Fallback to maestro_job_id if we have a repo method (which we haven't implemented yet, but for now we require chat_job_id in webhook data)
		return fmt.Errorf("invalid chat webhook data: missing internal chat_job_id")
	}

	status, ok := payload.Data["status"].(string)
	if !ok {
		return fmt.Errorf("invalid chat webhook data: missing status")
	}

	var pMaestroJobID *string
	if maestroJobID, ok := payload.Data["job_id"].(string); ok && maestroJobID != "" {
		pMaestroJobID = &maestroJobID
	}

	var answer *string
	if ans, ok := payload.Data["answer"].(string); ok {
		answer = &ans
	}

	// Translate maestro status to our status
	// StatusReady = "ready", JobCompleted = "completed", JobFailed = "failed"
	var mappedStatus string
	switch status {
	case "completed":
		mappedStatus = "COMPLETED"
	case "failed":
		mappedStatus = "FAILED"
	default:
		mappedStatus = "PROCESSING"
	}

	err := uc.chatJobRepo.UpdateJobStatus(ctx, internalJobID, mappedStatus, pMaestroJobID, answer, false)
	if err != nil {
		return fmt.Errorf("failed to update chat job status via webhook: %w", err)
	}

	uc.l.Infof(ctx, "Successfully processed chat webhook for internal job %s", internalJobID)
	return nil
}
