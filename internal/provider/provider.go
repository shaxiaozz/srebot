package provider

import "github.com/cloudwego/eino/components/model"

// ChatModel is an alias of eino's ToolCallingChatModel so callers can directly
// plug the provider into eino Graphs / ToolsNodes without extra wrapping.
type ChatModel = model.ToolCallingChatModel

// Config captures the fields we expose to our config layer.
// Only fields used by MVP.
type Config struct {
	BaseURL     string
	APIKey      string
	Model       string
	Temperature float64
	TimeoutSec  int
	MaxTokens   int
}
