package ui

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sort"
	"strconv"
	"time"

	"github.com/angstromsports/seven-test-tui/internal/aws"
	"github.com/angstromsports/seven-test-tui/internal/models"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var logger *log.Logger

func init() {
	// Create log file
	logFile, err := os.OpenFile("seven-test-tui.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatal(err)
	}
	logger = log.New(logFile, "", log.LstdFlags|log.Lshortfile)
	logger.Println("=== Seven Test TUI Started ===")
}

var (
	boxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("63")).
			Padding(1, 2)

	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("205")).
			MarginBottom(1)

	highlightStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("170")).
			Bold(true)

	normalStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240"))

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("245"))

	inputStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("86"))
)

type screenType int

const (
	authScreenType screenType = iota
	prefixScreenType
	gameweekScreenType
	fixtureScreenType
)

type focusPane int

const (
	focusTeam focusPane = iota
	focusGameWeek
	focusFixtures
)

type Model struct {
	state              *models.AppState
	authClient         *aws.AuthClient
	apiClient          *aws.APIClient
	dynamoClient       *aws.DynamoDBClient
	currentScreen      screenType
	authScreen         AuthScreen
	input              string
	inputMode          bool
	err                error
	width              int
	height             int
	focusedPane        focusPane
	prefix             string
	fixtureSelectMode  bool
	selectedFixtureIdx int
	editingFixture     bool
	editModalField     int // 0=period, 1=homeScore, 2=awayScore, 3=clockMin, 4=clockSec
	fixturesTable      table.Model
	showFieldSelect    bool
	fieldOptions       []string
	fieldOptionIdx     int
	showBatchModal     bool
	batchPresetIdx     int
	fixturePlayers     map[string]FixturePlayers // fixtureId -> players
	showGoalModal      bool
	goalModalIdx       int
	goalModalField     int // 0=player, 1=timeMin
	pendingGoals       []models.Goal
}

type FixturePlayers struct {
	HomePlayers []models.Player
	AwayPlayers []models.Player
}

func NewModel(authClient *aws.AuthClient, apiClient *aws.APIClient, dynamoClient *aws.DynamoDBClient) Model {
	return Model{
		state:         models.NewAppState(),
		authClient:    authClient,
		apiClient:     apiClient,
		dynamoClient:  dynamoClient,
		currentScreen: authScreenType,
		authScreen:    NewAuthScreen(),
		inputMode:     true,
	}
}

func (m Model) Init() tea.Cmd {
	return m.authScreen.Init()
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}

	case authSubmitMsg:
		// Authenticate
		return m, authenticateCmd(m.authClient, msg.username, msg.password)

	case authSuccessMsg:
		// Move to prefix screen
		m.apiClient.SetIDToken(msg.token)
		m.currentScreen = prefixScreenType
		m.inputMode = true
		return m, nil

	case authErrorMsg:
		m.err = msg.err
		m.authScreen = NewAuthScreen()
		return m, m.authScreen.Init()

	case gameWeeksLoadedMsg:
		if msg.err != nil {
			m.err = msg.err
		} else {
			m.state.SetGameWeeks(msg.gameWeeks)
			sort.Slice(m.state.GameWeeks, func(i, j int) bool {
				id1, _ := strconv.Atoi(m.state.GameWeeks[i].GameWeekID)
				id2, _ := strconv.Atoi(m.state.GameWeeks[j].GameWeekID)
				return id1 < id2
			})
		}
		return m, nil

	case fixturesLoadedMsg:
		if msg.err != nil {
			m.err = msg.err
		} else {
			m.state.SetFixtures(msg.fixtures)
			m.buildFixturesTable()
			// Fetch players for all fixtures
			if m.state.CurrentGameWeek != nil {
				return m, fetchPlayersCmd(m.apiClient, m.state.CurrentGameWeek.GameWeekID, msg.fixtures)
			}
		}
		return m, nil

	case playersLoadedMsg:
		if msg.err != nil {
			m.err = msg.err
		} else {
			// Build fixture players map
			m.fixturePlayers = make(map[string]FixturePlayers)
			for _, fixture := range m.state.Fixtures {
				var homePlayers, awayPlayers []models.Player
				for _, player := range msg.players {
					if player.FixtureID == fixture.FixtureID {
						if player.TeamID == fixture.Participants.Home.TeamID {
							homePlayers = append(homePlayers, player)
						} else if player.TeamID == fixture.Participants.Away.TeamID {
							awayPlayers = append(awayPlayers, player)
						}
					}
				}
				m.fixturePlayers[fixture.FixtureID] = FixturePlayers{
					HomePlayers: homePlayers,
					AwayPlayers: awayPlayers,
				}
			}
		}
		return m, nil

	case selectionLoadedMsg:
		if msg.err != nil {
			m.err = msg.err
		} else {
			m.state.SetSelection(msg.selection)
		}
		return m, nil

	case teamCreatedMsg:
		if msg.err != nil {
			m.err = msg.err
		} else {
			m.state.SetSelection(msg.selection)
		}
		return m, nil

	case gameWeekUpdatedMsg:
		if msg.err != nil {
			m.err = msg.err
		} else {
			// Update the gameweek in state
			m.state.SetCurrentGameWeek(msg.gameWeek)
			// Refresh gameweeks list
			return m, fetchGameWeeksCmd(m.apiClient)
		}
		return m, nil

	case fixtureUpdatedMsg:
		if msg.err != nil {
			m.err = msg.err
		} else {
			// Refresh fixtures
			if m.state.CurrentGameWeek != nil {
				return m, fetchFixturesCmd(m.apiClient, m.state.CurrentGameWeek.GameWeekID)
			}
		}
		return m, nil

	case batchUpdatedMsg:
		if msg.err != nil {
			m.err = msg.err
		} else {
			// Refresh fixtures
			if m.state.CurrentGameWeek != nil {
				return m, fetchFixturesCmd(m.apiClient, m.state.CurrentGameWeek.GameWeekID)
			}
		}
		return m, nil

	case githubActionMsg:
		if msg.err != nil {
			m.err = msg.err
		}
		// Success message handled by err being nil
		return m, nil
	}

	// Route to appropriate screen
	switch m.currentScreen {
	case authScreenType:
		var cmd tea.Cmd
		m.authScreen, cmd = m.authScreen.Update(msg)
		return m, cmd

	case prefixScreenType:
		return m.updatePrefixScreen(msg)

	case gameweekScreenType:
		return m.updateGameweekScreen(msg)

	case fixtureScreenType:
		return m.updateFixtureScreen(msg)
	}

	return m, nil
}

