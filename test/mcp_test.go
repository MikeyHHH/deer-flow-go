package test

import (
	"context"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"deer-flow-go/pkg/config"
	"deer-flow-go/pkg/mcp"
	"deer-flow-go/pkg/models"
	"deer-flow-go/pkg/search"
	"deer-flow-go/pkg/weather"
)

// TestMCPClient 测试MCP客户端功能
func TestMCPClient(t *testing.T) {
	// 加载配置
	cfg, err := config.LoadConfig()
	require.NoError(t, err, "Failed to load config")

	// 创建日志器
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	// 创建Tavily客户端
	tavilyClient := search.NewTavilyClient(&cfg.Tavily, logger)
	require.NotNil(t, tavilyClient, "Failed to create Tavily client")

	// 创建天气客户端
	weatherConfig := &weather.WeatherConfig{
		APIKey:  "test-key",
		BaseURL: "https://api.openweathermap.org/data/2.5",
		Timeout: 10,
	}
	weatherClient := weather.NewWeatherClient(weatherConfig, logger)
	require.NotNil(t, weatherClient, "Failed to create Weather client")

	// 创建MCP客户端
	mcpClient := mcp.NewMCPClient(&cfg.MCP, tavilyClient, weatherClient, logger)
	require.NotNil(t, mcpClient, "Failed to create MCP client")

	// 测试搜索请求处理
	t.Run("Search Request", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		// 创建搜索请求
		req := &models.MCPRequest{
			Method: "search",
			Params: map[string]interface{}{
				"query":        "Go语言并发编程",
				"max_results":  5,
				"search_depth": "advanced",
			},
		}

		// 处理请求
		resp, err := mcpClient.ProcessRequest(ctx, req)
		require.NoError(t, err, "Failed to process search request")
		require.NotNil(t, resp, "Response should not be nil")
		assert.Nil(t, resp.Error, "Response should not have error")
		assert.NotNil(t, resp.Result, "Response should have result")

		// 验证搜索结果
		searchResult, ok := resp.Result.(*models.SearchResponse)
		require.True(t, ok, "Result should be SearchResponse type")
		assert.Equal(t, "Go语言并发编程", searchResult.Query, "Query should match")
		assert.NotEmpty(t, searchResult.Results, "Results should not be empty")

		t.Logf("Search request processed successfully: %d results found", len(searchResult.Results))
		if searchResult.Answer != "" {
			t.Logf("Answer: %s", searchResult.Answer)
		}
	})

	// 测试直接响应请求处理
	t.Run("Direct Response Request", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		// 创建直接响应请求
		req := &models.MCPRequest{
			Method: "direct_response",
			Params: map[string]interface{}{
				"message": "你好，这是一个测试消息",
			},
		}

		// 处理请求
		resp, err := mcpClient.ProcessRequest(ctx, req)
		require.NoError(t, err, "Failed to process direct response request")
		require.NotNil(t, resp, "Response should not be nil")
		assert.Nil(t, resp.Error, "Response should not have error")
		assert.NotNil(t, resp.Result, "Response should have result")

		// 验证直接响应结果
		directResult, ok := resp.Result.(*models.SearchResponse)
		require.True(t, ok, "Result should be SearchResponse type")
		assert.Equal(t, "你好，这是一个测试消息", directResult.Query, "Query should match message")
		assert.Equal(t, "你好，这是一个测试消息", directResult.Answer, "Answer should match message")
		assert.Empty(t, directResult.Results, "Results should be empty for direct response")

		t.Logf("Direct response processed successfully: %s", directResult.Answer)
	})

	// 测试无效方法处理
	t.Run("Invalid Method", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// 创建无效方法请求
		req := &models.MCPRequest{
			Method: "invalid_method",
			Params: map[string]interface{}{},
		}

		// 处理请求
		resp, err := mcpClient.ProcessRequest(ctx, req)
		require.NoError(t, err, "Should not return error for invalid method")
		require.NotNil(t, resp, "Response should not be nil")
		assert.NotNil(t, resp.Error, "Response should have error")
		assert.Equal(t, -32601, resp.Error.Code, "Error code should be method not found")
		assert.Contains(t, resp.Error.Message, "Method not found", "Error message should mention method not found")

		t.Logf("Invalid method handled correctly: %s", resp.Error.Message)
	})

	// 测试无效参数处理
	t.Run("Invalid Parameters", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// 创建无效参数的搜索请求
		req := &models.MCPRequest{
			Method: "search",
			Params: map[string]interface{}{
				"invalid_param": "test",
				// 缺少必需的query参数
			},
		}

		// 处理请求
		resp, err := mcpClient.ProcessRequest(ctx, req)
		require.NoError(t, err, "Should not return error for invalid params")
		require.NotNil(t, resp, "Response should not be nil")
		assert.NotNil(t, resp.Error, "Response should have error")
		assert.Equal(t, -32602, resp.Error.Code, "Error code should be invalid params")
		assert.Contains(t, resp.Error.Message, "Missing or invalid query parameter", "Error message should mention missing query")

		t.Logf("Invalid parameters handled correctly: %s", resp.Error.Message)
	})
}

