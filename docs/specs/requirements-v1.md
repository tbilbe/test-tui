Implementation Plan - Seven Gameweek Fixture Testing TUI

Problem Statement:
Testers currently manually update DynamoDB tables to test gameweek fixture scenarios, which is error-prone, requires specific knowledge of data
shapes, and lacks visibility into the current state. The existing bash script only handles bulk period updates. We need a comprehensive TUI tool
that provides full fixture management capabilities with a user-friendly interface.

Requirements:

Functional Requirements:
1. Authenticate via AWS Cognito/Amplify on startup
2. List all available gameweeks from backend
3. Select a gameweek to view its fixtures
4. Display fixture details: teams, period, clock time, scores, start date
5. Edit fixture properties: period, clockTimeMin, clockTimeSec, homeScore, awayScore, startDate
6. Submit changes directly to DynamoDB SE7-2707-GameWeekFixtures table
7. Manual refresh capability via keyboard shortcuts
8. Lazygit-style interface: keyboard-driven, clickable panels, modal dialogs

Non-Functional Requirements:
1. Built with Go, Bubbletea, and Lipgloss
2. Support AWS SSO/credential paste authentication (like existing bash script)
3. Region: eu-west-2
4. Side-by-side workflow with iOS/Android simulator

Out of Scope (Future Iterations):
- Goal scorer management
- Customer team selection management
- Real-time auto-refresh/polling
- DynamoDB Streams integration

Background:

Current Architecture:
- React Native mobile app using AWS Amplify for auth
- Backend API endpoints:
  - GET /game-weeks - List all gameweeks
  - GET /game-weeks/:gameWeekId/fixtures - Get fixtures for a gameweek
- DynamoDB tables:
  - SE7-2707-GameWeekFixtures - Stores fixture data (partition key: gameWeekId, sort key: fixtureId)
  - Customer selections table (name TBD - for future iterations)
- AWS Cognito user pool for authentication
- Existing bash script: scripts/batch-update-fixtures.sh (bulk period updates only)

Data Structures:

typescript
// GameWeek
type GameWeek = {
  gameWeekId: string;
  label: string;
  fixturesStartDate: string;
  fixturesEndDate: string;
  customerStartDate: string;
  customerEndDate: string;
  competitionCalendarId: string;
  locked?: boolean;
}

// Fixture
type Fixture = {
  fixtureId: string;
  gameWeekId: string;
  startDate: string; // ISO 8601
  period: "PRE_MATCH" | "FIRST_HALF" | "HALF_TIME" | "SECOND_HALF" | "FULL_TIME";
  clockTimeMin: number;
  clockTimeSec: number;
  homeScore?: number;
  awayScore?: number;
  homeTeamId: string;
  awayTeamId: string;
  participants: {
    home: Team;
    away: Team;
  };
  goals?: Goal[]; // For future use
}

// Team
type Team = {
  teamId: string;
  teamName: string;
  teamNameShort: string;
  teamCode: string;
}

// Goal (future use)
type Goal = {
  goalId: string;
  timeMin: number;
  timeSec: number;
  type: "GOAL" | "OWN_GOAL" | "PENALTY_GOAL";
  playerId: string;
  playerName: string;
  period: string;
  teamId: string;
  sevenGoalType: "FIRST" | "LAST" | "INVALID";
}


Proposed Solution:

Build a Go TUI application using Bubbletea framework with the following architecture:

Component Structure:
1. Auth Layer - AWS Cognito authentication, credential management
2. Data Layer - DynamoDB client, API client for backend endpoints
3. UI Layer - Bubbletea models for different views
4. State Management - Application state for gameweeks, fixtures, selections

UI Layout (Lazygit-inspired):
┌─────────────────────────────────────────────────────────────┐
│ Seven Fixture Manager - GameWeek 27                         │
├──────────────────┬──────────────────────────────────────────┤
│ GameWeeks        │ Fixtures                                 │
│                  │                                          │
│ > GW 27 (Active) │ > Arsenal vs Chelsea                     │
│   GW 26          │   Period: FIRST_HALF | Time: 23:45      │
│   GW 25          │   Score: 1-0 | Start: 2026-02-20 15:00  │
│   GW 24          │                                          │
│                  │   Liverpool vs Man City                  │
│                  │   Period: PRE_MATCH | Time: 00:00        │
│                  │   Score: 0-0 | Start: 2026-02-20 17:30  │
│                  │                                          │
├──────────────────┴──────────────────────────────────────────┤
│ [e]dit [r]efresh [q]uit                                     │
└─────────────────────────────────────────────────────────────┘