func (m Model) updateFixtureScreen(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q":
			return m, tea.Quit
		case "g":
			// Go back to gameweek selection
			m.currentScreen = gameweekScreenType
			m.inputMode = true
			m.input = ""
			m.err = nil // Clear error
			m.state.SetFixtures([]models.Fixture{})
			m.state.SetSelection(nil)
			return m, nil
		case "m":
			// Make this gameweek current (update dates in DynamoDB)
			if m.state.CurrentGameWeek != nil {
				return m, makeGameWeekCurrentCmd(m.state.CurrentGameWeek, m.prefix, m.dynamoClient)
			}
			return m, nil
		// case "c":
		// 	// Create default team (only for current gameweek) - DISABLED
		// 	if m.state.CurrentGameWeek != nil {
		// 		// Check if this is the actual current gameweek
		// 		now := time.Now()
		// 		var currentGW *models.GameWeek
		// 		for i := range m.state.GameWeeks {
		// 			gw := &m.state.GameWeeks[i]
		// 			startDate, _ := time.Parse(time.RFC3339, gw.CustomerStartDate)
		// 			endDate, _ := time.Parse(time.RFC3339, gw.CustomerEndDate)
		// 			if now.After(startDate) && now.Before(endDate) {
		// 				currentGW = gw
		// 				break
		// 			}
		// 		}
		//
		// 		if currentGW == nil || currentGW.GameWeekID != m.state.CurrentGameWeek.GameWeekID {
		// 			m.err = fmt.Errorf("can only create team for current gameweek")
		// 			return m, nil
		// 		}
		// 	}
		// 	return m, createDefaultTeamCmd(m.apiClient)
		case "b":
			// Open batch update modal
			if m.state.CurrentGameWeek != nil && len(m.state.Fixtures) > 0 {
				m.showBatchModal = true
				m.batchPresetIdx = 0
			}
			return m, nil
		case "r":
			// Trigger GitHub Actions workflow to reset environment
			if m.prefix != "" && m.prefix != "dev" {
				return m, triggerGitHubActionCmd(m.prefix)
			}
			return m, nil
		case "tab":
			// In goal modal, switch between player and time fields
			if m.showGoalModal && !m.showFieldSelect {
				if m.goalModalField == 0 {
					m.goalModalField = 1
				} else {
					m.goalModalField = 0
				}
				return m, nil
			}
			// Cycle between gameweek and fixtures panes
			if m.focusedPane == focusGameWeek {
				m.focusedPane = focusFixtures
			} else {
				m.focusedPane = focusGameWeek
			}
			return m, nil
		case "enter":
			// Goal modal - select player or time
			if m.showGoalModal && !m.showFieldSelect {
				m.showFieldSelect = true
				m.fieldOptionIdx = 0

				if m.goalModalField == 0 {
					// Player selection - build list from home/away players
					fixture := m.state.Fixtures[m.selectedFixtureIdx]
					players := m.fixturePlayers[fixture.FixtureID]

					// Determine which team based on goal index
					homeGoals := 0
					if fixture.HomeScore != nil {
						homeGoals = *fixture.HomeScore
					}

					m.fieldOptions = []string{}
					if m.goalModalIdx < homeGoals {
						// Home goal
						for i, p := range players.HomePlayers {
							if i >= 11 {
								break
							}
							m.fieldOptions = append(m.fieldOptions, p.PlayerName)
						}
					} else {
						// Away goal
						for i, p := range players.AwayPlayers {
							if i >= 11 {
								break
							}
							m.fieldOptions = append(m.fieldOptions, p.PlayerName)
						}
					}
				} else {
					// Time selection - 5 minute increments
					m.fieldOptions = []string{}
					for i := 0; i <= 90; i += 5 {
						m.fieldOptions = append(m.fieldOptions, fmt.Sprintf("%d", i))
					}
				}
				return m, nil
			}
			// Apply selection in goal modal
			if m.showGoalModal && m.showFieldSelect {
				m.applyGoalFieldSelection()
				m.showFieldSelect = false
				return m, nil
			}
			// Batch modal submit
			if m.showBatchModal {
				m.showBatchModal = false
				presets := []string{"kickoff", "halftime", "secondhalf", "fulltime"}
				return m, batchUpdateFixturesCmd(m.state.Fixtures, presets[m.batchPresetIdx], m.prefix)
			}
			// Field select in edit modal
			if m.editingFixture && !m.showFieldSelect {
				m.showFieldSelect = true
				m.fieldOptionIdx = 0
				// Build options based on field
				switch m.editModalField {
				case 0: // Period
					m.fieldOptions = []string{"PRE_MATCH", "FIRST_HALF", "HALF_TIME", "SECOND_HALF", "FULL_TIME"}
				case 3: // Clock Min
					m.fieldOptions = []string{}
					for i := 0; i <= 90; i += 10 {
						m.fieldOptions = append(m.fieldOptions, fmt.Sprintf("%d", i))
					}
				case 4: // Clock Sec
					m.fieldOptions = []string{}
					for i := 0; i <= 50; i += 10 {
						m.fieldOptions = append(m.fieldOptions, fmt.Sprintf("%d", i))
					}
				}
				return m, nil
			}
			// Select option from dropdown
			if m.editingFixture && m.showFieldSelect {
				m.applyFieldSelection()
				m.showFieldSelect = false
				return m, nil
			}
			// Enter fixture selection mode when fixtures pane is focused
			if m.focusedPane == focusFixtures && !m.fixtureSelectMode && len(m.state.Fixtures) > 0 {
				m.fixtureSelectMode = true
				m.fixturesTable.Focus()
				m.selectedFixtureIdx = m.fixturesTable.Cursor()
				return m, nil
			} else if m.fixtureSelectMode && !m.editingFixture {
				// Open edit modal for selected fixture
				m.selectedFixtureIdx = m.fixturesTable.Cursor()
				m.editingFixture = true
				m.editModalField = 0
				return m, nil
			}
			return m, nil
		case "s":
			// Save goals from goal modal
			if m.showGoalModal {
				fixture := &m.state.Fixtures[m.selectedFixtureIdx]
				fixture.Goals = m.pendingGoals
				m.showGoalModal = false
				m.fixtureSelectMode = false
				m.pendingGoals = nil
				return m, updateFixtureCmd(fixture, m.prefix)
			}
			// Save fixture changes
			if m.editingFixture && m.selectedFixtureIdx < len(m.state.Fixtures) {
				fixture := m.state.Fixtures[m.selectedFixtureIdx]

				// Check if goals need to be assigned
				homeGoals := 0
				awayGoals := 0
				if fixture.HomeScore != nil {
					homeGoals = *fixture.HomeScore
				}
				if fixture.AwayScore != nil {
					awayGoals = *fixture.AwayScore
				}

				totalGoals := homeGoals + awayGoals
				currentGoals := len(fixture.Goals)

				// If goals changed, open goal assignment modal
				if totalGoals != currentGoals {
					logger.Printf("Goals changed: %d current, %d needed", currentGoals, totalGoals)
					m.editingFixture = false
					m.showGoalModal = true
					m.goalModalIdx = 0
					m.goalModalField = 0

					// Build pending goals array
					m.pendingGoals = make([]models.Goal, totalGoals)
					// Copy existing goals
					for i := 0; i < len(fixture.Goals) && i < totalGoals; i++ {
						m.pendingGoals[i] = fixture.Goals[i]
					}
					// Initialize new goals
					for i := len(fixture.Goals); i < totalGoals; i++ {
						m.pendingGoals[i] = models.Goal{
							GoalID:  fmt.Sprintf("%d", time.Now().UnixNano()+int64(i)),
							Type:    "GOAL",
							TimeMin: 0,
							TimeSec: 0,
						}
					}
					return m, nil
				}

				// No goal changes, save directly
				m.editingFixture = false
				m.fixtureSelectMode = false
				return m, updateFixtureCmd(&fixture, m.prefix)
			}
			return m, nil
		case "esc":
			// Exit selection mode or close modal
			if m.showGoalModal {
				m.showGoalModal = false
				m.pendingGoals = nil
				return m, nil
			} else if m.showFieldSelect {
				m.showFieldSelect = false
				return m, nil
			} else if m.showBatchModal {
				m.showBatchModal = false
				return m, nil
			} else if m.editingFixture {
				m.editingFixture = false
				return m, nil
			} else if m.fixtureSelectMode {
				m.fixtureSelectMode = false
				m.fixturesTable.Blur()
				return m, nil
			}
			return m, nil
		case "up", "k":
			if m.showGoalModal && !m.showFieldSelect {
				// Navigate between goals
				if m.goalModalIdx > 0 {
					m.goalModalIdx--
				}
			} else if m.showFieldSelect && m.fieldOptionIdx > 0 {
				m.fieldOptionIdx--
			} else if m.showBatchModal && m.batchPresetIdx > 0 {
				m.batchPresetIdx--
			} else if m.editingFixture {
				// Navigate modal fields
				if m.editModalField > 0 {
					m.editModalField--
				}
			} else if m.fixtureSelectMode {
				m.fixturesTable, _ = m.fixturesTable.Update(msg)
				m.selectedFixtureIdx = m.fixturesTable.Cursor()
			}
			return m, nil
		case "down", "j":
			if m.showGoalModal && !m.showFieldSelect {
				// Navigate between goals
				if m.goalModalIdx < len(m.pendingGoals)-1 {
					m.goalModalIdx++
				}
			} else if m.showFieldSelect && m.fieldOptionIdx < len(m.fieldOptions)-1 {
				m.fieldOptionIdx++
			} else if m.showBatchModal && m.batchPresetIdx < 3 {
				m.batchPresetIdx++
			} else if m.editingFixture {
				// Navigate modal fields
				maxField := 2
				if m.selectedFixtureIdx < len(m.state.Fixtures) {
					f := &m.state.Fixtures[m.selectedFixtureIdx]
					if f.Period == models.PeriodFirstHalf || f.Period == models.PeriodSecondHalf {
						maxField = 4
					}
				}
				if m.editModalField < maxField {
					m.editModalField++
				}
			} else if m.fixtureSelectMode {
				m.fixturesTable, _ = m.fixturesTable.Update(msg)
				m.selectedFixtureIdx = m.fixturesTable.Cursor()
			}
			return m, nil
		case "+", "=":
			// Increment field value
			if m.editingFixture && m.selectedFixtureIdx < len(m.state.Fixtures) {
				m.incrementFixtureField(1)
			}
			return m, nil
		case "-", "_":
			// Decrement field value
			if m.editingFixture && m.selectedFixtureIdx < len(m.state.Fixtures) {
				m.incrementFixtureField(-1)
			}
			return m, nil
		}
	case tea.MouseMsg:
		// Handle mouse clicks for pane focus (only row 1 has focusable panes)
		if msg.Type == tea.MouseLeft && msg.Y < m.height/2 {
			if msg.X < m.width/4 {
				m.focusedPane = focusGameWeek
			} else {
				m.focusedPane = focusFixtures
			}
		}
	}
	return m, nil
}

