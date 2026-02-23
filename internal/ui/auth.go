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
		lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render("tab: switch • enter: login • esc: quit")

	return authBoxStyle.Render(content)
}

func (a AuthScreen) GetCredentials() (string, string) {
	return a.usernameInput.Value(), a.passwordInput.Value()
}

type authSubmitMsg struct {
	username string
	password string
}
