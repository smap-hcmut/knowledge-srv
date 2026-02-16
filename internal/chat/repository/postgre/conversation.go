package postgre

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"knowledge-srv/internal/chat/repository"
	"knowledge-srv/internal/model"

	"github.com/google/uuid"
)

// CreateConversation - Tạo conversation mới
func (r *implRepository) CreateConversation(ctx context.Context, opt repository.CreateConversationOptions) (model.Conversation, error) {
	id := uuid.New().String()
	now := time.Now()

	query := `
		INSERT INTO knowledge.conversations (id, campaign_id, user_id, title, status, message_count, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, campaign_id, user_id, title, status, message_count, last_message_at, created_at, updated_at
	`

	var conv model.Conversation
	var lastMessageAt sql.NullTime

	err := r.db.QueryRowContext(ctx, query,
		id, opt.CampaignID, opt.UserID, opt.Title, "ACTIVE", 0, now, now,
	).Scan(
		&conv.ID, &conv.CampaignID, &conv.UserID, &conv.Title,
		&conv.Status, &conv.MessageCount, &lastMessageAt,
		&conv.CreatedAt, &conv.UpdatedAt,
	)
	if err != nil {
		return model.Conversation{}, fmt.Errorf("CreateConversation: %w", err)
	}

	if lastMessageAt.Valid {
		conv.LastMessageAt = &lastMessageAt.Time
	}

	return conv, nil
}

// GetConversationByID - Lấy conversation theo ID
func (r *implRepository) GetConversationByID(ctx context.Context, id string) (model.Conversation, error) {
	query := `
		SELECT id, campaign_id, user_id, title, status, message_count, last_message_at, created_at, updated_at
		FROM knowledge.conversations
		WHERE id = $1
	`

	var conv model.Conversation
	var lastMessageAt sql.NullTime

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&conv.ID, &conv.CampaignID, &conv.UserID, &conv.Title,
		&conv.Status, &conv.MessageCount, &lastMessageAt,
		&conv.CreatedAt, &conv.UpdatedAt,
	)
	if err != nil {
		return model.Conversation{}, fmt.Errorf("GetConversationByID: %w", err)
	}

	if lastMessageAt.Valid {
		conv.LastMessageAt = &lastMessageAt.Time
	}

	return conv, nil
}

// ListConversations - Liệt kê conversations theo campaign + user
func (r *implRepository) ListConversations(ctx context.Context, opt repository.ListConversationsOptions) ([]model.Conversation, error) {
	query := `
		SELECT id, campaign_id, user_id, title, status, message_count, last_message_at, created_at, updated_at
		FROM knowledge.conversations
		WHERE campaign_id = $1 AND user_id = $2
	`
	args := []interface{}{opt.CampaignID, opt.UserID}
	argIdx := 3

	if opt.Status != "" {
		query += fmt.Sprintf(" AND status = $%d", argIdx)
		args = append(args, opt.Status)
		argIdx++
	}

	query += " ORDER BY last_message_at DESC NULLS LAST, created_at DESC"

	if opt.Limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", argIdx)
		args = append(args, opt.Limit)
		argIdx++
	}
	if opt.Offset > 0 {
		query += fmt.Sprintf(" OFFSET $%d", argIdx)
		args = append(args, opt.Offset)
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("ListConversations: %w", err)
	}
	defer rows.Close()

	var conversations []model.Conversation
	for rows.Next() {
		var conv model.Conversation
		var lastMessageAt sql.NullTime

		if err := rows.Scan(
			&conv.ID, &conv.CampaignID, &conv.UserID, &conv.Title,
			&conv.Status, &conv.MessageCount, &lastMessageAt,
			&conv.CreatedAt, &conv.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("ListConversations scan: %w", err)
		}

		if lastMessageAt.Valid {
			conv.LastMessageAt = &lastMessageAt.Time
		}
		conversations = append(conversations, conv)
	}

	return conversations, rows.Err()
}

// UpdateConversationLastMessage - Cập nhật message_count và last_message_at
func (r *implRepository) UpdateConversationLastMessage(ctx context.Context, opt repository.UpdateLastMessageOptions) error {
	now := time.Now()
	query := `
		UPDATE knowledge.conversations
		SET message_count = $1, last_message_at = $2, updated_at = $3
		WHERE id = $4
	`

	_, err := r.db.ExecContext(ctx, query, opt.MessageCount, now, now, opt.ConversationID)
	if err != nil {
		return fmt.Errorf("UpdateConversationLastMessage: %w", err)
	}
	return nil
}

// ArchiveConversation - Archive conversation
func (r *implRepository) ArchiveConversation(ctx context.Context, id string) error {
	query := `
		UPDATE knowledge.conversations
		SET status = 'ARCHIVED', updated_at = $1
		WHERE id = $2
	`

	_, err := r.db.ExecContext(ctx, query, time.Now(), id)
	if err != nil {
		return fmt.Errorf("ArchiveConversation: %w", err)
	}
	return nil
}