func (m Model) updatePrefixScreen(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q":
			return m, tea.Quit
		case "enter":
			// Set prefix and update API URLs
			if m.input == "" {
				m.prefix = "dev"
				m.apiClient = aws.NewAPIClient("https://dev.api.playtheseven.com")
			} else {
				m.prefix = m.input
				m.apiClient = aws.NewAPIClient(fmt.Sprintf("https://%s.dev.api.playtheseven.com", m.input))
			}
			m.apiClient.SetIDToken(m.authClient.GetIDToken())

			// Recreate DynamoDB client with correct table name
			ctx := context.Background()
			tableName := m.prefix + "-GameWeek"
			if m.prefix == "dev" || m.prefix == "" {
				tableName = "int-dev-GameWeek"
			}
			dynamoClient, err := aws.NewDynamoDBClient(ctx, "eu-west-2", tableName)
			if err != nil {
				m.err = fmt.Errorf("failed to create dynamo client: %w", err)
				return m, nil
			}
			m.dynamoClient = dynamoClient

			m.input = ""
			m.currentScreen = gameweekScreenType
			return m, fetchGameWeeksCmd(m.apiClient)
		case "backspace":
			if len(m.input) > 0 {
				m.input = m.input[:len(m.input)-1]
			}
		default:
			// Allow alphanumeric and dash
			if len(msg.String()) == 1 {
				char := msg.String()[0]
				if (char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z') ||
					(char >= '0' && char <= '9') || char == '-' {
					m.input += msg.String()
				}
			}
		}
	}
	return m, nil
}

func (m *Model) incrementFixtureField(delta int) {
	if m.selectedFixtureIdx >= len(m.state.Fixtures) {
		return
	}

	f := &m.state.Fixtures[m.selectedFixtureIdx]

	switch m.editModalField {
	case 0: // Period
		periods := []models.FixturePeriod{
			models.PeriodPreMatch,
			models.PeriodFirstHalf,
			models.PeriodHalfTime,
			models.PeriodSecondHalf,
			models.PeriodFullTime,
		}
		currentIdx := 0
		for i, p := range periods {
			if p == f.Period {
				currentIdx = i
				break
			}
		}
		newIdx := currentIdx + delta
		if newIdx >= 0 && newIdx < len(periods) {
			f.Period = periods[newIdx]
		}
	case 1: // Home Score
		if f.HomeScore == nil {
			score := 0
			f.HomeScore = &score
		}
		newScore := *f.HomeScore + delta
		if newScore >= 0 {
			f.HomeScore = &newScore
		}
	case 2: // Away Score
		if f.AwayScore == nil {
			score := 0
			f.AwayScore = &score
		}
		newScore := *f.AwayScore + delta
		if newScore >= 0 {
			f.AwayScore = &newScore
		}
	case 3: // Clock Min
		newMin := f.ClockTimeMin + delta
		if newMin >= 0 && newMin <= 90 {
			f.ClockTimeMin = newMin
		}
	case 4: // Clock Sec
		newSec := f.ClockTimeSec + delta
		if newSec >= 0 && newSec <= 59 {
			f.ClockTimeSec = newSec
		}
	}
}

func (m *Model) applyFieldSelection() {
	if m.selectedFixtureIdx >= len(m.state.Fixtures) || len(m.fieldOptions) == 0 {
		return
	}

	f := &m.state.Fixtures[m.selectedFixtureIdx]
	selected := m.fieldOptions[m.fieldOptionIdx]

	switch m.editModalField {
	case 0: // Period
		f.Period = models.FixturePeriod(selected)
	case 3: // Clock Min
		val, _ := strconv.Atoi(selected)
		f.ClockTimeMin = val
	case 4: // Clock Sec
		val, _ := strconv.Atoi(selected)
		f.ClockTimeSec = val
	}
}

func (m *Model) applyGoalFieldSelection() {
	if m.goalModalIdx >= len(m.pendingGoals) || len(m.fieldOptions) == 0 {
		return
	}

	goal := &m.pendingGoals[m.goalModalIdx]
	selected := m.fieldOptions[m.fieldOptionIdx]
	fixture := m.state.Fixtures[m.selectedFixtureIdx]
	players := m.fixturePlayers[fixture.FixtureID]

	if m.goalModalField == 0 {
		// Player selection
		homeGoals := 0
		if fixture.HomeScore != nil {
			homeGoals = *fixture.HomeScore
		}

		var selectedPlayer *models.Player
		if m.goalModalIdx < homeGoals {
			// Home player
			for i := range players.HomePlayers {
				if i >= 11 {
					break
				}
				if players.HomePlayers[i].PlayerName == selected {
					selectedPlayer = &players.HomePlayers[i]
					break
				}
			}
		} else {
			// Away player
			for i := range players.AwayPlayers {
				if i >= 11 {
					break
				}
				if players.AwayPlayers[i].PlayerName == selected {
					selectedPlayer = &players.AwayPlayers[i]
					break
				}
			}
		}

		if selectedPlayer != nil {
			goal.PlayerID = selectedPlayer.PlayerID
			goal.PlayerName = selectedPlayer.PlayerName
			goal.TeamID = selectedPlayer.TeamID
		}
	} else {
		// Time selection
		timeMin, _ := strconv.Atoi(selected)
		goal.TimeMin = timeMin

		// Auto-assign period based on time
		if timeMin <= 45 {
			goal.Period = models.PeriodFirstHalf
		} else {
			goal.Period = models.PeriodSecondHalf
		}
	}
}

