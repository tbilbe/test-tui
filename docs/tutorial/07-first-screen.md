# Chapter 7: Building Your First Screen

## What You'll Learn

- How to build a complete screen from scratch
- Using Bubbles components (textinput)
- Styling with Lipgloss
- Handling focus and navigation

---

## The Auth Screen

We'll build the login screen step by step. This screen:
- Shows username and password inputs
- Handles tab navigation between fields
- Submits credentials on Enter
- Displays loading state

---

## Step 1: Define the Structure

Create `internal/ui/auth.go`:

```go
package ui

import (
    "github.com/charmbracelet/bubbles/textinput"
    tea "github.com/charmbracelet/bubbletea"
    "github.com/charmbracelet/lipgloss"
)

type AuthScreen struct {
    usernameInput textinput.Model
    passwordInput textinput.Model
    focusIndex    int
    submitted     bool
}
```

### What's Here

- **`textinput.Model`** - A Bubbles component for text input
- **`focusIndex`** - Which field is focused (0 = username, 1 = password)
- **`submitted`** - Whether form was submitted (for loading state)

---

## Step 2: The Constructor

```go
func NewAuthScreen() AuthScreen {
    username := textinput.New()
    username.Placeholder = "Username"
    username.Focus()  // Start with username focused
    username.CharLimit = 50
    username.Width = 40

    password := textinput.New()
    password.Placeholder = "Password"
    password.EchoMode = textinput.EchoPassword  // Hide characters
    password.EchoCharacter = '•'
    password.CharLimit = 50
    password.Width = 40

    return AuthScreen{
        usernameInput: username,
        passwordInput: password,
        focusIndex:    0,
    }
}
```

### Key Points

- **`textinput.New()`** - Creates a new text input component
- **`Focus()`** - Makes this input active (receives keystrokes)
- **`EchoMode`** - Password mode hides input
- **`EchoCharacter`** - What to show instead of actual characters

---

## Step 3: The Init Function

```go
func (a AuthScreen) Init() tea.Cmd {
    return textinput.Blink
}
```

`textinput.Blink` starts the cursor blinking animation. Without it, the cursor won't blink.

---

## Step 4: The Update Function

```go
func (a AuthScreen) Update(msg tea.Msg) (AuthScreen, tea.Cmd) {
    var cmd tea.Cmd

    switch msg := msg.(type) {
    case tea.KeyMsg:
        switch msg.String() {
        case "ctrl+c", "esc":
            return a, tea.Quit
            
        case "tab", "shift+tab", "up", "down":
            // Toggle focus between fields
            if a.focusIndex == 0 {
                a.focusIndex = 1
                a.usernameInput.Blur()
                a.passwordInput.Focus()
            } else {
                a.focusIndex = 0
                a.passwordInput.Blur()
                a.usernameInput.Focus()
            }
            
        case "enter":
            // Submit if both fields have values
            if a.usernameInput.Value() != "" && a.passwordInput.Value() != "" {
                a.submitted = true
                return a, func() tea.Msg {
                    return authSubmitMsg{
                        username: a.usernameInput.Value(),
                        password: a.passwordInput.Value(),
                    }
                }
            }
        }
    }

    // Update the focused input
    if a.focusIndex == 0 {
        a.usernameInput, cmd = a.usernameInput.Update(msg)
    } else {
        a.passwordInput, cmd = a.passwordInput.Update(msg)
    }

    return a, cmd
}
```

### Breaking It Down

**Focus Management:**
```go
a.usernameInput.Blur()   // Remove focus
a.passwordInput.Focus()  // Add focus
```

Only one input can be focused at a time. `Blur()` and `Focus()` manage this.

**Returning a Message:**
```go
return a, func() tea.Msg {
    return authSubmitMsg{
        username: a.usernameInput.Value(),
        password: a.passwordInput.Value(),
    }
}
```

This returns a command that immediately produces a message. The parent model will receive `authSubmitMsg`.

**Delegating to Child:**
```go
a.usernameInput, cmd = a.usernameInput.Update(msg)
```

The textinput component needs to handle keystrokes too (typing characters, backspace, etc.).

---

## Step 5: Define the Message Type

```go
type authSubmitMsg struct {
    username string
    password string
}
```

This message carries the credentials to the parent model.

---

## Step 6: Styling with Lipgloss

```go
var (
    authBoxStyle = lipgloss.NewStyle().
        Border(lipgloss.RoundedBorder()).
        BorderForeground(lipgloss.Color("63")).
        Padding(1, 2).
        Width(50)

    authTitleStyle = lipgloss.NewStyle().
        Bold(true).
        Foreground(lipgloss.Color("205")).
        MarginBottom(1)
)
```

### Lipgloss Basics

**Colors:**
```go
lipgloss.Color("205")     // ANSI color code
lipgloss.Color("#ff0000") // Hex color
lipgloss.Color("red")     // Named color
```

**Borders:**
```go
lipgloss.RoundedBorder()  // ╭───╮
lipgloss.NormalBorder()   // ┌───┐
lipgloss.DoubleBorder()   // ╔═══╗
```

**Spacing:**
```go
Padding(1, 2)      // 1 line top/bottom, 2 chars left/right
Margin(1)          // 1 unit on all sides
MarginBottom(1)    // 1 unit bottom only
```

---

## Step 7: The View Function

```go
func (a AuthScreen) View() string {
    if a.submitted {
        return authBoxStyle.Render(
            authTitleStyle.Render("Authenticating...") + "\n\n" +
                "Please wait...",
        )
    }

    content := authTitleStyle.Render("Seven Test TUI - Login") + "\n\n" +
        a.usernameInput.View() + "\n\n" +
        a.passwordInput.View() + "\n\n" +
        lipgloss.NewStyle().
            Foreground(lipgloss.Color("240")).
            Render("tab: switch • enter: login • esc: quit")

    return authBoxStyle.Render(content)
}
```

