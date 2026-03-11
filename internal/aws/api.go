package aws

import (
	"bytes"
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

func (a *APIClient) GetCurrentFixtures(ctx context.Context) (map[string]interface{}, error) {
	var fixtures map[string]interface{}
	if err := a.get(ctx, "/game-week/fixtures", &fixtures); err != nil {
		return nil, err
	}
	return fixtures, nil
}

func (a *APIClient) GetFixturesByGameWeek(ctx context.Context, gameWeekID string) (map[string]interface{}, error) {
	var fixtures map[string]interface{}
	path := fmt.Sprintf("/game-week/%s/fixtures", gameWeekID)
	if err := a.get(ctx, path, &fixtures); err != nil {
		return nil, err
	}
	return fixtures, nil
}

func (a *APIClient) GetGameWeekPlayers(ctx context.Context) (map[string]interface{}, error) {
	var players map[string]interface{}
	if err := a.get(ctx, "/game-week/players", &players); err != nil {
		return nil, err
	}
	return players, nil
}

func (a *APIClient) GetSelections(ctx context.Context) (map[string]interface{}, error) {
	var selections map[string]interface{}
	if err := a.get(ctx, "/game-week/selections", &selections); err != nil {
		return nil, err
	}
	return selections, nil
}

func (a *APIClient) PutSelections(ctx context.Context, selections map[string]interface{}) error {
	return a.put(ctx, "/game-week/selections", selections)
}

func (a *APIClient) put(ctx context.Context, path string, body interface{}) error {
	url := a.baseURL + path

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("failed to marshal request body: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "PUT", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if a.idToken != "" {
		req.Header.Set("Authorization", "Bearer "+a.idToken)
	}

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(bodyBytes))
	}

	return nil
}
