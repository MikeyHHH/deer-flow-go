package handlers

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"deer-flow-go/internal/workflow"
	"deer-flow-go/pkg/models"
	"deer-flow-go/pkg/queue"
)

// APIHandler API处理器
type APIHandler struct {
	agentWorkflow *workflow.AgentWorkflow
	queueManager  *queue.QueueManager
	logger        *logrus.Logger
}

// NewAPIHandler 创建新的API处理器
func NewAPIHandler(agentWorkflow *workflow.AgentWorkflow, queueManager *queue.QueueManager, logger *logrus.Logger) *APIHandler {
	return &APIHandler{
		agentWorkflow: agentWorkflow,
		queueManager:  queueManager,
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
		
		// 队列状态
		api.GET("/queue/status", h.QueueStatus)
		api.GET("/queue/stats", h.QueueStats)
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
	ctx, cancel := context.WithTimeout(c.Request.Context(), 60*time.Second)
	defer cancel()
	
	// 使用队列管理器处理请求
	resp, err := h.queueManager.SubmitRequest(ctx, req.Query)
	if err != nil {
		h.logger.WithError(err).Error("Failed to process query through queue")
		
		// 根据错误类型返回不同的HTTP状态码
		errorMsg := err.Error()
		if strings.Contains(errorMsg, "request queue is full") || strings.Contains(errorMsg, "timeout after") {
			// 队列超时 - 服务暂时不可用
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"error": "Service temporarily unavailable, please try again later",
				"code":  "QUEUE_TIMEOUT",
				"details": errorMsg,
			})
		} else if strings.Contains(errorMsg, "request timeout") {
			// 请求超时
			c.JSON(http.StatusRequestTimeout, gin.H{
				"error": "Request timeout, please try again",
				"code":  "REQUEST_TIMEOUT",
				"details": errorMsg,
			})
		} else if strings.Contains(errorMsg, "context canceled") || strings.Contains(errorMsg, "context deadline exceeded") {
			// 上下文取消或超时
			c.JSON(http.StatusRequestTimeout, gin.H{
				"error": "Request was cancelled or timed out",
				"code":  "CONTEXT_TIMEOUT",
				"details": errorMsg,
			})
		} else if strings.Contains(errorMsg, "queue manager is not running") {
			// 队列管理器未运行
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"error": "Service is currently unavailable",
				"code":  "SERVICE_UNAVAILABLE",
				"details": errorMsg,
			})
		} else {
			// 其他内部错误
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Internal server error",
				"code":  "INTERNAL_ERROR",
				"details": errorMsg,
			})
		}
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

// QueueStatus 队列状态处理器
func (h *APIHandler) QueueStatus(c *gin.Context) {
	status := map[string]interface{}{
		"healthy": h.queueManager.IsHealthy(),
		"timestamp": time.Now(),
	}
	
	c.JSON(http.StatusOK, status)
}

// QueueStats 队列统计处理器
func (h *APIHandler) QueueStats(c *gin.Context) {
	stats := h.queueManager.GetStats()
	stats["timestamp"] = time.Now()
	
	c.JSON(http.StatusOK, stats)
}