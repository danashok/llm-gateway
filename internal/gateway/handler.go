package gateway

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/ashokdan/llm-gateway/internal/compliance"
	"github.com/ashokdan/llm-gateway/internal/config"
	"github.com/ashokdan/llm-gateway/internal/middleware"
	"github.com/ashokdan/llm-gateway/internal/models"
	"github.com/ashokdan/llm-gateway/internal/provider"
	"github.com/ashokdan/llm-gateway/internal/ratelimit"
	"github.com/ashokdan/llm-gateway/internal/repository"
)

// ChatRequest represents the incoming chat request
type ChatRequest struct {
	ModelID   string `json:"model_id" binding:"required"`
	Prompt    string `json:"prompt" binding:"required"`
	SessionID string `json:"session_id,omitempty"`
}

// ChatHandler handles LLM chat requests
type ChatHandler struct {
	logger           *zap.Logger
	cfg              *config.Config
	complianceSvc    compliance.IComplianceService
	rateLimitSvc     ratelimit.IRateLimitService
	providerRouter   provider.IProviderRouter
	sessionRepo      repository.ISessionRepository
	auditLogRepo     repository.IAuditLogRepository
	teamBudgetRepo   repository.ITeamBudgetRepository
	costChan         chan costRecord
}

type costRecord struct {
	TraceID      uuid.UUID
	TeamID       uuid.UUID
	ModelID      string
	PromptTokens int
	OutputTokens int
}

// NewChatHandler creates a new chat handler
func NewChatHandler(
	logger *zap.Logger,
	cfg *config.Config,
	complianceSvc compliance.IComplianceService,
	rateLimitSvc ratelimit.IRateLimitService,
	providerRouter provider.IProviderRouter,
	sessionRepo repository.ISessionRepository,
	auditLogRepo repository.IAuditLogRepository,
	teamBudgetRepo repository.ITeamBudgetRepository,
) *ChatHandler {
	h := &ChatHandler{
		logger:           logger,
		cfg:              cfg,
		complianceSvc:    complianceSvc,
		rateLimitSvc:     rateLimitSvc,
		providerRouter:   providerRouter,
		sessionRepo:      sessionRepo,
		auditLogRepo:     auditLogRepo,
		teamBudgetRepo:   teamBudgetRepo,
		costChan:         make(chan costRecord, 1000),
	}

	// Start background cost processor
	go h.processCosts()

	return h
}

