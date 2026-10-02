package config

import (
	"fmt"
	"os"
)

type Config struct {
	APIEndpoint string
	APIKey      string
	AccessToken string
	Prefix      string
	AWSRegion   string
}

/**
 * Load reads environment configuration shared by bootstrap and the interactive TUI.
 */
func Load() (*Config, error) {
	return &Config{
		APIEndpoint: getEnv("API_ENDPOINT", "https://dev.cf.playtheseven.com"),
		APIKey:      getEnv("API_KEY", ""),
		AccessToken: getEnv("SEVEN_ACCESS_TOKEN", ""),
		Prefix:      getEnv("PREFIX", ""),
		AWSRegion:   getEnv("AWS_REGION", "eu-west-2"),
	}, nil
}

/**
 * ValidateForApplication rejects missing CloudFront credentials for read and write operations.
 */
func (c *Config) ValidateForApplication() error {
	if c.APIKey == "" {
		return fmt.Errorf("API_KEY is required.\nThis is the x-seven-api-key used by the CloudFront front door (WAF).\nUse the same value as the mobile app's .env (API_KEY).\nThen run: export API_KEY=\"your-api-key\" && seven-test-tui")
	}
	return nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
