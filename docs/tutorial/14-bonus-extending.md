# Bonus Chapter: Extending the Application

## Your Turn to Build

You've learned the patterns. Now it's time to apply them.

This chapter presents feature ideas for you to implement. Each idea includes:
- What the feature does
- Which chapters are relevant
- Hints to get you started
- Questions to consider

Don't just read these—pick one and build it.

---

## Feature Idea 1: Fixture Search

### The Problem

With many fixtures across gameweeks, finding a specific match is tedious. Users scroll through lists manually.

### The Feature

Add a search/filter capability:
- Press `/` to enter search mode
- Type to filter fixtures by team name
- Results update as you type
- Press Enter to select, Esc to cancel

### Relevant Chapters

- Chapter 7: Building screens with text input
- Chapter 6: State management in Update
- Chapter 9: Modal-like overlays

### Hints

```go
type Model struct {
    searchMode   bool
    searchQuery  string
    // ...
}

func (m Model) filteredFixtures() []models.Fixture {
    if m.searchQuery == "" {
        return m.state.Fixtures
    }
    
    var filtered []models.Fixture
    query := strings.ToLower(m.searchQuery)
    for _, f := range m.state.Fixtures {
        home := strings.ToLower(f.Participants.Home.TeamName)
        away := strings.ToLower(f.Participants.Away.TeamName)
        if strings.Contains(home, query) || strings.Contains(away, query) {
            filtered = append(filtered, f)
        }
    }
    return filtered
}
```

### Questions to Consider

- Should search persist when navigating away and back?
- How do you handle no results?
- Could you add fuzzy matching?

---

## Feature Idea 2: Keyboard Shortcuts Help Panel

### The Problem

Users don't know what keys do what. They have to remember or check documentation.

### The Feature

Add a help panel:
- Press `?` to toggle help overlay
- Shows all keyboard shortcuts for current screen
- Different shortcuts per screen
- Styled nicely with Lipgloss

### Relevant Chapters

- Chapter 7: Styling with Lipgloss
- Chapter 9: Modal overlays
- Chapter 6: Screen-specific state

### Hints

```go
type shortcut struct {
    key         string
    description string
}

func (m Model) shortcutsForScreen() []shortcut {
    switch m.currentScreen {
    case gameweekScreenType:
        return []shortcut{
            {"↑/k", "Move up"},
            {"↓/j", "Move down"},
            {"Enter", "Select gameweek"},
            {"q", "Quit"},
            {"?", "Toggle help"},
        }
    case fixtureScreenType:
        return []shortcut{
            {"Tab", "Switch pane"},
            {"e", "Edit fixture"},
            {"g", "Go back"},
            // ...
        }
    }
    return nil
}
```

### Questions to Consider

- Should help be a modal or a sidebar?
- How do you keep shortcuts in sync with actual handlers?
- Could shortcuts be configurable?

---

## Feature Idea 3: Fixture History / Audit Log

### The Problem

When testing, you make changes but forget what you changed. There's no record of modifications.

### The Feature

Track changes locally:
- Log every fixture update with timestamp
- Press `h` to view history for selected fixture
- Show what changed (before → after)
- Optionally persist to a local file

### Relevant Chapters

- Chapter 12: State management
- Chapter 8: Commands for file I/O
- Chapter 2: Structs for history entries

### Hints

```go
type ChangeRecord struct {
    Timestamp   time.Time
    FixtureID   string
    Field       string
    OldValue    string
    NewValue    string
}

type AppState struct {
    // ...
    ChangeHistory []ChangeRecord
}

func (s *AppState) RecordChange(fixtureID, field, oldVal, newVal string) {
    s.ChangeHistory = append(s.ChangeHistory, ChangeRecord{
        Timestamp: time.Now(),
        FixtureID: fixtureID,
        Field:     field,
        OldValue:  oldVal,
        NewValue:  newVal,
    })
}
```

### Questions to Consider

- How much history do you keep?
- Should history survive app restarts?
- Could you implement undo using this history?

---

## Feature Idea 4: Quick Actions / Presets

### The Problem

Testers often make the same changes repeatedly: "set all fixtures to half-time", "reset all scores to 0-0".

### The Feature

Add quick action presets:
- Press `p` to open preset menu
- Select from predefined actions
- Apply to single fixture or all fixtures
- Allow custom presets

### Relevant Chapters

- Chapter 9: Modal selection menus
- Chapter 8: Batch commands
- Chapter 4: Configuration for custom presets

### Hints

