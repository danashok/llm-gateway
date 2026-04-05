package provider

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"go.uber.org/zap"
)

const openaiProviderName = "openai"

type openaiAdapter struct {
	pool       ICredentialPool
	modelID    string
	httpClient *http.Client
}

// NewOpenAIAdapter creates a new OpenAI adapter with credential pool support
func NewOpenAIAdapter(pool ICredentialPool, modelID string) IProviderAdapter {
	return &openaiAdapter{
		pool:    pool,
		modelID: modelID,
		httpClient: &http.Client{
			Timeout: 120 * time.Second,
		},
	}
}

func (a *openaiAdapter) ModelID() string {
	return a.modelID
}

func (a *openaiAdapter) Chat(ctx context.Context, logger *zap.Logger, req ChatRequest) (<-chan Chunk, error) {
	chunkChan := make(chan Chunk, 100)

	// Acquire credential from pool
	cred, err := a.pool.Acquire(ctx, logger, openaiProviderName)
	if err != nil {
		close(chunkChan)
		return nil, fmt.Errorf("failed to acquire OpenAI credential: %w", err)
	}

	logger = logger.With(zap.String("key_hash", cred.KeyHash))

	// TODO: Implement OpenAI API integration
	// This is a stub implementation that returns an error indicating the adapter is not yet implemented
	//
	// Implementation notes:
	// 1. Build request to https://api.openai.com/v1/chat/completions
	// 2. Set Authorization header with Bearer token from cred.Key
	// 3. Enable streaming with "stream": true
	// 4. Parse SSE response chunks
	// 5. Convert to our Chunk format
	// 6. On 429 response, call a.pool.MarkExhausted(ctx, logger, cred, retryAfter)
	// 7. Always call a.pool.Release(ctx, logger, cred) when done
	//
	// Example request body:
	// {
	//   "model": "gpt-4",
	//   "messages": [{"role": "user", "content": "..."}],
	//   "max_tokens": 4096,
	//   "temperature": 0.7,
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
			Error: fmt.Errorf("OpenAI adapter not yet implemented for model: %s", a.modelID),
		}
	}()

	return chunkChan, nil
}