// Chat godoc
// @Summary Send a chat message to an LLM
// @Description Stream a chat completion from the specified LLM model
// @Tags chat
// @Accept json
// @Produce text/event-stream
// @Security ApiKeyAuth
// @Param request body ChatRequest true "Chat request"
// @Success 200 {string} string "SSE stream of response chunks"
// @Failure 400 {object} map[string]string "Bad request"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 422 {object} map[string]interface{} "Compliance violation"
// @Failure 429 {object} map[string]string "Rate limit exceeded"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /chat [post]
func (h *ChatHandler) Chat(c *gin.Context) {
	// Generate trace ID for this request
	traceID := uuid.New()
	logger := h.logger.With(zap.String("trace_id", traceID.String()))
	startTime := time.Now()

	// 1. Extract auth context
	authCtx := middleware.GetAuthContext(c)
	teamID := authCtx.TeamID
	userID := authCtx.UserID
	developerID := authCtx.DeveloperID

	logger = logger.With(
		zap.String("team_id", teamID.String()),
		zap.String("user_id", userID.String()),
		zap.String("developer_id", developerID.String()),
	)

	// Parse request
	var req ChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Warn("invalid request", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	logger = logger.With(zap.String("model_id", req.ModelID))
	logger.Info("chat request received")

	// 2. Check rate limit
	allowed, err := h.rateLimitSvc.Allow(c.Request.Context(), logger, teamID.String(), req.ModelID)
	if err != nil {
		logger.Error("rate limit check failed", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "rate limit check failed"})
		return
	}
	if !allowed {
		h.writeAuditLog(c.Request.Context(), logger, traceID, teamID, userID, nil, models.AuditEventRateLimitExceeded, req.ModelID, 0, 0, http.StatusTooManyRequests, "")
		c.JSON(http.StatusTooManyRequests, gin.H{"error": "rate limit exceeded"})
		return
	}

	// 3. Load or create session
	var sessionID *uuid.UUID
	var session *models.Session
	if req.SessionID != "" {
		sid, err := uuid.Parse(req.SessionID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid session_id"})
			return
		}
		sessionID = &sid
		session, err = h.sessionRepo.Get(c.Request.Context(), logger, sid, developerID)
		if err != nil {
			logger.Warn("failed to get session", zap.Error(err))
			c.JSON(http.StatusNotFound, gin.H{"error": "session not found"})
			return
		}
	}

	if session == nil {
		// Create new session
		newSessionID := uuid.New()
		sessionID = &newSessionID
		session = &models.Session{
			ID:          newSessionID,
			TeamID:      teamID,
			UserID:      userID,
			DeveloperID: developerID,
			ModelID:     req.ModelID,
			Messages:    []byte("[]"),
			ExpiresAt:   time.Now().Add(h.cfg.GatewayConfig.SessionTTL),
		}
		if err := h.sessionRepo.Create(c.Request.Context(), logger, session); err != nil {
			logger.Error("failed to create session", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create session"})
			return
		}
	}

	// 4. Run compliance check (parallel fan-out internally)
	complianceReq := compliance.ComplianceRequest{
		TraceID:   traceID,
		TeamID:    teamID,
		UserID:    userID,
		SessionID: sessionID,
		Prompt:    req.Prompt,
		ModelID:   req.ModelID,
	}

	complianceResult, err := h.complianceSvc.Check(c.Request.Context(), logger, complianceReq)
	if err != nil {
		logger.Error("compliance check failed", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "compliance check failed"})
		return
	}

	if !complianceResult.Passed {
		logger.Warn("compliance check rejected",
			zap.Int("violations", len(complianceResult.Violations)),
		)
		c.JSON(http.StatusUnprocessableEntity, gin.H{
			"error":      "compliance check failed",
			"violations": complianceResult.Violations,
		})
		return
	}

	// 5. Resolve provider adapter
	adapter, err := h.providerRouter.Route(c.Request.Context(), logger, req.ModelID)
	if err != nil {
		logger.Error("failed to route to provider", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("unsupported model: %s", req.ModelID)})
		return
	}

	// 6. Call provider and stream response
	chatReq := provider.ChatRequest{
		ModelID: req.ModelID,
		Prompt:  req.Prompt,
		Stream:  true,
	}

	// Add session messages if available
	if len(session.Messages) > 2 { // "[]" is empty
		var messages []provider.Message
		if err := json.Unmarshal(session.Messages, &messages); err == nil {
			chatReq.Messages = messages
		}
	}
	// Add current prompt as user message
	chatReq.Messages = append(chatReq.Messages, provider.Message{
		Role:    "user",
		Content: req.Prompt,
	})

	chunkChan, err := adapter.Chat(c.Request.Context(), logger, chatReq)
	if err != nil {
		logger.Error("failed to start chat", zap.Error(err))
		h.writeAuditLog(c.Request.Context(), logger, traceID, teamID, userID, sessionID, models.AuditEventProviderError, req.ModelID, 0, 0, http.StatusInternalServerError, err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to start chat"})
		return
	}

	// Write audit log for request
	promptHash := sha256.Sum256([]byte(req.Prompt))
	h.writeAuditLog(c.Request.Context(), logger, traceID, teamID, userID, sessionID, models.AuditEventChatRequest, req.ModelID, 0, 0, http.StatusOK, "")

	// 7. Stream SSE response
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Trace-ID", traceID.String())
	c.Header("X-Session-ID", sessionID.String())

	var responseContent string
	var promptTokens, outputTokens int

	c.Stream(func(w io.Writer) bool {
		select {
		case chunk, ok := <-chunkChan:
			if !ok {
				return false
			}

			if chunk.Error != nil {
				logger.Error("stream error", zap.Error(chunk.Error))
				data, _ := json.Marshal(gin.H{"error": chunk.Error.Error()})
				fmt.Fprintf(w, "data: %s\n\n", data)
				return false
			}

			if chunk.Content != "" {
				responseContent += chunk.Content
				data, _ := json.Marshal(gin.H{"content": chunk.Content})
				fmt.Fprintf(w, "data: %s\n\n", data)
			}

			if chunk.FinishReason != "" {
				promptTokens = chunk.PromptTokens
				outputTokens = chunk.OutputTokens
				data, _ := json.Marshal(gin.H{
					"finish_reason": chunk.FinishReason,
					"prompt_tokens": promptTokens,
					"output_tokens": outputTokens,
				})
				fmt.Fprintf(w, "data: %s\n\n", data)
				fmt.Fprintf(w, "data: [DONE]\n\n")
				return false
			}

			return true
		case <-c.Request.Context().Done():
			return false
		}
	})

	// 8. Update session and record costs async
	go func() {
		ctx := context.Background()
		bgLogger := h.logger.With(zap.String("trace_id", traceID.String()))

		// Append messages to session
		if err := h.sessionRepo.AppendMessage(ctx, bgLogger, *sessionID, developerID, models.Message{
			Role:      "user",
			Content:   req.Prompt,
			Timestamp: startTime,
		}); err != nil {
			bgLogger.Error("failed to append user message", zap.Error(err))
		}

		if responseContent != "" {
			if err := h.sessionRepo.AppendMessage(ctx, bgLogger, *sessionID, developerID, models.Message{
				Role:      "assistant",
				Content:   responseContent,
				Timestamp: time.Now(),
			}); err != nil {
				bgLogger.Error("failed to append assistant message", zap.Error(err))
			}
		}

		// Write response audit log
		duration := time.Since(startTime)
		auditLog := &models.AuditLog{
			TraceID:        traceID,
			TeamID:         teamID,
			UserID:         userID,
			SessionID:      sessionID,
			EventType:      models.AuditEventChatResponse,
			ModelID:        req.ModelID,
			PromptHash:     hex.EncodeToString(promptHash[:]),
			PromptTokens:   promptTokens,
			ResponseTokens: outputTokens,
			TotalTokens:    promptTokens + outputTokens,
			DurationMs:     duration.Milliseconds(),
			StatusCode:     http.StatusOK,
		}
		if err := h.auditLogRepo.Create(ctx, bgLogger, auditLog); err != nil {
			bgLogger.Error("failed to create audit log", zap.Error(err))
		}

		// Send cost record for async processing
		h.costChan <- costRecord{
			TraceID:      traceID,
			TeamID:       teamID,
			ModelID:      req.ModelID,
			PromptTokens: promptTokens,
			OutputTokens: outputTokens,
		}
	}()
}

