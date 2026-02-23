package ui

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/angstromsports/seven-test-tui/internal/aws"
	"github.com/angstromsports/seven-test-tui/internal/models"
)

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

	inputStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("86"))
)

type screenType int

const (
	authScreenType screenType = iota
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
	state         *models.AppState
	authClient    *aws.AuthClient
	apiClient     *aws.APIClient
	currentScreen screenType
	authScreen    AuthScreen
	input         string
	inputMode     bool
	err           error
	width         int
	height        int
	focusedPane   focusPane
}

func NewModel(authClient *aws.AuthClient, apiClient *aws.APIClient) Model {
	return Model{
		state:       models.NewAppState(),
		authClient:  authClient,
		apiClient:   apiClient,
		currentScreen: authScreenType,
		authScreen:  NewAuthScreen(),
		inputMode:   true,
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
		// Move to gameweek screen and fetch gameweeks
		m.apiClient.SetIDToken(msg.token)
		m.currentScreen = gameweekScreenType
		return m, fetchGameWeeksCmd(m.apiClient)

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
		}
		return m, nil
	}

	// Route to appropriate screen
	switch m.currentScreen {
	case authScreenType:
		var cmd tea.Cmd
		m.authScreen, cmd = m.authScreen.Update(msg)
		return m, cmd

	case gameweekScreenType:
		return m.updateGameweekScreen(msg)
	}

	return m, nil
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
							return m, fetchFixturesCmd(m.apiClient, m.input)
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

	case gameweekScreenType:
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, m.viewGameweekScreen())

	case fixtureScreenType:
		return m.viewFixtureScreen()
	}

	return ""
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

	leftPanel := m.viewGameweekPanel()
	rightPanel := m.viewFixturesPanel()

	return lipgloss.JoinHorizontal(lipgloss.Top, leftPanel, rightPanel)
}

func (m Model) viewGameweekPanel() string {
	gw := m.state.CurrentGameWeek
	content := titleStyle.Render(fmt.Sprintf("GameWeek %s", gw.GameWeekID)) + "\n\n"
	content += fmt.Sprintf("Label: %s\n", gw.Label)
	content += fmt.Sprintf("Start: %s\n", gw.CustomerStartDate)
	
	return boxStyle.Width(40).Height(m.height - 2).Render(content)
}

func (m Model) viewFixturesPanel() string {
	content := titleStyle.Render("Fixtures") + "\n\n"
	
	if m.err != nil {
		content += lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Render(fmt.Sprintf("Error: %v\n", m.err))
	} else if len(m.state.Fixtures) == 0 {
		content += "Loading fixtures..."
	} else {
		for i, f := range m.state.Fixtures {
			if i > 10 {
				content += fmt.Sprintf("\n... and %d more", len(m.state.Fixtures)-10)
				break
			}
			homeScore := "-"
			awayScore := "-"
			if f.HomeScore != nil {
				homeScore = fmt.Sprintf("%d", *f.HomeScore)
			}
			if f.AwayScore != nil {
				awayScore = fmt.Sprintf("%d", *f.AwayScore)
			}
			
			content += fmt.Sprintf("%s %s-%s %s | %s %d:%02d\n", 
				f.Participants.Home.TeamNameShort,
				homeScore,
				awayScore,
				f.Participants.Away.TeamNameShort,
				f.Period,
				f.ClockTimeMin,
				f.ClockTimeSec)
		}
	}

	return boxStyle.Width(m.width - 45).Height(m.height - 2).Render(content)
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
		FixtureID:    getString(m, "fixtureId"),
		GameWeekID:   getString(m, "gameWeekId"),
		StartDate:    getString(m, "startDate"),
		Period:       models.FixturePeriod(getString(m, "period")),
		ClockTimeMin: getInt(m, "clockTimeMin"),
		ClockTimeSec: getInt(m, "clockTimeSec"),
		HomeTeamID:   getString(m, "homeTeamId"),
		AwayTeamID:   getString(m, "awayTeamId"),
	}

	if participants, ok := m["participants"].(map[string]interface{}); ok {
		if home, ok := participants["home"].(map[string]interface{}); ok {
			fixture.Participants.Home = parseTeam(home)
		}
		if away, ok := participants["away"].(map[string]interface{}); ok {
			fixture.Participants.Away = parseTeam(away)
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
		ctx := context.Background()
		
		// Fetch available players
		data, err := client.GetGameWeekPlayers(ctx)
		if err != nil {
			return teamCreatedMsg{err: err}
		}

		var players []models.Player
		if playerList, ok := data["players"].([]interface{}); ok {
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
		}

		// Select 3 forwards + 4 others (one per team)
		usedTeams := make(map[string]bool)
		var selected []models.Player
		
		// Get 3 forwards
		for _, p := range players {
			if p.Position == models.PositionForward && !usedTeams[p.TeamID] && len(selected) < 3 {
				selected = append(selected, p)
				usedTeams[p.TeamID] = true
			}
		}
		
		// Get 4 more from remaining positions
		for _, p := range players {
			if p.Position != models.PositionForward && !usedTeams[p.TeamID] && len(selected) < 7 {
				selected = append(selected, p)
				usedTeams[p.TeamID] = true
			}
		}

		if len(selected) < 7 {
			return teamCreatedMsg{err: fmt.Errorf("not enough players available (found %d)", len(selected))}
		}

		// Create selection payload
		selections := make(map[string]interface{})
		for i, p := range selected {
			selections[fmt.Sprintf("player%d", i+1)] = map[string]string{"id": p.PlayerID}
		}

		// Submit selection
		if err := client.PutSelections(ctx, selections); err != nil {
			return teamCreatedMsg{err: err}
		}

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

func getInt(m map[string]interface{}, key string) int {
	if v, ok := m[key].(float64); ok {
		return int(v)
	}
	return 0
}
