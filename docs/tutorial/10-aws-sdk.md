# Chapter 10: AWS SDK Integration

## What You'll Learn

- How to use AWS SDK v2 in Go
- Cognito authentication
- DynamoDB operations
- Error handling with AWS services

---

## AWS SDK v2 Overview

AWS SDK for Go v2 is the current version. Key differences from v1:
- Context-first API design
- Modular packages (import only what you need)
- Better error handling
- Configuration via `config.LoadDefaultConfig`

---

## Setting Up AWS Clients

### The Pattern

```go
import (
    "context"
    "github.com/aws/aws-sdk-go-v2/config"
    "github.com/aws/aws-sdk-go-v2/service/dynamodb"
)

func NewDynamoDBClient(ctx context.Context, region string) (*dynamodb.Client, error) {
    cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(region))
    if err != nil {
        return nil, fmt.Errorf("failed to load AWS config: %w", err)
    }
    
    return dynamodb.NewFromConfig(cfg), nil
}
```

### What `LoadDefaultConfig` Does

It loads credentials from (in order):
1. Environment variables (`AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`)
2. Shared credentials file (`~/.aws/credentials`)
3. IAM role (if running on EC2/Lambda)
4. SSO credentials (if configured)

You don't need to handle credentials manually.

---

## The Auth Client

Create `internal/aws/auth.go`:

```go
package aws

import (
    "context"
    "fmt"

    "github.com/aws/aws-sdk-go-v2/aws"
    "github.com/aws/aws-sdk-go-v2/config"
    "github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider"
    "github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider/types"
)

type AuthClient struct {
    client       *cognitoidentityprovider.Client
    userPoolID   string
    clientID     string
    accessToken  string
    idToken      string
    refreshToken string
}

func NewAuthClient(ctx context.Context, region, userPoolID, clientID string) (*AuthClient, error) {
    cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(region))
    if err != nil {
        return nil, fmt.Errorf("failed to load AWS config: %w", err)
    }

    return &AuthClient{
        client:     cognitoidentityprovider.NewFromConfig(cfg),
        userPoolID: userPoolID,
        clientID:   clientID,
    }, nil
}
```

### The SignIn Method

```go
func (a *AuthClient) SignIn(ctx context.Context, username, password string) error {
    input := &cognitoidentityprovider.InitiateAuthInput{
        AuthFlow: types.AuthFlowTypeUserPasswordAuth,
        ClientId: aws.String(a.clientID),
        AuthParameters: map[string]string{
            "USERNAME": username,
            "PASSWORD": password,
        },
    }

    result, err := a.client.InitiateAuth(ctx, input)
    if err != nil {
        return fmt.Errorf("authentication failed: %w", err)
    }

    if result.AuthenticationResult != nil {
        a.accessToken = aws.ToString(result.AuthenticationResult.AccessToken)
        a.idToken = aws.ToString(result.AuthenticationResult.IdToken)
        a.refreshToken = aws.ToString(result.AuthenticationResult.RefreshToken)
    }

    return nil
}
```

### Key Points

**`aws.String()` and `aws.ToString()`**:
```go
aws.String("value")  // Returns *string (pointer to "value")
aws.ToString(ptr)    // Returns string (dereferences safely, returns "" if nil)
```

AWS SDK uses pointers for optional fields. These helpers make it easier.

**Auth Flow Types**:
```go
types.AuthFlowTypeUserPasswordAuth  // Username + password
types.AuthFlowTypeSrpAuth           // Secure Remote Password (more secure)
types.AuthFlowTypeRefreshToken      // Refresh an expired token
```

**Storing Tokens**:
```go
a.idToken = aws.ToString(result.AuthenticationResult.IdToken)
```

The ID token is used for API authentication. Store it for later use.

---

## Token Accessor Methods

```go
func (a *AuthClient) GetIDToken() string {
    return a.idToken
}

func (a *AuthClient) GetAccessToken() string {
    return a.accessToken
}

func (a *AuthClient) IsAuthenticated() bool {
    return a.idToken != ""
}
```

