package provider

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	openaiext "github.com/cloudwego/eino-ext/components/model/openai"
)

// NewOpenAICompatible constructs a ChatModel pointed at any OpenAI-compatible
// endpoint (OpenAI, DeepSeek, vLLM, Ollama with OpenAI shim, etc.).
func NewOpenAICompatible(cfg Config) (ChatModel, error) {
	if cfg.APIKey == "" {
		return nil, errors.New("provider: APIKey required")
	}
	if cfg.Model == "" {
		return nil, errors.New("provider: Model required")
	}
	if cfg.BaseURL == "" {
		return nil, errors.New("provider: BaseURL required")
	}

	timeout := time.Duration(cfg.TimeoutSec) * time.Second
	if cfg.TimeoutSec <= 0 {
		timeout = 120 * time.Second
	}

	httpClient := &http.Client{Timeout: timeout}

	ecfg := &openaiext.ChatModelConfig{
		APIKey:     cfg.APIKey,
		BaseURL:    cfg.BaseURL,
		Model:      cfg.Model,
		HTTPClient: httpClient,
	}
	if cfg.MaxTokens > 0 {
		mt := cfg.MaxTokens
		ecfg.MaxTokens = &mt
	}
	temp := float32(cfg.Temperature)
	ecfg.Temperature = &temp

	cm, err := openaiext.NewChatModel(context.Background(), ecfg)
	if err != nil {
		return nil, fmt.Errorf("provider: new chat model: %w", err)
	}
	return cm, nil
}
