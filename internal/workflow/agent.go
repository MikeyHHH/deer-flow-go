package workflow

import (
	"context"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"

	"deer-flow-go/pkg/config"
	"deer-flow-go/pkg/llm"
	"deer-flow-go/pkg/models"
	"deer-flow-go/pkg/weather"
)

// MCPClientInterface MCP客户端接口
type MCPClientInterface interface {
	ProcessRequest(ctx context.Context, req *models.MCPRequest) (*models.MCPResponse, error)
	HealthCheck(ctx context.Context) error
	GetCapabilities() map[string]interface{}
}

// AgentWorkflow 智能体工作流
type AgentWorkflow struct {
	llmClient *llm.AzureOpenAIClient
	mcpClient MCPClientInterface
	logger    *logrus.Logger
}

// NewAgentWorkflow 函数已被移除，请使用 NewAgentWorkflowWithMCP

// NewAgentWorkflowWithMCP 创建新的智能体工作流（使用真正的MCP客户端）
func NewAgentWorkflowWithMCP(cfg *config.Config, mcpClient MCPClientInterface, logger *logrus.Logger) *AgentWorkflow {
	// 创建LLM客户端
	llmClient := llm.NewAzureOpenAIClient(&cfg.AzureOpenAI, logger)
	
	return &AgentWorkflow{
		llmClient: llmClient,
		mcpClient: mcpClient,
		logger:    logger,
	}
}

// ProcessRequest 实现RequestProcessor接口
func (w *AgentWorkflow) ProcessRequest(ctx context.Context, query string) (*models.ChatResponse, error) {
	return w.ProcessQuery(ctx, query)
}

