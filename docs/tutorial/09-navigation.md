# Chapter 9: Multi-Screen Navigation

## What You'll Learn

- How to structure a multi-screen application
- Navigation patterns and state management
- Focus management across screens
- Modal dialogs and overlays

---

## Screen Types

Define your screens as constants:

```go
type screenType int

const (
    authScreenType screenType = iota
    prefixScreenType
    gameweekScreenType
    fixtureScreenType
)
```

Using `iota` gives you sequential integers (0, 1, 2, 3). This is Go's way of creating enums.

---

## The Navigation Model

```go
type Model struct {
    currentScreen screenType
    
    // Screen-specific state
    authScreen AuthScreen
    
    // Shared state
    state *models.AppState
    
    // Modal states
    editingFixture bool
    showBatchModal bool
}
```

### Screen State vs Shared State

**Screen-specific**: Only relevant to one screen
```go
authScreen AuthScreen  // Only used on auth screen
selectedIdx int        // Only used on list screens
```

**Shared state**: Used across screens
```go
state *models.AppState  // GameWeeks, Fixtures, etc.
apiClient *aws.APIClient
```

---

## Routing in Update

```go
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    // 1. Global handlers (work on any screen)
    switch msg := msg.(type) {
    case tea.KeyMsg:
        if msg.String() == "ctrl+c" {
            return m, tea.Quit
        }
    case tea.WindowSizeMsg:
        m.width = msg.Width
        m.height = msg.Height
    }
    
    // 2. Async message handlers (can arrive on any screen)
    switch msg := msg.(type) {
    case authSuccessMsg:
        m.apiClient.SetIDToken(msg.token)
        m.currentScreen = prefixScreenType
        return m, nil
    case gameWeeksLoadedMsg:
        m.state.SetGameWeeks(msg.gameWeeks)
        return m, nil
    }
    
    // 3. Route to screen-specific handler
    switch m.currentScreen {
    case authScreenType:
        return m.updateAuthScreen(msg)
    case prefixScreenType:
        return m.updatePrefixScreen(msg)
    case gameweekScreenType:
        return m.updateGameweekScreen(msg)
    case fixtureScreenType:
        return m.updateFixtureScreen(msg)
    }
    
    return m, nil
}
```

### Why This Order?

1. **Global handlers first** - Ctrl+C should always work
2. **Async handlers next** - Results can arrive regardless of current screen
3. **Screen routing last** - Screen-specific logic

---

## Screen-Specific Handlers

```go
func (m Model) updatePrefixScreen(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.KeyMsg:
        switch msg.String() {
        case "q":
            return m, tea.Quit
        case "enter":
            m.prefix = m.input
            m.currentScreen = gameweekScreenType
            return m, fetchGameWeeksCmd(m.apiClient)
        case "backspace":
            if len(m.input) > 0 {
                m.input = m.input[:len(m.input)-1]
            }
        default:
            if len(msg.String()) == 1 {
                m.input += msg.String()
            }
        }
    }
    return m, nil
}
```

Each handler:
- Only handles messages relevant to that screen
- Can change `currentScreen` to navigate
- Can return commands to fetch data

---

## Navigation Patterns

### Forward Navigation

```go
case "enter":
    m.currentScreen = nextScreen
    return m, fetchDataForNextScreen()
```

### Back Navigation

```go
case "g", "esc":
    m.currentScreen = previousScreen
    // Clear screen-specific state
    m.state.SetFixtures([]models.Fixture{})
    return m, nil
```

### Navigation with Data

```go
case "enter":
    // Save selection before navigating
    gw := m.state.GameWeeks[m.selectedIdx]
    m.state.SetCurrentGameWeek(&gw)
    m.currentScreen = fixtureScreenType
    return m, fetchFixturesCmd(m.apiClient, gw.GameWeekID)
```

---

## The View Router

```go
func (m Model) View() string {
    // Modal overlays take priority
    if m.showBatchModal {
        return m.renderBatchModal()
    }
    if m.editingFixture {
        return m.renderEditModal()
    }
    
    // Route to screen view
    switch m.currentScreen {
    case authScreenType:
        return m.authScreen.View()
    case prefixScreenType:
        return m.renderPrefixScreen()
    case gameweekScreenType:
        return m.renderGameweekScreen()
    case fixtureScreenType:
        return m.renderFixtureScreen()
    }
    
    return "Unknown screen"
}
```

---

## Focus Management

When you have multiple interactive elements:

```go
type focusPane int

const (
    focusGameWeek focusPane = iota
    focusFixtures
)

type Model struct {
    focusedPane focusPane
}

func (m Model) updateFixtureScreen(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.KeyMsg:
        switch msg.String() {
        case "tab":
            // Cycle focus
            if m.focusedPane == focusGameWeek {
                m.focusedPane = focusFixtures
            } else {
                m.focusedPane = focusGameWeek
            }
        case "up", "k":
            // Handle based on focused pane
            if m.focusedPane == focusFixtures {
                m.fixturesTable.MoveUp(1)
            }
        }
    }
}
```

