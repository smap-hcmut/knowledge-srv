package postgre

import (
	"context"

	"github.com/aarondl/sqlboiler/v4/boil"

	"knowledge-srv/internal/chat/repository"
	"knowledge-srv/internal/model"
	"knowledge-srv/internal/sqlboiler"
	"knowledge-srv/pkg/util"
)

// CreateMessage - Insert message record (returns created entity)
func (r *implRepository) CreateMessage(ctx context.Context, opt repository.CreateMessageOptions) (model.Message, error) {
	dbMsg := buildCreateMessage(opt)

	if err := dbMsg.Insert(ctx, r.db, boil.Infer()); err != nil {
		r.l.Errorf(ctx, "chat.repository.postgre.CreateMessage: Failed to insert message: %v", err)
		return model.Message{}, repository.ErrFailedToInsert
	}

	if msg := model.NewMessageFromDB(dbMsg); msg != nil {
		return *msg, nil
	}
	return model.Message{}, nil
}

// ListMessages - List messages by conversation
func (r *implRepository) ListMessages(ctx context.Context, opt repository.ListMessagesOptions) ([]model.Message, error) {
	mods := r.buildListMessagesQuery(opt)

	dbMsgs, err := sqlboiler.Messages(mods...).All(ctx, r.db)
	if err != nil {
		r.l.Errorf(ctx, "chat.repository.postgre.ListMessages: Failed to list messages: %v", err)
		return nil, repository.ErrFailedToList
	}

	return util.MapSlice(dbMsgs, model.NewMessageFromDB), nil
}
