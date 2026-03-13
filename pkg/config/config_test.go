package config

import (
	"os"
	"testing"
)

func TestLoad_AllEnvVarsSet(t *testing.T) {
	os.Setenv("API_ENDPOINT", "https://test.api.example.com")
	os.Setenv("USER_POOL_ID", "eu-west-2_test123")
	os.Setenv("CLIENT_ID", "test-client-id")
	os.Setenv("PREFIX", "test-prefix")
	os.Setenv("AWS_REGION", "us-east-1")
	defer func() {
		os.Unsetenv("API_ENDPOINT")
		os.Unsetenv("USER_POOL_ID")
		os.Unsetenv("CLIENT_ID")
		os.Unsetenv("PREFIX")
		os.Unsetenv("AWS_REGION")
	}()

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.APIEndpoint != "https://test.api.example.com" {
		t.Errorf("APIEndpoint = %q, want %q", cfg.APIEndpoint, "https://test.api.example.com")
	}
	if cfg.UserPoolID != "eu-west-2_test123" {
		t.Errorf("UserPoolID = %q, want %q", cfg.UserPoolID, "eu-west-2_test123")
	}
	if cfg.ClientID != "test-client-id" {
		t.Errorf("ClientID = %q, want %q", cfg.ClientID, "test-client-id")
	}
	if cfg.Prefix != "test-prefix" {
		t.Errorf("Prefix = %q, want %q", cfg.Prefix, "test-prefix")
	}
	if cfg.AWSRegion != "us-east-1" {
		t.Errorf("AWSRegion = %q, want %q", cfg.AWSRegion, "us-east-1")
	}
}

func TestLoad_DefaultValues(t *testing.T) {
	// Only CLIENT_ID is required, others have defaults
	os.Setenv("CLIENT_ID", "test-client-id")
	os.Unsetenv("API_ENDPOINT")
	os.Unsetenv("USER_POOL_ID")
	os.Unsetenv("PREFIX")
	os.Unsetenv("AWS_REGION")
	defer os.Unsetenv("CLIENT_ID")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.APIEndpoint != "https://dev.api.playtheseven.com" {
		t.Errorf("APIEndpoint default = %q, want %q", cfg.APIEndpoint, "https://dev.api.playtheseven.com")
	}
	if cfg.UserPoolID != "eu-west-2_uqwEOLO5d" {
		t.Errorf("UserPoolID default = %q, want %q", cfg.UserPoolID, "eu-west-2_uqwEOLO5d")
	}
	if cfg.Prefix != "" {
		t.Errorf("Prefix default = %q, want empty", cfg.Prefix)
	}
	if cfg.AWSRegion != "eu-west-2" {
		t.Errorf("AWSRegion default = %q, want %q", cfg.AWSRegion, "eu-west-2")
	}
}

func TestLoad_MissingClientID(t *testing.T) {
	os.Unsetenv("CLIENT_ID")

	_, err := Load()
	if err == nil {
		t.Error("Load() should return error when CLIENT_ID is missing")
	}
}

func TestGetEnv_WithValue(t *testing.T) {
	os.Setenv("TEST_VAR", "test-value")
	defer os.Unsetenv("TEST_VAR")

	result := getEnv("TEST_VAR", "default")
	if result != "test-value" {
		t.Errorf("getEnv() = %q, want %q", result, "test-value")
	}
}

func TestGetEnv_WithDefault(t *testing.T) {
	os.Unsetenv("TEST_VAR_MISSING")

	result := getEnv("TEST_VAR_MISSING", "default-value")
	if result != "default-value" {
		t.Errorf("getEnv() = %q, want %q", result, "default-value")
	}
}

func TestGetEnv_EmptyValue(t *testing.T) {
	os.Setenv("TEST_VAR_EMPTY", "")
	defer os.Unsetenv("TEST_VAR_EMPTY")

	result := getEnv("TEST_VAR_EMPTY", "default")
	if result != "default" {
		t.Errorf("getEnv() with empty value = %q, want %q (default)", result, "default")
	}
}
