package maestro

import (
	"context"
	"encoding/json"
	"fmt"
)

// CreateNotebook creates a new NotebookLM notebook (async, returns job).
func (m *maestroImpl) CreateNotebook(ctx context.Context, sessionID string, req CreateNotebookReq) (JobEnqueued, error) {
	path := PathNotebooks
	body, statusCode, err := m.doPost(ctx, path, sessionID, req)
	if err != nil {
		return JobEnqueued{}, fmt.Errorf("create notebook: %w", err)
	}
	if err := m.checkStatus(statusCode, body); err != nil {
		return JobEnqueued{}, fmt.Errorf("create notebook: %w", err)
	}

	var resp SuccessResponse[JobEnqueued]
	if err := json.Unmarshal(body, &resp); err != nil {
		return JobEnqueued{}, fmt.Errorf("create notebook: unmarshal: %w", err)
	}
	return resp.Data, nil
}

// ListNotebooks lists all notebooks in the current session.
func (m *maestroImpl) ListNotebooks(ctx context.Context, sessionID string) ([]NotebookData, error) {
	path := PathNotebooks
	body, statusCode, err := m.doGet(ctx, path, sessionID)
	if err != nil {
		return nil, fmt.Errorf("list notebooks: %w", err)
	}
	if err := m.checkStatus(statusCode, body); err != nil {
		return nil, fmt.Errorf("list notebooks: %w", err)
	}

	var resp SuccessResponse[[]NotebookData]
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("list notebooks: unmarshal: %w", err)
	}
	return resp.Data, nil
}

// UploadSources uploads sources to a notebook (async, returns job).
func (m *maestroImpl) UploadSources(ctx context.Context, sessionID, notebookID string, req UploadSourcesReq) (JobEnqueued, error) {
	path := fmt.Sprintf("%s/%s/sources", PathNotebooks, notebookID)
	body, statusCode, err := m.doPost(ctx, path, sessionID, req)
	if err != nil {
		return JobEnqueued{}, fmt.Errorf("upload sources: %w", err)
	}
	if err := m.checkStatus(statusCode, body); err != nil {
		return JobEnqueued{}, fmt.Errorf("upload sources: %w", err)
	}

	var resp SuccessResponse[JobEnqueued]
	if err := json.Unmarshal(body, &resp); err != nil {
		return JobEnqueued{}, fmt.Errorf("upload sources: unmarshal: %w", err)
	}
	return resp.Data, nil
}

// ChatNotebook sends a chat prompt to a notebook (async, returns job).
func (m *maestroImpl) ChatNotebook(ctx context.Context, sessionID, notebookID string, req ChatNotebookReq) (JobEnqueued, error) {
	path := fmt.Sprintf("%s/%s/chat", PathNotebooks, notebookID)
	body, statusCode, err := m.doPost(ctx, path, sessionID, req)
	if err != nil {
		return JobEnqueued{}, fmt.Errorf("chat notebook: %w", err)
	}
	if err := m.checkStatus(statusCode, body); err != nil {
		return JobEnqueued{}, fmt.Errorf("chat notebook: %w", err)
	}

	var resp SuccessResponse[JobEnqueued]
	if err := json.Unmarshal(body, &resp); err != nil {
		return JobEnqueued{}, fmt.Errorf("chat notebook: unmarshal: %w", err)
	}
	return resp.Data, nil
}
