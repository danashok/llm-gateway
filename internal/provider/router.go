package provider

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	"github.com/ashokdan/llm-gateway/internal/config"
)

// IProviderRouter defines the interface for routing requests to providers
type IProviderRouter interface {
	Route(ctx context.Context, logger *zap.Logger, modelID string) (IProviderAdapter, error)
	ListModels() []string
}

type providerRouter struct {
	registry *AdapterRegistry
}

// NewProviderRouter creates a new provider router with all adapters registered
func NewProviderRouter(cfg *config.ProviderConfig, pool ICredentialPool) IProviderRouter {
	registry := NewAdapterRegistry()

	// Register internal adapter (always available, no credentials needed)
	registry.Register(NewInternalAdapter(cfg.InternalAPIURL))

	// Register OpenAI adapters if API keys are configured
	openaiKeys := cfg.GetOpenAIKeys()
	if len(openaiKeys) > 0 {
		registry.Register(NewOpenAIAdapter(pool, "gpt-4"))
		registry.Register(NewOpenAIAdapter(pool, "gpt-4-turbo"))
		registry.Register(NewOpenAIAdapter(pool, "gpt-3.5-turbo"))
	}

	// Register Anthropic adapters if API keys are configured
	anthropicKeys := cfg.GetAnthropicKeys()
	if len(anthropicKeys) > 0 {
		registry.Register(NewAnthropicAdapter(pool, "claude-3-opus"))
		registry.Register(NewAnthropicAdapter(pool, "claude-3-sonnet"))
		registry.Register(NewAnthropicAdapter(pool, "claude-3-haiku"))
	}

	return &providerRouter{
		registry: registry,
	}
}

func (r *providerRouter) Route(ctx context.Context, logger *zap.Logger, modelID string) (IProviderAdapter, error) {
	adapter, ok := r.registry.Get(modelID)
	if !ok {
		logger.Warn("model not found", zap.String("model_id", modelID))
		return nil, fmt.Errorf("model not found: %s", modelID)
	}
	return adapter, nil
}

func (r *providerRouter) ListModels() []string {
	return r.registry.List()
}
