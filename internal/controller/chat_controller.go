package controller

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/ashokdan/llm-gateway/internal/gateway"
)

// ChatController handles chat-related HTTP requests
type ChatController struct {
	handler *gateway.ChatHandler
	logger  *zap.Logger
}

// NewChatController creates a new chat controller
func NewChatController(handler *gateway.ChatHandler, logger *zap.Logger) *ChatController {
	return &ChatController{handler: handler, logger: logger}
}

// Chat godoc
// @Summary Send a chat message to an LLM
// @Description Stream a chat completion from the specified LLM model
// @Tags chat
// @Accept json
// @Produce text/event-stream
// @Security ApiKeyAuth
// @Param request body gateway.ChatRequest true "Chat request"
// @Success 200 {string} string "SSE stream of response chunks"
// @Failure 400 {object} map[string]string "Bad request"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 422 {object} map[string]interface{} "Compliance violation"
// @Failure 429 {object} map[string]string "Rate limit exceeded"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /chat [post]
func (ctrl *ChatController) Chat(c *gin.Context) {
	ctrl.handler.Chat(c)
}

// ListModels godoc
// @Summary List available LLM models
// @Description Returns a list of all available LLM models
// @Tags chat
// @Produce json
// @Security ApiKeyAuth
// @Success 200 {object} map[string][]string "List of model IDs"
// @Router /models [get]
func (ctrl *ChatController) ListModels(c *gin.Context) {
	ctrl.handler.ListModels(c)
}

// BindRoutes binds chat routes to the router group
func (ctrl *ChatController) BindRoutes(rg *gin.RouterGroup) {
	rg.POST("/chat", ctrl.Chat)
	rg.GET("/models", ctrl.ListModels)
}
