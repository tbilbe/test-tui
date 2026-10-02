package config

import "testing"

func TestLoad_AllEnvVarsSet(t *testing.T) {
	t.Setenv("API_ENDPOINT", "https://test.api.example.com")
	t.Setenv("API_KEY", "test-key-123")
	t.Setenv("SEVEN_ACCESS_TOKEN", "manual-seven-jwt")
	t.Setenv("PREFIX", "test-prefix")
	t.Setenv("AWS_REGION", "us-east-1")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.APIEndpoint != "https://test.api.example.com" {
		t.Errorf("APIEndpoint = %q, want %q", cfg.APIEndpoint, "https://test.api.example.com")
	}
	if cfg.APIKey != "test-key-123" {
		t.Errorf("APIKey = %q, want %q", cfg.APIKey, "test-key-123")
	}
	if cfg.AccessToken != "manual-seven-jwt" {
		t.Errorf("AccessToken = %q, want %q", cfg.AccessToken, "manual-seven-jwt")
	}
	if cfg.Prefix != "test-prefix" {
		t.Errorf("Prefix = %q, want %q", cfg.Prefix, "test-prefix")
	}
	if cfg.AWSRegion != "us-east-1" {
		t.Errorf("AWSRegion = %q, want %q", cfg.AWSRegion, "us-east-1")
	}
}

func TestLoad_DefaultValues(t *testing.T) {
	t.Setenv("API_ENDPOINT", "")
	t.Setenv("API_KEY", "")
	t.Setenv("SEVEN_ACCESS_TOKEN", "")
	t.Setenv("PREFIX", "")
	t.Setenv("AWS_REGION", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.APIEndpoint != "https://dev.cf.playtheseven.com" {
		t.Errorf("APIEndpoint default = %q, want %q", cfg.APIEndpoint, "https://dev.cf.playtheseven.com")
	}
	if cfg.AWSRegion != "eu-west-2" {
		t.Errorf("AWSRegion default = %q, want %q", cfg.AWSRegion, "eu-west-2")
	}
}

func TestValidateForApplication(t *testing.T) {
	if err := (&Config{}).ValidateForApplication(); err == nil {
		t.Error("ValidateForApplication() should reject a missing API key")
	}
	if err := (&Config{APIKey: "test-api-key"}).ValidateForApplication(); err != nil {
		t.Fatalf("ValidateForApplication() error = %v", err)
	}
}

func TestGetEnv(t *testing.T) {
	t.Setenv("TEST_VAR", "test-value")
	if value := getEnv("TEST_VAR", "default"); value != "test-value" {
		t.Errorf("getEnv() = %q, want %q", value, "test-value")
	}
	t.Setenv("TEST_VAR", "")
	if value := getEnv("TEST_VAR", "default"); value != "default" {
		t.Errorf("getEnv() = %q, want %q", value, "default")
	}
}