func (m *Model) buildFixturesTable() {
	columns := []table.Column{
		{Title: "Home", Width: 15},
		{Title: "Score", Width: 7},
		{Title: "Away", Width: 15},
		{Title: "Period", Width: 15},
		{Title: "Time", Width: 8},
	}

	rows := []table.Row{}
	for _, f := range m.state.Fixtures {
		// Score display
		score := "---"
		if f.Period != models.PeriodPreMatch {
			// Show 0-0 if no scores set
			homeScore := 0
			awayScore := 0
			if f.HomeScore != nil {
				homeScore = *f.HomeScore
			}
			if f.AwayScore != nil {
				awayScore = *f.AwayScore
			}
			score = fmt.Sprintf("%d-%d", homeScore, awayScore)
		}

		// Time display
		clockTime := ""
		if f.Period == models.PeriodPreMatch {
			// Show kickoff time for pre-match
			if startTime, err := time.Parse(time.RFC3339, f.StartDate); err == nil {
				clockTime = startTime.Format("15:04")
			}
		} else if f.Period == models.PeriodFirstHalf || f.Period == models.PeriodSecondHalf {
			clockTime = fmt.Sprintf("%d:%02d", f.ClockTimeMin, f.ClockTimeSec)
		}

		rows = append(rows, table.Row{
			f.Participants.Home.TeamNameShort,
			score,
			f.Participants.Away.TeamNameShort,
			string(f.Period),
			clockTime,
		})
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithRows(rows),
		table.WithFocused(m.fixtureSelectMode),
		table.WithHeight(15),
	)

	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		BorderBottom(true).
		Bold(false)
	s.Selected = s.Selected.
		Foreground(lipgloss.Color("229")).
		Background(lipgloss.Color("205")).
		Bold(true)

	t.SetStyles(s)
	m.fixturesTable = t
}

func (m Model) updateGameweekScreen(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.inputMode {
			switch msg.String() {
			case "q":
				return m, tea.Quit
			case "enter":
				if m.input != "" {
					for i, gw := range m.state.GameWeeks {
						if gw.GameWeekID == m.input {
							m.state.SetCurrentGameWeek(&m.state.GameWeeks[i])
							m.currentScreen = fixtureScreenType
							m.inputMode = false
							// Fetch both fixtures and selection
							return m, tea.Batch(
								fetchFixturesCmd(m.apiClient, m.input),
								fetchSelectionCmd(m.apiClient),
							)
						}
					}
					m.err = fmt.Errorf("gameweek %s not found", m.input)
				}
			case "backspace":
				if len(m.input) > 0 {
					m.input = m.input[:len(m.input)-1]
				}
			default:
				if len(msg.String()) == 1 && msg.String()[0] >= '0' && msg.String()[0] <= '9' {
					m.input += msg.String()
				}
			}
		}
	}
	return m, nil
}

func (m Model) View() string {
	switch m.currentScreen {
	case authScreenType:
		view := m.authScreen.View()
		if m.err != nil {
			errMsg := lipgloss.NewStyle().
				Foreground(lipgloss.Color("196")).
				Render(fmt.Sprintf("\n\nError: %v", m.err))
			view += errMsg
		}
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, view)

	case prefixScreenType:
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, m.viewPrefixScreen())

	case gameweekScreenType:
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, m.viewGameweekScreen())

	case fixtureScreenType:
		view := m.viewFixtureScreen()
		// Overlay batch modal
		if m.showBatchModal {
			modal := m.viewBatchModal()
			view = lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, modal)
		}
		// Overlay goal modal
		if m.showGoalModal {
			modal := m.viewGoalModal()
			view = lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, modal)
		}
		// Overlay edit modal if editing
		if m.editingFixture && m.selectedFixtureIdx < len(m.state.Fixtures) {
			modal := m.viewEditModal()
			view = lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, modal)
		}
		return view
	}

	return ""
}

func (m Model) viewPrefixScreen() string {
	content := titleStyle.Render("Environment Setup") + "\n\n"
	content += "Enter branch prefix (e.g., SE7-2701)\n"
	content += "Leave blank for dev environment\n\n"
	content += highlightStyle.Render("⚠ Case sensitive!") + " Match AWS table names exactly\n\n"

	content += "Prefix: " + inputStyle.Render(m.input) + "\n\n"

	if m.input == "" {
		content += normalStyle.Render("Will use: https://dev.api.playtheseven.com\n")
		content += normalStyle.Render("Tables: int-dev-GameWeek, int-dev-GameWeekFixtures\n\n")
	} else {
		content += normalStyle.Render(fmt.Sprintf("Will use: https://%s.dev.api.playtheseven.com\n", m.input))
		content += normalStyle.Render(fmt.Sprintf("Tables: %s-GameWeek, %s-GameWeekFixtures\n\n", m.input, m.input))
	}

	content += normalStyle.Render("enter: continue • q: quit")

	return boxStyle.Width(70).Render(content)
}

func (m Model) viewGameweekScreen() string {
	if len(m.state.GameWeeks) == 0 {
		return boxStyle.Width(50).Render(
			titleStyle.Render("Loading GameWeeks...") + "\n\n" +
				"Please wait...",
		)
	}

	content := titleStyle.Render("Select GameWeek") + "\n\n"

	now := time.Now()
	content += fmt.Sprintf("Today: %s\n\n", now.Format("Monday, 2 Jan 2006"))

	var nextGW *models.GameWeek
	for i := range m.state.GameWeeks {
		gw := &m.state.GameWeeks[i]
		startDate, err := time.Parse(time.RFC3339, gw.CustomerStartDate)
		if err == nil && startDate.After(now) {
			nextGW = gw
			break
		}
	}

	if nextGW != nil {
		startDate, _ := time.Parse(time.RFC3339, nextGW.CustomerStartDate)
		content += highlightStyle.Render(fmt.Sprintf("Next: GameWeek %s (%s)\n", nextGW.GameWeekID, startDate.Format("Mon 2 Jan")))
		content += normalStyle.Render(fmt.Sprintf("Starts: %s\n", nextGW.CustomerStartDate)) + "\n"
	}

	content += "\nEnter GameWeek ID: " + inputStyle.Render(m.input+"_") + "\n\n"
	content += normalStyle.Render("Type number • enter: select • q: quit")

	return boxStyle.Width(60).Render(content)
}

func (m Model) viewFixtureScreen() string {
	if m.state.CurrentGameWeek == nil {
		return "No gameweek selected"
	}

	// Row 1: GameWeek (40%) | Fixtures (60%)
	gwPanel := m.viewGameweekPanel()
	fixturesPanel := m.viewFixturesPanel()
	row1 := lipgloss.JoinHorizontal(lipgloss.Top, gwPanel, fixturesPanel)

	// Row 2: Players Panel (full width)
	playersPanel := m.viewPlayersPanel()

	return lipgloss.JoinVertical(lipgloss.Left, row1, playersPanel)
}

func (m Model) viewPlayersPanel() string {
	borderColor := lipgloss.Color("63")

	style := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(borderColor).
		Padding(1, 2).
		Width(m.width - 4).
		Height((m.height / 2) - 4)

	// Get players for selected fixture
	if m.selectedFixtureIdx >= len(m.state.Fixtures) {
		return style.Render("No fixture selected")
	}

	fixture := m.state.Fixtures[m.selectedFixtureIdx]
	players, ok := m.fixturePlayers[fixture.FixtureID]
	if !ok {
		return style.Render("Loading players...")
	}

	// Build home and away columns
	homeContent := titleStyle.Render(fixture.Participants.Home.TeamNameShort) + "\n\n"
	for i, p := range players.HomePlayers {
		if i >= 11 {
			break
		}
		homeContent += fmt.Sprintf("#%-2d %-20s\n", p.ShirtNumber, p.PlayerName)
	}

	awayContent := titleStyle.Render(fixture.Participants.Away.TeamNameShort) + "\n\n"
	for i, p := range players.AwayPlayers {
		if i >= 11 {
			break
		}
		awayContent += fmt.Sprintf("#%-2d %-20s\n", p.ShirtNumber, p.PlayerName)
	}

	// Create two columns
	homeStyle := lipgloss.NewStyle().Width((m.width - 8) / 2)
	awayStyle := lipgloss.NewStyle().Width((m.width - 8) / 2)

	columns := lipgloss.JoinHorizontal(lipgloss.Top,
		homeStyle.Render(homeContent),
		awayStyle.Render(awayContent))

	return style.Render(columns)
}

