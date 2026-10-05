package config

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
)

const credentialTimeout = time.Second

type Config struct {
	APIEndpoint string
	APIKey      string
	AccessToken string
	Prefix      string
	AWSRegion   string
	AllowWrites bool
}

// CredentialsProvider retrieves AWS credentials from a loaded AWS configuration.
type CredentialsProvider interface {
	Retrieve(context.Context) (awssdk.Credentials, error)
}

type PreflightProblem struct {
	Variable string
	What     string
	Source   string
	Export   string
}

type Preflight struct {
	APIEndpoint string
	AWSRegion   string
	Errors      []PreflightProblem
	Warnings    []PreflightProblem
}

// Load reads environment configuration for the interactive TUI.
func Load() (*Config, error) {
	return &Config{
		APIEndpoint: getEnv("API_ENDPOINT", "https://dev.cf.playtheseven.com"),
		APIKey:      getEnv("API_KEY", ""),
		AccessToken: getEnv("SEVEN_ACCESS_TOKEN", ""),
		Prefix:      getEnv("PREFIX", ""),
		AWSRegion:   getEnv("AWS_REGION", "eu-west-2"),
		AllowWrites: os.Getenv("SEVEN_TUI_ALLOW_WRITES") == "true",
	}, nil
}

// Preflight reports configuration errors and write capability warnings together.
func (c *Config) Preflight(provider CredentialsProvider) Preflight {
	result := Preflight{APIEndpoint: c.APIEndpoint, AWSRegion: c.AWSRegion}
	if c.APIKey == "" {
		result.Errors = append(result.Errors, PreflightProblem{
			Variable: "API_KEY",
			What:     "The x-seven-api-key for the CloudFront/WAF front door.",
			Source:   "Get API_KEY from the mobile app repository's .env file.",
			Export:   "export API_KEY=\"your-api-key\"",
		})
	}
	if c.AccessToken == "" {
		result.Errors = append(result.Errors, PreflightProblem{
			Variable: "SEVEN_ACCESS_TOKEN",
			What:     "Seven's KMS-signed JWT, not a bwin/Entain IdP token. It is short-lived.",
			Source:   "Run the mobile app in a simulator, sign in, open dev tools, find a /game-weeks request, and copy Authorization without the Bearer prefix. Re-copy it when it expires.",
			Export:   "export SEVEN_ACCESS_TOKEN=\"your-seven-access-token\"",
		})
	}
	if !c.AllowWrites {
		result.Warnings = append(result.Warnings, PreflightProblem{
			Variable: "SEVEN_TUI_ALLOW_WRITES",
			What:     "The TUI is read-only and all writes will be refused.",
			Source:   "Writes also require a prefix that canonicalises to SE7-* or is exactly int-dev.",
			Export:   "export SEVEN_TUI_ALLOW_WRITES=true",
		})
	}
	if !credentialsResolvable(provider) {
		result.Warnings = append(result.Warnings, PreflightProblem{
			Variable: "AWS_PROFILE",
			What:     "AWS credentials are not resolvable, so direct DynamoDB writes will fail.",
			Source:   "Sign in with the Seven development SSO profile before enabling writes.",
			Export:   "aws sso login --profile seven_engineer_seven_dev-339713102567 && export AWS_PROFILE=seven_engineer_seven_dev-339713102567",
		})
	}
	return result
}

// HasErrors reports whether the TUI must not start.
func (p Preflight) HasErrors() bool {
	return len(p.Errors) > 0
}

// String renders the startup configuration report.
func (p Preflight) String() string {
	var builder strings.Builder
	fmt.Fprintln(&builder, "Startup preflight")
	fmt.Fprintf(&builder, "API_ENDPOINT: %s\n", p.APIEndpoint)
	fmt.Fprintf(&builder, "AWS_REGION: %s\n", p.AWSRegion)
	writeProblems(&builder, "Errors", p.Errors)
	writeProblems(&builder, "Warnings", p.Warnings)
	return builder.String()
}

func credentialsResolvable(provider CredentialsProvider) bool {
	if provider == nil {
		return false
	}
	ctx, cancel := context.WithTimeout(context.Background(), credentialTimeout)
	defer cancel()
	result := make(chan error, 1)
	go func() {
		_, err := provider.Retrieve(ctx)
		result <- err
	}()
	select {
	case err := <-result:
		return err == nil
	case <-ctx.Done():
		return false
	}
}

func writeProblems(builder *strings.Builder, heading string, problems []PreflightProblem) {
	if len(problems) == 0 {
		return
	}
	fmt.Fprintf(builder, "%s:\n", heading)
	for _, problem := range problems {
		fmt.Fprintf(builder, "- %s: %s\n  Get it: %s\n  Run: %s\n", problem.Variable, problem.What, problem.Source, problem.Export)
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