// TestMCPClientCapabilities 测试MCP客户端能力
func TestMCPClientCapabilities(t *testing.T) {
	// 加载配置
	cfg, err := config.LoadConfig()
	require.NoError(t, err, "Failed to load config")

	// 创建日志器
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	// 创建Tavily客户端
	tavilyClient := search.NewTavilyClient(&cfg.Tavily, logger)
	require.NotNil(t, tavilyClient, "Failed to create Tavily client")

	// 创建天气客户端
	weatherConfig := &weather.WeatherConfig{
		APIKey:  "test-key",
		BaseURL: "https://api.openweathermap.org/data/2.5",
		Timeout: 10,
	}
	weatherClient := weather.NewWeatherClient(weatherConfig, logger)
	require.NotNil(t, weatherClient, "Failed to create Weather client")

	// 创建MCP客户端
	mcpClient := mcp.NewMCPClient(&cfg.MCP, tavilyClient, weatherClient, logger)
	require.NotNil(t, mcpClient, "Failed to create MCP client")

	// 获取能力信息
	capabilities := mcpClient.GetCapabilities()
	require.NotNil(t, capabilities, "Capabilities should not be nil")

	// 验证能力信息
	assert.Equal(t, cfg.MCP.Enabled, capabilities["enabled"], "Enabled status should match config")
	assert.Equal(t, cfg.MCP.Timeout, capabilities["timeout_seconds"], "Timeout should match config")
	assert.Equal(t, "tavily", capabilities["search_engine"], "Search engine should be tavily")

	// 验证支持的方法
	methods, ok := capabilities["methods"].([]string)
	require.True(t, ok, "Methods should be string array")
	assert.Contains(t, methods, "search", "Should support search method")
	assert.Contains(t, methods, "direct_response", "Should support direct_response method")

	t.Logf("MCP capabilities: %+v", capabilities)
}

// TestMCPClientHealthCheck 测试MCP客户端健康检查
func TestMCPClientHealthCheck(t *testing.T) {
	// 加载配置
	cfg, err := config.LoadConfig()
	require.NoError(t, err, "Failed to load config")

	// 创建日志器
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	// 创建Tavily客户端
	tavilyClient := search.NewTavilyClient(&cfg.Tavily, logger)
	require.NotNil(t, tavilyClient, "Failed to create Tavily client")

	// 创建天气客户端
	weatherConfig := &weather.WeatherConfig{
		APIKey:  "test-key",
		BaseURL: "https://api.openweathermap.org/data/2.5",
		Timeout: 10,
	}
	weatherClient := weather.NewWeatherClient(weatherConfig, logger)
	require.NotNil(t, weatherClient, "Failed to create Weather client")

	// 创建MCP客户端
	mcpClient := mcp.NewMCPClient(&cfg.MCP, tavilyClient, weatherClient, logger)
	require.NotNil(t, mcpClient, "Failed to create MCP client")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 执行健康检查
	err = mcpClient.HealthCheck(ctx)
	assert.NoError(t, err, "Health check should pass")

	t.Log("MCP client health check passed")
}

// TestMCPClientDisabled 测试MCP客户端禁用状态
func TestMCPClientDisabled(t *testing.T) {
	// 创建禁用MCP的配置
	disabledConfig := &config.MCPConfig{
		Enabled: false,
		Timeout: 60,
	}

	// 创建日志器
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	// 创建Tavily客户端（使用默认配置）
	cfg, err := config.LoadConfig()
	require.NoError(t, err, "Failed to load config")
	tavilyClient := search.NewTavilyClient(&cfg.Tavily, logger)
	require.NotNil(t, tavilyClient, "Failed to create Tavily client")

	// 创建天气客户端
	weatherConfig := &weather.WeatherConfig{
		APIKey:  "test-key",
		BaseURL: "https://api.openweathermap.org/data/2.5",
		Timeout: 10,
	}
	weatherClient := weather.NewWeatherClient(weatherConfig, logger)
	require.NotNil(t, weatherClient, "Failed to create Weather client")

	// 创建禁用的MCP客户端
	mcpClient := mcp.NewMCPClient(disabledConfig, tavilyClient, weatherClient, logger)
	require.NotNil(t, mcpClient, "Failed to create MCP client")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 测试禁用状态下的请求处理
	req := &models.MCPRequest{
		Method: "search",
		Params: map[string]interface{}{
			"query": "test",
		},
	}

	resp, err := mcpClient.ProcessRequest(ctx, req)
	require.NoError(t, err, "Should not return error when disabled")
	require.NotNil(t, resp, "Response should not be nil")
	assert.NotNil(t, resp.Error, "Response should have error when disabled")
	assert.Equal(t, -1, resp.Error.Code, "Error code should be -1 for disabled")
	assert.Contains(t, resp.Error.Message, "MCP is disabled", "Error message should mention disabled")

	// 测试禁用状态下的健康检查
	err = mcpClient.HealthCheck(ctx)
	assert.Error(t, err, "Health check should fail when disabled")
	assert.Contains(t, err.Error(), "MCP is disabled", "Error should mention disabled")

	t.Log("MCP client disabled state handled correctly")
}