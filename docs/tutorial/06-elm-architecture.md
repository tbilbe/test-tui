# Chapter 6: The Elm Architecture Deep Dive

## What You'll Learn

- Why unidirectional data flow matters
- How to structure complex state
- Message design patterns
- Composing models and updates

---

## The Problem with Traditional UI

In traditional UI programming:

```go
// Pseudo-code - DON'T DO THIS
button.OnClick(func() {
    label.SetText("Clicked!")
    counter++
    updateDatabase(counter)
    if counter > 10 {
        otherLabel.SetText("Too many!")
    }
})
```

Problems:
- State scattered everywhere
- Hard to track what changed
- Race conditions with async operations
- Difficult to test

---

## Unidirectional Data Flow

The Elm Architecture enforces one-way data flow:

```
User Input → Message → Update → New Model → View → Screen
                ↑                              │
                └──────────────────────────────┘
```

Rules:
1. **View never modifies state** - It only reads the model
2. **Update is the only place state changes** - All mutations go through Update
3. **Messages describe what happened** - Not what to do

---

## Designing Your Model

### Start with What You Need to Display

Ask: "What does the UI need to show?"

For our app:
- List of gameweeks
- Currently selected gameweek
- List of fixtures for that gameweek
- Loading states
- Error messages

```go
type Model struct {
    // Data
    gameWeeks       []models.GameWeek
    currentGameWeek *models.GameWeek
    fixtures        []models.Fixture
    
    // UI State
    isLoading      bool
    loadingMessage string
    errorMessage   string
    
    // Navigation
    currentScreen screenType
    focusedPane   focusPane
}
```

### Separate Data from UI State

```go
// Data - comes from API/database
gameWeeks []models.GameWeek
fixtures  []models.Fixture

// UI State - local to the application
isLoading    bool
selectedIdx  int
editingField int
```

This separation helps when:
- Refreshing data (keep UI state, replace data)
- Testing (mock data, test UI state changes)

---

## The AppState Pattern

Our app uses a separate `AppState` struct:

```go
// internal/models/state.go
type AppState struct {
    GameWeeks       []GameWeek
    CurrentGameWeek *GameWeek
    Fixtures        []Fixture
    IsLoading       bool
    ErrorMessage    string
}

func NewAppState() *AppState {
    return &AppState{
        GameWeeks: []GameWeek{},
        Fixtures:  []Fixture{},
    }
}

func (s *AppState) SetError(msg string) {
    s.ErrorMessage = msg
    s.SuccessMessage = ""  // Clear success when setting error
}
```

### Why a Separate Struct?

1. **Reusability** - Same state structure could be used by different UIs
2. **Encapsulation** - State mutations in one place
3. **Side effects** - `SetError` clears success message automatically

### Using It in the Model

```go
type Model struct {
    state *models.AppState  // Pointer to shared state
    
    // UI-specific state
    currentScreen screenType
    focusedPane   focusPane
}
```

---

## Designing Messages

Messages describe events, not actions:

```go
// Good - describes what happened
type authSuccessMsg struct {
    token string
}

type gameWeeksLoadedMsg struct {
    gameWeeks []models.GameWeek
    err       error
}

// Bad - describes what to do
type setTokenMsg struct {  // "set" is an action
    token string
}
```

### Message Naming Convention

```go
type [noun][verb]Msg struct {}

// Examples
type authSuccessMsg struct{}      // Auth succeeded
type authErrorMsg struct{}        // Auth failed
type gameWeeksLoadedMsg struct{}  // GameWeeks finished loading
type fixtureUpdatedMsg struct{}   // Fixture was updated
```

### Including Error in Messages

```go
type gameWeeksLoadedMsg struct {
    gameWeeks []models.GameWeek
    err       error  // nil if successful
}
```

This pattern lets Update handle both success and failure:

```go
case gameWeeksLoadedMsg:
    if msg.err != nil {
        m.state.SetError(msg.err.Error())
        return m, nil
    }
    m.state.SetGameWeeks(msg.gameWeeks)
    return m, nil
```

---

## The Update Function Structure

### Type Switch Pattern

```go
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.KeyMsg:
        // Handle keyboard
    case tea.WindowSizeMsg:
        // Handle resize
    case authSuccessMsg:
        // Handle auth success
    case gameWeeksLoadedMsg:
        // Handle data loaded
    }
    return m, nil
}
```

### Routing to Sub-Updates

For complex apps, route to screen-specific handlers:

