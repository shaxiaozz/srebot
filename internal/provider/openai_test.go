package provider

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func newStubServer(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
}

func TestNewOpenAICompatible_MissingAPIKey(t *testing.T) {
	_, err := NewOpenAICompatible(Config{Model: "x", BaseURL: "http://a"})
	if err == nil || !strings.Contains(err.Error(), "APIKey") {
		t.Fatalf("expected APIKey error, got %v", err)
	}
}

func TestNewOpenAICompatible_MissingModel(t *testing.T) {
	_, err := NewOpenAICompatible(Config{APIKey: "k", BaseURL: "http://a"})
	if err == nil || !strings.Contains(err.Error(), "Model") {
		t.Fatalf("expected Model error, got %v", err)
	}
}

func TestNewOpenAICompatible_MissingBaseURL(t *testing.T) {
	_, err := NewOpenAICompatible(Config{APIKey: "k", Model: "gpt"})
	if err == nil || !strings.Contains(err.Error(), "BaseURL") {
		t.Fatalf("expected BaseURL error, got %v", err)
	}
}

func TestNewOpenAICompatible_Success(t *testing.T) {
	ts := newStubServer(t)
	defer ts.Close()

	cm, err := NewOpenAICompatible(Config{
		APIKey:  "k",
		Model:   "gpt",
		BaseURL: ts.URL,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cm == nil {
		t.Fatal("expected non-nil ChatModel")
	}
}

func TestNewOpenAICompatible_DefaultTimeout(t *testing.T) {
	ts := newStubServer(t)
	defer ts.Close()

	cm, err := NewOpenAICompatible(Config{
		APIKey:     "k",
		Model:      "gpt",
		BaseURL:    ts.URL,
		TimeoutSec: 0,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cm == nil {
		t.Fatal("expected non-nil ChatModel")
	}
}

func TestNewOpenAICompatible_Temperature(t *testing.T) {
	ts := newStubServer(t)
	defer ts.Close()

	cm, err := NewOpenAICompatible(Config{
		APIKey:      "k",
		Model:       "gpt",
		BaseURL:     ts.URL,
		Temperature: 0.7,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cm == nil {
		t.Fatal("expected non-nil ChatModel")
	}
}

func TestNewOpenAICompatible_MaxTokens(t *testing.T) {
	ts := newStubServer(t)
	defer ts.Close()

	cm, err := NewOpenAICompatible(Config{
		APIKey:    "k",
		Model:     "gpt",
		BaseURL:   ts.URL,
		MaxTokens: 1000,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cm == nil {
		t.Fatal("expected non-nil ChatModel")
	}
}
