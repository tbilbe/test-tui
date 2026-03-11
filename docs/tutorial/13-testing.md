# Chapter 13: Testing TUI Applications

## What You'll Learn

- Testing strategies for TUI apps
- Unit testing models and state
- Testing commands and async operations
- Integration testing with mock services

---

## Testing Pyramid for TUI Apps

```
        /\
       /  \     E2E Tests (few)
      /----\    - Full app with real services
     /      \
    /--------\  Integration Tests (some)
   /          \ - Commands with mock services
  /------------\
 /              \ Unit Tests (many)
/________________\ - Models, state, validation
```

Most tests should be unit tests. They're fast and reliable.

---

## Unit Testing Models

### Testing Validation

```go
// internal/models/models_test.go
package models

import "testing"

func TestValidatePeriod(t *testing.T) {
    tests := []struct {
        name    string
        period  FixturePeriod
        wantErr bool
    }{
        {"valid PRE_MATCH", PeriodPreMatch, false},
        {"valid FIRST_HALF", PeriodFirstHalf, false},
        {"valid HALF_TIME", PeriodHalfTime, false},
        {"valid SECOND_HALF", PeriodSecondHalf, false},
        {"valid FULL_TIME", PeriodFullTime, false},
        {"invalid period", FixturePeriod("INVALID"), true},
        {"empty period", FixturePeriod(""), true},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := ValidatePeriod(tt.period)
            if (err != nil) != tt.wantErr {
                t.Errorf("ValidatePeriod(%q) error = %v, wantErr %v", tt.period, err, tt.wantErr)
            }
        })
    }
}
```

### Table-Driven Tests

The pattern above is called "table-driven tests". Benefits:
- Easy to add new cases
- Clear what's being tested
- Consistent structure

### Testing Clock Time Validation

```go
func TestValidateClockTime(t *testing.T) {
    tests := []struct {
        name    string
        period  FixturePeriod
        min     int
        sec     int
        wantErr bool
    }{
        {"PRE_MATCH at 0:00", PeriodPreMatch, 0, 0, false},
        {"PRE_MATCH at 1:00", PeriodPreMatch, 1, 0, true},
        {"FIRST_HALF at 0:00", PeriodFirstHalf, 0, 0, false},
        {"FIRST_HALF at 45:00", PeriodFirstHalf, 45, 0, false},
        {"FIRST_HALF at 46:00", PeriodFirstHalf, 46, 0, true},
        {"SECOND_HALF at 45:00", PeriodSecondHalf, 45, 0, false},
        {"SECOND_HALF at 90:00", PeriodSecondHalf, 90, 0, false},
        {"SECOND_HALF at 44:00", PeriodSecondHalf, 44, 0, true},
        {"invalid seconds", PeriodFirstHalf, 30, 60, true},
        {"negative seconds", PeriodFirstHalf, 30, -1, true},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := ValidateClockTime(tt.period, tt.min, tt.sec)
            if (err != nil) != tt.wantErr {
                t.Errorf("ValidateClockTime(%q, %d, %d) error = %v, wantErr %v",
                    tt.period, tt.min, tt.sec, err, tt.wantErr)
            }
        })
    }
}
```

---

## Testing State

```go
// internal/models/state_test.go
package models

import "testing"

func TestAppState_SetError(t *testing.T) {
    state := NewAppState()
    state.SuccessMessage = "Previous success"
    
    state.SetError("Something went wrong")
    
    if state.ErrorMessage != "Something went wrong" {
        t.Errorf("ErrorMessage = %q, want %q", state.ErrorMessage, "Something went wrong")
    }
    if state.SuccessMessage != "" {
        t.Errorf("SuccessMessage should be cleared, got %q", state.SuccessMessage)
    }
}

func TestAppState_SetSuccess(t *testing.T) {
    state := NewAppState()
    state.ErrorMessage = "Previous error"
    
    state.SetSuccess("Operation completed")
    
    if state.SuccessMessage != "Operation completed" {
        t.Errorf("SuccessMessage = %q, want %q", state.SuccessMessage, "Operation completed")
    }
    if state.ErrorMessage != "" {
        t.Errorf("ErrorMessage should be cleared, got %q", state.ErrorMessage)
    }
}

func TestAppState_SetLoading(t *testing.T) {
    state := NewAppState()
    state.ErrorMessage = "Previous error"
    state.SuccessMessage = "Previous success"
    
    state.SetLoading(true, "Loading data...")
    
    if !state.IsLoading {
        t.Error("IsLoading should be true")
    }
    if state.LoadingMessage != "Loading data..." {
        t.Errorf("LoadingMessage = %q, want %q", state.LoadingMessage, "Loading data...")
    }
    if state.ErrorMessage != "" || state.SuccessMessage != "" {
        t.Error("Messages should be cleared when loading starts")
    }
}
```

