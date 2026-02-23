package main

import (
	"context"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/angstromsports/seven-test-tui/internal/aws"
	"github.com/angstromsports/seven-test-tui/internal/ui"
	"github.com/angstromsports/seven-test-tui/pkg/config"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		fmt.Printf("Configuration error: %v\n", err)
		os.Exit(1)
	}

	ctx := context.Background()

	// Create auth client
	authClient, err := aws.NewAuthClient(ctx, cfg.AWSRegion, cfg.UserPoolID, cfg.ClientID)
	if err != nil {
		fmt.Printf("Failed to create auth client: %v\n", err)
		os.Exit(1)
	}

	// Create API client
	apiClient := aws.NewAPIClient(cfg.APIEndpoint)

	// Run TUI with auth screen
	p := tea.NewProgram(
		ui.NewModel(authClient, apiClient),
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)
	
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}

