package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"deer-flow-go/internal/workflow"
	"deer-flow-go/pkg/models"
)

// APIHandler API处理器
type APIHandler struct {
	agentWorkflow *workflow.AgentWorkflow
	logger        *logrus.Logger
}

// NewAPIHandler 创建新的API处理器
func NewAPIHandler(agentWorkflow *workflow.AgentWorkflow, logger *logrus.Logger) *APIHandler {
	return &APIHandler{
		agentWorkflow: agentWorkflow,
		logger:        logger,
	}
}

// SetupRoutes 设置API路由
func (h *APIHandler) SetupRoutes(router *gin.Engine) {
	// 健康检查
	router.GET("/health", h.HealthCheck)
	
	// API路由组
	api := router.Group("/api")
	{
		// 聊天相关
		api.POST("/chat", h.Chat)
		
		// 工作流状态
		api.GET("/workflow/status", h.WorkflowStatus)
	}
}

// HealthCheck 健康检查处理器
func (h *APIHandler) HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "ok",
		"timestamp": time.Now(),
	})
}

// Chat 聊天处理器
func (h *APIHandler) Chat(c *gin.Context) {
	var req models.ChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request format",
		})
		return
	}
	
	h.logger.WithFields(logrus.Fields{
		"query":         req.Query,
		"messages_count": len(req.Messages),
	}).Info("Received chat request")
	
	// 创建上下文
	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()
	
	// 处理查询
	resp, err := h.agentWorkflow.ProcessQuery(ctx, req.Query)
	if err != nil {
		h.logger.WithError(err).Error("Failed to process query")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}
	
	c.JSON(http.StatusOK, resp)
}

// WorkflowStatus 工作流状态处理器
func (h *APIHandler) WorkflowStatus(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()
	
	status, err := h.agentWorkflow.GetWorkflowStatus(ctx)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get workflow status")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}
	
	c.JSON(http.StatusOK, status)
}