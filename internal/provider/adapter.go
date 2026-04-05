package provider

import (
	"context"

	"go.uber.org/zap"
)

// ChatRequest represents a request to an LLM provider
type ChatRequest struct {
	ModelID    string    `json:"model_id"`
	Prompt     string    `json:"prompt"`
	Messages   []Message `json:"messages,omitempty"`
	MaxTokens  int       `json:"max_tokens,omitempty"`
	Temperature float64  `json:"temperature,omitempty"`
	Stream     bool      `json:"stream"`
}

// Message represents a chat message
type Message struct {
	Role    string `json:"role"`    // "system", "user", "assistant"
	Content string `json:"content"`
}

// Chunk represents a streaming response chunk from an LLM
type Chunk struct {
	Content      string `json:"content"`
	FinishReason string `json:"finish_reason,omitempty"`
	PromptTokens int    `json:"prompt_tokens,omitempty"`
	OutputTokens int    `json:"output_tokens,omitempty"`
	Error        error  `json:"-"`
}

// IProviderAdapter defines the interface for LLM provider adapters
type IProviderAdapter interface {
	Chat(ctx context.Context, logger *zap.Logger, req ChatRequest) (<-chan Chunk, error)
	ModelID() string
}

// AdapterRegistry holds all registered provider adapters
type AdapterRegistry struct {
	adapters map[string]IProviderAdapter
}

// NewAdapterRegistry creates a new adapter registry
func NewAdapterRegistry() *AdapterRegistry {
	return &AdapterRegistry{
		adapters: make(map[string]IProviderAdapter),
	}
}

// Register adds an adapter to the registry
func (r *AdapterRegistry) Register(adapter IProviderAdapter) {
	r.adapters[adapter.ModelID()] = adapter
}

// Get retrieves an adapter by model ID
func (r *AdapterRegistry) Get(modelID string) (IProviderAdapter, bool) {
	adapter, ok := r.adapters[modelID]
	return adapter, ok
}

// List returns all registered model IDs
func (r *AdapterRegistry) List() []string {
	ids := make([]string, 0, len(r.adapters))
	for id := range r.adapters {
		ids = append(ids, id)
	}
	return ids
}
