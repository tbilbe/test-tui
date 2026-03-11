# Chapter 12: State Management

## What You'll Learn

- Centralizing application state
- State mutation patterns
- Derived state and computed values
- Debugging state changes

---

## The Problem with Scattered State

Without centralized state:

```go
type Model struct {
    gameWeeks       []models.GameWeek
    currentGameWeek *models.GameWeek
    fixtures        []models.Fixture
    isLoading       bool
    errorMessage    string
    successMessage  string
    // ... 20 more fields
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    case dataLoadedMsg:
        m.gameWeeks = msg.gameWeeks
        m.isLoading = false
        m.errorMessage = ""  // Did I remember to clear this?
        // Easy to forget related state changes
}
```

Problems:
- Easy to forget to update related fields
- State changes scattered across Update
- Hard to track what changed

---

## The AppState Pattern

Create `internal/models/state.go`:

```go
package models

type AppState struct {
    // Data
    GameWeeks       []GameWeek
    CurrentGameWeek *GameWeek
    Fixtures        []Fixture
    SelectedFixture *Fixture
    Selection       *Selection
    
    // Auth
    IsAuthenticated bool
    Username        string
    
    // UI Feedback
    ErrorMessage   string
    SuccessMessage string
    IsLoading      bool
    LoadingMessage string
}

func NewAppState() *AppState {
    return &AppState{
        GameWeeks: []GameWeek{},
        Fixtures:  []Fixture{},
    }
}
```

### Why a Separate Struct?

1. **Single source of truth** - All data in one place
2. **Encapsulated mutations** - Changes go through methods
3. **Reusable** - Same state could be used by different UIs
4. **Testable** - Test state logic independently

---

## State Mutation Methods

```go
func (s *AppState) SetGameWeeks(gameWeeks []GameWeek) {
    s.GameWeeks = gameWeeks
}

func (s *AppState) SetCurrentGameWeek(gw *GameWeek) {
    s.CurrentGameWeek = gw
}

func (s *AppState) SetFixtures(fixtures []Fixture) {
    s.Fixtures = fixtures
}

func (s *AppState) SetSelectedFixture(fixture *Fixture) {
    s.SelectedFixture = fixture
}
```

### Methods with Side Effects

```go
func (s *AppState) SetError(msg string) {
    s.ErrorMessage = msg
    s.SuccessMessage = ""  // Clear success when setting error
}

func (s *AppState) SetSuccess(msg string) {
    s.SuccessMessage = msg
    s.ErrorMessage = ""  // Clear error when setting success
}

func (s *AppState) ClearMessages() {
    s.ErrorMessage = ""
    s.SuccessMessage = ""
}

func (s *AppState) SetLoading(loading bool, message string) {
    s.IsLoading = loading
    s.LoadingMessage = message
    if loading {
        s.ClearMessages()  // Clear messages when starting to load
    }
}
```

### Why Methods Instead of Direct Assignment?

```go
// Without methods - easy to forget
m.state.ErrorMessage = "Something went wrong"
// Oops, forgot to clear SuccessMessage

// With methods - side effects handled
m.state.SetError("Something went wrong")
// SuccessMessage automatically cleared
```

---

## Using State in the Model

```go
type Model struct {
    state *models.AppState  // Pointer to shared state
    
    // UI-specific state (not in AppState)
    currentScreen screenType
    focusedPane   focusPane
    selectedIdx   int
}

func NewModel(authClient *aws.AuthClient, apiClient *aws.APIClient, dynamoClient *aws.DynamoDBClient) Model {
    return Model{
        state:         models.NewAppState(),
        currentScreen: authScreenType,
    }
}
```

### What Goes in AppState vs Model?

**AppState** - Data and feedback:
- Domain data (gameweeks, fixtures)
- User feedback (errors, success messages)
- Loading states

**Model** - UI state:
- Current screen
- Focus state
- Selection indices
- Modal visibility

---

## State in Update

```go
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case gameWeeksLoadedMsg:
        if msg.err != nil {
            m.state.SetError(msg.err.Error())
        } else {
            m.state.SetGameWeeks(msg.gameWeeks)
            m.state.ClearMessages()
        }
        return m, nil
        
    case fixtureUpdatedMsg:
        if msg.err != nil {
            m.state.SetError(msg.err.Error())
        } else {
            m.state.SetSuccess("Fixture updated successfully")
            // Refresh data
            return m, fetchFixturesCmd(m.apiClient, m.state.CurrentGameWeek.GameWeekID)
        }
        return m, nil
    }
}
```

---

## Computed/Derived State

Sometimes you need values computed from state:

