package aws

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type fakeHTTPClient struct {
	do func(*http.Request) (*http.Response, error)
}

func (c fakeHTTPClient) Do(request *http.Request) (*http.Response, error) {
	return c.do(request)
}

type fakeTokenStore struct {
	token   string
	saved   string
	loadErr error
	saveErr error
}

func (s *fakeTokenStore) Load() (string, error) {
	return s.token, s.loadErr
}

func (s *fakeTokenStore) Save(token string) error {
	s.saved = token
	return s.saveErr
}

type fakeKeyring struct {
	getToken string
	getErr   error
	setErr   error
}

func (k fakeKeyring) Get(string, string) (string, error) {
	return k.getToken, k.getErr
}

func (k fakeKeyring) Set(string, string, string) error {
	return k.setErr
}

func TestExchangeAuthorizationCodeSendsFormAndPersistsRefreshToken(t *testing.T) {
	store := &fakeTokenStore{}
	client := NewTokenClient("https://api.example.com", "seven-test-tui-host", store)
	authorizationURL, err := client.AuthorizationURL()
	if err != nil {
		t.Fatalf("AuthorizationURL() error = %v", err)
	}
	state, err := url.Parse(authorizationURL)
	if err != nil {
		t.Fatalf("parse authorization URL: %v", err)
	}
	client.httpClient = fakeHTTPClient{do: func(request *http.Request) (*http.Response, error) {
		if request.URL.String() != "https://api.example.com/oauth2/token" {
			t.Errorf("request URL = %q", request.URL)
		}
		if got := request.Header.Get("x-device-id"); got != "seven-test-tui-host" {
			t.Errorf("x-device-id = %q", got)
		}
		body, _ := io.ReadAll(request.Body)
		values, _ := url.ParseQuery(string(body))
		if got := values.Get("grant_type"); got != "authorization_code" {
			t.Errorf("grant_type = %q", got)
		}
		if got := values.Get("code"); got != "browser-code" {
			t.Errorf("code = %q", got)
		}
		return tokenHTTPResponse(http.StatusOK, `{"access_token":"seven-jwt","refresh_token":"new-refresh"}`), nil
	}}

	accessToken, err := client.ExchangeAuthorizationCode(
		context.Background(),
		"seven://auth?code=browser-code&state="+state.Query().Get("state"),
	)
	if err != nil {
		t.Fatalf("ExchangeAuthorizationCode() error = %v", err)
	}
	if accessToken != "seven-jwt" {
		t.Errorf("access token = %q", accessToken)
	}
	if store.saved != "new-refresh" {
		t.Errorf("stored refresh token = %q", store.saved)
	}
}

func TestRefreshDoesNotReturnRotatedAccessTokenWhenPersistenceFails(t *testing.T) {
	store := &fakeTokenStore{token: "old-refresh", saveErr: errors.New("keyring unavailable")}
	client := NewTokenClient("https://api.example.com", "seven-test-tui-host", store)
	client.httpClient = fakeHTTPClient{do: func(request *http.Request) (*http.Response, error) {
		if got := request.Header.Get("x-device-id"); got != "" {
			t.Errorf("refresh sent x-device-id = %q", got)
		}
		body, _ := io.ReadAll(request.Body)
		values, _ := url.ParseQuery(string(body))
		if got := values.Get("refresh_token"); got != "old-refresh" {
			t.Errorf("refresh_token = %q", got)
		}
		return tokenHTTPResponse(http.StatusOK, `{"access_token":"must-not-be-used","refresh_token":"rotated-refresh"}`), nil
	}}

	accessToken, err := client.Refresh(context.Background())
	if err == nil {
		t.Fatal("Refresh() error = nil, want persistence error")
	}
	if accessToken != "" {
		t.Errorf("access token = %q, want empty when persistence fails", accessToken)
	}
	if store.saved != "rotated-refresh" {
		t.Errorf("stored refresh token = %q", store.saved)
	}
}

func TestRefreshReportsRevokedSession(t *testing.T) {
	store := &fakeTokenStore{token: "old-refresh"}
	client := NewTokenClient("https://api.example.com", "seven-test-tui-host", store)
	client.httpClient = fakeHTTPClient{do: func(*http.Request) (*http.Response, error) {
		return tokenHTTPResponse(http.StatusBadRequest, `{"error":"invalid_grant","error_description":"Refresh token is no longer valid"}`), nil
	}}

	_, err := client.Refresh(context.Background())
	if !errors.Is(err, ErrSessionRevoked) {
		t.Fatalf("Refresh() error = %v, want ErrSessionRevoked", err)
	}
}