func (m Model) viewTeamPanel() string {
	style := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("63")).
		Padding(1, 2).
		Width(m.width - 4).
		Height((m.height / 2) - 2)

	content := titleStyle.Render("Customer Team") + "\n\n"

	if m.state.Selection == nil || m.state.Selection.Player1 == nil {
		content += normalStyle.Render("No team set  ")
		content += highlightStyle.Render("Press 'c'") + normalStyle.Render(" to create default team")
	} else {
		players := []*models.Player{
			m.state.Selection.Player1,
			m.state.Selection.Player2,
			m.state.Selection.Player3,
			m.state.Selection.Player4,
			m.state.Selection.Player5,
			m.state.Selection.Player6,
			m.state.Selection.Player7,
		}

		// Display in a row format
		for i, p := range players {
			if p != nil {
				content += fmt.Sprintf("%-3d %-15s %-12s %-15s  ",
					i+1,
					fmt.Sprintf("%s %s", p.FirstName, p.LastName),
					p.Position,
					p.TeamName)
			}
		}
	}

	return style.Render(content)
}

func (m Model) viewGameweekPanel() string {
	borderColor := lipgloss.Color("63")
	if m.focusedPane == focusGameWeek {
		borderColor = lipgloss.Color("170")
	}

	style := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(borderColor).
		Padding(1, 2).
		Width((m.width * 2 / 5) - 2).
		Height((m.height / 2) - 2)

	gw := m.state.CurrentGameWeek

	// Show prefix
	prefixDisplay := "dev"
	if m.prefix != "" && m.prefix != "dev" {
		prefixDisplay = m.prefix
	}
	content := normalStyle.Render(fmt.Sprintf("Prefix: %s\n", prefixDisplay))
	content += titleStyle.Render(fmt.Sprintf("GameWeek %s", gw.GameWeekID))

	// Check if this is the current gameweek
	now := time.Now()
	startDate, _ := time.Parse(time.RFC3339, gw.CustomerStartDate)
	endDate, _ := time.Parse(time.RFC3339, gw.CustomerEndDate)
	isCurrent := now.After(startDate) && now.Before(endDate)

	if isCurrent {
		content += " " + highlightStyle.Render("(CURRENT)")
	}

	content += "\n\n"
	content += fmt.Sprintf("Label: %s\n\n", gw.Label)
	content += fmt.Sprintf("Start: %s\n", gw.CustomerStartDate[:10])
	content += fmt.Sprintf("End: %s\n\n", gw.CustomerEndDate[:10])

	content += normalStyle.Render("Controls:\n")
	// if isCurrent {
	// 	content += helpStyle.Render("c: create team\n")
	// } else {
	if !isCurrent {
		content += helpStyle.Foreground(lipgloss.Color("170")).Render("m: make current\n")
	}
	content += helpStyle.Render("b: batch update\n")
	content += helpStyle.Render("g: change gameweek\n")
	content += helpStyle.Render("r: reset environment\n")
	content += helpStyle.Render("q: quit")

	return style.Render(content)
}

func (m Model) viewFixturesPanel() string {
	borderColor := lipgloss.Color("63")
	borderStyle := lipgloss.RoundedBorder()

	if m.focusedPane == focusFixtures {
		borderColor = lipgloss.Color("170")
	}

	if m.fixtureSelectMode {
		borderStyle = lipgloss.ThickBorder()
		borderColor = lipgloss.Color("205")
	}

	style := lipgloss.NewStyle().
		Border(borderStyle).
		BorderForeground(borderColor).
		Padding(1, 2).
		Width((m.width * 3 / 5) - 2).
		Height((m.height / 2) - 2)

	content := titleStyle.Render("Fixtures")
	if m.fixtureSelectMode {
		content += " " + highlightStyle.Render("(SELECT MODE)")
	}
	content += "\n\n"

	if m.err != nil {
		content += lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Render(fmt.Sprintf("Error: %v\n", m.err))
	} else if len(m.state.Fixtures) == 0 {
		content += "Loading fixtures..."
	} else {
		content += m.fixturesTable.View() + "\n"

		if m.fixtureSelectMode {
			content += "\n" + normalStyle.Render("↑↓: navigate • enter: edit • esc: exit")
		} else if m.focusedPane == focusFixtures {
			content += "\n" + normalStyle.Render("enter: select fixture")
		}
	}

	return style.Render(content)
}

func (m Model) viewEditModal() string {
	if m.selectedFixtureIdx >= len(m.state.Fixtures) {
		return ""
	}

	f := &m.state.Fixtures[m.selectedFixtureIdx]

	modalStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("205")).
		Padding(1, 2).
		Width(50)

	content := titleStyle.Render("Edit Fixture") + "\n\n"
	content += fmt.Sprintf("%s vs %s\n\n",
		f.Participants.Home.TeamNameShort,
		f.Participants.Away.TeamNameShort)

	// Period selection
	periodLabel := "Period: "
	if m.editModalField == 0 {
		periodLabel = highlightStyle.Render("> Period: ")
	}
	content += periodLabel + string(f.Period) + "\n"

	// Scores
	homeScoreLabel := "Home Score: "
	if m.editModalField == 1 {
		homeScoreLabel = highlightStyle.Render("> Home Score: ")
	}
	homeScore := "-"
	if f.HomeScore != nil {
		homeScore = fmt.Sprintf("%d", *f.HomeScore)
	}
	content += homeScoreLabel + homeScore + "\n"

	awayScoreLabel := "Away Score: "
	if m.editModalField == 2 {
		awayScoreLabel = highlightStyle.Render("> Away Score: ")
	}
	awayScore := "-"
	if f.AwayScore != nil {
		awayScore = fmt.Sprintf("%d", *f.AwayScore)
	}
	content += awayScoreLabel + awayScore + "\n"

	// Clock time (only for live matches)
	if f.Period == models.PeriodFirstHalf || f.Period == models.PeriodSecondHalf {
		clockMinLabel := "Clock Min: "
		if m.editModalField == 3 {
			clockMinLabel = highlightStyle.Render("> Clock Min: ")
		}
		content += clockMinLabel + fmt.Sprintf("%d\n", f.ClockTimeMin)

		clockSecLabel := "Clock Sec: "
		if m.editModalField == 4 {
			clockSecLabel = highlightStyle.Render("> Clock Sec: ")
		}
		content += clockSecLabel + fmt.Sprintf("%d\n", f.ClockTimeSec)
	}

	// Show dropdown if field select is active
	if m.showFieldSelect {
		content += "\n" + titleStyle.Render("Select value:") + "\n"
		for i, opt := range m.fieldOptions {
			if i == m.fieldOptionIdx {
				content += highlightStyle.Render("> "+opt) + "\n"
			} else {
				content += normalStyle.Render("  "+opt) + "\n"
			}
		}
		content += "\n" + normalStyle.Render("↑↓: select • enter: apply • esc: cancel")
	} else {
		content += "\n" + normalStyle.Render("↑↓: navigate • enter: select • +/-: scores • s: save • esc: cancel")
	}

	return modalStyle.Render(content)
}

func (m Model) viewBatchModal() string {
	modalStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("205")).
		Padding(1, 2).
		Width(60)

	content := titleStyle.Render("Batch Update Fixtures") + "\n\n"
	content += fmt.Sprintf("GameWeek: %s\n", m.state.CurrentGameWeek.GameWeekID)
	content += fmt.Sprintf("Fixtures: %d\n\n", len(m.state.Fixtures))
	content += "Select preset:\n\n"

	presets := []struct {
		name string
		desc string
	}{
		{"Kickoff", "FIRST_HALF, clockTime:0, startDate:-5min"},
		{"Half Time", "HALF_TIME, clockTime:45"},
		{"Second Half", "SECOND_HALF, clockTime:45"},
		{"Full Time", "FULL_TIME, clockTime:90"},
	}

	for i, preset := range presets {
		if i == m.batchPresetIdx {
			content += highlightStyle.Render("> "+preset.name) + "\n"
			content += normalStyle.Render("  "+preset.desc) + "\n"
		} else {
			content += normalStyle.Render("  "+preset.name) + "\n"
			content += normalStyle.Render("  "+preset.desc) + "\n"
		}
	}

	content += "\n" + normalStyle.Render("↑↓: select • enter: apply • esc: cancel")

	return modalStyle.Render(content)
}

