# Chapter 11: API Client Patterns

## What You'll Learn

- Building HTTP API clients in Go
- Request/response handling
- Authentication headers
- Error handling for HTTP APIs

---

## The API Client Structure

Create `internal/aws/api.go`:

```go
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
```

### Key Design Decisions

**Configurable base URL**:
```go
baseURL: baseURL,
```
Different environments have different URLs. Don't hardcode.

**Custom HTTP client**:
```go
httpClient: &http.Client{
    Timeout: 30 * time.Second,
},
```
Always set a timeout. The default `http.Client` has no timeout - requests can hang forever.

**Token storage**:
```go
idToken: string,
```
Store the auth token so it can be added to requests.

---

## Setting the Auth Token

```go
func (a *APIClient) SetIDToken(token string) {
    a.idToken = token
}
```

Called after successful authentication:

```go
case authSuccessMsg:
    m.apiClient.SetIDToken(msg.token)
```

---

## The Generic GET Method

```go
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
```

### Breaking It Down

**Create request with context**:
```go
req, err := http.NewRequestWithContext(ctx, "GET", a.baseURL+path, nil)
```
Context allows cancellation and timeouts.

**Add auth header**:
```go
req.Header.Set("Authorization", "Bearer "+a.idToken)
```
Bearer token authentication - standard for JWTs.

**Always close the body**:
```go
defer resp.Body.Close()
```
Prevents resource leaks.

**Check status code**:
```go
if resp.StatusCode != http.StatusOK {
    return fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
}
```
Don't assume success. Check the status.

**Unmarshal into result**:
```go
json.Unmarshal(body, result)
```
The `result` parameter is a pointer. JSON unmarshals into it.

---

## Specific API Methods

```go
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
```

### Why `map[string]interface{}`?

The API returns JSON with varying structures. Using `map[string]interface{}` is flexible:

```go
// API returns: {"gameWeekId": "1", "label": "Week 1", ...}
// We get: map[string]interface{}{"gameWeekId": "1", "label": "Week 1", ...}
```

For type safety, you could define response structs:

```go
type GameWeekResponse struct {
    GameWeekID string `json:"gameWeekId"`
    Label      string `json:"label"`
}

func (a *APIClient) GetGameWeeks(ctx context.Context) ([]GameWeekResponse, error) {
    var gameWeeks []GameWeekResponse
    if err := a.get(ctx, "/game-weeks", &gameWeeks); err != nil {
        return nil, err
    }
    return gameWeeks, nil
}
```

---

## The Generic PUT Method

```go
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
```

### Key Differences from GET

**Marshal body to JSON**:
```go
jsonBody, err := json.Marshal(body)
req, err := http.NewRequestWithContext(ctx, "PUT", url, bytes.NewBuffer(jsonBody))
```

**Set Content-Type**:
```go
req.Header.Set("Content-Type", "application/json")
```

**Check for 4xx/5xx errors**:
```go
if resp.StatusCode >= 400 {
```

---

## Using PUT

```go
func (a *APIClient) PutSelections(ctx context.Context, selections map[string]interface{}) error {
    return a.put(ctx, "/game-week/selections", selections)
}
```

---

## Error Handling Patterns

### Pattern 1: Return Error Details

```go
type APIError struct {
    StatusCode int
    Message    string
    Body       string
}

func (e *APIError) Error() string {
    return fmt.Sprintf("API error %d: %s", e.StatusCode, e.Message)
}

func (a *APIClient) get(ctx context.Context, path string, result interface{}) error {
    // ...
    if resp.StatusCode != http.StatusOK {
        return &APIError{
            StatusCode: resp.StatusCode,
            Message:    http.StatusText(resp.StatusCode),
            Body:       string(body),
        }
    }
    // ...
}
```

Usage:
```go
data, err := client.GetGameWeeks(ctx)
if err != nil {
    var apiErr *APIError
    if errors.As(err, &apiErr) {
        if apiErr.StatusCode == 401 {
            // Token expired, re-authenticate
        }
    }
}
```