---

## Testing the Update Function

You can test Update by sending messages and checking the result:

```go
// internal/ui/gameweeks_test.go
package ui

import (
    "testing"
    
    tea "github.com/charmbracelet/bubbletea"
    "github.com/angstromsports/seven-test-tui/internal/models"
)

func TestModel_Update_GameWeeksLoaded(t *testing.T) {
    // Setup
    model := NewModel(nil, nil, nil)  // nil clients for unit test
    
    gameWeeks := []models.GameWeek{
        {GameWeekID: "1", Label: "Week 1"},
        {GameWeekID: "2", Label: "Week 2"},
    }
    
    // Send message
    msg := gameWeeksLoadedMsg{gameWeeks: gameWeeks}
    newModel, cmd := model.Update(msg)
    
    // Assert
    m := newModel.(Model)
    if len(m.state.GameWeeks) != 2 {
        t.Errorf("GameWeeks count = %d, want 2", len(m.state.GameWeeks))
    }
    if cmd != nil {
        t.Error("Expected no command")
    }
}

func TestModel_Update_GameWeeksLoadedError(t *testing.T) {
    model := NewModel(nil, nil, nil)
    
    msg := gameWeeksLoadedMsg{err: fmt.Errorf("network error")}
    newModel, _ := model.Update(msg)
    
    m := newModel.(Model)
    if m.state.ErrorMessage == "" {
        t.Error("Expected error message to be set")
    }
}
```

### Testing Key Presses

```go
func TestModel_Update_QuitKey(t *testing.T) {
    model := NewModel(nil, nil, nil)
    model.currentScreen = gameweekScreenType
    
    msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}}
    _, cmd := model.Update(msg)
    
    // tea.Quit returns a special command
    if cmd == nil {
        t.Error("Expected quit command")
    }
}

func TestModel_Update_NavigationKeys(t *testing.T) {
    model := NewModel(nil, nil, nil)
    model.currentScreen = gameweekScreenType
    model.state.SetGameWeeks([]models.GameWeek{
        {GameWeekID: "1"},
        {GameWeekID: "2"},
        {GameWeekID: "3"},
    })
    model.selectedIdx = 0
    
    // Press down
    msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}
    newModel, _ := model.Update(msg)
    m := newModel.(Model)
    
    if m.selectedIdx != 1 {
        t.Errorf("selectedIdx = %d, want 1", m.selectedIdx)
    }
}
```

---

## Testing Commands

Commands are functions that return messages. Test them by calling and checking the result:

```go
func TestAuthenticateCmd_Success(t *testing.T) {
    // Create mock auth client
    mockClient := &MockAuthClient{
        signInFunc: func(ctx context.Context, username, password string) error {
            return nil
        },
        getIDTokenFunc: func() string {
            return "mock-token"
        },
    }
    
    // Get the command
    cmd := authenticateCmd(mockClient, "user", "pass")
    
    // Execute it
    msg := cmd()
    
    // Check result
    successMsg, ok := msg.(authSuccessMsg)
    if !ok {
        t.Fatalf("Expected authSuccessMsg, got %T", msg)
    }
    if successMsg.token != "mock-token" {
        t.Errorf("token = %q, want %q", successMsg.token, "mock-token")
    }
}

func TestAuthenticateCmd_Error(t *testing.T) {
    mockClient := &MockAuthClient{
        signInFunc: func(ctx context.Context, username, password string) error {
            return fmt.Errorf("invalid credentials")
        },
    }
    
    cmd := authenticateCmd(mockClient, "user", "wrong")
    msg := cmd()
    
    errorMsg, ok := msg.(authErrorMsg)
    if !ok {
        t.Fatalf("Expected authErrorMsg, got %T", msg)
    }
    if errorMsg.err == nil {
        t.Error("Expected error to be set")
    }
}
```

### Creating Mock Clients

```go
// internal/aws/mock_auth.go
type MockAuthClient struct {
    signInFunc     func(ctx context.Context, username, password string) error
    getIDTokenFunc func() string
}

func (m *MockAuthClient) SignIn(ctx context.Context, username, password string) error {
    if m.signInFunc != nil {
        return m.signInFunc(ctx, username, password)
    }
    return nil
}

func (m *MockAuthClient) GetIDToken() string {
    if m.getIDTokenFunc != nil {
        return m.getIDTokenFunc()
    }
    return ""
}
```