```go
// In AppState
func (s *AppState) CurrentGameWeekLabel() string {
    if s.CurrentGameWeek == nil {
        return "No gameweek selected"
    }
    return s.CurrentGameWeek.Label
}

func (s *AppState) FixtureCount() int {
    return len(s.Fixtures)
}

func (s *AppState) HasError() bool {
    return s.ErrorMessage != ""
}

func (s *AppState) IsCurrentGameWeek(gw *GameWeek) bool {
    if s.CurrentGameWeek == nil || gw == nil {
        return false
    }
    return s.CurrentGameWeek.GameWeekID == gw.GameWeekID
}
```

Usage in View:

```go
func (m Model) View() string {
    if m.state.HasError() {
        return errorStyle.Render(m.state.ErrorMessage)
    }
    
    title := fmt.Sprintf("Fixtures (%d)", m.state.FixtureCount())
    // ...
}
```

---

## State Validation

Add validation methods:

```go
func (s *AppState) CanSelectFixture() bool {
    return s.CurrentGameWeek != nil && len(s.Fixtures) > 0
}

func (s *AppState) CanEditFixture() bool {
    return s.SelectedFixture != nil
}

func (s *AppState) ValidateFixtureEdit(fixture *Fixture) error {
    if fixture.Period == "" {
        return fmt.Errorf("period is required")
    }
    if fixture.HomeScore != nil && *fixture.HomeScore < 0 {
        return fmt.Errorf("home score cannot be negative")
    }
    return nil
}
```

Usage:

```go
case "e":  // Edit
    if !m.state.CanEditFixture() {
        return m, nil  // Ignore if can't edit
    }
    m.editingFixture = true
```

---

## Debugging State

### Logging State Changes

```go
var logger *log.Logger

func init() {
    logFile, _ := os.OpenFile("app.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
    logger = log.New(logFile, "", log.LstdFlags|log.Lshortfile)
}

func (s *AppState) SetGameWeeks(gameWeeks []GameWeek) {
    logger.Printf("SetGameWeeks: %d gameweeks", len(gameWeeks))
    s.GameWeeks = gameWeeks
}

func (s *AppState) SetError(msg string) {
    logger.Printf("SetError: %s", msg)
    s.ErrorMessage = msg
    s.SuccessMessage = ""
}
```

### State Snapshot

```go
func (s *AppState) Debug() string {
    return fmt.Sprintf(
        "AppState{GameWeeks: %d, CurrentGW: %v, Fixtures: %d, Loading: %v, Error: %q}",
        len(s.GameWeeks),
        s.CurrentGameWeek != nil,
        len(s.Fixtures),
        s.IsLoading,
        s.ErrorMessage,
    )
}
```

Usage:

```go
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    logger.Printf("Before: %s", m.state.Debug())
    // ... handle message
    logger.Printf("After: %s", m.state.Debug())
}
```

---

## State Reset

For navigation or error recovery:

```go
func (s *AppState) Reset() {
    s.GameWeeks = []GameWeek{}
    s.CurrentGameWeek = nil
    s.Fixtures = []Fixture{}
    s.SelectedFixture = nil
    s.ClearMessages()
    s.IsLoading = false
}

func (s *AppState) ResetFixtures() {
    s.Fixtures = []Fixture{}
    s.SelectedFixture = nil
}
```

Usage:

```go
case "g":  // Go back
    m.state.ResetFixtures()
    m.currentScreen = gameweekScreenType
```

---

## Immutability Considerations

Go doesn't enforce immutability, but you can design for it:

```go
// Instead of modifying in place
func (s *AppState) SetGameWeeks(gameWeeks []GameWeek) {
    s.GameWeeks = gameWeeks
}

// Return a new state (more functional)
func (s AppState) WithGameWeeks(gameWeeks []GameWeek) AppState {
    s.GameWeeks = gameWeeks
    return s
}
```

The second approach is more "pure" but less common in Go. The first approach is pragmatic and works well with Bubbletea.

---

## Key Takeaways

1. **Centralize state** - One struct for all application data
2. **Use methods for mutations** - Encapsulate side effects
3. **Separate data from UI state** - AppState vs Model fields
4. **Computed properties** - Methods that derive values
5. **Validation methods** - Check if actions are allowed
6. **Logging for debugging** - Track state changes
7. **Reset methods** - Clean state for navigation

---

## Exercise

1. Add a `GetFixtureByID(id string) *Fixture` method to AppState
2. Implement an "undo" feature that remembers the last state change
3. Add a `StateChanged` channel that notifies when state changes (for external observers)
4. Create a `Snapshot()` method that returns a deep copy of the state

---

[← Previous: API Client Patterns](./11-api-client.md) | [Next: Chapter 13 - Testing TUI Applications →](./13-testing.md)