### Pattern 2: Retry on Specific Errors

```go
func (a *APIClient) getWithRetry(ctx context.Context, path string, result interface{}) error {
    var lastErr error
    for attempt := 0; attempt < 3; attempt++ {
        err := a.get(ctx, path, result)
        if err == nil {
            return nil
        }
        
        var apiErr *APIError
        if errors.As(err, &apiErr) && apiErr.StatusCode == 429 {
            // Rate limited, wait and retry
            time.Sleep(time.Duration(attempt+1) * time.Second)
            lastErr = err
            continue
        }
        
        return err  // Don't retry other errors
    }
    return lastErr
}
```

---

## Parsing API Responses

When the API returns `map[string]interface{}`, you need to parse it:

```go
func parseGameWeeks(data []map[string]interface{}) []models.GameWeek {
    var gameWeeks []models.GameWeek
    for _, item := range data {
        gw := models.GameWeek{
            GameWeekID: item["gameWeekId"].(string),
            Label:      item["label"].(string),
        }
        
        // Handle optional fields
        if startDate, ok := item["fixturesStartDate"].(string); ok {
            gw.FixturesStartDate = startDate
        }
        
        gameWeeks = append(gameWeeks, gw)
    }
    return gameWeeks
}
```

### Safe Type Assertions

```go
// Unsafe - panics if wrong type
value := item["field"].(string)

// Safe - check ok
if value, ok := item["field"].(string); ok {
    // Use value
}

// With default
value, _ := item["field"].(string)  // Empty string if not string
```

---

## Testing API Clients

### Mock Server

```go
func TestGetGameWeeks(t *testing.T) {
    // Create mock server
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        if r.URL.Path != "/game-weeks" {
            t.Errorf("unexpected path: %s", r.URL.Path)
        }
        
        w.Header().Set("Content-Type", "application/json")
        w.Write([]byte(`[{"gameWeekId": "1", "label": "Week 1"}]`))
    }))
    defer server.Close()
    
    // Create client pointing to mock
    client := NewAPIClient(server.URL)
    
    // Test
    gameWeeks, err := client.GetGameWeeks(context.Background())
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    
    if len(gameWeeks) != 1 {
        t.Errorf("expected 1 gameweek, got %d", len(gameWeeks))
    }
}
```

---

## Best Practices

### 1. Always Use Context

```go
func (a *APIClient) GetData(ctx context.Context) error {
    req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
}
```

### 2. Set Timeouts

```go
httpClient: &http.Client{
    Timeout: 30 * time.Second,
}
```

### 3. Close Response Bodies

```go
resp, err := a.httpClient.Do(req)
if err != nil {
    return err
}
defer resp.Body.Close()  // Always!
```

### 4. Check Status Codes

```go
if resp.StatusCode != http.StatusOK {
    return fmt.Errorf("unexpected status: %d", resp.StatusCode)
}
```

### 5. Wrap Errors with Context

```go
return fmt.Errorf("failed to fetch gameweeks: %w", err)
```

---

## Key Takeaways

1. **Configurable base URL** - Different environments, same code
2. **Custom HTTP client** - Always set timeouts
3. **Generic methods** - `get()` and `put()` reduce duplication
4. **Bearer token auth** - Standard for JWT authentication
5. **Always close bodies** - Prevent resource leaks
6. **Check status codes** - Don't assume success
7. **Safe type assertions** - Use the `ok` pattern

---

## Exercise

1. Add a `Delete` method for removing resources
2. Implement request logging (log URL, method, status code, duration)
3. Add a `WithTimeout` option to override the default timeout per-request
4. Create response structs instead of using `map[string]interface{}`

---

[← Previous: AWS SDK Integration](./10-aws-sdk.md) | [Next: Chapter 12 - State Management →](./12-state-management.md)