These provide controlled access to the tokens.

---

## The DynamoDB Client

Create `internal/aws/dynamodb.go`:

```go
package aws

import (
    "context"
    "fmt"

    "github.com/aws/aws-sdk-go-v2/aws"
    "github.com/aws/aws-sdk-go-v2/config"
    "github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
    "github.com/aws/aws-sdk-go-v2/service/dynamodb"
    "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

type DynamoDBClient struct {
    client    *dynamodb.Client
    tableName string
}

func NewDynamoDBClient(ctx context.Context, region, tableName string) (*DynamoDBClient, error) {
    cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(region))
    if err != nil {
        return nil, fmt.Errorf("failed to load AWS config: %w", err)
    }

    return &DynamoDBClient{
        client:    dynamodb.NewFromConfig(cfg),
        tableName: tableName,
    }, nil
}
```

### Query Operation

```go
func (d *DynamoDBClient) QueryFixturesByGameWeek(ctx context.Context, gameWeekID string) ([]map[string]types.AttributeValue, error) {
    input := &dynamodb.QueryInput{
        TableName:              aws.String(d.tableName),
        KeyConditionExpression: aws.String("gameWeekId = :gw"),
        ExpressionAttributeValues: map[string]types.AttributeValue{
            ":gw": &types.AttributeValueMemberS{Value: gameWeekID},
        },
    }

    result, err := d.client.Query(ctx, input)
    if err != nil {
        return nil, fmt.Errorf("failed to query fixtures: %w", err)
    }

    return result.Items, nil
}
```

### Understanding DynamoDB Types

DynamoDB has its own type system:

```go
// String
&types.AttributeValueMemberS{Value: "hello"}

// Number (stored as string)
&types.AttributeValueMemberN{Value: "42"}

// Boolean
&types.AttributeValueMemberBOOL{Value: true}

// List
&types.AttributeValueMemberL{Value: []types.AttributeValue{...}}

// Map
&types.AttributeValueMemberM{Value: map[string]types.AttributeValue{...}}
```

### Put Operation

```go
func (d *DynamoDBClient) PutFixture(ctx context.Context, fixture map[string]types.AttributeValue) error {
    input := &dynamodb.PutItemInput{
        TableName: aws.String(d.tableName),
        Item:      fixture,
    }

    _, err := d.client.PutItem(ctx, input)
    if err != nil {
        return fmt.Errorf("failed to put fixture: %w", err)
    }

    return nil
}
```

### Using attributevalue for Marshalling

Instead of manually building `map[string]types.AttributeValue`, use the helper:

```go
import "github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"

func (d *DynamoDBClient) UpdateFixture(ctx context.Context, item interface{}) error {
    // Convert Go struct to DynamoDB attribute values
    av, err := attributevalue.MarshalMap(item)
    if err != nil {
        return fmt.Errorf("failed to marshal fixture: %w", err)
    }

    return d.PutFixture(ctx, av)
}
```

This uses the `dynamodbav` struct tags:

```go
type Fixture struct {
    FixtureID string `dynamodbav:"fixtureId"`
    Period    string `dynamodbav:"period"`
}

// MarshalMap produces:
// map[string]types.AttributeValue{
//     "fixtureId": &types.AttributeValueMemberS{Value: "123"},
//     "period":    &types.AttributeValueMemberS{Value: "FIRST_HALF"},
// }
```

---

## Error Handling

### AWS Errors

```go
import "github.com/aws/smithy-go"

result, err := client.Query(ctx, input)
if err != nil {
    var apiErr smithy.APIError
    if errors.As(err, &apiErr) {
        fmt.Printf("AWS error: %s - %s\n", apiErr.ErrorCode(), apiErr.ErrorMessage())
    }
    return nil, err
}
```

### Common Cognito Errors

