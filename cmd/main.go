package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

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
	// Version must work before config, so an unconfigured install can still be identified.
	if len(os.Args) > 1 && (os.Args[1] == "version" || os.Args[1] == "--version") {
		fmt.Printf("seven-test-tui %s (commit %s, built %s)\n", version, commit, date)
		return
	}

	cfg, err := config.Load()
	if err != nil {
		fmt.Printf("Configuration error: %v\n", err)
		os.Exit(1)
	}
	if len(os.Args) > 1 {
		if os.Args[1] != "bootstrap" {
			fmt.Fprintln(os.Stderr, "Usage: seven-test-tui [bootstrap|version]")
			os.Exit(2)
		}
		bootstrap(cfg)
		return
	}
	if err := cfg.ValidateForApplication(); err != nil {
		fmt.Printf("Configuration error: %v\n", err)
		os.Exit(1)
	}
	run(cfg)
}

func bootstrap(cfg *config.Config) {
	client, err := newTokenClient(cfg)
	if err != nil {
		fmt.Printf("Failed to initialise token storage: %v\n", err)
		return
	}

	authorizationURL, err := client.AuthorizationURL()
	if err != nil {
		fmt.Printf("Failed to create authorization URL: %v\n", err)
		return
	}
	fmt.Printf("Open this URL in a browser and complete sign-in:\n%s\n\n", authorizationURL)
	fmt.Println("The seven:// deep link will fail. Paste its full URL here:")
	authorizationResponse, err := bufio.NewReader(os.Stdin).ReadString('\n')
	if err != nil {
		fmt.Printf("Failed to read authorization response: %v\n", err)
		return
	}
	if _, err := client.ExchangeAuthorizationCode(context.Background(), authorizationResponse); err != nil {
		fmt.Printf("Bootstrap failed: %v\n", err)
		return
	}
	fmt.Println("Session saved. Start seven-test-tui normally.")
}

func run(cfg *config.Config) {
	ctx := context.Background()
	accessToken := cfg.AccessToken
	var tokenClient *aws.TokenClient
	if accessToken == "" {
		var err error
		tokenClient, err = newTokenClient(cfg)
		if err == nil {
			accessToken, err = tokenClient.Refresh(ctx)
		}
		if err != nil {
			fmt.Printf("Authentication error: %v\n", err)
			os.Exit(1)
		}
	}

	apiClient := aws.NewAPIClient(cfg.APIEndpoint)
	apiClient.SetAPIKey(cfg.APIKey)
	apiClient.SetIDToken(accessToken)
	var tokenRefresher aws.TokenRefresher
	if tokenClient != nil {
		tokenRefresher = tokenClient
		apiClient.SetTokenRefresher(tokenRefresher)
	}

	dynamoClient, err := aws.NewDynamoDBClient(ctx, cfg.AWSRegion, cfg.Prefix, cfg.Prefix+"-GameWeek")
	if err != nil {
		fmt.Printf("Failed to create DynamoDB client: %v\n", err)
		os.Exit(1)
	}

	program := tea.NewProgram(
		ui.NewModel(apiClient, dynamoClient, cfg.APIKey, accessToken, tokenRefresher),
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)
	if _, err := program.Run(); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}

func newTokenClient(cfg *config.Config) (*aws.TokenClient, error) {
	store, err := aws.NewRefreshTokenStore(cfg.APIEndpoint)
	if err != nil {
		return nil, err
	}
	return aws.NewTokenClient(strings.TrimRight(cfg.APIEndpoint, "/"), aws.DeviceID(), store), nil
}
