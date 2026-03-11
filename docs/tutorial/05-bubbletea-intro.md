# Chapter 5: Introduction to Bubbletea

## What You'll Learn

- What Bubbletea is and why we use it
- The Model-Update-View pattern
- How terminal UIs work differently from web UIs
- Your first Bubbletea program

---

## What is Bubbletea?

Bubbletea is a Go framework for building terminal user interfaces. It's made by [Charm](https://charm.sh/) and is the most popular TUI framework in the Go ecosystem.

Think of it like React, but for terminals:
- **React**: Browser renders HTML/CSS based on component state
- **Bubbletea**: Terminal renders text based on model state

---

## Why Terminal UIs?

Terminal UIs are useful when:
- Users are already in a terminal (developers, sysadmins)
- You need to work over SSH
- You want fast, keyboard-driven interfaces
- You don't want to build a web frontend

Examples: `vim`, `htop`, `lazygit`, `k9s`

---

## How Terminal UIs Work

Terminals display a grid of characters. Your program:
1. Clears the screen (or part of it)
2. Prints characters at specific positions
3. Waits for input
4. Repeats

Bubbletea handles the low-level details. You just describe what to show.

---

## The Elm Architecture

Bubbletea uses the **Elm Architecture**, named after the Elm programming language. It has three parts:

```
┌─────────────────────────────────────────────────────────┐
│                                                         │
│    ┌─────────┐     ┌─────────┐     ┌─────────┐         │
│    │  Model  │────▶│  View   │────▶│ Screen  │         │
│    └─────────┘     └─────────┘     └─────────┘         │
│         ▲                                               │
│         │                                               │
│    ┌─────────┐     ┌─────────┐                         │
│    │ Update  │◀────│   Msg   │◀──── User Input         │
│    └─────────┘     └─────────┘                         │
│                                                         │
└─────────────────────────────────────────────────────────┘
```

1. **Model**: Your application state (a struct)
2. **Update**: A function that takes a message and returns a new model
3. **View**: A function that takes the model and returns a string to display

---

## Your First Bubbletea Program

Create a file `simple/main.go`:

```go
package main

import (
    "fmt"
    "os"

    tea "github.com/charmbracelet/bubbletea"
)

// MODEL - holds all application state
type model struct {
    count int
}

// INIT - returns initial command (none for now)
func (m model) Init() tea.Cmd {
    return nil
}

// UPDATE - handles messages, returns new model
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.KeyMsg:
        switch msg.String() {
        case "q", "ctrl+c":
            return m, tea.Quit
        case "up", "k":
            m.count++
        case "down", "j":
            m.count--
        }
    }
    return m, nil
}

// VIEW - returns string to display
func (m model) View() string {
    return fmt.Sprintf(
        "Count: %d\n\n↑/k: increment • ↓/j: decrement • q: quit",
        m.count,
    )
}

func main() {
    p := tea.NewProgram(model{count: 0})
    if _, err := p.Run(); err != nil {
        fmt.Printf("Error: %v\n", err)
        os.Exit(1)
    }
}
```

Run it:

```bash
go run simple/main.go
```

Press up/down arrows or j/k to change the count. Press q to quit.

---

## Breaking Down the Code

### The Model

```go
type model struct {
    count int
}
```

This is your entire application state. Everything the UI needs to render must be here.

### The Init Function

```go
func (m model) Init() tea.Cmd {
    return nil
}
```

Called once when the program starts. Returns a command to run (or nil for nothing).

### The Update Function

```go
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.KeyMsg:
        switch msg.String() {
        case "q":
            return m, tea.Quit
        case "up":
            m.count++
        }
    }
    return m, nil
}
```

This is the heart of your application. It:
1. Receives a message (keyboard input, timer, custom event)
2. Updates the model based on the message
3. Returns the new model and optionally a command

### The View Function

```go
func (m model) View() string {
    return fmt.Sprintf("Count: %d\n\n...", m.count)
}
```

Takes the model, returns a string. That's it. No side effects, no state changes.

---

