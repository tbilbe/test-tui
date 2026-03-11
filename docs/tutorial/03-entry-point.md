# Chapter 3: The Entry Point Pattern

## What You'll Learn

- Why `main()` should be thin
- Dependency injection in Go
- Error handling patterns
- The "fail fast" principle

---

## The Problem with Fat Main Functions

Beginners often write everything in `main()`:

```go
// DON'T DO THIS
func main() {
    // Load config
    apiEndpoint := os.Getenv("API_ENDPOINT")
    if apiEndpoint == "" {
        apiEndpoint = "http://localhost:8080"
    }
    userPoolID := os.Getenv("USER_POOL_ID")
    // ... 20 more lines of config
    
    // Create HTTP client
    client := &http.Client{Timeout: 30 * time.Second}
    
    // Authenticate
    // ... 50 lines of auth code
    
    // Start UI
    // ... 100 lines of UI code
}
```

Problems:
- Hard to test (can't test pieces in isolation)
- Hard to read (everything mixed together)
- Hard to change (modifications ripple everywhere)

---

## The Thin Main Pattern

Look at our actual `cmd/main.go`:

```go
package main

import (
    "context"
    "fmt"
    "os"

    tea "github.com/charmbracelet/bubbletea"

    "github.com/angstromsports/seven-test-tui/internal/aws"
    "github.com/angstromsports/seven-test-tui/internal/ui"
    "github.com/angstromsports/seven-test-tui/pkg/config"
)

func main() {
    // 1. Load configuration
    cfg, err := config.Load()
    if err != nil {
        fmt.Printf("Configuration error: %v\n", err)
        os.Exit(1)
    }

    ctx := context.Background()

    // 2. Create dependencies
    authClient, err := aws.NewAuthClient(ctx, cfg.AWSRegion, cfg.UserPoolID, cfg.ClientID)
    if err != nil {
        fmt.Printf("Failed to create auth client: %v\n", err)
        os.Exit(1)
    }

    apiClient := aws.NewAPIClient(cfg.APIEndpoint)

    dynamoClient, err := aws.NewDynamoDBClient(ctx, cfg.AWSRegion, cfg.Prefix+"-GameWeek")
    if err != nil {
        fmt.Printf("Failed to create DynamoDB client: %v\n", err)
        os.Exit(1)
    }

    // 3. Wire everything together and run
    p := tea.NewProgram(
        ui.NewModel(authClient, apiClient, dynamoClient),
        tea.WithAltScreen(),
        tea.WithMouseCellMotion(),
    )
    
    if _, err := p.Run(); err != nil {
        fmt.Printf("Error: %v\n", err)
        os.Exit(1)
    }
}
```

### What Main Does

1. **Load configuration** - Get settings from environment
2. **Create dependencies** - Build the objects the app needs
3. **Wire and run** - Connect everything and start

That's it. No business logic. No UI code. Just assembly.

---

## Dependency Injection

Notice how we create clients and pass them to the UI:

```go
authClient, err := aws.NewAuthClient(ctx, cfg.AWSRegion, cfg.UserPoolID, cfg.ClientID)
apiClient := aws.NewAPIClient(cfg.APIEndpoint)
dynamoClient, err := aws.NewDynamoDBClient(ctx, cfg.AWSRegion, cfg.Prefix+"-GameWeek")

// Inject dependencies
ui.NewModel(authClient, apiClient, dynamoClient)
```

### Why Inject Dependencies?

**Testability**: In tests, you can pass mock clients:

```go
// In production
ui.NewModel(realAuthClient, realAPIClient, realDynamoClient)

// In tests
ui.NewModel(mockAuthClient, mockAPIClient, mockDynamoClient)
```

**Flexibility**: Easy to swap implementations:

```go
// Today: AWS Cognito
authClient := aws.NewAuthClient(...)

// Tomorrow: Different auth provider
authClient := auth0.NewClient(...)

// UI code doesn't change
ui.NewModel(authClient, ...)
```

**Clarity**: Dependencies are explicit, not hidden:

```go
// Bad: Hidden dependency
func NewModel() Model {
    client := aws.NewAuthClient(...)  // Where does config come from?
    return Model{client: client}
}

// Good: Explicit dependency
func NewModel(client *aws.AuthClient) Model {
    return Model{client: client}
}
```

---

## Error Handling: Fail Fast

Look at the error handling pattern:

```go
cfg, err := config.Load()
if err != nil {
    fmt.Printf("Configuration error: %v\n", err)
    os.Exit(1)
}
```

### The Pattern

```go
result, err := doSomething()
if err != nil {
    // Handle error immediately
    // Don't continue with invalid state
}
// Continue with valid result
```

### Why Fail Fast?

During startup, if something is wrong, we want to know immediately:

```go
// Good: Fail at startup
authClient, err := aws.NewAuthClient(...)
if err != nil {
    fmt.Printf("Failed to create auth client: %v\n", err)
    os.Exit(1)  // App won't start with broken auth
}

// Bad: Fail later
authClient, _ := aws.NewAuthClient(...)  // Ignore error
// ... app starts ...
// ... user tries to login ...
// ... cryptic error because authClient is nil
```

Failing fast means:
- Errors are caught early
- Error messages are clear
- Invalid states don't propagate

---

## Context: The First Parameter

Notice `ctx := context.Background()`:

```go
ctx := context.Background()
authClient, err := aws.NewAuthClient(ctx, cfg.AWSRegion, cfg.UserPoolID, cfg.ClientID)
```

### What is Context?

Context carries:
- **Cancellation signals** - "Stop what you're doing"
- **Deadlines** - "Finish by this time"
- **Request-scoped values** - Trace IDs, auth tokens, etc.

### Why Pass It?

AWS SDK operations can be cancelled:

```go
// If context is cancelled, the AWS call stops
result, err := client.InitiateAuth(ctx, input)
```

In `main()`, we use `context.Background()` - a never-cancelled context. In a web server, you'd use the request's context so operations cancel if the client disconnects.

---

## The Constructor Pattern

Look at how we create clients:

```go
authClient, err := aws.NewAuthClient(ctx, cfg.AWSRegion, cfg.UserPoolID, cfg.ClientID)
```

### The `NewXxx` Convention

Go doesn't have constructors like Java/C#. Instead, we write functions:

```go
// Constructor function
func NewAuthClient(ctx context.Context, region, userPoolID, clientID string) (*AuthClient, error) {
    cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(region))
    if err != nil {
        return nil, fmt.Errorf("failed to load AWS config: %w", err)
    }

    return &AuthClient{
        client:     cognitoidentityprovider.NewFromConfig(cfg),
        userPoolID: userPoolID,
        clientID:   clientID,
    }, nil
}
```

### Return Patterns

**`(*Type, error)`** - When construction can fail:

```go
func NewAuthClient(...) (*AuthClient, error)
func NewDynamoDBClient(...) (*DynamoDBClient, error)
```

**`*Type`** - When construction can't fail:

```go
func NewAPIClient(baseURL string) *APIClient {
    return &APIClient{
        baseURL: baseURL,
        httpClient: &http.Client{Timeout: 30 * time.Second},
    }
}
```

---

## Bubbletea Program Setup

The final piece:

```go
p := tea.NewProgram(
    ui.NewModel(authClient, apiClient, dynamoClient),
    tea.WithAltScreen(),
    tea.WithMouseCellMotion(),
)

if _, err := p.Run(); err != nil {
    fmt.Printf("Error: %v\n", err)
    os.Exit(1)
}
```

### What's Happening

1. **`ui.NewModel(...)`** - Create our application model with injected dependencies
2. **`tea.WithAltScreen()`** - Use alternate screen buffer (UI disappears when app exits)
3. **`tea.WithMouseCellMotion()`** - Enable mouse support
4. **`p.Run()`** - Start the event loop (blocks until app exits)

---

## The Complete Pattern

```go
func main() {
    // 1. Configuration
    cfg, err := loadConfig()
    exitOnError(err, "config")

    // 2. Dependencies
    dep1, err := createDep1(cfg)
    exitOnError(err, "dep1")
    
    dep2, err := createDep2(cfg)
    exitOnError(err, "dep2")

    // 3. Wire and run
    app := newApp(dep1, dep2)
    exitOnError(app.Run(), "run")
}

func exitOnError(err error, context string) {
    if err != nil {
        fmt.Fprintf(os.Stderr, "%s error: %v\n", context, err)
        os.Exit(1)
    }
}
```

---

## Key Takeaways

1. **Keep main thin** - Only configuration, dependency creation, and wiring
2. **Inject dependencies** - Pass them in, don't create them inside
3. **Fail fast** - Check errors immediately, exit on startup failures
4. **Use context** - Pass it as the first parameter to cancellable operations
5. **`NewXxx` convention** - Constructor functions return `(*Type, error)` or `*Type`
6. **Explicit over implicit** - Dependencies should be visible in function signatures

---

## Exercise

1. What happens if you remove `tea.WithAltScreen()`? Try it.
2. Add a helper function `exitOnError(err error, msg string)` to reduce repetition.
3. What would you change to support multiple environments (dev, staging, prod)?

---

[← Previous: Understanding Structs](./02-structs.md) | [Next: Chapter 4 - Configuration Management →](./04-configuration.md)
