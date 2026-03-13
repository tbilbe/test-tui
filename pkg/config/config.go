package config

import (
	"fmt"
	"os"
)

type Config struct {
	APIEndpoint  string
	UserPoolID   string
	ClientID     string
	Prefix       string
	AWSRegion    string
	FixtureTable string
}

func Load() (*Config, error) {
	cfg := &Config{
		APIEndpoint: getEnv("API_ENDPOINT", "https://dev.api.playtheseven.com"),
		UserPoolID:  getEnv("USER_POOL_ID", "eu-west-2_uqwEOLO5d"),
		ClientID:    getEnv("CLIENT_ID", ""),
		Prefix:      getEnv("PREFIX", ""),
		AWSRegion:   getEnv("AWS_REGION", "eu-west-2"),
	}

	// Validate required fields - only CLIENT_ID is required
	if cfg.ClientID == "" {
		return nil, fmt.Errorf("CLIENT_ID is required.\nGet it from AWS Console → Cognito → User Pools → App Clients\nThen run: export CLIENT_ID=\"your-client-id\" && seven-test-tui")
	}

	return cfg, nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
