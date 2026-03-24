package maestro

import (
	"context"
	"encoding/json"
	"fmt"
)

// SubmitPipeline submits a multi-step pipeline (async, returns job).
func (m *maestroImpl) SubmitPipeline(ctx context.Context, sessionID string, req PipelineReq) (PipelineData, error) {
	path := PathPipelines
	body, statusCode, err := m.doPost(ctx, path, sessionID, req)
	if err != nil {
		return PipelineData{}, fmt.Errorf("submit pipeline: %w", err)
	}
	if err := m.checkStatus(statusCode, body); err != nil {
		return PipelineData{}, fmt.Errorf("submit pipeline: %w", err)
	}

	var resp SuccessResponse[PipelineData]
	if err := json.Unmarshal(body, &resp); err != nil {
		return PipelineData{}, fmt.Errorf("submit pipeline: unmarshal: %w", err)
	}
	return resp.Data, nil
}
