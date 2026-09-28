package aws

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGet_SendsApiKeyAndAuthHeaders(t *testing.T) {
	var gotAPIKey, gotAuth, gotPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAPIKey = r.Header.Get("x-seven-api-key")
		gotAuth = r.Header.Get("Authorization")
		gotPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`[]`))
	}))
	defer server.Close()

	client := NewAPIClient(server.URL)
	client.SetIDToken("id-token-abc")
	client.SetAPIKey("secret-key-123")

	if _, err := client.GetGameWeeks(context.Background()); err != nil {
		t.Fatalf("GetGameWeeks() error = %v", err)
	}

	if gotAPIKey != "secret-key-123" {
		t.Errorf("x-seven-api-key = %q, want %q", gotAPIKey, "secret-key-123")
	}
	if gotAuth != "Bearer id-token-abc" {
		t.Errorf("Authorization = %q, want %q", gotAuth, "Bearer id-token-abc")
	}
	if gotPath != "/game-weeks" {
		t.Errorf("path = %q, want %q", gotPath, "/game-weeks")
	}
}

func TestGet_OmitsApiKeyWhenUnset(t *testing.T) {
	var hasAPIKey bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, hasAPIKey = r.Header["X-Seven-Api-Key"]
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`[]`))
	}))
	defer server.Close()

	client := NewAPIClient(server.URL)
	if _, err := client.GetGameWeeks(context.Background()); err != nil {
		t.Fatalf("GetGameWeeks() error = %v", err)
	}

	if hasAPIKey {
		t.Error("x-seven-api-key header should not be set when API key is empty")
	}
}

func TestPut_SendsApiKeyHeader(t *testing.T) {
	var gotAPIKey string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAPIKey = r.Header.Get("x-seven-api-key")
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client := NewAPIClient(server.URL)
	client.SetAPIKey("put-key-456")

	if err := client.PutSelections(context.Background(), map[string]interface{}{"a": 1}); err != nil {
		t.Fatalf("PutSelections() error = %v", err)
	}

	if gotAPIKey != "put-key-456" {
		t.Errorf("x-seven-api-key = %q, want %q", gotAPIKey, "put-key-456")
	}
}
