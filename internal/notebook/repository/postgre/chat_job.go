package postgre

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"knowledge-srv/internal/notebook"
	"knowledge-srv/internal/notebook/repository"
)

type chatJobRepo struct {
	db *sql.DB
}

// NewChatJobRepo creates a new chat job repository.
func NewChatJobRepo(db *sql.DB) repository.ChatJobRepo {
	return &chatJobRepo{db: db}
}

func (r *chatJobRepo) CreateJob(ctx context.Context, job notebook.ChatJob) (notebook.ChatJob, error) {
	query := `
		INSERT INTO notebook_chat_jobs (
			conversation_id, campaign_id, user_message, status, fallback_used
		) VALUES (
			$1, $2, $3, $4, false
		) RETURNING id, created_at, expires_at
	`

	err := r.db.QueryRowContext(ctx, query,
		job.ConversationID,
		job.CampaignID,
		job.UserMessage,
		job.Status,
	).Scan(&job.ID, &job.CreatedAt, &job.ExpiresAt)

	if err != nil {
		return notebook.ChatJob{}, fmt.Errorf("failed to create chat job: %w", err)
	}

	return job, nil
}

func (r *chatJobRepo) GetJobByID(ctx context.Context, jobID string) (notebook.ChatJob, error) {
	query := `
		SELECT 
			id, conversation_id, campaign_id, user_message, maestro_job_id, 
			status, notebook_answer, fallback_used, created_at, completed_at, expires_at
		FROM notebook_chat_jobs
		WHERE id = $1
	`

	var job notebook.ChatJob
	var maestroJobID, notebookAnswer sql.NullString
	var completedAt sql.NullTime

	var createdAt, expiresAt time.Time

	err := r.db.QueryRowContext(ctx, query, jobID).Scan(
		&job.ID, &job.ConversationID, &job.CampaignID, &job.UserMessage, &maestroJobID,
		&job.Status, &notebookAnswer, &job.FallbackUsed, &createdAt, &completedAt, &expiresAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return notebook.ChatJob{}, errors.New("chat job not found")
		}
		return notebook.ChatJob{}, fmt.Errorf("failed to get chat job: %w", err)
	}

	if maestroJobID.Valid {
		job.MaestroJobID = maestroJobID.String
	}
	if notebookAnswer.Valid {
		job.NotebookAnswer = notebookAnswer.String
	}

	job.CreatedAt = createdAt.Format(time.RFC3339)
	job.ExpiresAt = expiresAt.Format(time.RFC3339)
	if completedAt.Valid {
		parsed := completedAt.Time.Format(time.RFC3339)
		job.CompletedAt = &parsed
	}

	return job, nil
}

func (r *chatJobRepo) UpdateJobStatus(ctx context.Context, jobID, status string, maestroJobID, answer *string, fallbackUsed bool) error {
	query := `
		UPDATE notebook_chat_jobs
		SET status = $1, maestro_job_id = COALESCE($2, maestro_job_id), notebook_answer = COALESCE($3, notebook_answer), fallback_used = $4
	`
	
	args := []interface{}{status, maestroJobID, answer, fallbackUsed}
	
	if status == "COMPLETED" || status == "FAILED" || status == "EXPIRED" {
		query += `, completed_at = NOW()`
	}

	query += ` WHERE id = $5`
	args = append(args, jobID)

	_, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to update chat job status: %w", err)
	}
	return nil
}
