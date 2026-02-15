package scope

import (
	"context"

	"identity-srv/internal/model"
)

type PayloadCtxKey struct{}
type ScopeCtxKey struct{}
type ThirdPartyScopeKey struct{}
type SessionUserCtxKey struct{}

// SetPayloadToContext sets the payload to context
func SetPayloadToContext(ctx context.Context, payload Payload) context.Context {
	return context.WithValue(ctx, PayloadCtxKey{}, payload)
}

// GetPayloadFromContext gets the payload from context
func GetPayloadFromContext(ctx context.Context) (Payload, bool) {
	payload, ok := ctx.Value(PayloadCtxKey{}).(Payload)
	return payload, ok
}

// GetSubFromContext gets the subject from context
func GetUserIDFromContext(ctx context.Context) (string, bool) {
	payload, ok := GetPayloadFromContext(ctx)
	if !ok {
		return "", false
	}
	return payload.UserID, true
}

// GetUsernameFromContext gets the username from context
func GetUsernameFromContext(ctx context.Context) (string, bool) {
	payload, ok := GetPayloadFromContext(ctx)
	if !ok {
		return "", false
	}
	return payload.Username, true
}

func SetScopeToContext(ctx context.Context, scope model.Scope) context.Context {
	return context.WithValue(ctx, ScopeCtxKey{}, scope)
}

// GetScopeFromContext gets the scope from context
func GetScopeFromContext(ctx context.Context) (model.Scope, bool) {
	scope, ok := ctx.Value(ScopeCtxKey{}).(model.Scope)
	return scope, ok
}