## Message Types

Bubbletea sends different message types:

```go
switch msg := msg.(type) {
case tea.KeyMsg:
    // Keyboard input
    // msg.String() returns "a", "enter", "ctrl+c", etc.

case tea.MouseMsg:
    // Mouse input (if enabled)
    // msg.X, msg.Y for position
    // msg.Type for click type

case tea.WindowSizeMsg:
    // Terminal was resized
    // msg.Width, msg.Height
}
```

You can also define custom message types (covered in Chapter 8).

---

## The tea.Cmd Type

Commands are functions that return messages:

```go
type Cmd func() Msg
```

Built-in commands:

```go
tea.Quit          // Exit the program
tea.ClearScreen   // Clear the terminal
tea.Batch(...)    // Run multiple commands
```

Custom commands (covered in Chapter 8):

```go
func fetchData() tea.Msg {
    // Do async work
    return dataLoadedMsg{data: result}
}
```

---

## Adding Styling with Lipgloss

Bubbletea works with Lipgloss for styling. Update the view:

```go
import "github.com/charmbracelet/lipgloss"

var (
    titleStyle = lipgloss.NewStyle().
        Bold(true).
        Foreground(lipgloss.Color("205"))
    
    countStyle = lipgloss.NewStyle().
        Foreground(lipgloss.Color("86")).
        Bold(true)
)

func (m model) View() string {
    title := titleStyle.Render("Counter App")
    count := countStyle.Render(fmt.Sprintf("%d", m.count))
    
    return fmt.Sprintf(
        "%s\n\nCount: %s\n\n↑/k: increment • ↓/j: decrement • q: quit",
        title,
        count,
    )
}
```

Lipgloss provides:
- Colors (foreground, background)
- Text styles (bold, italic, underline)
- Borders and padding
- Layout (width, height, alignment)

---

## The Program Options

When creating the program:

```go
p := tea.NewProgram(
    model{},
    tea.WithAltScreen(),      // Use alternate screen buffer
    tea.WithMouseCellMotion(), // Enable mouse tracking
)
```

### Alternate Screen

Without `WithAltScreen()`:
- UI renders in the normal terminal
- Previous output stays visible
- UI remains after exit

With `WithAltScreen()`:
- UI renders in a separate buffer
- Clean slate
- Returns to previous terminal state on exit

Most TUI apps use alternate screen.

### Mouse Support

`WithMouseCellMotion()` enables mouse events. Without it, `tea.MouseMsg` is never sent.

---

## Common Patterns

### Handling Multiple Keys

```go
case tea.KeyMsg:
    switch msg.String() {
    case "q", "ctrl+c", "esc":
        return m, tea.Quit
    case "up", "k":
        m.count++
    case "down", "j":
        m.count--
    case "enter", " ":
        m.selected = true
    }
```

### Conditional Rendering

```go
func (m model) View() string {
    if m.loading {
        return "Loading..."
    }
    
    if m.error != nil {
        return fmt.Sprintf("Error: %v", m.error)
    }
    
    return m.renderMainView()
}
```

### Window Size

```go
type model struct {
    width  int
    height int
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.WindowSizeMsg:
        m.width = msg.Width
        m.height = msg.Height
    }
    return m, nil
}
```

---

## Key Takeaways

1. **Model holds all state** - Everything the UI needs
2. **Update handles messages** - Returns new model + optional command
3. **View renders to string** - Pure function, no side effects
4. **Messages drive changes** - Keyboard, mouse, custom events
5. **Commands for async work** - Functions that return messages
6. **Lipgloss for styling** - Colors, borders, layout

---

## Exercise

1. Add a "reset" key (r) that sets count back to 0
2. Add bounds checking so count stays between 0 and 100
3. Change the color of the count based on its value (red if negative, green if positive)
4. Add a "help" toggle (?) that shows/hides the keyboard shortcuts

---

[← Previous: Configuration Management](./04-configuration.md) | [Next: Chapter 6 - The Elm Architecture Deep Dive →](./06-elm-architecture.md)
