package postgre

import (
	"context"
	"fmt"
	"time"

	"knowledge-srv/internal/chat/repository"
	"knowledge-srv/internal/model"

	"github.com/google/uuid"
)

// CreateMessage - Tạo message mới
func (r *implRepository) CreateMessage(ctx context.Context, opt repository.CreateMessageOptions) (model.Message, error) {
	id := uuid.New().String()
	now := time.Now()

	query := `
		INSERT INTO knowledge.messages (id, conversation_id, role, content, citations, search_metadata, suggestions, filters_used, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, conversation_id, role, content, citations, search_metadata, suggestions, filters_used, created_at
	`

	var msg model.Message

	err := r.db.QueryRowContext(ctx, query,
		id, opt.ConversationID, opt.Role, opt.Content,
		nullJSON(opt.Citations), nullJSON(opt.SearchMetadata),
		nullJSON(opt.Suggestions), nullJSON(opt.FiltersUsed),
		now,
	).Scan(
		&msg.ID, &msg.ConversationID, &msg.Role, &msg.Content,
		&msg.Citations, &msg.SearchMetadata, &msg.Suggestions, &msg.FiltersUsed,
		&msg.CreatedAt,
	)
	if err != nil {
		return model.Message{}, fmt.Errorf("CreateMessage: %w", err)
	}

	return msg, nil
}

// ListMessages - Liệt kê messages theo conversation
func (r *implRepository) ListMessages(ctx context.Context, opt repository.ListMessagesOptions) ([]model.Message, error) {
	query := `
		SELECT id, conversation_id, role, content, citations, search_metadata, suggestions, filters_used, created_at
		FROM knowledge.messages
		WHERE conversation_id = $1
	`

	order := "DESC"
	if opt.OrderASC {
		order = "ASC"
	}
	query += fmt.Sprintf(" ORDER BY created_at %s", order)

	if opt.Limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", opt.Limit)
	}

	rows, err := r.db.QueryContext(ctx, query, opt.ConversationID)
	if err != nil {
		return nil, fmt.Errorf("ListMessages: %w", err)
	}
	defer rows.Close()

	var messages []model.Message
	for rows.Next() {
		var msg model.Message
		if err := rows.Scan(
			&msg.ID, &msg.ConversationID, &msg.Role, &msg.Content,
			&msg.Citations, &msg.SearchMetadata, &msg.Suggestions, &msg.FiltersUsed,
			&msg.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("ListMessages scan: %w", err)
		}
		messages = append(messages, msg)
	}

	return messages, rows.Err()
}

// nullJSON - Convert empty/nil json.RawMessage to database-compatible value
func nullJSON(data []byte) interface{} {
	if len(data) == 0 || string(data) == "null" {
		return nil
	}
	return data
}
