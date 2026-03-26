package maestro

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// GetJob retrieves the status and result of an async job.
func (m *maestroImpl) GetJob(ctx context.Context, jobID string) (JobData, error) {
	path := fmt.Sprintf("%s/%s", PathJobs, jobID)
	body, statusCode, err := m.doGet(ctx, path, "")
	if err != nil {
		return JobData{}, fmt.Errorf("get job: %w", err)
	}
	if statusCode == http.StatusNotFound {
		return JobData{}, ErrJobNotFound
	}
	if err := m.checkStatus(statusCode, body); err != nil {
		return JobData{}, fmt.Errorf("get job: %w", err)
	}

	var resp SuccessResponse[JobData]
	if err := json.Unmarshal(body, &resp); err != nil {
		return JobData{}, fmt.Errorf("get job: unmarshal: %w", err)
	}
	return resp.Data, nil
}
