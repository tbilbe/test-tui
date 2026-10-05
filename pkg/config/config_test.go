package config

import (
	"context"
	"errors"
	"strings"
	"testing"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
)

type fakeCredentialsProvider struct {
	err error
}

func (p fakeCredentialsProvider) Retrieve(context.Context) (awssdk.Credentials, error) {
	if p.err != nil {
		return awssdk.Credentials{}, p.err
	}
	return awssdk.Credentials{AccessKeyID: "access-key"}, nil
}

func TestLoad_AllEnvVarsSet(t *testing.T) {
	t.Setenv("API_ENDPOINT", "https://test.api.example.com")
	t.Setenv("API_KEY", "test-key-123")
	t.Setenv("SEVEN_ACCESS_TOKEN", "manual-seven-jwt")
	t.Setenv("PREFIX", "test-prefix")
	t.Setenv("AWS_REGION", "us-east-1")
	t.Setenv("SEVEN_TUI_ALLOW_WRITES", "true")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.APIEndpoint != "https://test.api.example.com" || cfg.APIKey != "test-key-123" || cfg.AccessToken != "manual-seven-jwt" || cfg.Prefix != "test-prefix" || cfg.AWSRegion != "us-east-1" || !cfg.AllowWrites {
		t.Errorf("Load() = %#v, want all configured values", cfg)
	}
}

func TestLoad_DefaultValues(t *testing.T) {
	t.Setenv("API_ENDPOINT", "")
	t.Setenv("API_KEY", "")
	t.Setenv("SEVEN_ACCESS_TOKEN", "")
	t.Setenv("PREFIX", "")
	t.Setenv("AWS_REGION", "")
	t.Setenv("SEVEN_TUI_ALLOW_WRITES", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.APIEndpoint != "https://dev.cf.playtheseven.com" || cfg.AWSRegion != "eu-west-2" || cfg.AllowWrites {
		t.Errorf("Load() = %#v, want defaults", cfg)
	}
}

func TestPreflight_EveryRequiredVariableMissing(t *testing.T) {
	result := (&Config{}).Preflight(fakeCredentialsProvider{err: errors.New("unavailable")})
	if len(result.Errors) != 2 {
		t.Fatalf("error count = %d, want 2", len(result.Errors))
	}
	if result.Errors[0].Variable != "API_KEY" || result.Errors[1].Variable != "SEVEN_ACCESS_TOKEN" {
		t.Errorf("errors = %#v, want API_KEY and SEVEN_ACCESS_TOKEN", result.Errors)
	}
	if len(result.Warnings) != 2 || !result.HasErrors() {
		t.Errorf("warnings = %#v, HasErrors() = %t", result.Warnings, result.HasErrors())
	}
}

func TestPreflight_OnlyAccessTokenMissing(t *testing.T) {
	result := (&Config{APIKey: "api-key", AllowWrites: true}).Preflight(fakeCredentialsProvider{})
	if len(result.Errors) != 1 || result.Errors[0].Variable != "SEVEN_ACCESS_TOKEN" {
		t.Errorf("errors = %#v, want only SEVEN_ACCESS_TOKEN", result.Errors)
	}
	if len(result.Warnings) != 0 {
		t.Errorf("warnings = %#v, want none", result.Warnings)
	}
}

func TestPreflight_AllPresent(t *testing.T) {
	result := (&Config{
		APIEndpoint: "https://se7-test.dev.cf.playtheseven.com",
		APIKey:      "api-key",
		AccessToken: "seven-jwt",
		AWSRegion:   "eu-west-2",
		AllowWrites: true,
	}).Preflight(fakeCredentialsProvider{})
	if result.HasErrors() || len(result.Warnings) != 0 {
		t.Errorf("Preflight() = %#v, want no errors or warnings", result)
	}
	output := result.String()
	for _, want := range []string{"API_ENDPOINT: https://se7-test.dev.cf.playtheseven.com", "AWS_REGION: eu-west-2"} {
		if !strings.Contains(output, want) {
			t.Errorf("preflight output missing %q:\n%s", want, output)
		}
	}
}

func TestPreflight_WarningOnlyCases(t *testing.T) {
	cfg := &Config{APIKey: "api-key", AccessToken: "seven-jwt"}
	result := cfg.Preflight(fakeCredentialsProvider{err: errors.New("unavailable")})
	if result.HasErrors() || len(result.Warnings) != 2 {
		t.Errorf("Preflight() = %#v, want two warnings only", result)
	}
	output := result.String()
	for _, want := range []string{"SEVEN_TUI_ALLOW_WRITES", "AWS_PROFILE", "export SEVEN_TUI_ALLOW_WRITES=true", "aws sso login --profile seven_engineer_seven_dev-339713102567"} {
		if !strings.Contains(output, want) {
			t.Errorf("preflight output missing %q:\n%s", want, output)
		}
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
