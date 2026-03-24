package maestro

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// CreateSession creates a new browser automation session.
func (m *maestroImpl) CreateSession(ctx context.Context, req CreateSessionReq) (SessionData, error) {
	path := PathSessions
	body, statusCode, err := m.doPost(ctx, path, "", req)
	if err != nil {
		return SessionData{}, fmt.Errorf("create session: %w", err)
	}
	if err := m.checkStatus(statusCode, body); err != nil {
		return SessionData{}, fmt.Errorf("create session: %w", err)
	}

	var resp SuccessResponse[SessionData]
	if err := json.Unmarshal(body, &resp); err != nil {
		return SessionData{}, fmt.Errorf("create session: unmarshal: %w", err)
	}
	return resp.Data, nil
}

// GetSession retrieves the status of an existing session.
func (m *maestroImpl) GetSession(ctx context.Context, sessionID string) (SessionData, error) {
	path := fmt.Sprintf("%s/%s", PathSessions, sessionID)
	body, statusCode, err := m.doGet(ctx, path, "")
	if err != nil {
		return SessionData{}, fmt.Errorf("get session: %w", err)
	}
	if statusCode == http.StatusNotFound {
		return SessionData{}, ErrSessionNotFound
	}
	if err := m.checkStatus(statusCode, body); err != nil {
		return SessionData{}, fmt.Errorf("get session: %w", err)
	}

	var resp SuccessResponse[SessionData]
	if err := json.Unmarshal(body, &resp); err != nil {
		return SessionData{}, fmt.Errorf("get session: unmarshal: %w", err)
	}
	return resp.Data, nil
}

// DeleteSession tears down a browser session.
func (m *maestroImpl) DeleteSession(ctx context.Context, sessionID string) error {
	path := fmt.Sprintf("%s/%s", PathSessions, sessionID)
	body, statusCode, err := m.doDelete(ctx, path)
	if err != nil {
		return fmt.Errorf("delete session: %w", err)
	}
	if statusCode == http.StatusNotFound {
		return ErrSessionNotFound
	}
	if err := m.checkStatus(statusCode, body); err != nil {
		return fmt.Errorf("delete session: %w", err)
	}
	return nil
}