func (m Model) viewGoalModal() string {
	modalStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("205")).
		Padding(1, 2).
		Width(70)

	fixture := m.state.Fixtures[m.selectedFixtureIdx]

	content := titleStyle.Render("Assign Goal Scorers") + "\n\n"
	content += fmt.Sprintf("%s vs %s\n\n",
		fixture.Participants.Home.TeamNameShort,
		fixture.Participants.Away.TeamNameShort)

	homeGoals := 0
	if fixture.HomeScore != nil {
		homeGoals = *fixture.HomeScore
	}
	_ = homeGoals // Used for determining team assignment

	// Display goals
	for i, goal := range m.pendingGoals {
		isHome := i < homeGoals
		teamName := fixture.Participants.Away.TeamNameShort
		if isHome {
			teamName = fixture.Participants.Home.TeamNameShort
		}

		prefix := "  "
		if i == m.goalModalIdx {
			prefix = "> "
		}

		playerName := goal.PlayerName
		if playerName == "" {
			playerName = "[Select Player]"
		}

		timeStr := fmt.Sprintf("%d'", goal.TimeMin)
		if goal.TimeMin == 0 {
			timeStr = "[Select Time]"
		}

		// Highlight current field
		if i == m.goalModalIdx {
			if m.goalModalField == 0 {
				playerName = highlightStyle.Render(playerName)
			} else {
				timeStr = highlightStyle.Render(timeStr)
			}
		}

		content += fmt.Sprintf("%s%s: %s - %s\n", prefix, teamName, playerName, timeStr)
	}

	// Show dropdown if field select is active
	if m.showFieldSelect {
		content += "\n" + titleStyle.Render("Select:") + "\n"
		for i, opt := range m.fieldOptions {
			if i == m.fieldOptionIdx {
				content += highlightStyle.Render("> "+opt) + "\n"
			} else {
				content += normalStyle.Render("  "+opt) + "\n"
			}
			if i >= 10 {
				content += normalStyle.Render(fmt.Sprintf("  ... (%d more)\n", len(m.fieldOptions)-i-1))
				break
			}
		}
		content += "\n" + helpStyle.Render("↑↓: select • enter: apply • esc: cancel")
	} else {
		content += "\n" + helpStyle.Render("↑↓: navigate goals • tab: switch field • enter: select • s: save • esc: cancel")
	}

	return modalStyle.Render(content)
}

// Messages
type authSuccessMsg struct {
	token string
}

type authErrorMsg struct {
	err error
}

type gameWeeksLoadedMsg struct {
	gameWeeks []models.GameWeek
	err       error
}

type fixturesLoadedMsg struct {
	fixtures []models.Fixture
	err      error
}

type selectionLoadedMsg struct {
	selection *models.Selection
	err       error
}

type playersLoadedMsg struct {
	players []models.Player
	err     error
}

type teamCreatedMsg struct {
	selection *models.Selection
	err       error
}

type gameWeekUpdatedMsg struct {
	gameWeek *models.GameWeek
	err      error
}

type fixtureUpdatedMsg struct {
	fixture *models.Fixture
	err     error
}

type batchUpdatedMsg struct {
	count int
	err   error
}

type githubActionMsg struct {
	success bool
	err     error
}

// Commands
func authenticateCmd(client *aws.AuthClient, username, password string) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		if err := client.SignIn(ctx, username, password); err != nil {
			return authErrorMsg{err: err}
		}
		return authSuccessMsg{token: client.GetIDToken()}
	}
}

func fetchGameWeeksCmd(client *aws.APIClient) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		data, err := client.GetGameWeeks(ctx)
		if err != nil {
			return gameWeeksLoadedMsg{err: err}
		}

		gameWeeks := make([]models.GameWeek, 0, len(data))
		for _, item := range data {
			gw := models.GameWeek{
				GameWeekID:        getString(item, "gameWeekId"),
				Label:             getString(item, "label"),
				FixturesStartDate: getString(item, "fixturesStartDate"),
				FixturesEndDate:   getString(item, "fixturesEndDate"),
				CustomerStartDate: getString(item, "customerStartDate"),
				CustomerEndDate:   getString(item, "customerEndDate"),
			}
			gameWeeks = append(gameWeeks, gw)
		}

		return gameWeeksLoadedMsg{gameWeeks: gameWeeks}
	}
}

func fetchFixturesCmd(client *aws.APIClient, gameWeekID string) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		data, err := client.GetFixturesByGameWeek(ctx, gameWeekID)
		if err != nil {
			return fixturesLoadedMsg{err: err}
		}

		// Parse fixtures from response
		// Response format: { gameWeekId: "27", fixtures: [...] }
		fixtures := []models.Fixture{}
		if fixtureList, ok := data["fixtures"].([]interface{}); ok {
			for _, item := range fixtureList {
				if fixtureMap, ok := item.(map[string]interface{}); ok {
					fixture := parseFixture(fixtureMap)
					fixtures = append(fixtures, fixture)
				}
			}
		}

		return fixturesLoadedMsg{fixtures: fixtures}
	}
}

func parseFixture(m map[string]interface{}) models.Fixture {
	fixture := models.Fixture{
		FixtureID:     getString(m, "fixtureId"),
		GameWeekID:    getString(m, "gameWeekId"),
		StartDate:     getString(m, "startDate"),
		Period:        models.FixturePeriod(getString(m, "period")),
		ClockTimeMin:  getInt(m, "clockTimeMin"),
		ClockTimeSec:  getInt(m, "clockTimeSec"),
		HomeTeamID:    getString(m, "homeTeamId"),
		AwayTeamID:    getString(m, "awayTeamId"),
		FixtureStatus: getString(m, "fixtureStatus"),
	}

	if participants, ok := m["participants"].(map[string]interface{}); ok {
		if home, ok := participants["home"].(map[string]interface{}); ok {
			fixture.Participants.Home = parseTeam(home)
		}
		if away, ok := participants["away"].(map[string]interface{}); ok {
			fixture.Participants.Away = parseTeam(away)
		}
	}

	// Parse scores
	if homeScore, ok := m["homeScore"].(float64); ok {
		score := int(homeScore)
		fixture.HomeScore = &score
	}
	if awayScore, ok := m["awayScore"].(float64); ok {
		score := int(awayScore)
		fixture.AwayScore = &score
	}

	// Parse metadata (preserve as-is)
	if metadata, ok := m["metadata"].(map[string]interface{}); ok {
		fixture.Metadata = metadata
	}

	// Parse goals (preserve as-is)
	if goals, ok := m["goals"].([]interface{}); ok {
		for _, g := range goals {
			if goalMap, ok := g.(map[string]interface{}); ok {
				goal := models.Goal{
					GoalID:        getString(goalMap, "goalId"),
					TimeMin:       getInt(goalMap, "timeMin"),
					TimeSec:       getInt(goalMap, "timeSec"),
					Type:          getString(goalMap, "type"),
					PlayerID:      getString(goalMap, "playerId"),
					PlayerName:    getString(goalMap, "playerName"),
					Period:        models.FixturePeriod(getString(goalMap, "period")),
					TeamID:        getString(goalMap, "teamId"),
					SevenGoalType: getString(goalMap, "sevenGoalType"),
				}
				fixture.Goals = append(fixture.Goals, goal)
			}
		}
	}

	return fixture
}