### View Patterns

**Conditional Rendering:**
```go
if a.submitted {
    return "Loading..."
}
return "Normal view"
```

**Composing Strings:**
```go
title + "\n\n" + input1 + "\n\n" + input2
```

**Inline Styling:**
```go
lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render("help text")
```

---

## Step 8: Helper Method

```go
func (a AuthScreen) GetCredentials() (string, string) {
    return a.usernameInput.Value(), a.passwordInput.Value()
}
```

Useful for the parent to access values without knowing internal structure.

---

## The Complete Auth Screen

```go
package ui

import (
    "github.com/charmbracelet/bubbles/textinput"
    tea "github.com/charmbracelet/bubbletea"
    "github.com/charmbracelet/lipgloss"
)

var (
    authBoxStyle = lipgloss.NewStyle().
        Border(lipgloss.RoundedBorder()).
        BorderForeground(lipgloss.Color("63")).
        Padding(1, 2).
        Width(50)

    authTitleStyle = lipgloss.NewStyle().
        Bold(true).
        Foreground(lipgloss.Color("205")).
        MarginBottom(1)
)

type AuthScreen struct {
    usernameInput textinput.Model
    passwordInput textinput.Model
    focusIndex    int
    submitted     bool
}

func NewAuthScreen() AuthScreen {
    username := textinput.New()
    username.Placeholder = "Username"
    username.Focus()
    username.CharLimit = 50
    username.Width = 40

    password := textinput.New()
    password.Placeholder = "Password"
    password.EchoMode = textinput.EchoPassword
    password.EchoCharacter = '•'
    password.CharLimit = 50
    password.Width = 40

    return AuthScreen{
        usernameInput: username,
        passwordInput: password,
        focusIndex:    0,
    }
}

func (a AuthScreen) Init() tea.Cmd {
    return textinput.Blink
}

func (a AuthScreen) Update(msg tea.Msg) (AuthScreen, tea.Cmd) {
    var cmd tea.Cmd

    switch msg := msg.(type) {
    case tea.KeyMsg:
        switch msg.String() {
        case "ctrl+c", "esc":
            return a, tea.Quit
        case "tab", "shift+tab", "up", "down":
            if a.focusIndex == 0 {
                a.focusIndex = 1
                a.usernameInput.Blur()
                a.passwordInput.Focus()
            } else {
                a.focusIndex = 0
                a.passwordInput.Blur()
                a.usernameInput.Focus()
            }
        case "enter":
            if a.usernameInput.Value() != "" && a.passwordInput.Value() != "" {
                a.submitted = true
                return a, func() tea.Msg {
                    return authSubmitMsg{
                        username: a.usernameInput.Value(),
                        password: a.passwordInput.Value(),
                    }
                }
            }
        }
    }

    if a.focusIndex == 0 {
        a.usernameInput, cmd = a.usernameInput.Update(msg)
    } else {
        a.passwordInput, cmd = a.passwordInput.Update(msg)
    }

    return a, cmd
}

func (a AuthScreen) View() string {
    if a.submitted {
        return authBoxStyle.Render(
            authTitleStyle.Render("Authenticating...") + "\n\n" +
                "Please wait...",
        )
    }

    content := authTitleStyle.Render("Seven Test TUI - Login") + "\n\n" +
        a.usernameInput.View() + "\n\n" +
        a.passwordInput.View() + "\n\n" +
        lipgloss.NewStyle().
            Foreground(lipgloss.Color("240")).
            Render("tab: switch • enter: login • esc: quit")

    return authBoxStyle.Render(content)
}

type authSubmitMsg struct {
    username string
    password string
}
```

---

## Integrating with the Main Model

In `internal/ui/gameweeks.go`:

```go
type Model struct {
    authScreen    AuthScreen
    currentScreen screenType
    // ...
}

func NewModel(...) Model {
    return Model{
        authScreen:    NewAuthScreen(),
        currentScreen: authScreenType,
        // ...
    }
}

func (m Model) Init() tea.Cmd {
    return m.authScreen.Init()
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    // Handle auth submit message
    switch msg := msg.(type) {
    case authSubmitMsg:
        return m, authenticateCmd(m.authClient, msg.username, msg.password)
    }
    
    // Route to auth screen
    if m.currentScreen == authScreenType {
        var cmd tea.Cmd
        m.authScreen, cmd = m.authScreen.Update(msg)
        return m, cmd
    }
    
    // ...
}

func (m Model) View() string {
    if m.currentScreen == authScreenType {
        return m.authScreen.View()
    }
    // ...
}
```

---

## Key Takeaways

1. **Components are self-contained** - Own state, update, view
2. **Focus management is explicit** - Call `Focus()` and `Blur()`
3. **Messages communicate up** - Child sends message, parent handles
4. **Lipgloss for styling** - Declarative, chainable API
5. **Conditional rendering** - Different views for different states
6. **Delegate to children** - Pass messages to child components

---

## Exercise

1. Add a "Show password" toggle (press 's' to toggle between hidden and visible)
2. Add validation - show error if username is less than 3 characters
3. Add a "Remember me" checkbox (hint: it's just a boolean you toggle)
4. Center the auth box in the terminal (hint: use `lipgloss.Place()`)

---

[← Previous: The Elm Architecture Deep Dive](./06-elm-architecture.md) | [Next: Chapter 8 - Commands & Async Operations →](./08-commands.md)