func (h *ChatHandler) writeAuditLog(ctx context.Context, logger *zap.Logger, traceID, teamID, userID uuid.UUID, sessionID *uuid.UUID, eventType models.AuditEventType, modelID string, promptTokens, responseTokens, statusCode int, errorMsg string) {
	auditLog := &models.AuditLog{
		TraceID:        traceID,
		TeamID:         teamID,
		UserID:         userID,
		SessionID:      sessionID,
		EventType:      eventType,
		ModelID:        modelID,
		PromptTokens:   promptTokens,
		ResponseTokens: responseTokens,
		TotalTokens:    promptTokens + responseTokens,
		StatusCode:     statusCode,
		ErrorMessage:   errorMsg,
	}
	if err := h.auditLogRepo.Create(ctx, logger, auditLog); err != nil {
		logger.Error("failed to create audit log", zap.Error(err))
	}
}

func (h *ChatHandler) processCosts() {
	for record := range h.costChan {
		ctx := context.Background()
		logger := h.logger.With(zap.String("trace_id", record.TraceID.String()))

		totalTokens := record.PromptTokens + record.OutputTokens
		if totalTokens > 0 {
			// Update team budget
			if err := h.teamBudgetRepo.IncrementUsage(ctx, logger, record.TeamID, totalTokens); err != nil {
				logger.Error("failed to increment team usage", zap.Error(err))
			}

			// Consume rate limit tokens
			if err := h.rateLimitSvc.Consume(ctx, logger, record.TeamID.String(), totalTokens); err != nil {
				logger.Error("failed to consume rate limit tokens", zap.Error(err))
			}
		}
	}
}

// ListModels godoc
// @Summary List available LLM models
// @Description Returns a list of all available LLM models
// @Tags chat
// @Produce json
// @Security ApiKeyAuth
// @Success 200 {object} map[string][]string "List of model IDs"
// @Router /models [get]
func (h *ChatHandler) ListModels(c *gin.Context) {
	models := h.providerRouter.ListModels()
	c.JSON(http.StatusOK, gin.H{"models": models})
}