func fetchSelectionCmd(client *aws.APIClient) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		data, err := client.GetSelections(ctx)
		if err != nil {
			return selectionLoadedMsg{err: err}
		}

		selection := &models.Selection{
			GameWeekID: getString(data, "gameWeekId"),
		}

		// Parse player slots
		for i := 1; i <= 7; i++ {
			key := fmt.Sprintf("player%d", i)
			if playerData, ok := data[key].(map[string]interface{}); ok {
				if id := getString(playerData, "id"); id != "" {
					player := &models.Player{PlayerID: id}
					switch i {
					case 1:
						selection.Player1 = player
					case 2:
						selection.Player2 = player
					case 3:
						selection.Player3 = player
					case 4:
						selection.Player4 = player
					case 5:
						selection.Player5 = player
					case 6:
						selection.Player6 = player
					case 7:
						selection.Player7 = player
					}
				}
			}
		}

		return selectionLoadedMsg{selection: selection}
	}
}

func createDefaultTeamCmd(client *aws.APIClient) tea.Cmd {
	return func() tea.Msg {
		logger.Println("Creating default team...")

		ctx := context.Background()

		// Fetch available players
		logger.Println("Fetching available players...")
		data, err := client.GetGameWeekPlayers(ctx)
		if err != nil {
			logger.Printf("ERROR: Failed to fetch players: %v", err)
			return teamCreatedMsg{err: err}
		}

		var players []models.Player
		if playerList, ok := data["players"].([]interface{}); ok {
			logger.Printf("Found %d players", len(playerList))
			for _, item := range playerList {
				if playerMap, ok := item.(map[string]interface{}); ok {
					players = append(players, models.Player{
						PlayerID:  getString(playerMap, "playerId"),
						FirstName: getString(playerMap, "firstName"),
						LastName:  getString(playerMap, "lastName"),
						Position:  models.Position(getString(playerMap, "position")),
						TeamID:    getString(playerMap, "teamId"),
						TeamName:  getString(playerMap, "teamName"),
					})
				}
			}
		} else {
			logger.Printf("ERROR: Invalid players data structure")
			return teamCreatedMsg{err: fmt.Errorf("invalid players data")}
		}

		// Select 3 forwards + 4 mids/defenders (one per team)
		usedTeams := make(map[string]bool)
		var forwards []models.Player
		var others []models.Player

		// Get 3 forwards first
		logger.Println("Selecting 3 forwards...")
		for _, p := range players {
			if p.Position == models.PositionForward && !usedTeams[p.TeamID] && len(forwards) < 3 {
				forwards = append(forwards, p)
				usedTeams[p.TeamID] = true
				logger.Printf("  Selected forward: %s %s (%s)", p.FirstName, p.LastName, p.TeamName)
			}
		}

		if len(forwards) < 3 {
			logger.Printf("ERROR: Not enough forwards (found %d)", len(forwards))
			return teamCreatedMsg{err: fmt.Errorf("not enough forwards available (found %d)", len(forwards))}
		}

		// Get 4 mids/defenders
		logger.Println("Selecting 4 mids/defenders...")
		for _, p := range players {
			if (p.Position == models.PositionMidfielder || p.Position == models.PositionDefender) &&
				!usedTeams[p.TeamID] && len(others) < 4 {
				others = append(others, p)
				usedTeams[p.TeamID] = true
				logger.Printf("  Selected %s: %s %s (%s)", p.Position, p.FirstName, p.LastName, p.TeamName)
			}
		}

		if len(others) < 4 {
			logger.Printf("ERROR: Not enough mids/defenders (found %d)", len(others))
			return teamCreatedMsg{err: fmt.Errorf("not enough mids/defenders available (found %d)", len(others))}
		}

		// Combine: 3 forwards + 4 others = 7 players
		selected := append(forwards, others...)
		logger.Printf("Total selected: %d players (3F + 4M/D)", len(selected))

		// Create selection payload
		selections := make(map[string]interface{})
		for i, p := range selected {
			selections[fmt.Sprintf("player%d", i+1)] = map[string]string{"id": p.PlayerID}
		}

		// Submit selection
		logger.Println("Submitting team selection...")
		if err := client.PutSelections(ctx, selections); err != nil {
			logger.Printf("ERROR: Failed to submit selections: %v", err)
			return teamCreatedMsg{err: err}
		}

		logger.Println("Team created successfully!")
		// Build selection object
		selection := &models.Selection{}
		for i, p := range selected {
			player := p
			switch i {
			case 0:
				selection.Player1 = &player
			case 1:
				selection.Player2 = &player
			case 2:
				selection.Player3 = &player
			case 3:
				selection.Player4 = &player
			case 4:
				selection.Player5 = &player
			case 5:
				selection.Player6 = &player
			case 6:
				selection.Player7 = &player
			}
		}

		return teamCreatedMsg{selection: selection}
	}
}

func makeGameWeekCurrentCmd(gw *models.GameWeek, prefix string, dynamoClient *aws.DynamoDBClient) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()

		// Set dates to make this gameweek current
		now := time.Now()
		gw.CustomerStartDate = now.Add(-24 * time.Hour).Format(time.RFC3339)
		gw.CustomerEndDate = now.Add(6 * 24 * time.Hour).Format(time.RFC3339)
		gw.FixturesStartDate = now.Add(-12 * time.Hour).Format(time.RFC3339)
		gw.FixturesEndDate = now.Add(5 * 24 * time.Hour).Format(time.RFC3339)

		// Update in DynamoDB using the passed client (already configured for GameWeeks table)
		if err := dynamoClient.UpdateGameWeek(ctx, gw); err != nil {
			return gameWeekUpdatedMsg{err: err}
		}

		return gameWeekUpdatedMsg{gameWeek: gw}
	}
}

func updateFixtureCmd(fixture *models.Fixture, prefix string) tea.Cmd {
	return func() tea.Msg {
		logger.Printf("Updating fixture: %s", fixture.FixtureID)

		// Validate before saving
		if err := fixture.Validate(); err != nil {
			logger.Printf("ERROR: Validation failed: %v", err)
			return fixtureUpdatedMsg{err: fmt.Errorf("validation failed: %w", err)}
		}

		ctx := context.Background()

		// Create DynamoDB client for Fixtures table
		tableName := prefix + "-GameWeekFixtures"
		if prefix == "dev" || prefix == "" {
			tableName = "int-dev-GameWeekFixtures"
		}

		logger.Printf("Using table: %s", tableName)
		dynamoClient, err := aws.NewDynamoDBClient(ctx, "eu-west-2", tableName)
		if err != nil {
			logger.Printf("ERROR: Failed to create dynamo client: %v", err)
			return fixtureUpdatedMsg{err: fmt.Errorf("failed to create dynamo client: %w", err)}
		}

		// Ensure FixtureStatus is set
		if fixture.FixtureStatus == "" {
			fixture.FixtureStatus = "FIXTURE"
		}

		// Update startDate if going live (not PRE_MATCH)
		if fixture.Period != models.PeriodPreMatch {
			// Set startDate to 5 minutes ago to enable websocket
			now := time.Now().UTC()
			fixture.StartDate = now.Add(-5 * time.Minute).Format(time.RFC3339)
		}

		// Ensure metadata exists with default periods
		if fixture.Metadata == nil {
			fixture.Metadata = map[string]interface{}{
				"periods": map[string]interface{}{
					"FIRST_HALF": map[string]interface{}{
						"expectedLengthMins": 45,
					},
					"SECOND_HALF": map[string]interface{}{
						"expectedLengthMins": 45,
					},
				},
			}
		}

		logger.Printf("Fixture data: Period=%s, HomeScore=%v, AwayScore=%v, Clock=%d:%d",
			fixture.Period, fixture.HomeScore, fixture.AwayScore, fixture.ClockTimeMin, fixture.ClockTimeSec)

		// Update in DynamoDB
		if err := dynamoClient.UpdateFixture(ctx, fixture); err != nil {
			logger.Printf("ERROR: Failed to update fixture in DynamoDB: %v", err)
			return fixtureUpdatedMsg{err: err}
		}

		logger.Printf("Successfully updated fixture: %s", fixture.FixtureID)
		return fixtureUpdatedMsg{fixture: fixture}
	}
}

