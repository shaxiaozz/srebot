package mcp

import "testing"

func TestIsHTTP_URLSet(t *testing.T) {
	s := Server{Name: "x", URL: "https://example.com/mcp"}
	if !s.IsHTTP() {
		t.Error("expected IsHTTP() == true when URL is set")
	}
}

func TestIsHTTP_CommandSet(t *testing.T) {
	s := Server{Name: "x", Command: "mcp-bin"}
	if s.IsHTTP() {
		t.Error("expected IsHTTP() == false when only Command is set")
	}
}

func TestIsHTTP_BothEmpty(t *testing.T) {
	if (Server{}).IsHTTP() {
		t.Error("zero Server should not be HTTP")
	}
}