Keyboard Shortcuts:
- ↑/↓ or j/k - Navigate lists
- Enter - Select gameweek/fixture
- e - Edit selected fixture (opens modal)
- r - Refresh current view
- q - Quit application
- Esc - Close modal/go back
- Tab - Switch between panels

Task Breakdown:

Task 1: Project Setup & Dependencies
- Initialize Go module for the project
- Add dependencies: bubbletea, lipgloss, AWS SDK for Go v2
- Create project structure: cmd/, internal/, pkg/
- Set up basic main.go with bubbletea skeleton

Implementation Guidance:
- Use Go modules (go mod init)
- Dependencies: github.com/charmbracelet/bubbletea, github.com/charmbracelet/lipgloss, github.com/aws/aws-sdk-go-v2
- Follow standard Go project layout

Demo: Run go run cmd/main.go and see a basic TUI window appear

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━


Task 2: AWS Authentication Module
- Implement AWS credential loading (environment variables, SSO profiles)
- Create Cognito client for user authentication
- Support username/password authentication flow
- Store and manage auth tokens (access token, ID token, refresh token)

Implementation Guidance:
- Use AWS SDK's config.LoadDefaultConfig for credential chain
- Implement Cognito InitiateAuth API call for username/password
- Store tokens in memory (no persistence for MVP)
- Handle token expiration gracefully

Test Requirements:
- Verify credential loading from environment variables
- Test Cognito authentication with valid/invalid credentials
- Verify token storage and retrieval

Demo: Successfully authenticate and display user info in TUI

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━


Task 3: DynamoDB Client Module
- Create DynamoDB client wrapper
- Implement query operation for GameWeekFixtures table (by gameWeekId)
- Implement put operation for updating fixtures
- Add error handling and retry logic

Implementation Guidance:
- Use AWS SDK DynamoDB client
- Query with partition key (gameWeekId)
- Use PutItem for updates (overwrites entire item)
- Region: eu-west-2

Test Requirements:
- Mock DynamoDB calls and verify query parameters
- Test error handling for network failures
- Verify data marshaling/unmarshaling

Demo: Query fixtures for a gameweek and display raw JSON output

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━


Task 4: Backend API Client Module
- Create HTTP client for backend API
- Implement GET /game-weeks endpoint
- Implement GET /game-weeks/:gameWeekId/fixtures endpoint
- Add authentication header injection (Bearer token)

Implementation Guidance:
- Use standard Go net/http package
- Base URL from environment variable (API_URL)
- Add Authorization header with Cognito ID token
- Parse JSON responses into Go structs

Test Requirements:
- Mock HTTP responses and verify request construction
- Test authentication header injection
- Verify JSON unmarshaling

Demo: Fetch gameweeks from API and display list in terminal

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━


Task 5: Data Models & State Management
- Define Go structs for GameWeek, Fixture, Team, Goal
- Create application state struct
- Implement state update functions
- Add validation logic for fixture updates

Implementation Guidance:
- Use struct tags for JSON marshaling
- Validate period transitions (PRE_MATCH → FIRST_HALF → HALF_TIME → SECOND_HALF → FULL_TIME)
- Validate clock time ranges (0-45 for first half, 45-90+ for second half)
- Ensure scores are non-negative

Test Requirements:
- Test JSON marshaling/unmarshaling
- Verify validation rules for period, time, scores
- Test state update functions

Demo: Load fixture data, modify it, validate changes, and display updated state

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━


Task 6: GameWeek List View
- Create bubbletea model for gameweek list
- Implement list rendering with lipgloss styling
- Add keyboard navigation (up/down, j/k)
- Highlight selected gameweek
- Handle Enter key to select gameweek

Implementation Guidance:
- Use lipgloss for styling (borders, colors, padding)
- Maintain cursor position in model state
- Fetch gameweeks on view initialization
- Display: gameWeekId, label, date range, locked status

Test Requirements:
- Test keyboard navigation logic
- Verify cursor wrapping at list boundaries
- Test gameweek selection

Demo: Display list of gameweeks, navigate with keyboard, select one

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━


Task 7: Fixture List View
- Create bubbletea model for fixture list
- Display fixtures for selected gameweek
- Show: teams, period, clock time, scores, start date
- Add keyboard navigation
- Handle Enter key to open edit modal

Implementation Guidance:
- Fetch fixtures when gameweek is selected
- Format display: "Arsenal vs Chelsea | FIRST_HALF 23:45 | 1-0"
- Use color coding for periods (PRE_MATCH: gray, IN_PLAY: green, FULL_TIME: blue)
- Handle empty fixture list gracefully