// ProcessQuery 处理用户查询的完整工作流
func (w *AgentWorkflow) ProcessQuery(ctx context.Context, query string) (*models.ChatResponse, error) {
	startTime := time.Now()
	
	w.logger.WithFields(logrus.Fields{
		"query": query,
	}).Info("Starting agent workflow")
	
	// 步骤1: 使用LLM将用户查询解析为MCP请求
	w.logger.Debug("Step 1: Parsing query to MCP request")
	mcpRequest, err := w.llmClient.ParseQueryToMCP(ctx, query)
	if err != nil {
		w.logger.WithError(err).Error("Failed to parse query to MCP")
		return &models.ChatResponse{
			Response:  "抱歉，处理您的查询时出现错误。",
			Timestamp: time.Now(),
			Success:   false,
			Error:     err.Error(),
		}, nil
	}
	
	w.logger.WithFields(logrus.Fields{
		"mcp_method": mcpRequest.Method,
	}).Debug("Query parsed to MCP request")
	
	// 步骤2: 使用MCP客户端处理请求
	w.logger.Debug("Step 2: Processing MCP request")
	mcpResponse, err := w.mcpClient.ProcessRequest(ctx, mcpRequest)
	if err != nil {
		w.logger.WithError(err).Error("Failed to process MCP request")
		return &models.ChatResponse{
			Response:  "抱歉，搜索过程中出现错误。",
			Timestamp: time.Now(),
			Success:   false,
			Error:     err.Error(),
		}, nil
	}
	
	// 检查MCP响应是否有错误
	if mcpResponse.Error != nil {
		w.logger.WithFields(logrus.Fields{
			"error_code":    mcpResponse.Error.Code,
			"error_message": mcpResponse.Error.Message,
		}).Error("MCP request returned error")
		return &models.ChatResponse{
			Response:  fmt.Sprintf("处理请求时出现错误：%s", mcpResponse.Error.Message),
			Timestamp: time.Now(),
			Success:   false,
			Error:     mcpResponse.Error.Message,
		}, nil
	}
	
	// 步骤3: 处理搜索结果或直接响应
	w.logger.Debug("Step 3: Processing MCP response")
	var finalResponse string
	
	if mcpRequest.Method == "direct_response" {
		// 直接响应，不需要进一步处理
		if resultMap, ok := mcpResponse.Result.(map[string]interface{}); ok {
			if content, exists := resultMap["content"]; exists {
				if contentStr, ok := content.(string); ok {
					finalResponse = contentStr
				} else {
					finalResponse = fmt.Sprintf("%v", content)
				}
			} else {
				finalResponse = "处理完成"
			}
		} else if searchResp, ok := mcpResponse.Result.(*models.SearchResponse); ok {
			finalResponse = searchResp.Answer
		} else {
			finalResponse = "处理完成"
		}
	} else if mcpRequest.Method == "get_weather" || mcpRequest.Method == "get_weather_forecast" {
		// 天气响应，处理真正的MCP协议返回的格式
		if resultMap, ok := mcpResponse.Result.(map[string]interface{}); ok {
			if content, exists := resultMap["content"]; exists {
				// 真正的MCP协议返回格式化的文本内容
				if contentStr, ok := content.(string); ok {
					finalResponse = contentStr
				} else {
					finalResponse = fmt.Sprintf("%v", content)
				}
			} else {
				// 兼容其他格式
				finalResponse = fmt.Sprintf("天气信息: %v", resultMap)
			}
		} else if weatherData, ok := mcpResponse.Result.(*weather.WeatherData); ok {
			// 兼容伪MCP客户端的格式
			finalResponse = fmt.Sprintf("🌤️ %s 当前天气:\n" +
				"🌡️ 温度: %.1f°C\n" +
				"☁️ 天气: %s\n" +
				"💧 湿度: %d%%\n" +
				"💨 风速: %.1f m/s\n" +
				"⏰ 更新时间: %s",
				weatherData.Location,
				weatherData.Temperature,
				weatherData.Description,
				weatherData.Humidity,
				weatherData.WindSpeed,
				weatherData.Timestamp)
		} else {
			w.logger.WithField("result_type", fmt.Sprintf("%T", mcpResponse.Result)).Debug("Weather response format")
			finalResponse = fmt.Sprintf("天气信息: %v", mcpResponse.Result)
		}
	} else {
		// 搜索结果处理，支持真正的MCP协议格式
		if resultMap, ok := mcpResponse.Result.(map[string]interface{}); ok {
			if content, exists := resultMap["content"]; exists {
				// 真正的MCP协议返回格式化的文本内容
				if contentStr, ok := content.(string); ok {
					finalResponse = contentStr
				} else {
					finalResponse = fmt.Sprintf("%v", content)
				}
			} else {
				// 兼容其他格式
				finalResponse = fmt.Sprintf("搜索结果: %v", resultMap)
			}
		} else if searchResp, ok := mcpResponse.Result.(*models.SearchResponse); ok {
			// 兼容伪MCP客户端的格式
			w.logger.WithFields(logrus.Fields{
				"results_count": len(searchResp.Results),
				"has_answer":    searchResp.Answer != "",
			}).Debug("Formatting search results with LLM")
			
			// 步骤4: 使用LLM格式化搜索结果
			finalResponse, err = w.llmClient.FormatSearchResults(ctx, query, searchResp)
			if err != nil {
				w.logger.WithError(err).Error("Failed to format search results")
				// 如果格式化失败，使用原始答案
				if searchResp.Answer != "" {
					finalResponse = searchResp.Answer
				} else {
					finalResponse = "抱歉，无法格式化搜索结果。"
				}
			}
		} else {
			w.logger.WithField("result_type", fmt.Sprintf("%T", mcpResponse.Result)).Error("Invalid MCP response format")
			return &models.ChatResponse{
				Response:  "抱歉，响应格式错误。",
				Timestamp: time.Now(),
				Success:   false,
				Error:     "Invalid response format",
			}, nil
		}
	}
	
	processingTime := time.Since(startTime)
	w.logger.WithFields(logrus.Fields{
		"processing_time": processingTime,
		"response_length": len(finalResponse),
	}).Info("Agent workflow completed successfully")
	
	return &models.ChatResponse{
		Response:  finalResponse,
		Timestamp: time.Now(),
		Success:   true,
	}, nil
}

// GetWorkflowStatus 获取工作流状态
func (w *AgentWorkflow) GetWorkflowStatus(ctx context.Context) (*models.WorkflowState, error) {
	// 检查MCP客户端健康状态
	err := w.mcpClient.HealthCheck(ctx)
	mcpHealthy := err == nil
	
	return &models.WorkflowState{
		Step:        "ready",
		Query:       "",
		MCPRequest:  nil,
		SearchData:  map[string]interface{}{
			"mcp_healthy": mcpHealthy,
			"capabilities": w.mcpClient.GetCapabilities(),
		},
		FinalResult: "",
	}, nil
}

// ValidateWorkflow 验证工作流配置
func (w *AgentWorkflow) ValidateWorkflow(ctx context.Context) error {
	w.logger.Debug("Validating workflow configuration")
	
	// 检查MCP客户端
	if err := w.mcpClient.HealthCheck(ctx); err != nil {
		return fmt.Errorf("MCP client validation failed: %w", err)
	}
	
	w.logger.Info("Workflow validation completed successfully")
	return nil
}