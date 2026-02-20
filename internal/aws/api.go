package aws

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type APIClient struct {
	baseURL    string
	httpClient *http.Client
	idToken    string
}

func NewAPIClient(baseURL string) *APIClient {
	return &APIClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (a *APIClient) SetIDToken(token string) {
	a.idToken = token
}

func (a *APIClient) get(ctx context.Context, path string, result interface{}) error {
	req, err := http.NewRequestWithContext(ctx, "GET", a.baseURL+path, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	if a.idToken != "" {
		req.Header.Set("Authorization", "Bearer "+a.idToken)
	}

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	if err := json.Unmarshal(body, result); err != nil {
		return fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return nil
}

func (a *APIClient) GetGameWeeks(ctx context.Context) ([]map[string]interface{}, error) {
	var gameWeeks []map[string]interface{}
	if err := a.get(ctx, "/game-weeks", &gameWeeks); err != nil {
		return nil, err
	}
	return gameWeeks, nil
}

func (a *APIClient) GetFixturesByGameWeek(ctx context.Context, gameWeekID string) (map[string]interface{}, error) {
	var fixtures map[string]interface{}
	path := fmt.Sprintf("/game-week/%s/fixtures", gameWeekID)
	if err := a.get(ctx, path, &fixtures); err != nil {
		return nil, err
	}
	return fixtures, nil
}