Test Requirements:
- Test fixture list rendering
- Verify navigation and selection
- Test empty state display

Demo: Select a gameweek, view its fixtures with formatted display

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━


Task 8: Split Panel Layout
- Implement two-panel layout (gameweeks | fixtures)
- Add panel switching with Tab key
- Highlight active panel
- Maintain independent scroll positions

Implementation Guidance:
- Use lipgloss JoinHorizontal for side-by-side layout
- Track active panel in state
- Apply border styling to active panel
- Calculate panel widths based on terminal size

Test Requirements:
- Test panel switching logic
- Verify scroll position preservation
- Test responsive layout on different terminal sizes

Demo: Navigate between gameweek and fixture panels, see active panel highlighted

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━


Task 9: Fixture Edit Modal
- Create modal dialog for editing fixture
- Display form fields: period, clockTimeMin, clockTimeSec, homeScore, awayScore, startDate
- Implement field navigation (Tab, Shift+Tab)
- Add input validation
- Handle save (Enter) and cancel (Esc)

Implementation Guidance:
- Use lipgloss for modal overlay (centered, bordered)
- Implement form field state (focused field, values)
- Validate inputs on field blur
- Show validation errors inline
- Period: dropdown/cycle through values
- Time: numeric input
- Scores: numeric input
- StartDate: ISO 8601 string input (or date picker)

Test Requirements:
- Test field navigation
- Verify validation rules
- Test save/cancel actions
- Verify modal overlay rendering

Demo: Open edit modal, modify fixture values, save changes (in-memory only)

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━


Task 10: DynamoDB Update Integration
- Wire edit modal save to DynamoDB put operation
- Show loading state during update
- Display success/error messages
- Refresh fixture list after successful update

Implementation Guidance:
- Call DynamoDB PutItem with updated fixture
- Use bubbletea Cmd for async operation
- Show spinner during save
- Display toast notification for success/error
- Refresh fixture data from DynamoDB after save

Test Requirements:
- Test DynamoDB update call
- Verify error handling
- Test UI state during async operation

Demo: Edit a fixture, save changes, see them reflected in DynamoDB and refreshed in UI

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━


Task 11: Refresh & Keyboard Shortcuts
- Implement refresh command (r key)
- Add quit command (q key)
- Add help overlay (? key)
- Display keyboard shortcuts in footer

Implementation Guidance:
- Refresh: re-fetch current view data (gameweeks or fixtures)
- Show loading indicator during refresh
- Help overlay: modal with keyboard shortcut reference
- Footer: always visible, shows available actions

Test Requirements:
- Test refresh functionality
- Verify help overlay display
- Test quit command

Demo: Press 'r' to refresh data, '?' to see help, 'q' to quit

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━


Task 12: Error Handling & User Feedback
- Add error display for API/DynamoDB failures
- Implement toast notifications for actions
- Add loading states for async operations
- Handle network timeouts gracefully

Implementation Guidance:
- Display errors in a dedicated panel or modal
- Toast notifications: auto-dismiss after 3 seconds
- Loading states: spinner with message
- Retry logic for transient failures

Test Requirements:
- Test error display for various failure scenarios
- Verify toast notification behavior
- Test loading state rendering

Demo: Trigger various errors (network failure, invalid data), see appropriate error messages

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━


Task 13: Configuration & Environment Setup
- Create config file structure (.env support)
- Add configuration for: API_URL, AWS region, DynamoDB table name, Cognito pool IDs
- Implement config validation on startup
- Add command-line flags for overrides

Implementation Guidance:
- Use godotenv for .env file loading
- Support environment variables
- Command-line flags override env vars
- Validate required config on startup

Test Requirements:
- Test config loading from various sources
- Verify validation logic
- Test flag overrides

Demo: Run with different configurations, see appropriate behavior

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━


Task 14: Polish & Documentation
- Add README with setup instructions
- Document keyboard shortcuts
- Add inline help text in UI
- Improve styling and visual consistency
- Add version display

Implementation Guidance:
- README: prerequisites, installation, configuration, usage
- Consistent color scheme throughout UI
- Smooth animations for transitions
- Version flag: --version

Demo: Complete, polished TUI ready for testing use

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━


Future Enhancements (Post-MVP):
- Goal scorer management
- Customer team selection (7-player team assignment)
- Real-time updates via DynamoDB Streams
- Fixture presets (quick actions like "Set to Kickoff")
- Bulk operations (update multiple fixtures)
- Search/filter fixtures
- Export fixture state to JSON
- Undo/redo functionality

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━