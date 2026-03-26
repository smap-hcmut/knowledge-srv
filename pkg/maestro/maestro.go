package maestro

import (
	"context"
	"fmt"
	"io"
	"net/http"
)

// doPost performs a POST request with JSON body.
func (m *maestroImpl) doPost(ctx context.Context, path, sessionID string, reqBody any) ([]byte, int, error) {
	url := m.baseURL + path
	headers := m.buildHeaders(sessionID)
	return m.httpClient.Post(ctx, url, reqBody, headers)
}

// doGet performs a GET request.
func (m *maestroImpl) doGet(ctx context.Context, path, sessionID string) ([]byte, int, error) {
	url := m.baseURL + path
	headers := m.buildHeaders(sessionID)
	return m.httpClient.Get(ctx, url, headers)
}

// doDelete performs a DELETE request using net/http directly (pkghttp.Client has no Delete method).
func (m *maestroImpl) doDelete(ctx context.Context, path string) ([]byte, int, error) {
	url := m.baseURL + path
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, url, nil)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set(HeaderAPIKey, m.apiKey)

	client := &http.Client{Timeout: DefaultTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, fmt.Errorf("failed to read response: %w", err)
	}
	return body, resp.StatusCode, nil
}

// buildHeaders returns common headers for Maestro API requests.
func (m *maestroImpl) buildHeaders(sessionID string) map[string]string {
	headers := map[string]string{
		HeaderAPIKey: m.apiKey,
	}
	if sessionID != "" {
		headers[HeaderSessionID] = sessionID
	}
	return headers
}

// checkStatus maps HTTP status codes to sentinel errors.
func (m *maestroImpl) checkStatus(statusCode int, body []byte) error {
	switch {
	case statusCode >= 200 && statusCode < 300:
		return nil
	case statusCode == http.StatusUnauthorized:
		return ErrUnauthorized
	case statusCode == http.StatusNotFound:
		return ErrNotebookNotFound
	case statusCode == http.StatusTooManyRequests:
		return ErrRateLimited
	case statusCode == http.StatusBadRequest:
		return fmt.Errorf("%w: %s", ErrBadRequest, string(body))
	case statusCode >= 500:
		return fmt.Errorf("%w: status=%d body=%s", ErrServerError, statusCode, string(body))
	default:
		return fmt.Errorf("maestro: unexpected status %d: %s", statusCode, string(body))
	}
}
