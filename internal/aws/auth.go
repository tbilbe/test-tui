package aws

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/zalando/go-keyring"
)

const (
	keyringService           = "github.com/angstromsports/seven-test-tui"
	authorizationRedirectURI = "seven://auth"
	authorizationClientID    = "seven-test-tui"
)

var (
	ErrRefreshTokenNotFound = errors.New("no stored session — run seven-test-tui bootstrap")
	ErrSessionRevoked       = errors.New("session revoked — re-run bootstrap")
	ErrNoCurrentGameWeek    = errors.New("no current game week found — seed an active game week and retry")
)

type HTTPClient interface {
	Do(*http.Request) (*http.Response, error)
}

type TokenStore interface {
	Load() (string, error)
	Save(string) error
}

type keyringClient interface {
	Get(service, user string) (string, error)
	Set(service, user, password string) error
}

type systemKeyring struct{}

func (systemKeyring) Get(service, user string) (string, error) {
	return keyring.Get(service, user)
}

func (systemKeyring) Set(service, user, password string) error {
	return keyring.Set(service, user, password)
}

type refreshTokenStore struct {
	keyring keyringClient
	account string
	file    string
}

// NewRefreshTokenStore stores refresh tokens in the operating-system keyring,
// with a private config file only where a keyring is unavailable.
func NewRefreshTokenStore(apiEndpoint string) (TokenStore, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return nil, fmt.Errorf("find user config directory: %w", err)
	}

	endpointHash := sha256.Sum256([]byte(apiEndpoint))
	endpointID := hex.EncodeToString(endpointHash[:8])
	return newRefreshTokenStore(
		systemKeyring{},
		"refresh-token-"+endpointID,
		filepath.Join(configDir, "seven-test-tui", endpointID+".refresh-token"),
	), nil
}

func newRefreshTokenStore(keyring keyringClient, account, file string) *refreshTokenStore {
	return &refreshTokenStore{keyring: keyring, account: account, file: file}
}

func (s *refreshTokenStore) Load() (string, error) {
	token, keyringErr := s.keyring.Get(keyringService, s.account)
	if keyringErr == nil && token != "" {
		return token, nil
	}

	fileToken, fileErr := os.ReadFile(s.file)
	if fileErr == nil && len(fileToken) > 0 {
		return string(fileToken), nil
	}
	if errors.Is(fileErr, os.ErrNotExist) {
		return "", ErrRefreshTokenNotFound
	}
	if fileErr != nil && !errors.Is(fileErr, os.ErrNotExist) {
		return "", fmt.Errorf("read refresh-token fallback: %w", fileErr)
	}
	if keyringErr != nil && !errors.Is(keyringErr, keyring.ErrNotFound) {
		return "", fmt.Errorf("read refresh token from keyring: %w", keyringErr)
	}
	return "", ErrRefreshTokenNotFound
}

func (s *refreshTokenStore) Save(token string) error {
	if err := s.keyring.Set(keyringService, s.account, token); err == nil {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(s.file), 0700); err != nil {
		return fmt.Errorf("create refresh-token fallback directory: %w", err)
	}
	if err := os.WriteFile(s.file, []byte(token), 0600); err != nil {
		return fmt.Errorf("write refresh-token fallback: %w", err)
	}
	if err := os.Chmod(s.file, 0600); err != nil {
		return fmt.Errorf("restrict refresh-token fallback permissions: %w", err)
	}
	return nil
}

type TokenClient struct {
	apiEndpoint        string
	deviceID           string
	authorizationState string
	httpClient         HTTPClient
	store              TokenStore
}

type tokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

type tokenErrorResponse struct {
	Error            string `json:"error"`
	ErrorDescription string `json:"error_description"`
	ErrorMessage     string `json:"errorMessage"`
}

// NewTokenClient creates an auth-server client for one dedicated TUI device.
func NewTokenClient(apiEndpoint, deviceID string, store TokenStore) *TokenClient {
	return &TokenClient{
		apiEndpoint: strings.TrimRight(apiEndpoint, "/"),
		deviceID:    deviceID,
		httpClient:  http.DefaultClient,
		store:       store,
	}
}

// DeviceID returns the stable identifier reserved for this TUI, never a mobile-device identifier.
func DeviceID() string {
	hostname, err := os.Hostname()
	if err != nil || hostname == "" {
		hostname = "unknown-host"
	}
	return "seven-test-tui-" + strings.Map(func(character rune) rune {
		if (character >= 'a' && character <= 'z') || (character >= 'A' && character <= 'Z') || (character >= '0' && character <= '9') || character == '-' || character == '_' || character == '.' {
			return character
		}
		return '-'
	}, hostname)
}

