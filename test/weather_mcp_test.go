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

// TestWeatherMCPIntegration 测试天气服务MCP集成
func TestWeatherMCPIntegration(t *testing.T) {
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

	// 测试天气查询功能
	t.Run("Get Weather Request", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		// 构造天气查询请求
		request := &models.MCPRequest{
			Method: "get_weather",
			Params: map[string]interface{}{
				"city": "Beijing",
			},
		}

		// 处理请求
		response, err := mcpClient.ProcessRequest(ctx, request)
		
		// 由于使用测试API密钥，可能会失败，但应该有适当的错误处理
		if err != nil {
			t.Logf("Weather request failed as expected with test API key: %v", err)
			assert.Contains(t, err.Error(), "weather", "Error should be weather-related")
		} else {
			require.NotNil(t, response, "Response should not be nil")
			t.Logf("Weather response: %+v", response)
		}
	})

	// 测试天气预报功能
	t.Run("Get Weather Forecast Request", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		// 构造天气预报请求
		request := &models.MCPRequest{
			Method: "get_weather_forecast",
			Params: map[string]interface{}{
				"city": "Shanghai",
				"days": 3,
			},
		}

		// 处理请求
		response, err := mcpClient.ProcessRequest(ctx, request)
		
		// 由于使用测试API密钥，可能会失败，但应该有适当的错误处理
		if err != nil {
			t.Logf("Weather forecast request failed as expected with test API key: %v", err)
			assert.Contains(t, err.Error(), "weather", "Error should be weather-related")
		} else {
			require.NotNil(t, response, "Response should not be nil")
			t.Logf("Weather forecast response: %+v", response)
		}
	})

	// 测试无效参数处理
	t.Run("Invalid Parameters", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		// 测试缺少城市参数
		request := &models.MCPRequest{
			Method: "get_weather",
			Params: map[string]interface{}{},
		}

		response, err := mcpClient.ProcessRequest(ctx, request)
		require.NoError(t, err, "ProcessRequest should not return Go error")
		require.NotNil(t, response, "Response should not be nil")
		require.NotNil(t, response.Error, "Response should contain MCP error")
		assert.Contains(t, response.Error.Message, "city", "Error should mention missing city parameter")
	})

	// 测试天气预报无效天数
	t.Run("Invalid Forecast Days", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		// 测试无效天数（超出范围）
		request := &models.MCPRequest{
			Method: "get_weather_forecast",
			Params: map[string]interface{}{
				"city": "Beijing",
				"days": float64(10), // 超出1-5天的范围
			},
		}

		response, err := mcpClient.ProcessRequest(ctx, request)
		require.NoError(t, err, "ProcessRequest should not return Go error")
		require.NotNil(t, response, "Response should not be nil")
		require.NotNil(t, response.Error, "Response should contain MCP error")
		assert.Contains(t, response.Error.Message, "days", "Error should mention invalid days parameter")
	})
}

// TestWeatherMCPCapabilities 测试天气服务能力声明
func TestWeatherMCPCapabilities(t *testing.T) {
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

	// 验证天气相关方法是否包含在能力列表中
	methods, ok := capabilities["methods"].([]string)
	require.True(t, ok, "Methods should be a string slice")

	assert.Contains(t, methods, "get_weather", "Should support get_weather method")
	assert.Contains(t, methods, "get_weather_forecast", "Should support get_weather_forecast method")

	t.Logf("MCP capabilities with weather support: %+v", capabilities)
}

// TestWeatherClientHealthCheck 测试天气客户端健康检查
func TestWeatherClientHealthCheck(t *testing.T) {
	// 创建日志器
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	// 创建天气客户端
	weatherConfig := &weather.WeatherConfig{
		APIKey:  "test-key",
		BaseURL: "https://api.openweathermap.org/data/2.5",
		Timeout: 10,
	}
	weatherClient := weather.NewWeatherClient(weatherConfig, logger)
	require.NotNil(t, weatherClient, "Failed to create Weather client")

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// 执行健康检查
	err := weatherClient.HealthCheck(ctx)
	
	// 由于使用测试API密钥，健康检查可能会失败，但应该有适当的错误处理
	if err != nil {
		t.Logf("Weather client health check failed as expected with test API key: %v", err)
		assert.Contains(t, err.Error(), "weather", "Error should be weather-related")
	} else {
		t.Log("Weather client health check passed")
	}
}