```go
import "github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider/types"

_, err := client.InitiateAuth(ctx, input)
if err != nil {
    var notAuth *types.NotAuthorizedException
    if errors.As(err, &notAuth) {
        return fmt.Errorf("invalid username or password")
    }
    
    var userNotFound *types.UserNotFoundException
    if errors.As(err, &userNotFound) {
        return fmt.Errorf("user does not exist")
    }
    
    return fmt.Errorf("authentication failed: %w", err)
}
```

### Common DynamoDB Errors

```go
import "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"

_, err := client.PutItem(ctx, input)
if err != nil {
    var condFailed *types.ConditionalCheckFailedException
    if errors.As(err, &condFailed) {
        return fmt.Errorf("item already exists or condition not met")
    }
    
    var throughput *types.ProvisionedThroughputExceededException
    if errors.As(err, &throughput) {
        return fmt.Errorf("rate limit exceeded, try again later")
    }
    
    return fmt.Errorf("database error: %w", err)
}
```

---

## Using Clients in Commands

```go
func authenticateCmd(client *aws.AuthClient, username, password string) tea.Cmd {
    return func() tea.Msg {
        ctx := context.Background()
        err := client.SignIn(ctx, username, password)
        if err != nil {
            return authErrorMsg{err: err}
        }
        return authSuccessMsg{token: client.GetIDToken()}
    }
}

func updateFixtureCmd(fixture *models.Fixture, prefix string) tea.Cmd {
    return func() tea.Msg {
        ctx := context.Background()
        
        tableName := prefix + "-GameWeekFixtures"
        client, err := aws.NewDynamoDBClient(ctx, "eu-west-2", tableName)
        if err != nil {
            return fixtureUpdatedMsg{err: err}
        }
        
        err = client.UpdateFixture(ctx, fixture)
        return fixtureUpdatedMsg{err: err}
    }
}
```

---

## Best Practices

### 1. Create Clients Once

```go
// Good - create in main, pass around
func main() {
    client, _ := aws.NewDynamoDBClient(ctx, region, table)
    ui.NewModel(client)
}

// Bad - create in every command
func fetchDataCmd() tea.Cmd {
    return func() tea.Msg {
        client, _ := aws.NewDynamoDBClient(...)  // Wasteful
    }
}
```

### 2. Use Context for Timeouts

```go
func (d *DynamoDBClient) Query(ctx context.Context, ...) error {
    ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
    defer cancel()
    
    result, err := d.client.Query(ctx, input)
    // ...
}
```

### 3. Wrap Errors with Context

```go
if err != nil {
    return nil, fmt.Errorf("failed to query fixtures for gameweek %s: %w", gameWeekID, err)
}
```

### 4. Don't Expose SDK Types

```go
// Bad - leaks SDK types
func (d *DynamoDBClient) Query() ([]map[string]types.AttributeValue, error)

// Good - returns domain types
func (d *DynamoDBClient) GetFixtures(gameWeekID string) ([]models.Fixture, error)
```

---

## Key Takeaways

1. **`config.LoadDefaultConfig`** - Handles credential loading automatically
2. **`aws.String()` / `aws.ToString()`** - Helpers for pointer conversion
3. **`attributevalue.MarshalMap`** - Convert structs to DynamoDB format
4. **Struct tags** - `dynamodbav:"fieldName"` controls marshalling
5. **Error handling** - Use `errors.As` for specific AWS errors
6. **Create clients once** - Pass them as dependencies

---

## Exercise

1. Add a `RefreshToken` method to `AuthClient` that uses the refresh token to get new access/ID tokens
2. Add a `GetItem` method to `DynamoDBClient` that fetches a single fixture by ID
3. Implement retry logic for `ProvisionedThroughputExceededException`
4. Add logging to track how long AWS operations take

---

[← Previous: Multi-Screen Navigation](./09-navigation.md) | [Next: Chapter 11 - API Client Patterns →](./11-api-client.md)
