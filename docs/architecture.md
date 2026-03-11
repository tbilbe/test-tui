# Application Architecture

## Overview

The Seven Test TUI is a terminal user interface application built with Go, using the Bubbletea framework for the TUI and AWS SDK for backend integration.

## Architecture Diagram

```
┌─────────────────────────────────────────────────────────────┐
│                         cmd/main.go                         │
│                    (Application Entry)                      │
└────────────────────────┬────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────────┐
│                    internal/ui/                             │
│              (Bubbletea TUI Components)                     │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐       │
│  │ GameWeek     │  │  Fixture     │  │    Edit      │       │
│  │ List View    │  │  List View   │  │    Modal     │       │
│  └──────────────┘  └──────────────┘  └──────────────┘       │
└────────────────────────┬────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────────┐
│                  internal/models/                           │
│              (Data Models & State)                          │
│  • GameWeek, Fixture, Team structs                          │
│  • AppState (application state management)                  │
│  • Validation logic                                         │
└────────────────────────┬────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────────┐
│                    internal/aws/                            │
│                 (AWS Integration)                           │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐       │
│  │   Auth       │  │     API      │  │  DynamoDB    │       │
│  │  (Cognito)   │  │   Client     │  │   Client     │       │
│  └──────────────┘  └──────────────┘  └──────────────┘       │
└────────────────────────┬────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────────┐
│                   pkg/config/                               │
│            (Configuration Management)                       │
│  • Environment variable loading                             │
│  • Configuration validation                                 │
└─────────────────────────────────────────────────────────────┘
```

## Module Breakdown

### 1. `cmd/main.go`
**Purpose**: Application entry point

**Responsibilities**:
- Initialize configuration
- Set up AWS clients (Auth, API, DynamoDB)
- Create and run the Bubbletea program
- Handle graceful shutdown

**Flow**:
```go
main() → Load Config → Initialize Clients → Run TUI
```

### 2. `internal/ui/`
**Purpose**: Terminal user interface components

**Components** (to be implemented):
- **GameWeek List View**: Display and navigate gameweeks
- **Fixture List View**: Display fixtures for selected gameweek
- **Edit Modal**: Form for editing fixture properties
- **Split Panel Layout**: Two-panel layout with keyboard navigation

**Key Concepts**:
- Uses Bubbletea's Model-View-Update (MVU) pattern
- Each component implements `tea.Model` interface:
  - `Init()`: Initialize component
  - `Update(msg)`: Handle messages (keyboard, data updates)
  - `View()`: Render component to string
- Lipgloss for styling (colors, borders, layout)

### 3. `internal/models/`
**Purpose**: Data structures and business logic

**Files**:
- `models.go`: Core data structures (GameWeek, Fixture, Team, Goal)
- `state.go`: Application state management

**Key Features**:
- Struct tags for JSON and DynamoDB marshaling
- Validation functions for fixture data
- State management methods (SetGameWeeks, SetError, etc.)

**Example**:
```go
type Fixture struct {
    FixtureID    string        `json:"fixtureId" dynamodbav:"fixtureId"`
    Period       FixturePeriod `json:"period" dynamodbav:"period"`
    ClockTimeMin int           `json:"clockTimeMin" dynamodbav:"clockTimeMin"`
    // ...
}

func (f *Fixture) Validate() error {
    // Validation logic
}
```

### 4. `internal/aws/`
**Purpose**: AWS service integration

**Files**:
- `auth.go`: Cognito authentication
- `api.go`: Backend REST API client
- `dynamodb.go`: DynamoDB operations

**Auth Client**:
```go
authClient.SignIn(ctx, username, password) → Returns tokens
authClient.GetIDToken() → Used for API calls
```

**API Client**:
```go
apiClient.GetGameWeeks(ctx) → Fetch gameweeks
apiClient.GetFixturesByGameWeek(ctx, gameWeekID) → Fetch fixtures
```

**DynamoDB Client**:
```go
dynamoClient.QueryFixturesByGameWeek(ctx, gameWeekID) → Read fixtures
dynamoClient.UpdateFixture(ctx, fixture) → Write fixture updates
```

### 5. `pkg/config/`
**Purpose**: Configuration management

**Responsibilities**:
- Load environment variables
- Validate required configuration
- Provide default values
- Construct derived values (e.g., table names)

**Usage**:
```go
cfg, err := config.Load()
// cfg.APIEndpoint, cfg.UserPoolID, etc.
```

## Data Flow

### Reading Data (Startup)
```
User starts TUI
    ↓
Load Config (env vars)
    ↓
Authenticate with Cognito (username/password)
    ↓
Get ID Token
    ↓
Fetch GameWeeks from API (with Bearer token)
    ↓
Display GameWeek List
    ↓
User selects GameWeek
    ↓
Fetch Fixtures from API
    ↓
Display Fixture List
```

### Updating Data (Edit Fixture)
```
User selects Fixture
    ↓
Open Edit Modal
    ↓
User modifies fields (period, time, scores)
    ↓
Validate changes
    ↓
Update DynamoDB directly (PutItem)
    ↓
Refresh Fixture List from API
    ↓
Display success message
```

## Bubbletea MVU Pattern

Bubbletea uses the Model-View-Update (MVU) architecture:

### Model
The application state (what data we have):
```go
type model struct {
    state      *models.AppState
    authClient *aws.AuthClient
    apiClient  *aws.APIClient
    dbClient   *aws.DynamoDBClient
    // UI state
    cursor     int
    activePanel string
}
```

### Update
How the state changes in response to messages:
```go
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.KeyMsg:
        switch msg.String() {
        case "up", "k":
            m.cursor--
        case "enter":
            // Select item, fetch data
            return m, fetchFixturesCmd
        }
    case fixturesLoadedMsg:
        m.state.SetFixtures(msg.fixtures)
    }
    return m, nil
}
```

### View
How to display the state:
```go
func (m model) View() string {
    // Render UI using lipgloss
    return lipgloss.JoinVertical(
        lipgloss.Left,
        titleStyle.Render("Seven Test TUI"),
        gameWeekListView(m),
        fixtureListView(m),
    )
}
```

### Commands (Async Operations)
Commands handle side effects (API calls, DB updates):
```go
func fetchGameWeeksCmd() tea.Msg {
    gameWeeks, err := apiClient.GetGameWeeks(ctx)
    return gameWeeksLoadedMsg{gameWeeks, err}
}
```

## Current Implementation Status

✅ **Completed**:
- Project structure
- Go module initialization
- Dependencies (Bubbletea, Lipgloss, AWS SDK)
- AWS Auth client (Cognito)
- AWS API client (REST endpoints)
- AWS DynamoDB client
- Data models with validation
- Application state management
- Configuration module
- Basic TUI skeleton

🚧 **In Progress**:
- UI components (GameWeek list, Fixture list, Edit modal)
- Split panel layout
- Keyboard navigation
- Error handling and user feedback

📋 **TODO**:
- Complete UI implementation
- Integration testing
- Documentation
- Polish and styling
