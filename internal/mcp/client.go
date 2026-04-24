package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
	mcpclient "github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
)

// defaultToolTimeout applies when Server.ToolTimeout is 0.
const defaultToolTimeout = 60 * time.Second

// Client wraps a single mcp-go client and exposes the remote tools as
// eino InvokableTool implementations.
type Client struct {
	name    string
	timeout time.Duration
	mc      *mcpclient.Client
	tools   []tool.InvokableTool
}

// newClient builds a Client by selecting the proper mcp-go transport based
// on Server.IsHTTP(). It does NOT call Connect() — callers must do that.
func newClient(s Server) (*Client, error) {
	var (
		mc  *mcpclient.Client
		err error
	)
	if s.IsHTTP() {
		mc, err = mcpclient.NewStreamableHttpClient(s.URL)
	} else {
		env := make([]string, 0, len(s.Env))
		for k, v := range s.Env {
			env = append(env, k+"="+v)
		}
		mc, err = mcpclient.NewStdioMCPClient(s.Command, env, s.Args...)
	}
	if err != nil {
		return nil, fmt.Errorf("mcp: create client for %q: %w", s.Name, err)
	}
	timeout := s.ToolTimeout
	if timeout == 0 {
		timeout = defaultToolTimeout
	}
	return &Client{name: s.Name, timeout: timeout, mc: mc}, nil
}

// Connect starts the underlying transport, performs MCP initialize and
// caches the remote tool list as InvokableTool wrappers.
func (c *Client) Connect(ctx context.Context) error {
	if err := c.mc.Start(ctx); err != nil {
		return fmt.Errorf("mcp: start %q: %w", c.name, err)
	}
	if _, err := c.mc.Initialize(ctx, mcp.InitializeRequest{}); err != nil {
		return fmt.Errorf("mcp: initialize %q: %w", c.name, err)
	}
	resp, err := c.mc.ListTools(ctx, mcp.ListToolsRequest{})
	if err != nil {
		return fmt.Errorf("mcp: list tools %q: %w", c.name, err)
	}
	c.tools = make([]tool.InvokableTool, 0, len(resp.Tools))
	for _, remote := range resp.Tools {
		c.tools = append(c.tools, &mcpTool{
			mc:         c.mc,
			timeout:    c.timeout,
			localName:  fmt.Sprintf("mcp_%s_%s", c.name, remote.Name),
			remoteName: remote.Name,
			desc:       remote.Description,
			schema:     remote.RawInputSchema,
		})
	}
	return nil
}

// Tools returns the InvokableTool wrappers discovered by Connect().
func (c *Client) Tools() []tool.InvokableTool { return c.tools }

// Close releases the underlying mcp-go client transport.
func (c *Client) Close() error { return c.mc.Close() }

// mcpTool adapts a single remote MCP tool to eino's InvokableTool.
//
// Schema conversion from MCP JSON Schema to eino's schema.ParameterInfo is
// non-trivial; for MVP we omit ParamsOneOf and rely on the description so
// the LLM still knows what to pass. The schema field is reserved for future
// best-effort conversion.
type mcpTool struct {
	mc         *mcpclient.Client
	timeout    time.Duration
	localName  string
	remoteName string
	desc       string
	schema     json.RawMessage
}

func (t *mcpTool) Info(_ context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: t.localName,
		Desc: t.desc,
	}, nil
}

func (t *mcpTool) InvokableRun(ctx context.Context, argsJSON string, _ ...tool.Option) (string, error) {
	args := map[string]any{}
	if argsJSON != "" {
		if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
			return "", fmt.Errorf("mcp tool %s: invalid arguments json: %w", t.localName, err)
		}
	}

	callCtx, cancel := context.WithTimeout(ctx, t.timeout)
	defer cancel()

	req := mcp.CallToolRequest{}
	req.Params.Name = t.remoteName
	req.Params.Arguments = args

	res, err := t.mc.CallTool(callCtx, req)
	if err != nil {
		return "", fmt.Errorf("mcp tool %s: call failed: %w", t.localName, err)
	}

	var sb strings.Builder
	for _, c := range res.Content {
		if tc, ok := c.(mcp.TextContent); ok {
			if sb.Len() > 0 {
				sb.WriteString("\n")
			}
			sb.WriteString(tc.Text)
		}
	}
	return sb.String(), nil
}
