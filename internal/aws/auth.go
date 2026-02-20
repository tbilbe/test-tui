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

func (a *AuthClient) GetIDToken() string {
	return a.idToken
}

func (a *AuthClient) GetAccessToken() string {
	return a.accessToken
}

func (a *AuthClient) IsAuthenticated() bool {
	return a.idToken != ""
}
