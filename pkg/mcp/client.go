package mcp

import (
	"context"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"

	"deer-flow-go/pkg/config"
	"deer-flow-go/pkg/models"
	"deer-flow-go/pkg/search"
	"deer-flow-go/pkg/weather"
)

// MCPClient MCP协议客户端
type MCPClient struct {
	config        *config.MCPConfig
	tavilyClient  *search.TavilyClient
	weatherClient *weather.WeatherClient
	logger        *logrus.Logger
}

// NewMCPClient 创建新的MCP客户端
func NewMCPClient(cfg *config.MCPConfig, tavilyClient *search.TavilyClient, weatherClient *weather.WeatherClient, logger *logrus.Logger) *MCPClient {
	return &MCPClient{
		config:        cfg,
		tavilyClient:  tavilyClient,
		weatherClient: weatherClient,
		logger:        logger,
	}
}

// ProcessRequest 处理MCP请求
func (c *MCPClient) ProcessRequest(ctx context.Context, req *models.MCPRequest) (*models.MCPResponse, error) {
	if !c.config.Enabled {
		return &models.MCPResponse{
			Error: &models.MCPError{
				Code:    -1,
				Message: "MCP is disabled",
			},
		}, nil
	}

	c.logger.WithFields(logrus.Fields{
		"method": req.Method,
	}).Debug("Processing MCP request")

	// 设置超时
	ctx, cancel := context.WithTimeout(ctx, time.Duration(c.config.Timeout)*time.Second)
	defer cancel()

	switch req.Method {
	case "search":
		return c.handleSearchRequest(ctx, req)
	case "direct_response":
		return c.handleDirectResponse(ctx, req)
	case "get_weather":
		return c.handleGetWeatherRequest(ctx, req)
	case "get_weather_forecast":
		return c.handleGetWeatherForecastRequest(ctx, req)
	default:
		return &models.MCPResponse{
			Error: &models.MCPError{
				Code:    -32601,
				Message: fmt.Sprintf("Method not found: %s", req.Method),
			},
		}, nil
	}
}

// handleSearchRequest 处理搜索请求
func (c *MCPClient) handleSearchRequest(ctx context.Context, req *models.MCPRequest) (*models.MCPResponse, error) {
	// 解析参数
	params, ok := req.Params.(map[string]interface{})
	if !ok {
		return &models.MCPResponse{
			Error: &models.MCPError{
				Code:    -32602,
				Message: "Invalid params format",
			},
		}, nil
	}

	query, ok := params["query"].(string)
	if !ok || query == "" {
		return &models.MCPResponse{
			Error: &models.MCPError{
				Code:    -32602,
				Message: "Missing or invalid query parameter",
			},
		}, nil
	}

	c.logger.WithFields(logrus.Fields{
		"query": query,
	}).Debug("Executing search via Tavily")

	// 执行搜索
	searchResp, err := c.tavilyClient.Search(ctx, query)
	if err != nil {
		c.logger.WithError(err).Error("Search failed")
		return &models.MCPResponse{
			Error: &models.MCPError{
				Code:    -32603,
				Message: fmt.Sprintf("Search failed: %v", err),
			},
		}, nil
	}

	// 清理结果
	cleanedResp := c.tavilyClient.CleanResults(searchResp)

	c.logger.WithFields(logrus.Fields{
		"results_count": len(cleanedResp.Results),
		"has_answer":    cleanedResp.Answer != "",
	}).Debug("Search completed successfully")

	return &models.MCPResponse{
		Result: cleanedResp,
	}, nil
}

// handleDirectResponse 处理直接响应请求
func (c *MCPClient) handleDirectResponse(ctx context.Context, req *models.MCPRequest) (*models.MCPResponse, error) {
	// 解析参数
	params, ok := req.Params.(map[string]interface{})
	if !ok {
		return &models.MCPResponse{
			Error: &models.MCPError{
				Code:    -32602,
				Message: "Invalid params format",
			},
		}, nil
	}

	message, ok := params["message"].(string)
	if !ok || message == "" {
		return &models.MCPResponse{
			Error: &models.MCPError{
				Code:    -32602,
				Message: "Missing or invalid message parameter",
			},
		}, nil
	}

	c.logger.WithFields(logrus.Fields{
		"message": message,
	}).Debug("Processing direct response")

	// 构建直接响应
	directResp := &models.SearchResponse{
		Query:   message,
		Answer:  message,
		Results: []models.SearchResult{},
	}

	return &models.MCPResponse{
		Result: directResp,
	}, nil
}