---

## Testing API Client

Use `httptest` to create a mock server:

```go
func TestAPIClient_GetGameWeeks(t *testing.T) {
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Verify request
        if r.URL.Path != "/game-weeks" {
            t.Errorf("unexpected path: %s", r.URL.Path)
        }
        if r.Header.Get("Authorization") != "Bearer test-token" {
            t.Error("missing or wrong auth header")
        }
        
        // Return mock response
        w.Header().Set("Content-Type", "application/json")
        w.Write([]byte(`[{"gameWeekId": "1", "label": "Week 1"}]`))
    }))
    defer server.Close()
    
    client := NewAPIClient(server.URL)
    client.SetIDToken("test-token")
    
    gameWeeks, err := client.GetGameWeeks(context.Background())
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    
    if len(gameWeeks) != 1 {
        t.Errorf("gameWeeks count = %d, want 1", len(gameWeeks))
    }
}

func TestAPIClient_GetGameWeeks_Error(t *testing.T) {
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(http.StatusInternalServerError)
        w.Write([]byte(`{"error": "internal error"}`))
    }))
    defer server.Close()
    
    client := NewAPIClient(server.URL)
    
    _, err := client.GetGameWeeks(context.Background())
    if err == nil {
        t.Error("expected error for 500 response")
    }
}
```

---

## Testing View Output

You can test that View returns expected content:

```go
func TestAuthScreen_View(t *testing.T) {
    screen := NewAuthScreen()
    
    view := screen.View()
    
    if !strings.Contains(view, "Login") {
        t.Error("View should contain 'Login'")
    }
    if !strings.Contains(view, "Username") {
        t.Error("View should contain 'Username'")
    }
}

func TestAuthScreen_View_Submitted(t *testing.T) {
    screen := NewAuthScreen()
    screen.submitted = true
    
    view := screen.View()
    
    if !strings.Contains(view, "Authenticating") {
        t.Error("View should show loading state")
    }
}
```

---

## Running Tests

```bash
# Run all tests
go test ./...

# Run tests with verbose output
go test -v ./...

# Run specific package
go test ./internal/models/...

# Run specific test
go test -run TestValidatePeriod ./internal/models/

# Run with coverage
go test -cover ./...

# Generate coverage report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

---

## Test Organization

```
internal/
├── models/
│   ├── models.go
│   ├── models_test.go    # Tests for models.go
│   ├── state.go
│   └── state_test.go     # Tests for state.go
├── aws/
│   ├── auth.go
│   ├── auth_test.go
│   ├── api.go
│   ├── api_test.go
│   └── mock_auth.go      # Mock implementations
└── ui/
    ├── gameweeks.go
    └── gameweeks_test.go
```

Convention: `foo_test.go` tests `foo.go`.

---

## Key Takeaways

1. **Table-driven tests** - Easy to add cases, clear structure
2. **Test Update with messages** - Send message, check new model
3. **Mock external services** - Use interfaces and mock implementations
4. **httptest for API tests** - Create mock HTTP servers
5. **Test commands directly** - Call the function, check the message
6. **Coverage reports** - Find untested code

---

## Exercise

1. Write tests for `ValidateScore` function
2. Create a mock `DynamoDBClient` and test `updateFixtureCmd`
3. Write a test that simulates a full login flow (auth screen → submit → success)
4. Add tests for edge cases: empty lists, nil pointers, boundary values

---

## Conclusion

You've now learned how to build a production-ready TUI application in Go:

1. **Project structure** - Standard Go layout with `cmd/`, `internal/`, `pkg/`
2. **Structs and types** - Domain modeling with validation
3. **Entry point** - Thin main, dependency injection
4. **Configuration** - Environment variables, validation
5. **Bubbletea** - Model-Update-View pattern
6. **Elm Architecture** - Unidirectional data flow
7. **Screens** - Components with focus management
8. **Commands** - Async operations without blocking
9. **Navigation** - Multi-screen apps with modals
10. **AWS SDK** - Cognito auth, DynamoDB operations
11. **API clients** - HTTP with proper error handling
12. **State management** - Centralized, encapsulated state
13. **Testing** - Unit tests, mocks, integration tests

Keep building. The best way to learn is to write code.

---

[← Previous: State Management](./12-state-management.md) | [Back to Introduction](./00-introduction.md)
