package provider

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"go.uber.org/zap"
)

const anthropicProviderName = "anthropic"

type anthropicAdapter struct {
	pool       ICredentialPool
	modelID    string
	httpClient *http.Client
}

// NewAnthropicAdapter creates a new Anthropic adapter with credential pool support
func NewAnthropicAdapter(pool ICredentialPool, modelID string) IProviderAdapter {
	return &anthropicAdapter{
		pool:    pool,
		modelID: modelID,
		httpClient: &http.Client{
			Timeout: 120 * time.Second,
		},
	}
}

func (a *anthropicAdapter) ModelID() string {
	return a.modelID
}

func (a *anthropicAdapter) Chat(ctx context.Context, logger *zap.Logger, req ChatRequest) (<-chan Chunk, error) {
	chunkChan := make(chan Chunk, 100)

	// Acquire credential from pool
	cred, err := a.pool.Acquire(ctx, logger, anthropicProviderName)
	if err != nil {
		close(chunkChan)
		return nil, fmt.Errorf("failed to acquire Anthropic credential: %w", err)
	}

	logger = logger.With(zap.String("key_hash", cred.KeyHash))

	// TODO: Implement Anthropic API integration
	// This is a stub implementation that returns an error indicating the adapter is not yet implemented
	//
	// Implementation notes:
	// 1. Build request to https://api.anthropic.com/v1/messages
	// 2. Set x-api-key header from cred.Key and anthropic-version header
	// 3. Enable streaming with "stream": true
	// 4. Parse SSE response events (content_block_delta, message_delta, etc.)
	// 5. Convert to our Chunk format
	// 6. On 429 response, call a.pool.MarkExhausted(ctx, logger, cred, retryAfter)
	// 7. Always call a.pool.Release(ctx, logger, cred) when done
	//
	// Example request body:
	// {
	//   "model": "claude-3-opus-20240229",
	//   "max_tokens": 4096,
	//   "messages": [{"role": "user", "content": "..."}],
	//   "stream": true
	// }

	go func() {
		defer close(chunkChan)
		defer func() {
			if releaseErr := a.pool.Release(ctx, logger, cred); releaseErr != nil {
				logger.Error("failed to release credential", zap.Error(releaseErr))
			}
		}()

		chunkChan <- Chunk{
			Error: fmt.Errorf("Anthropic adapter not yet implemented for model: %s", a.modelID),
		}
	}()

	return chunkChan, nil
}