// AuthorizationURL creates the auth-server entry point for browser bootstrap.
func (c *TokenClient) AuthorizationURL() (string, error) {
	state, err := newAuthorizationState()
	if err != nil {
		return "", fmt.Errorf("generate authorization state: %w", err)
	}
	c.authorizationState = state

	authorizeURL, err := url.Parse(c.apiEndpoint + "/oauth2/authorize")
	if err != nil {
		return "", fmt.Errorf("parse authorization endpoint: %w", err)
	}
	query := authorizeURL.Query()
	query.Set("state", state)
	query.Set("device_id", c.deviceID)
	query.Set("redirect_uri", authorizationRedirectURI)
	query.Set("client_id", authorizationClientID)
	authorizeURL.RawQuery = query.Encode()
	return authorizeURL.String(), nil
}

func newAuthorizationState() (string, error) {
	value := make([]byte, 32)
	if _, err := rand.Read(value); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(value), nil
}

// ExchangeAuthorizationCode validates and exchanges a browser authorization response.
func (c *TokenClient) ExchangeAuthorizationCode(ctx context.Context, authorizationResponse string) (string, error) {
	code, state := extractAuthorizationResponse(authorizationResponse)
	if strings.TrimSpace(code) == "" {
		return "", fmt.Errorf("authorization code is required")
	}
	if c.authorizationState == "" {
		return "", fmt.Errorf("authorization state not initialized — open the bootstrap URL first")
	}
	if subtle.ConstantTimeCompare([]byte(state), []byte(c.authorizationState)) != 1 {
		return "", fmt.Errorf("authorization state mismatch")
	}
	return c.requestAndPersist(ctx, url.Values{
		"grant_type": {"authorization_code"},
		"code":       {code},
	}, true)
}

// Refresh persists the one-time rotated refresh token before returning its access token.
func (c *TokenClient) Refresh(ctx context.Context) (string, error) {
	if c == nil || c.store == nil {
		return "", ErrRefreshTokenNotFound
	}
	refreshToken, err := c.store.Load()
	if err != nil {
		return "", err
	}
	return c.requestAndPersist(ctx, url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {refreshToken},
	}, false)
}

func (c *TokenClient) requestAndPersist(ctx context.Context, values url.Values, includeDeviceID bool) (string, error) {
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		c.apiEndpoint+"/oauth2/token",
		strings.NewReader(values.Encode()),
	)
	if err != nil {
		return "", fmt.Errorf("create token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if includeDeviceID {
		req.Header.Set("x-device-id", c.deviceID)
	}

	response, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("request token: %w", err)
	}
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return "", fmt.Errorf("read token response: %w", err)
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return "", tokenRequestError(response.StatusCode, body)
	}

	var token tokenResponse
	if err := json.Unmarshal(body, &token); err != nil {
		return "", fmt.Errorf("decode token response: %w", err)
	}
	if token.AccessToken == "" || token.RefreshToken == "" {
		return "", fmt.Errorf("token response did not contain access_token and refresh_token")
	}
	if err := c.store.Save(token.RefreshToken); err != nil {
		return "", fmt.Errorf("persist rotated refresh token: %w", err)
	}
	return token.AccessToken, nil
}

func tokenRequestError(statusCode int, body []byte) error {
	var response tokenErrorResponse
	_ = json.Unmarshal(body, &response)
	if response.Error == "invalid_grant" {
		return ErrSessionRevoked
	}
	message := response.ErrorDescription
	if message == "" {
		message = response.ErrorMessage
	}
	if statusCode == http.StatusServiceUnavailable && strings.Contains(strings.ToLower(message), "no current game week found") {
		return ErrNoCurrentGameWeek
	}
	if message == "" {
		message = strings.TrimSpace(string(body))
	}
	return fmt.Errorf("token request failed with status %d: %s", statusCode, message)
}

// ExtractAuthorizationCode accepts either a deep-link URL or its code parameter.
func ExtractAuthorizationCode(input string) string {
	code, _ := extractAuthorizationResponse(input)
	return code
}

func extractAuthorizationResponse(input string) (string, string) {
	value := strings.TrimSpace(input)
	parsed, err := url.Parse(value)
	if err != nil {
		return value, ""
	}
	if code := parsed.Query().Get("code"); code != "" {
		return code, parsed.Query().Get("state")
	}
	return value, ""
}
