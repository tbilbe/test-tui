# Chapter 8: Commands & Async Operations

## What You'll Learn

- What commands are and how they work
- Fetching data asynchronously
- Error handling in async operations
- The command pattern for side effects

---

## The Problem: Blocking the UI

If you fetch data synchronously:

```go
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    case "enter":
        // DON'T DO THIS - blocks the UI
        data, err := m.apiClient.GetGameWeeks(context.Background())
        m.gameWeeks = data
        return m, nil
}
```

The UI freezes while waiting for the network. Users can't even quit.

---

## Commands: Async Without Blocking

A command is a function that:
1. Runs in a goroutine (doesn't block)
2. Returns a message when done
3. The message gets fed back into Update

```go
// Command definition
func fetchGameWeeksCmd(client *aws.APIClient) tea.Cmd {
    return func() tea.Msg {
        ctx := context.Background()
        data, err := client.GetGameWeeks(ctx)
        if err != nil {
            return gameWeeksLoadedMsg{err: err}
        }
        // Parse data...
        return gameWeeksLoadedMsg{gameWeeks: gameWeeks}
    }
}

// Usage in Update
case "enter":
    return m, fetchGameWeeksCmd(m.apiClient)  // Returns immediately

// Handle result later
case gameWeeksLoadedMsg:
    if msg.err != nil {
        m.state.SetError(msg.err.Error())
    } else {
        m.state.SetGameWeeks(msg.gameWeeks)
    }
    return m, nil
```

---

## Anatomy of a Command

```go
func fetchGameWeeksCmd(client *aws.APIClient) tea.Cmd {
    return func() tea.Msg {
        // This runs in a goroutine
        // Do slow/blocking work here
        return someMsg{}
    }
}
```

### The Outer Function

```go
func fetchGameWeeksCmd(client *aws.APIClient) tea.Cmd
```

- Takes dependencies as parameters
- Returns a `tea.Cmd` (which is `func() tea.Msg`)
- Called from Update

### The Inner Function

```go
return func() tea.Msg {
    // Async work
    return someMsg{}
}
```

- Runs asynchronously
- Has access to outer function's parameters (closure)
- Must return a message

---

## Real Example: Authentication

```go
// Message types
type authSuccessMsg struct {
    token string
}

type authErrorMsg struct {
    err error
}

// Command
func authenticateCmd(client *aws.AuthClient, username, password string) tea.Cmd {
    return func() tea.Msg {
        ctx := context.Background()
        err := client.SignIn(ctx, username, password)
        if err != nil {
            return authErrorMsg{err: err}
        }
        return authSuccessMsg{token: client.GetIDToken()}
    }
}

// In Update
case authSubmitMsg:
    return m, authenticateCmd(m.authClient, msg.username, msg.password)

case authSuccessMsg:
    m.apiClient.SetIDToken(msg.token)
    m.currentScreen = gameweekScreenType
    return m, fetchGameWeeksCmd(m.apiClient)  // Chain another command

case authErrorMsg:
    m.err = msg.err
    m.authScreen = NewAuthScreen()  // Reset form
    return m, m.authScreen.Init()
```

---

## Real Example: Fetching Data

```go
// Message type
type gameWeeksLoadedMsg struct {
    gameWeeks []models.GameWeek
    err       error
}

// Command
func fetchGameWeeksCmd(client *aws.APIClient) tea.Cmd {
    return func() tea.Msg {
        ctx := context.Background()
        data, err := client.GetGameWeeks(ctx)
        if err != nil {
            return gameWeeksLoadedMsg{err: err}
        }

        // Parse the response
        var gameWeeks []models.GameWeek
        for _, item := range data {
            gw := models.GameWeek{
                GameWeekID: item["gameWeekId"].(string),
                Label:      item["label"].(string),
            }
            gameWeeks = append(gameWeeks, gw)
        }

        return gameWeeksLoadedMsg{gameWeeks: gameWeeks}
    }
}

// In Update
case gameWeeksLoadedMsg:
    if msg.err != nil {
        m.err = msg.err
    } else {
        m.state.SetGameWeeks(msg.gameWeeks)
    }
    return m, nil
```

---

## Error Handling Patterns

### Pattern 1: Error in Message

```go
type dataLoadedMsg struct {
    data []Item
    err  error  // nil if successful
}

// In command
return dataLoadedMsg{err: err}  // or
return dataLoadedMsg{data: data}

// In Update
case dataLoadedMsg:
    if msg.err != nil {
        m.state.SetError(msg.err.Error())
        return m, nil
    }
    m.state.SetData(msg.data)
```

### Pattern 2: Separate Error Message

```go
type dataLoadedMsg struct {
    data []Item
}

type dataErrorMsg struct {
    err error
}

// In command
if err != nil {
    return dataErrorMsg{err: err}
}
return dataLoadedMsg{data: data}

// In Update
case dataLoadedMsg:
    m.state.SetData(msg.data)
case dataErrorMsg:
    m.state.SetError(msg.err.Error())
```

Pattern 1 is more common and concise. Pattern 2 is clearer when success and error handling are very different.

---

## Chaining Commands

Sometimes you need to run commands in sequence:

```go
case authSuccessMsg:
    m.apiClient.SetIDToken(msg.token)
    m.currentScreen = gameweekScreenType
    // After auth succeeds, fetch gameweeks
    return m, fetchGameWeeksCmd(m.apiClient)

case gameWeeksLoadedMsg:
    m.state.SetGameWeeks(msg.gameWeeks)
    // After gameweeks load, fetch fixtures for first one
    if len(msg.gameWeeks) > 0 {
        return m, fetchFixturesCmd(m.apiClient, msg.gameWeeks[0].GameWeekID)
    }
```

Each message handler can return another command.

---

## Parallel Commands with tea.Batch

Run multiple commands simultaneously:

```go
func (m Model) Init() tea.Cmd {
    return tea.Batch(
        fetchGameWeeksCmd(m.apiClient),
        fetchUserProfileCmd(m.apiClient),
        textinput.Blink,
    )
}
```

All three commands run in parallel. Messages arrive as each completes.

---

## Loading States

Show loading indicators while commands run:

```go
// In model
type Model struct {
    isLoading      bool
    loadingMessage string
}

// When starting async operation
case "enter":
    m.isLoading = true
    m.loadingMessage = "Loading gameweeks..."
    return m, fetchGameWeeksCmd(m.apiClient)

// When operation completes
case gameWeeksLoadedMsg:
    m.isLoading = false
    m.loadingMessage = ""
    // Handle data...

// In View
func (m Model) View() string {
    if m.isLoading {
        return fmt.Sprintf("⏳ %s", m.loadingMessage)
    }
    return m.renderMainView()
}
```

---

## Real Example: Updating Data

```go
// Message
type fixtureUpdatedMsg struct {
    err error
}

// Command
func updateFixtureCmd(fixture *models.Fixture, prefix string) tea.Cmd {
    return func() tea.Msg {
        ctx := context.Background()
        
        // Create DynamoDB client
        tableName := prefix + "-GameWeekFixtures"
        client, err := aws.NewDynamoDBClient(ctx, "eu-west-2", tableName)
        if err != nil {
            return fixtureUpdatedMsg{err: err}
        }
        
        // Update the fixture
        err = client.UpdateFixture(ctx, fixture)
        return fixtureUpdatedMsg{err: err}
    }
}

// In Update
case "s":  // Save
    if m.editingFixture {
        fixture := m.state.Fixtures[m.selectedIdx]
        m.editingFixture = false
        return m, updateFixtureCmd(&fixture, m.prefix)
    }

case fixtureUpdatedMsg:
    if msg.err != nil {
        m.state.SetError(msg.err.Error())
    } else {
        // Refresh data after successful update
        return m, fetchFixturesCmd(m.apiClient, m.state.CurrentGameWeek.GameWeekID)
    }
```

---

## Command Best Practices

### 1. Keep Commands Pure

Commands should only:
- Read from parameters (closure)
- Make network/IO calls
- Return a message

They should NOT:
- Modify the model directly
- Access global state
- Have side effects beyond their purpose

### 2. Handle All Errors

```go
func fetchDataCmd(client *APIClient) tea.Cmd {
    return func() tea.Msg {
        data, err := client.GetData(context.Background())
        if err != nil {
            return dataLoadedMsg{err: err}  // Always handle errors
        }
        return dataLoadedMsg{data: data}
    }
}
```

### 3. Use Context for Cancellation

```go
func fetchDataCmd(client *APIClient) tea.Cmd {
    return func() tea.Msg {
        ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
        defer cancel()
        
        data, err := client.GetData(ctx)
        // ...
    }
}
```

### 4. Name Commands Consistently

```go
fetchGameWeeksCmd()    // Fetches gameweeks
updateFixtureCmd()     // Updates a fixture
authenticateCmd()      // Authenticates user
triggerGitHubActionCmd() // Triggers GitHub action
```

Pattern: `[verb][noun]Cmd`

---

## Key Takeaways

1. **Commands run async** - Don't block the UI
2. **Commands return messages** - Results come back through Update
3. **Closures capture dependencies** - Pass what the command needs
4. **Always handle errors** - Include error in message or use separate error message
5. **Chain commands** - Return new command from message handler
6. **tea.Batch for parallel** - Run multiple commands simultaneously
7. **Show loading states** - Set flag before command, clear on result

---

## Exercise

1. Add a timeout to `fetchGameWeeksCmd` - if it takes more than 10 seconds, return an error
2. Create a `refreshCmd` that fetches both gameweeks and fixtures in parallel
3. Add a retry mechanism - if fetch fails, automatically retry up to 3 times
4. Implement a "cancel" feature - pressing Escape while loading cancels the operation

---

[← Previous: Building Your First Screen](./07-first-screen.md) | [Next: Chapter 9 - Multi-Screen Navigation →](./09-navigation.md)
