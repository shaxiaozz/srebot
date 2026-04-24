// Package mcp wires Model Context Protocol servers (HTTP or stdio) into
// srebot's tools.Registry. The package owns its own Server type so it does
// not import config — callers are expected to translate config.MCPServer
// into mcp.Server at composition time.
package mcp

import "time"

// Server describes a single MCP server connection.
// IsHTTP() distinguishes streamable-HTTP from stdio transports.
type Server struct {
	Name        string
	URL         string
	Command     string
	Args        []string
	Env         map[string]string
	ToolTimeout time.Duration
}

// IsHTTP reports whether the server should be reached via Streamable HTTP.
func (s Server) IsHTTP() bool { return s.URL != "" }
