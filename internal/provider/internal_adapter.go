package provider

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"go.uber.org/zap"
)

type internalAdapter struct {
	baseURL    string
	httpClient *http.Client
	modelID    string
}

// NewInternalAdapter creates a new adapter for internally hosted models
func NewInternalAdapter(baseURL string) IProviderAdapter {
	return &internalAdapter{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 120 * time.Second,
		},
		modelID: "internal",
	}
}

func (a *internalAdapter) ModelID() string {
	return a.modelID
}

// internalRequest represents the request format for internal LLM service
type internalRequest struct {
	Prompt      string    `json:"prompt"`
	Messages    []Message `json:"messages,omitempty"`
	MaxTokens   int       `json:"max_tokens,omitempty"`
	Temperature float64   `json:"temperature,omitempty"`
	Stream      bool      `json:"stream"`
}

// internalStreamResponse represents a streaming response from internal LLM
type internalStreamResponse struct {
	Content      string `json:"content"`
	Done         bool   `json:"done"`
	PromptTokens int    `json:"prompt_tokens,omitempty"`
	OutputTokens int    `json:"output_tokens,omitempty"`
}

func (a *internalAdapter) Chat(ctx context.Context, logger *zap.Logger, req ChatRequest) (<-chan Chunk, error) {
	chunkChan := make(chan Chunk, 100)

	// Build request body
	internalReq := internalRequest{
		Prompt:      req.Prompt,
		Messages:    req.Messages,
		MaxTokens:   req.MaxTokens,
		Temperature: req.Temperature,
		Stream:      true,
	}

	if internalReq.MaxTokens == 0 {
		internalReq.MaxTokens = 4096
	}

	body, err := json.Marshal(internalReq)
	if err != nil {
		close(chunkChan)
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, a.baseURL+"/v1/chat/completions", bytes.NewReader(body))
	if err != nil {
		close(chunkChan)
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "text/event-stream")

	// Start streaming in goroutine
	go func() {
		defer close(chunkChan)

		resp, err := a.httpClient.Do(httpReq)
		if err != nil {
			chunkChan <- Chunk{Error: fmt.Errorf("request failed: %w", err)}
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			chunkChan <- Chunk{Error: fmt.Errorf("provider returned status %d: %s", resp.StatusCode, string(body))}
			return
		}

		a.processStream(ctx, logger, resp.Body, chunkChan)
	}()

	return chunkChan, nil
}

func (a *internalAdapter) processStream(ctx context.Context, logger *zap.Logger, body io.Reader, chunkChan chan<- Chunk) {
	scanner := bufio.NewScanner(body)
	var totalPromptTokens, totalOutputTokens int

	for scanner.Scan() {
		select {
		case <-ctx.Done():
			chunkChan <- Chunk{Error: ctx.Err()}
			return
		default:
		}

		line := scanner.Text()

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, ":") {
			continue
		}

		// Parse SSE data line
		if strings.HasPrefix(line, "data: ") {
			data := strings.TrimPrefix(line, "data: ")

			// Check for stream end
			if data == "[DONE]" {
				chunkChan <- Chunk{
					FinishReason: "stop",
					PromptTokens: totalPromptTokens,
					OutputTokens: totalOutputTokens,
				}
				return
			}

			var streamResp internalStreamResponse
			if err := json.Unmarshal([]byte(data), &streamResp); err != nil {
				logger.Warn("failed to parse stream response", zap.Error(err), zap.String("data", data))
				continue
			}

			if streamResp.PromptTokens > 0 {
				totalPromptTokens = streamResp.PromptTokens
			}
			if streamResp.OutputTokens > 0 {
				totalOutputTokens = streamResp.OutputTokens
			}

			if streamResp.Done {
				chunkChan <- Chunk{
					Content:      streamResp.Content,
					FinishReason: "stop",
					PromptTokens: totalPromptTokens,
					OutputTokens: totalOutputTokens,
				}
				return
			}

			if streamResp.Content != "" {
				chunkChan <- Chunk{Content: streamResp.Content}
			}
		}
	}

	if err := scanner.Err(); err != nil {
		chunkChan <- Chunk{Error: fmt.Errorf("stream read error: %w", err)}
	}
}
