# Chapter 4: Configuration Management

## What You'll Learn

- How to handle application configuration
- Environment variables vs config files
- Validation at load time
- The options pattern

---

## Why Configuration Matters

Your app needs to know:
- Which API endpoint to call
- Which AWS region to use
- Which environment (dev, staging, prod)

Hardcoding these is bad:

```go
// DON'T DO THIS
apiClient := NewAPIClient("https://dev.api.playtheseven.com")  // What about prod?
```

---

## Our Configuration Package

Create `pkg/config/config.go`:

```go
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

    // Derived values
    cfg.FixtureTable = fmt.Sprintf("%s-GameWeekFixtures", cfg.Prefix)

    // Validation
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
```

---

## Breaking It Down

### The Config Struct

```go
type Config struct {
    APIEndpoint  string
    UserPoolID   string
    ClientID     string
    Prefix       string
    AWSRegion    string
    FixtureTable string
}
```

All configuration in one place. Benefits:
- Easy to see what the app needs
- Type-safe (can't misspell field names)
- IDE autocomplete works

### The Helper Function

```go
func getEnv(key, defaultValue string) string {
    if value := os.Getenv(key); value != "" {
        return value
    }
    return defaultValue
}
```

This pattern appears in almost every Go application. It:
- Gets an environment variable
- Returns a default if not set
- Keeps the main code clean

### Loading with Defaults

```go
cfg := &Config{
    APIEndpoint: getEnv("API_ENDPOINT", ""),      // Required, no default
    Prefix:      getEnv("PREFIX", "int-dev"),     // Optional, has default
    AWSRegion:   getEnv("AWS_REGION", "eu-west-2"), // Optional, has default
}
```

Notice:
- Required fields have empty string default (we validate later)
- Optional fields have sensible defaults

### Derived Values

```go
cfg.FixtureTable = fmt.Sprintf("%s-GameWeekFixtures", cfg.Prefix)
```

Some values are computed from others. Do this in `Load()` so callers don't need to know the formula.

### Validation

```go
if cfg.APIEndpoint == "" {
    return nil, fmt.Errorf("API_ENDPOINT is required")
}
```

Validate early. If config is invalid, the app shouldn't start.

---

## Using the Config

In `main.go`:

```go
cfg, err := config.Load()
if err != nil {
    fmt.Printf("Configuration error: %v\n", err)
    os.Exit(1)
}

// Use config values
authClient, err := aws.NewAuthClient(ctx, cfg.AWSRegion, cfg.UserPoolID, cfg.ClientID)
apiClient := aws.NewAPIClient(cfg.APIEndpoint)
```

---

## Environment Variables vs Config Files

### Environment Variables (What We Use)

```bash
export API_ENDPOINT="https://dev.api.playtheseven.com"
export USER_POOL_ID="eu-west-2_abc123"
./seven-test-tui
```

Pros:
- Standard in cloud environments (Docker, Kubernetes, Lambda)
- Easy to change without rebuilding
- Secrets don't end up in git

Cons:
- Can't see all config in one place
- Easy to forget to set one

### Config Files

```yaml
# config.yaml
api_endpoint: https://dev.api.playtheseven.com
user_pool_id: eu-west-2_abc123
```

Pros:
- All config visible in one file
- Can commit non-secret config to git
- Supports complex nested structures

Cons:
- Need to manage file paths
- Secrets in files are risky

### The Hybrid Approach

Many apps do both:

```go
func Load() (*Config, error) {
    // 1. Load defaults from file
    cfg := loadFromFile("config.yaml")
    
    // 2. Override with environment variables
    if endpoint := os.Getenv("API_ENDPOINT"); endpoint != "" {
        cfg.APIEndpoint = endpoint
    }
    
    return cfg, nil
}
```

Environment variables override file values. This gives you:
- Defaults in a file (committed to git)
- Secrets in environment (not in git)
- Easy overrides for different environments

---

## The .env File Pattern

For local development, we use a `.env` file:

```bash
# .env
API_ENDPOINT="https://se7-int-dev.dev.api.playtheseven.com"
USER_POOL_ID="eu-west-2_uqwEOLO5d"
CLIENT_ID="your-client-id"
PREFIX="int-dev"
```

Load it before running:

```bash
export $(cat .env | grep -v '^#' | xargs)
./seven-test-tui
```

Or use the `run.sh` script that does this automatically.

**Important**: Add `.env` to `.gitignore`. Never commit secrets.

---

## Validation Patterns

### Required vs Optional

```go
// Required: empty default, validate
APIEndpoint: getEnv("API_ENDPOINT", ""),
// ...
if cfg.APIEndpoint == "" {
    return nil, fmt.Errorf("API_ENDPOINT is required")
}

// Optional: sensible default, no validation needed
AWSRegion: getEnv("AWS_REGION", "eu-west-2"),
```

### Format Validation

```go
func Load() (*Config, error) {
    cfg := &Config{
        APIEndpoint: getEnv("API_ENDPOINT", ""),
    }
    
    // Validate URL format
    if _, err := url.Parse(cfg.APIEndpoint); err != nil {
        return nil, fmt.Errorf("API_ENDPOINT is not a valid URL: %w", err)
    }
    
    return cfg, nil
}
```

### Range Validation

```go
timeout := getEnvInt("TIMEOUT_SECONDS", 30)
if timeout < 1 || timeout > 300 {
    return nil, fmt.Errorf("TIMEOUT_SECONDS must be between 1 and 300")
}
```

---

## Helper Functions for Types

Environment variables are always strings. For other types:

```go
func getEnvInt(key string, defaultValue int) int {
    if value := os.Getenv(key); value != "" {
        if intVal, err := strconv.Atoi(value); err == nil {
            return intVal
        }
    }
    return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
    if value := os.Getenv(key); value != "" {
        if boolVal, err := strconv.ParseBool(value); err == nil {
            return boolVal
        }
    }
    return defaultValue
}

func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
    if value := os.Getenv(key); value != "" {
        if duration, err := time.ParseDuration(value); err == nil {
            return duration
        }
    }
    return defaultValue
}
```

Usage:

```go
cfg := &Config{
    Timeout:     getEnvDuration("TIMEOUT", 30*time.Second),
    MaxRetries:  getEnvInt("MAX_RETRIES", 3),
    DebugMode:   getEnvBool("DEBUG", false),
}
```

---

## Key Takeaways

1. **Struct for configuration** - Groups related values, provides type safety
2. **`getEnv` helper** - Standard pattern for environment variables with defaults
3. **Validate at load time** - Fail fast if config is invalid
4. **Derived values in Load()** - Compute once, use everywhere
5. **Environment variables for secrets** - Never commit secrets to git
6. **`.env` for local development** - Convenient but gitignored

---

## Exercise

1. Add a `DEBUG` boolean config option with default `false`
2. Add validation that `PREFIX` can only contain alphanumeric characters and hyphens
3. Create a `getEnvInt` helper and add a `TIMEOUT_SECONDS` config option
4. What happens if someone sets `AWS_REGION=""` (empty string)? Is that different from not setting it?

---

[← Previous: The Entry Point Pattern](./03-entry-point.md) | [Next: Chapter 5 - Introduction to Bubbletea →](./05-bubbletea-intro.md)
