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
		APIEndpoint: getEnv("API_ENDPOINT", ""),
		UserPoolID:  getEnv("USER_POOL_ID", ""),
		ClientID:    getEnv("CLIENT_ID", ""),
		Prefix:      getEnv("PREFIX", "int-dev"),
		AWSRegion:   getEnv("AWS_REGION", "eu-west-2"),
	}

	// Construct fixture table name from prefix
	cfg.FixtureTable = fmt.Sprintf("%s-GameWeekFixtures", cfg.Prefix)

	// Validate required fields
	if cfg.APIEndpoint == "" {
		return nil, fmt.Errorf("API_ENDPOINT is required")
	}
	if cfg.UserPoolID == "" {
		return nil, fmt.Errorf("USER_POOL_ID is required")
	}
	if cfg.ClientID == "" {
		return nil, fmt.Errorf("CLIENT_ID is required")
	}

	return cfg, nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