```go
type Preset struct {
    Name        string
    Description string
    Apply       func(*models.Fixture)
}

var defaultPresets = []Preset{
    {
        Name:        "Kick Off",
        Description: "Set to FIRST_HALF, 0:00, scores 0-0",
        Apply: func(f *models.Fixture) {
            f.Period = models.PeriodFirstHalf
            f.ClockTimeMin = 0
            f.ClockTimeSec = 0
            zero := 0
            f.HomeScore = &zero
            f.AwayScore = &zero
        },
    },
    {
        Name:        "Half Time",
        Description: "Set to HALF_TIME, 45:00",
        Apply: func(f *models.Fixture) {
            f.Period = models.PeriodHalfTime
            f.ClockTimeMin = 45
            f.ClockTimeSec = 0
        },
    },
}
```

### Questions to Consider

- How do you handle presets that conflict with validation?
- Could users create and save their own presets?
- Should presets be undoable?

---

## Feature Idea 5: Connection Status Indicator

### The Problem

Network issues happen. Users don't know if they're connected or if requests are failing silently.

### The Feature

Add a status indicator:
- Show connection status in the UI (connected/disconnected/error)
- Ping the API periodically
- Visual indicator (green/yellow/red)
- Show last successful sync time

### Relevant Chapters

- Chapter 8: Commands for periodic tasks
- Chapter 11: API client health checks
- Chapter 7: Styling status indicators

### Hints

```go
type connectionStatus int

const (
    statusConnected connectionStatus = iota
    statusDisconnected
    statusError
)

// Periodic health check command
func healthCheckCmd(client *aws.APIClient) tea.Cmd {
    return tea.Tick(30*time.Second, func(t time.Time) tea.Msg {
        ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
        defer cancel()
        
        _, err := client.GetGameWeeks(ctx)
        if err != nil {
            return healthCheckMsg{status: statusError, err: err}
        }
        return healthCheckMsg{status: statusConnected, checkedAt: t}
    })
}
```

### Questions to Consider

- How often should you check?
- What happens to pending changes when disconnected?
- Should you queue changes and retry?

---

## Feature Idea 6: Data Export

### The Problem

Sometimes you need fixture data outside the TUI—for reports, sharing, or backup.

### The Feature

Export data to files:
- Press `x` to open export menu
- Export current gameweek fixtures to JSON/CSV
- Choose destination path
- Show success/failure message

### Relevant Chapters

- Chapter 8: Commands for file operations
- Chapter 2: Struct serialization with tags
- Chapter 9: Modal for options

### Hints

```go
func exportToJSONCmd(fixtures []models.Fixture, path string) tea.Cmd {
    return func() tea.Msg {
        data, err := json.MarshalIndent(fixtures, "", "  ")
        if err != nil {
            return exportCompleteMsg{err: err}
        }
        
        err = os.WriteFile(path, data, 0644)
        return exportCompleteMsg{err: err, path: path}
    }
}

func exportToCSV(fixtures []models.Fixture) string {
    var buf strings.Builder
    buf.WriteString("FixtureID,HomeTeam,AwayTeam,Period,HomeScore,AwayScore\n")
    for _, f := range fixtures {
        buf.WriteString(fmt.Sprintf("%s,%s,%s,%s,%d,%d\n",
            f.FixtureID,
            f.Participants.Home.TeamName,
            f.Participants.Away.TeamName,
            f.Period,
            safeInt(f.HomeScore),
            safeInt(f.AwayScore),
        ))
    }
    return buf.String()
}
```

### Questions to Consider

- What formats are most useful?
- Should you include all fields or let users choose?
- How do you handle file path input in a TUI?

---

## How to Approach Your Feature

### Step 1: Define the Scope

Write down:
- What does the feature do?
- What's the minimum viable version?
- What can you add later?

### Step 2: Design the State

Ask:
- What new fields does Model need?
- Does AppState need changes?
- What messages will you create?

### Step 3: Implement Update Logic

Start with:
- Key handler to trigger the feature
- Message handlers for async results
- State transitions

### Step 4: Build the View

Add:
- New UI elements
- Conditional rendering
- Styling

### Step 5: Test

Write tests for:
- State changes
- Edge cases
- Error handling

### Step 6: Refine

Consider:
- Is the UX intuitive?
- Are there edge cases you missed?
- Can you simplify the code?

---

## Challenge: Combine Features

Once you've built one feature, try combining concepts:

- **Search + Export**: Export only filtered results
- **History + Presets**: Undo preset applications
- **Status + History**: Show sync status per change

---

## Share Your Work

Built something cool? Consider:
- Adding it to the main project
- Writing about what you learned
- Helping others who are learning

The best way to solidify knowledge is to teach it.

---

## Final Thoughts

You now have the tools to build TUI applications in Go. The patterns you've learned—Elm Architecture, commands, state management—apply beyond this project.

Keep building. Break things. Fix them. That's how you grow.

Good luck.

---

[← Previous: Testing TUI Applications](./13-testing.md) | [Back to Introduction](./00-introduction.md)