```go
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    // Global handlers first
    switch msg := msg.(type) {
    case tea.KeyMsg:
        if msg.String() == "ctrl+c" {
            return m, tea.Quit
        }
    case tea.WindowSizeMsg:
        m.width = msg.Width
        m.height = msg.Height
    }
    
    // Handle async results (these can come from any screen)
    switch msg := msg.(type) {
    case authSuccessMsg:
        m.apiClient.SetIDToken(msg.token)
        m.currentScreen = gameweekScreenType
        return m, fetchGameWeeksCmd(m.apiClient)
    case gameWeeksLoadedMsg:
        if msg.err != nil {
            m.state.SetError(msg.err.Error())
        } else {
            m.state.SetGameWeeks(msg.gameWeeks)
        }
        return m, nil
    }
    
    // Route to screen-specific handler
    switch m.currentScreen {
    case authScreenType:
        return m.updateAuthScreen(msg)
    case gameweekScreenType:
        return m.updateGameweekScreen(msg)
    case fixtureScreenType:
        return m.updateFixtureScreen(msg)
    }
    
    return m, nil
}
```

### Screen-Specific Handlers

```go
func (m Model) updateGameweekScreen(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.KeyMsg:
        switch msg.String() {
        case "q":
            return m, tea.Quit
        case "enter":
            // Select gameweek
            if m.selectedIdx < len(m.state.GameWeeks) {
                gw := &m.state.GameWeeks[m.selectedIdx]
                m.state.SetCurrentGameWeek(gw)
                m.currentScreen = fixtureScreenType
                return m, fetchFixturesCmd(m.apiClient, gw.GameWeekID)
            }
        case "up", "k":
            if m.selectedIdx > 0 {
                m.selectedIdx--
            }
        case "down", "j":
            if m.selectedIdx < len(m.state.GameWeeks)-1 {
                m.selectedIdx++
            }
        }
    }
    return m, nil
}
```

---

## Composing Components

### Embedding Sub-Models

```go
type Model struct {
    authScreen AuthScreen  // Embedded component
    // ...
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    if m.currentScreen == authScreenType {
        var cmd tea.Cmd
        m.authScreen, cmd = m.authScreen.Update(msg)
        return m, cmd
    }
    // ...
}
```

### The AuthScreen Component

```go
type AuthScreen struct {
    usernameInput textinput.Model
    passwordInput textinput.Model
    focusIndex    int
}

func NewAuthScreen() AuthScreen {
    username := textinput.New()
    username.Placeholder = "Username"
    username.Focus()
    
    password := textinput.New()
    password.Placeholder = "Password"
    password.EchoMode = textinput.EchoPassword
    
    return AuthScreen{
        usernameInput: username,
        passwordInput: password,
    }
}

func (a AuthScreen) Update(msg tea.Msg) (AuthScreen, tea.Cmd) {
    // Handle input switching, submission, etc.
    // Returns updated AuthScreen, not tea.Model
}

func (a AuthScreen) View() string {
    return boxStyle.Render(
        a.usernameInput.View() + "\n" +
        a.passwordInput.View(),
    )
}
```

### Why Components Return Themselves

```go
// Component returns its own type
func (a AuthScreen) Update(msg tea.Msg) (AuthScreen, tea.Cmd)

// Not tea.Model
func (a AuthScreen) Update(msg tea.Msg) (tea.Model, tea.Cmd)
```

This allows the parent to update its field:

```go
m.authScreen, cmd = m.authScreen.Update(msg)
```

If it returned `tea.Model`, you'd need a type assertion:

```go
result, cmd := m.authScreen.Update(msg)
m.authScreen = result.(AuthScreen)  // Ugly and error-prone
```

---

## State Transitions

### Screen Navigation

```go
type screenType int

const (
    authScreenType screenType = iota
    prefixScreenType
    gameweekScreenType
    fixtureScreenType
)
```

Using `iota` for sequential constants. Transitions happen in Update:

```go
case authSuccessMsg:
    m.currentScreen = prefixScreenType  // Navigate to next screen
    return m, nil

case "enter":
    if m.currentScreen == prefixScreenType {
        m.currentScreen = gameweekScreenType
        return m, fetchGameWeeksCmd(m.apiClient)
    }
```

### Modal States

```go
type Model struct {
    // ...
    editingFixture bool
    showBatchModal bool
    showGoalModal  bool
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    // Modal takes priority
    if m.showBatchModal {
        return m.updateBatchModal(msg)
    }
    if m.editingFixture {
        return m.updateEditModal(msg)
    }
    // Normal screen handling
}
```

---

## Key Takeaways

1. **One-way data flow** - Messages → Update → Model → View
2. **Model holds everything** - All state in one place
3. **Messages describe events** - Not actions
4. **Route by screen** - Keep Update organized
5. **Components return themselves** - Not tea.Model
6. **Separate data from UI state** - Easier to reason about

---

## Exercise

1. Add a `confirmQuit` boolean to the model. When user presses 'q', show "Press q again to quit" instead of quitting immediately.
2. Create a `StatusBar` component that shows the current screen name and any error messages.
3. Implement a "back" navigation that remembers the previous screen.

---

[← Previous: Introduction to Bubbletea](./05-bubbletea-intro.md) | [Next: Chapter 7 - Building Your First Screen →](./07-first-screen.md)
