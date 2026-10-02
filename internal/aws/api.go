package aws

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"reflect"
	"strings"
	"time"
)

// apiLogger writes API request/response diagnostics to seven-test-tui.log
// so failures are visible even when the UI shows a loading state.
var apiLogger *log.Logger

func init() {
	f, err := os.OpenFile("seven-test-tui.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		apiLogger = log.New(io.Discard, "", 0)
		return
	}
	apiLogger = log.New(f, "[api] ", log.LstdFlags)
}

type TokenRefresher interface {
	Refresh(context.Context) (string, error)
}

type apiStatusError struct {
	statusCode int
	body       string
}

func (e *apiStatusError) Error() string {
	return fmt.Sprintf("API error (status %d): %s", e.statusCode, e.body)
}

type APIClient struct {
	baseURL        string
	httpClient     HTTPClient
	idToken        string
	apiKey         string
	tokenRefresher TokenRefresher
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

func (a *APIClient) SetAPIKey(key string) {
	a.apiKey = key
}

/**
 * SetTokenRefresher enables a single token refresh and retry for unauthorized reads.
 */
func (a *APIClient) SetTokenRefresher(refresher TokenRefresher) {
	a.tokenRefresher = refresher
}

func (a *APIClient) get(ctx context.Context, path string, result interface{}) error {
	err := a.getOnce(ctx, path, result)
	var statusError *apiStatusError
	if !errors.As(err, &statusError) || statusError.statusCode != http.StatusUnauthorized {
		return err
	}
	if !hasTokenRefresher(a.tokenRefresher) {
		return fmt.Errorf("refresh session after unauthorized response: %w", ErrRefreshTokenNotFound)
	}

	token, refreshErr := a.tokenRefresher.Refresh(ctx)
	if refreshErr != nil {
		return fmt.Errorf("refresh session after unauthorized response: %w", refreshErr)
	}
	a.SetIDToken(token)
	return a.getOnce(ctx, path, result)
}

func hasTokenRefresher(refresher TokenRefresher) bool {
	if refresher == nil {
		return false
	}
	value := reflect.ValueOf(refresher)
	switch value.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return !value.IsNil()
	default:
		return true
	}
}

func (a *APIClient) getOnce(ctx context.Context, path string, result interface{}) error {
	url := a.baseURL + path
	apiLogger.Printf("GET %s (auth=%t apiKey=%t)", url, a.idToken != "", a.apiKey != "")

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		apiLogger.Printf("GET %s: build request error: %v", url, err)
		return fmt.Errorf("failed to create request: %w", err)
	}
	a.setHeaders(req)

	resp, err := a.httpClient.Do(req)
	if err != nil {
		apiLogger.Printf("GET %s: transport error: %v", url, err)
		return fmt.Errorf("request to %s failed: %w", url, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		apiLogger.Printf("GET %s: read body error: %v", url, err)
		return fmt.Errorf("failed to read response: %w", err)
	}
	apiLogger.Printf("GET %s -> %d (%d bytes)", url, resp.StatusCode, len(body))

	if resp.StatusCode != http.StatusOK {
		apiLogger.Printf("GET %s: error body: %s", url, string(body))
		if strings.Contains(strings.ToLower(string(body)), "no current game week found") {
			return ErrNoCurrentGameWeek
		}
		return &apiStatusError{statusCode: resp.StatusCode, body: string(body)}
	}
	if err := json.Unmarshal(body, result); err != nil {
		apiLogger.Printf("GET %s: unmarshal error: %v", url, err)
		return fmt.Errorf("failed to unmarshal response: %w", err)
	}
	return nil
}

func (a *APIClient) setHeaders(req *http.Request) {
	if a.idToken != "" {
		req.Header.Set("Authorization", "Bearer "+a.idToken)
	}
	if a.apiKey != "" {
		req.Header.Set("x-seven-api-key", a.apiKey)
	}
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
	path := fmt.Sprintf("/game-weeks/%s/fixtures", gameWeekID)
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

func (a *APIClient) PutSelections(ctx context.Context, prefix string, selections map[string]interface{}) error {
	if err := AssertWritable(prefix); err != nil {
		return err
	}
	return a.put(ctx, "/game-week/selections", selections)
}

func (a *APIClient) put(ctx context.Context, path string, body interface{}) error {
	url := a.baseURL + path

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("failed to marshal request body: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	a.setHeaders(req)

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