// handleGetWeatherRequest 处理获取天气请求
func (c *MCPClient) handleGetWeatherRequest(ctx context.Context, req *models.MCPRequest) (*models.MCPResponse, error) {
	c.logger.Debug("Processing get_weather request")

	// 解析参数
	params, ok := req.Params.(map[string]interface{})
	if !ok {
		return &models.MCPResponse{
			Error: &models.MCPError{
				Code:    -32602,
				Message: "Invalid params format",
			},
		}, nil
	}

	city, ok := params["city"].(string)
	if !ok || city == "" {
		return &models.MCPResponse{
			Error: &models.MCPError{
				Code:    -32602,
				Message: "Missing or invalid city parameter",
			},
		}, nil
	}

	// 获取天气数据
	weatherData, err := c.weatherClient.GetWeather(ctx, city)
	if err != nil {
		c.logger.WithError(err).Error("Failed to get weather data")
		return &models.MCPResponse{
			Error: &models.MCPError{
				Code:    -32603,
				Message: fmt.Sprintf("Failed to get weather data: %v", err),
			},
		}, nil
	}

	c.logger.WithField("city", city).Info("Successfully retrieved weather data")

	return &models.MCPResponse{
		Result: weatherData,
	}, nil
}

// handleGetWeatherForecastRequest 处理获取天气预报请求
func (c *MCPClient) handleGetWeatherForecastRequest(ctx context.Context, req *models.MCPRequest) (*models.MCPResponse, error) {
	c.logger.Debug("Processing get_weather_forecast request")

	// 解析参数
	params, ok := req.Params.(map[string]interface{})
	if !ok {
		return &models.MCPResponse{
			Error: &models.MCPError{
				Code:    -32602,
				Message: "Invalid params format",
			},
		}, nil
	}

	city, ok := params["city"].(string)
	if !ok || city == "" {
		return &models.MCPResponse{
			Error: &models.MCPError{
				Code:    -32602,
				Message: "Missing or invalid city parameter",
			},
		}, nil
	}

	// 解析天数参数，默认为1天
	days := 1
	if daysParam, exists := params["days"]; exists {
		if daysFloat, ok := daysParam.(float64); ok {
			days = int(daysFloat)
		}
	}

	if days <= 0 || days > 5 {
		return &models.MCPResponse{
			Error: &models.MCPError{
				Code:    -32602,
				Message: "Invalid days parameter: must be between 1 and 5",
			},
		}, nil
	}

	// 获取天气预报数据
	forecastData, err := c.weatherClient.GetForecast(ctx, city, days)
	if err != nil {
		c.logger.WithError(err).Error("Failed to get weather forecast data")
		return &models.MCPResponse{
			Error: &models.MCPError{
				Code:    -32603,
				Message: fmt.Sprintf("Failed to get weather forecast data: %v", err),
			},
		}, nil
	}

	c.logger.WithFields(logrus.Fields{
		"city": city,
		"days": days,
	}).Info("Successfully retrieved weather forecast data")

	return &models.MCPResponse{
		Result: forecastData,
	}, nil
}

// GetCapabilities 获取MCP客户端能力
func (c *MCPClient) GetCapabilities() map[string]interface{} {
	return map[string]interface{}{
		"enabled": c.config.Enabled,
		"methods": []string{"search", "direct_response", "get_weather", "get_weather_forecast"},
		"search_engine": "tavily",
		"timeout_seconds": c.config.Timeout,
	}
}

// HealthCheck 健康检查
func (c *MCPClient) HealthCheck(ctx context.Context) error {
	if !c.config.Enabled {
		return fmt.Errorf("MCP is disabled")
	}

	// 测试搜索功能
	testReq := &models.MCPRequest{
		Method: "search",
		Params: map[string]interface{}{
			"query": "test",
		},
	}

	_, err := c.ProcessRequest(ctx, testReq)
	if err != nil {
		return fmt.Errorf("MCP health check failed: %w", err)
	}

	return nil
}