### Visual Focus Indicators

```go
func (m Model) renderPane(title string, content string, focused bool) string {
    style := boxStyle.Copy()
    if focused {
        style = style.BorderForeground(lipgloss.Color("205"))  // Highlight
    } else {
        style = style.BorderForeground(lipgloss.Color("240"))  // Dim
    }
    return style.Render(titleStyle.Render(title) + "\n" + content)
}
```

---

## Modal Dialogs

Modals are screens that overlay the current screen:

```go
type Model struct {
    // Modal states
    editingFixture bool
    editModalField int
    showBatchModal bool
    batchPresetIdx int
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    // Modal handlers take priority
    if m.showBatchModal {
        return m.updateBatchModal(msg)
    }
    if m.editingFixture {
        return m.updateEditModal(msg)
    }
    
    // Normal screen handling
    // ...
}

func (m Model) updateEditModal(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.KeyMsg:
        switch msg.String() {
        case "esc":
            m.editingFixture = false  // Close modal
            return m, nil
        case "s":
            // Save and close
            m.editingFixture = false
            return m, updateFixtureCmd(&fixture, m.prefix)
        case "up", "k":
            if m.editModalField > 0 {
                m.editModalField--
            }
        case "down", "j":
            if m.editModalField < maxField {
                m.editModalField++
            }
        }
    }
    return m, nil
}
```

### Modal View

```go
func (m Model) renderEditModal() string {
    // Render the background (dimmed)
    background := m.renderFixtureScreen()
    
    // Render the modal
    modal := modalStyle.Render(
        titleStyle.Render("Edit Fixture") + "\n\n" +
        m.renderEditFields() + "\n\n" +
        "↑/↓: navigate • s: save • esc: cancel",
    )
    
    // Center the modal over the background
    return lipgloss.Place(
        m.width, m.height,
        lipgloss.Center, lipgloss.Center,
        modal,
        lipgloss.WithWhitespaceBackground(lipgloss.Color("0")),
    )
}
```

---

## Selection Modes

Sometimes you need a "selection mode" within a screen:

```go
type Model struct {
    fixtureSelectMode  bool
    selectedFixtureIdx int
}

func (m Model) updateFixtureScreen(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.KeyMsg:
        switch msg.String() {
        case "enter":
            if !m.fixtureSelectMode {
                // Enter selection mode
                m.fixtureSelectMode = true
                m.selectedFixtureIdx = 0
            } else {
                // Open edit modal for selected item
                m.editingFixture = true
            }
        case "esc":
            if m.fixtureSelectMode {
                // Exit selection mode
                m.fixtureSelectMode = false
            }
        case "up", "k":
            if m.fixtureSelectMode && m.selectedFixtureIdx > 0 {
                m.selectedFixtureIdx--
            }
        }
    }
}
```

---

## State Cleanup on Navigation

When leaving a screen, clean up:

```go
case "g":  // Go back
    m.currentScreen = gameweekScreenType
    
    // Clear fixture-specific state
    m.state.SetFixtures([]models.Fixture{})
    m.state.SetCurrentGameWeek(nil)
    m.fixtureSelectMode = false
    m.selectedFixtureIdx = 0
    m.editingFixture = false
    
    return m, nil
```

---

## Navigation History (Optional)

For complex apps, you might want back/forward:

```go
type Model struct {
    screenHistory []screenType
    currentScreen screenType
}

func (m *Model) navigateTo(screen screenType) {
    m.screenHistory = append(m.screenHistory, m.currentScreen)
    m.currentScreen = screen
}

func (m *Model) goBack() bool {
    if len(m.screenHistory) == 0 {
        return false
    }
    m.currentScreen = m.screenHistory[len(m.screenHistory)-1]
    m.screenHistory = m.screenHistory[:len(m.screenHistory)-1]
    return true
}
```

---

## Key Takeaways

1. **Use constants for screens** - `iota` for sequential values
2. **Route in Update** - Global → Async → Screen-specific
3. **Modals are just state** - Boolean flags control visibility
4. **Clean up on navigation** - Reset screen-specific state
5. **Focus is explicit** - Track which element is focused
6. **Visual feedback** - Show which pane/element is focused

---

## Exercise

1. Add a "breadcrumb" display showing: Auth → Prefix → GameWeek → Fixtures
2. Implement a confirmation dialog when pressing 'q' to quit
3. Add keyboard shortcuts to jump directly to screens (e.g., '1' for gameweeks, '2' for fixtures)
4. Implement a "help" modal that shows all keyboard shortcuts for the current screen

---

[← Previous: Commands & Async Operations](./08-commands.md) | [Next: Chapter 10 - AWS SDK Integration →](./10-aws-sdk.md)
