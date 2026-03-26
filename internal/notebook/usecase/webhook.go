package usecase

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"knowledge-srv/internal/model"
	"knowledge-srv/internal/notebook"
	"knowledge-srv/internal/notebook/repository"
)

// HandleWebhook processes webhook callbacks from Maestro.
func (uc *implUseCase) HandleWebhook(ctx context.Context, sc model.Scope, payload notebook.WebhookPayload) error {
	switch payload.Action {
	case "upload_sources":
		return uc.handleUploadSourcesWebhook(ctx, payload)
	case "create_notebook":
		uc.l.Infof(ctx, "Received create_notebook webhook (campaign already persisted by sync): action=%s jobId=%s", payload.Action, payload.JobID)
		return nil
	case "chat":
		// Phase 3: Update notebook_chat_jobs
		return uc.handleChatWebhook(ctx, payload)
	default:
		return fmt.Errorf("unknown webhook action: %s", payload.Action)
	}
}

func (uc *implUseCase) handleUploadSourcesWebhook(ctx context.Context, payload notebook.WebhookPayload) error {
	if uc.sourceRepo == nil {
		return nil
	}
	jobID := payload.JobID
	if jobID == "" && payload.Data != nil {
		if v, ok := payload.Data["jobId"].(string); ok {
			jobID = v
		}
		if jobID == "" {
			if v, ok := payload.Data["job_id"].(string); ok {
				jobID = v
			}
		}
	}
	if jobID == "" {
		return fmt.Errorf("upload_sources webhook: missing job id")
	}
	status := strings.ToLower(strings.TrimSpace(payload.Status))
	if status == "" && payload.Data != nil {
		if v, ok := payload.Data["status"].(string); ok {
			status = strings.ToLower(strings.TrimSpace(v))
		}
	}
	var errMsg *string
	st := "UPLOADING"
	switch status {
	case "completed", "success", "done":
		st = "SYNCED"
	case "failed", "error":
		st = "FAILED"
		if payload.Data != nil {
			s := fmt.Sprintf("%v", payload.Data["error"])
			errMsg = &s
		}
	default:
		return nil
	}
	err := uc.sourceRepo.UpdateStatusByMaestroJobID(ctx, jobID, st, errMsg)
	if err != nil && errors.Is(err, repository.ErrNotFound) {
		uc.l.Warnf(ctx, "upload_sources webhook: no row for maestro job %s", jobID)
		return nil
	}
	return err
}

func (uc *implUseCase) handleChatWebhook(ctx context.Context, payload notebook.WebhookPayload) error {
	if payload.Data == nil {
		return fmt.Errorf("invalid chat webhook data: missing data")
	}
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