func batchUpdateFixturesCmd(fixtures []models.Fixture, preset string, prefix string) tea.Cmd {
	return func() tea.Msg {
		logger.Printf("Batch update started: preset=%s, prefix=%s, fixtures=%d", preset, prefix, len(fixtures))

		ctx := context.Background()

		tableName := prefix + "-GameWeekFixtures"
		if prefix == "dev" || prefix == "" {
			tableName = "int-dev-GameWeekFixtures"
		}

		logger.Printf("Creating DynamoDB client for table: %s", tableName)
		dynamoClient, err := aws.NewDynamoDBClient(ctx, "eu-west-2", tableName)
		if err != nil {
			logger.Printf("ERROR: Failed to create dynamo client: %v", err)
			return batchUpdatedMsg{err: fmt.Errorf("failed to create dynamo client: %w", err)}
		}

		now := time.Now().UTC()

		for i := range fixtures {
			f := &fixtures[i]
			logger.Printf("Updating fixture %d/%d: %s", i+1, len(fixtures), f.FixtureID)

			switch preset {
			case "kickoff":
				f.Period = models.PeriodFirstHalf
				f.FixtureStatus = "IN_PLAY"
				f.ClockTimeMin = 0
				f.ClockTimeSec = 0
				f.StartDate = now.Add(-5 * time.Minute).Format(time.RFC3339)
			case "halftime":
				f.Period = models.PeriodHalfTime
				f.ClockTimeMin = 45
				f.ClockTimeSec = 0
			case "secondhalf":
				f.Period = models.PeriodSecondHalf
				f.ClockTimeMin = 45
				f.ClockTimeSec = 0
			case "fulltime":
				f.Period = models.PeriodFullTime
				f.ClockTimeMin = 90
				f.ClockTimeSec = 0
			}

			logger.Printf("  Period: %s, Clock: %d:%d", f.Period, f.ClockTimeMin, f.ClockTimeSec)

			// Validate before saving
			if err := f.Validate(); err != nil {
				logger.Printf("ERROR: Validation failed for fixture %s: %v", f.FixtureID, err)
				return batchUpdatedMsg{err: fmt.Errorf("validation failed for %s: %w", f.FixtureID, err)}
			}

			// Ensure metadata
			if f.Metadata == nil {
				f.Metadata = map[string]interface{}{
					"periods": map[string]interface{}{
						"FIRST_HALF":  map[string]interface{}{"expectedLengthMins": 45},
						"SECOND_HALF": map[string]interface{}{"expectedLengthMins": 45},
					},
				}
			}

			if err := dynamoClient.UpdateFixture(ctx, f); err != nil {
				logger.Printf("ERROR: Failed to update fixture %s: %v", f.FixtureID, err)
				return batchUpdatedMsg{err: fmt.Errorf("failed to update fixture %s: %w", f.FixtureID, err)}
			}
			logger.Printf("  Successfully updated fixture %s", f.FixtureID)
		}

		logger.Printf("Batch update completed successfully: %d fixtures", len(fixtures))
		return batchUpdatedMsg{count: len(fixtures)}
	}
}

func parseTeam(m map[string]interface{}) models.Team {
	return models.Team{
		TeamID:        getString(m, "teamId"),
		TeamName:      getString(m, "teamName"),
		TeamNameShort: getString(m, "teamNameShort"),
		TeamCode:      getString(m, "teamCode"),
	}
}

func getString(m map[string]interface{}, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

func getStringFromAttr(m map[string]types.AttributeValue, key string) string {
	if v, ok := m[key].(*types.AttributeValueMemberS); ok {
		return v.Value
	}
	return ""
}

func getIntFromAttr(m map[string]types.AttributeValue, key string) int {
	if v, ok := m[key].(*types.AttributeValueMemberN); ok {
		val, _ := strconv.Atoi(v.Value)
		return val
	}
	return 0
}

func getInt(m map[string]interface{}, key string) int {
	if v, ok := m[key].(float64); ok {
		return int(v)
	}
	return 0
}

func fetchPlayersCmd(apiClient *aws.APIClient, gameWeekID string, fixtures []models.Fixture) tea.Cmd {
	return func() tea.Msg {
		logger.Printf("Fetching players for gameweek: %s via API", gameWeekID)

		ctx := context.Background()

		// Call API to get players
		data, err := apiClient.GetGameWeekPlayers(ctx)
		if err != nil {
			logger.Printf("ERROR: Failed to fetch players from API: %v", err)
			return playersLoadedMsg{err: err}
		}

		// Parse players from response
		var players []models.Player
		if playerList, ok := data["players"].([]interface{}); ok {
			logger.Printf("Found %d players in API response", len(playerList))
			for _, item := range playerList {
				if playerMap, ok := item.(map[string]interface{}); ok {
					player := models.Player{
						PlayerID:    getString(playerMap, "playerId"),
						PlayerName:  getString(playerMap, "playerName"),
						FirstName:   getString(playerMap, "playerFirstName"),
						LastName:    getString(playerMap, "playerShortName"),
						Position:    models.Position(getString(playerMap, "position")),
						TeamID:      getString(playerMap, "teamId"),
						TeamName:    getString(playerMap, "teamName"),
						ShirtNumber: getInt(playerMap, "shirtNumber"),
						FixtureID:   getString(playerMap, "fixtureId"),
					}
					players = append(players, player)
				}
			}
		} else {
			logger.Println("ERROR: Invalid players data structure from API")
			return playersLoadedMsg{err: fmt.Errorf("invalid players data structure")}
		}

		logger.Printf("Parsed %d players from API", len(players))
		return playersLoadedMsg{players: players}
	}
}

func triggerGitHubActionCmd(prefix string) tea.Cmd {
	return func() tea.Msg {
		logger.Printf("Triggering GitHub Action for prefix: %s", prefix)

		// Get GitHub token from environment
		token := os.Getenv("GITHUB_TOKEN")
		if token == "" {
			logger.Println("ERROR: GITHUB_TOKEN not set")
			return githubActionMsg{err: fmt.Errorf("GITHUB_TOKEN environment variable not set")}
		}

		// Prepare workflow dispatch payload
		payload := map[string]interface{}{
			"ref": "develop",
			"inputs": map[string]string{
				"prefix": prefix,
			},
		}

		payloadBytes, _ := json.Marshal(payload)

		// GitHub API endpoint for workflow dispatch
		url := "https://api.github.com/repos/angstromsports/aws-node-lambdas/actions/workflows/clone-environment-data.yml/dispatches"

		req, err := http.NewRequest("POST", url, bytes.NewBuffer(payloadBytes))
		if err != nil {
			logger.Printf("ERROR: Failed to create request: %v", err)
			return githubActionMsg{err: err}
		}

		req.Header.Set("Accept", "application/vnd.github+json")
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
		req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
		req.Header.Set("Content-Type", "application/json")

		client := &http.Client{Timeout: 10 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			logger.Printf("ERROR: Failed to trigger workflow: %v", err)
			return githubActionMsg{err: err}
		}
		defer resp.Body.Close()

		if resp.StatusCode != 204 {
			body, _ := io.ReadAll(resp.Body)
			logger.Printf("ERROR: GitHub API returned status %d: %s", resp.StatusCode, string(body))
			return githubActionMsg{err: fmt.Errorf("GitHub API error (status %d)", resp.StatusCode)}
		}

		logger.Printf("Successfully triggered environment reset for prefix: %s", prefix)
		return githubActionMsg{success: true}
	}
}