func TestRefreshTokenStoreFallsBackToPrivateFile(t *testing.T) {
	file := filepath.Join(t.TempDir(), "config", "refresh-token")
	store := newRefreshTokenStore(fakeKeyring{setErr: errors.New("no keyring")}, "account", file)

	if err := store.Save("refresh-token"); err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	info, err := os.Stat(file)
	if err != nil {
		t.Fatalf("Stat() error = %v", err)
	}
	if info.Mode().Perm() != 0600 {
		t.Errorf("file mode = %o, want 600", info.Mode().Perm())
	}
	token, err := store.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if token != "refresh-token" {
		t.Errorf("token = %q", token)
	}
}

type fakeRefresher struct {
	token string
	err   error
	calls int
}

func (r *fakeRefresher) Refresh(context.Context) (string, error) {
	r.calls++
	return r.token, r.err
}

func TestAPIClientRefreshesAndRetriesOneUnauthorizedRead(t *testing.T) {
	requests := 0
	client := NewAPIClient("https://api.example.com")
	client.SetIDToken("expired-token")
	client.SetTokenRefresher(&fakeRefresher{token: "fresh-token"})
	client.httpClient = fakeHTTPClient{do: func(request *http.Request) (*http.Response, error) {
		requests++
		switch requests {
		case 1:
			if got := request.Header.Get("Authorization"); got != "Bearer expired-token" {
				t.Errorf("first authorization = %q", got)
			}
			return tokenHTTPResponse(http.StatusUnauthorized, `{"errorMessage":"Unauthorized"}`), nil
		case 2:
			if got := request.Header.Get("Authorization"); got != "Bearer fresh-token" {
				t.Errorf("retry authorization = %q", got)
			}
			return tokenHTTPResponse(http.StatusOK, `[]`), nil
		default:
			return nil, errors.New("unexpected request")
		}
	}}

	if _, err := client.GetGameWeeks(context.Background()); err != nil {
		t.Fatalf("GetGameWeeks() error = %v", err)
	}
	if requests != 2 {
		t.Errorf("request count = %d, want 2", requests)
	}
}

func TestExtractAuthorizationCode(t *testing.T) {
	code := ExtractAuthorizationCode("seven://auth?code=browser-code&state=state")
	if code != "browser-code" {
		t.Errorf("code = %q", code)
	}
	if code := ExtractAuthorizationCode("plain-code"); strings.TrimSpace(code) != "plain-code" {
		t.Errorf("plain code = %q", code)
	}
}

func tokenHTTPResponse(statusCode int, body string) *http.Response {
	return &http.Response{
		StatusCode: statusCode,
		Body:       io.NopCloser(strings.NewReader(body)),
		Header:     make(http.Header),
	}
}

func TestAPIClientStopsAfterRetryingOneUnauthorizedRead(t *testing.T) {
	requests := 0
	refresher := &fakeRefresher{token: "fresh-token"}
	client := NewAPIClient("https://api.example.com")
	client.SetIDToken("expired-token")
	client.SetTokenRefresher(refresher)
	client.httpClient = fakeHTTPClient{do: func(*http.Request) (*http.Response, error) {
		requests++
		return tokenHTTPResponse(http.StatusUnauthorized, `{"errorMessage":"Unauthorized"}`), nil
	}}

	if _, err := client.GetGameWeeks(context.Background()); err == nil {
		t.Fatal("GetGameWeeks() error = nil, want second unauthorized response")
	}
	if requests != 2 {
		t.Errorf("request count = %d, want 2", requests)
	}
	if refresher.calls != 1 {
		t.Errorf("refresh count = %d, want 1", refresher.calls)
	}
}

func TestAuthorizationURLIncludesBootstrapParametersAndValidatesState(t *testing.T) {
	client := NewTokenClient("https://api.example.com", "seven-test-tui-host", &fakeTokenStore{})
	authorizationURL, err := client.AuthorizationURL()
	if err != nil {
		t.Fatalf("AuthorizationURL() error = %v", err)
	}
	parsed, err := url.Parse(authorizationURL)
	if err != nil {
		t.Fatalf("parse authorization URL: %v", err)
	}
	if parsed.Path != "/oauth2/authorize" {
		t.Errorf("authorization path = %q", parsed.Path)
	}
	query := parsed.Query()
	if query.Get("state") == "" {
		t.Error("authorization URL omitted state")
	}
	if query.Get("device_id") != "seven-test-tui-host" {
		t.Errorf("device_id = %q", query.Get("device_id"))
	}
	if query.Get("redirect_uri") != authorizationRedirectURI {
		t.Errorf("redirect_uri = %q", query.Get("redirect_uri"))
	}
	if query.Get("client_id") != authorizationClientID {
		t.Errorf("client_id = %q", query.Get("client_id"))
	}

	_, err = client.ExchangeAuthorizationCode(context.Background(), "seven://auth?code=browser-code&state=other")
	if err == nil || !strings.Contains(err.Error(), "state mismatch") {
		t.Fatalf("ExchangeAuthorizationCode() error = %v, want state mismatch", err)
	}
}
