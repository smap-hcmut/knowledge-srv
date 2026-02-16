package postgre

import (
	"context"
	"database/sql"
	"time"

	"github.com/aarondl/null/v8"
	"github.com/aarondl/sqlboiler/v4/boil"

	"knowledge-srv/internal/chat/repository"
	"knowledge-srv/internal/model"
	"knowledge-srv/internal/sqlboiler"
	"knowledge-srv/pkg/util"
)

// CreateConversation - Insert conversation record (returns created entity)
func (r *implRepository) CreateConversation(ctx context.Context, opt repository.CreateConversationOptions) (model.Conversation, error) {
	dbConv := buildCreateConversation(opt)

	if err := dbConv.Insert(ctx, r.db, boil.Infer()); err != nil {
		r.l.Errorf(ctx, "chat.repository.postgre.CreateConversation: Failed to insert conversation: %v", err)
		return model.Conversation{}, repository.ErrFailedToInsert
	}

	if conv := model.NewConversationFromDB(dbConv); conv != nil {
		return *conv, nil
	}
	return model.Conversation{}, nil
}

// GetConversationByID - Get conversation by primary key
func (r *implRepository) GetConversationByID(ctx context.Context, id string) (model.Conversation, error) {
	dbConv, err := sqlboiler.FindConversation(ctx, r.db, id)
	if err == sql.ErrNoRows {
		return model.Conversation{}, nil // Not found
	}
	if err != nil {
		r.l.Errorf(ctx, "chat.repository.postgre.GetConversationByID: Failed to get conversation: %v", err)
		return model.Conversation{}, repository.ErrFailedToGet
	}

	if conv := model.NewConversationFromDB(dbConv); conv != nil {
		return *conv, nil
	}
	return model.Conversation{}, nil
}

// ListConversations - List conversations by campaign + user
func (r *implRepository) ListConversations(ctx context.Context, opt repository.ListConversationsOptions) ([]model.Conversation, error) {
	mods := r.buildListConversationsQuery(opt)

	dbConvs, err := sqlboiler.Conversations(mods...).All(ctx, r.db)
	if err != nil {
		r.l.Errorf(ctx, "chat.repository.postgre.ListConversations: Failed to list conversations: %v", err)
		return nil, repository.ErrFailedToList
	}

	return util.MapSlice(dbConvs, model.NewConversationFromDB), nil
}

// UpdateConversationLastMessage - Update message_count and last_message_at
func (r *implRepository) UpdateConversationLastMessage(ctx context.Context, opt repository.UpdateLastMessageOptions) error {
	dbConv, err := sqlboiler.FindConversation(ctx, r.db, opt.ConversationID)
	if err != nil {
		r.l.Errorf(ctx, "chat.repository.postgre.UpdateConversationLastMessage: Failed to find conversation: %v", err)
		return repository.ErrFailedToUpdate
	}

	now := time.Now()
	dbConv.MessageCount = opt.MessageCount
	dbConv.LastMessageAt = null.TimeFrom(now)
	dbConv.UpdatedAt = null.TimeFrom(now)

	_, err = dbConv.Update(ctx, r.db, boil.Infer())
	if err != nil {
		r.l.Errorf(ctx, "chat.repository.postgre.UpdateConversationLastMessage: Failed to update conversation: %v", err)
		return repository.ErrFailedToUpdate
	}

	return nil
}

// ArchiveConversation - Archive conversation by setting status to ARCHIVED
func (r *implRepository) ArchiveConversation(ctx context.Context, id string) error {
	dbConv, err := sqlboiler.FindConversation(ctx, r.db, id)
	if err != nil {
		r.l.Errorf(ctx, "chat.repository.postgre.ArchiveConversation: Failed to find conversation: %v", err)
		return repository.ErrFailedToUpdate
	}

	dbConv.Status = "ARCHIVED"
	dbConv.UpdatedAt = null.TimeFrom(time.Now())

	_, err = dbConv.Update(ctx, r.db, boil.Infer())
	if err != nil {
		r.l.Errorf(ctx, "chat.repository.postgre.ArchiveConversation: Failed to archive conversation: %v", err)
		return repository.ErrFailedToUpdate
	}

	return nil
}
