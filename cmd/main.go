package main

import (
	"context"
	"fmt"
	"os"
	"time"

	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/angstromsports/seven-test-tui/internal/aws"
	"github.com/angstromsports/seven-test-tui/internal/ui"
	"github.com/angstromsports/seven-test-tui/pkg/config"
)

// Overwritten via -ldflags at release time; "dev" means a local build.
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	if len(os.Args) > 1 && (os.Args[1] == "version" || os.Args[1] == "--version") {
		fmt.Printf("seven-test-tui %s (commit %s, built %s)\n", version, commit, date)
		return
	}
	if len(os.Args) > 1 {
		fmt.Fprintln(os.Stderr, "Usage: seven-test-tui [version]")
		os.Exit(2)
	}

	cfg, err := config.Load()
	if err != nil {
		fmt.Printf("Configuration error: %v\n", err)
		os.Exit(1)
	}

	credentialContext, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	awsConfig, err := awsconfig.LoadDefaultConfig(credentialContext, awsconfig.WithRegion(cfg.AWSRegion))
	var credentialsProvider config.CredentialsProvider
	if err == nil {
		credentialsProvider = awsConfig.Credentials
	}
	preflight := cfg.Preflight(credentialsProvider)
	fmt.Print(preflight.String())
	if preflight.HasErrors() {
		os.Exit(1)
	}

	run(cfg)
}

func run(cfg *config.Config) {
	apiClient := aws.NewAPIClient(cfg.APIEndpoint)
	apiClient.SetAPIKey(cfg.APIKey)
	apiClient.SetIDToken(cfg.AccessToken)

	// The DynamoDB client is built on the prefix screen, once a prefix is known.
	// Building one here would use the wrong table and abort startup on an
	// unresolved AWS profile, contradicting the preflight's read-only warning.
	program := tea.NewProgram(
		ui.NewModel(apiClient, nil, cfg.APIKey, cfg.AccessToken),
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)
	if _, err := program.Run(); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}
