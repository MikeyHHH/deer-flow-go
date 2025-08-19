package main

import (
	"context"
	"fmt"
	"log"

	"deer-flow-go/pkg/config"
	"deer-flow-go/pkg/search"
	"deer-flow-go/pkg/weather"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/sirupsen/logrus"
)

func main() {
	// 加载配置
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// 创建日志记录器
	logger := logrus.New()
	logger.SetLevel(logrus.InfoLevel)

	// 初始化服务客户端
	tavilyClient := search.NewTavilyClient(&cfg.Tavily, logger)
	// 转换配置类型
	weatherConfig := &weather.WeatherConfig{
		APIKey:  cfg.Weather.APIKey,
		BaseURL: cfg.Weather.BaseURL,
		Timeout: cfg.Weather.Timeout,
	}
	weatherClient := weather.NewWeatherClient(weatherConfig, logger)

	// 创建统一的MCP服务器
	mcpServer := server.NewMCPServer("unified-server", "1.0.0")

	// 注册天气工具
	registerWeatherTools(mcpServer, weatherClient, logger)

	// 注册搜索工具
	registerSearchTools(mcpServer, tavilyClient, logger)

	// 启动统一的MCP服务器
	logger.Info("Starting unified MCP server with weather and search tools...")
	if err := server.ServeStdio(mcpServer); err != nil {
		log.Fatalf("Failed to start MCP server: %v", err)
	}
}

// registerWeatherTools 注册天气相关工具
func registerWeatherTools(mcpServer *server.MCPServer, weatherClient *weather.WeatherClient, logger *logrus.Logger) {
	// 注册获取当前天气工具
	getWeatherTool := mcp.NewTool("get_weather",
		mcp.WithDescription("获取指定城市的当前天气信息"),
		mcp.WithString("city",
			mcp.Required(),
			mcp.Description("城市名称，例如：北京、上海、New York"),
		),
	)
	mcpServer.AddTool(getWeatherTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleGetWeather(ctx, request, weatherClient, logger)
	})

	// 注册获取天气预报工具
	getForecastTool := mcp.NewTool("get_weather_forecast",
		mcp.WithDescription("获取指定城市的天气预报信息"),
		mcp.WithString("city",
			mcp.Required(),
			mcp.Description("城市名称，例如：北京、上海、New York"),
		),
		mcp.WithNumber("days",
			mcp.Description("预报天数，默认为1天"),
		),
	)
	mcpServer.AddTool(getForecastTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleGetWeatherForecast(ctx, request, weatherClient, logger)
	})
}

// registerSearchTools 注册搜索相关工具
func registerSearchTools(mcpServer *server.MCPServer, searchClient *search.TavilyClient, logger *logrus.Logger) {
	// 注册搜索工具
	searchTool := mcp.NewTool("search",
		mcp.WithDescription("搜索互联网信息，返回相关的搜索结果"),
		mcp.WithString("query",
			mcp.Required(),
			mcp.Description("搜索查询关键词"),
		),
		mcp.WithNumber("max_results",
			mcp.Description("最大返回结果数量，默认为5"),
		),
	)
	mcpServer.AddTool(searchTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleSearch(ctx, request, searchClient, logger)
	})
}

// handleGetWeather 处理获取当前天气请求
func handleGetWeather(ctx context.Context, request mcp.CallToolRequest, weatherClient *weather.WeatherClient, logger *logrus.Logger) (*mcp.CallToolResult, error) {
	logger.WithFields(logrus.Fields{
		"tool": "get_weather",
	}).Debug("Processing get_weather request")

	// 解析请求参数
	city, err := request.RequireString("city")
	if err != nil {
		logger.WithError(err).Error("Failed to parse city parameter")
		return mcp.NewToolResultError(fmt.Sprintf("参数解析失败: %v", err)), nil
	}

	if city == "" {
		return mcp.NewToolResultError("城市名称不能为空"), nil
	}

	// 获取天气数据
	weatherData, err := weatherClient.GetWeather(ctx, city)
	if err != nil {
		logger.WithError(err).Error("Failed to get weather data")
		return mcp.NewToolResultError(fmt.Sprintf("获取天气信息失败: %v", err)), nil
	}

	// 格式化响应
	weatherText := fmt.Sprintf("🌤️ %s 当前天气:\n"+
		"🌡️ 温度: %.1f°C\n"+
		"☁️ 天气: %s\n"+
		"💧 湿度: %d%%\n"+
		"💨 风速: %.1f m/s\n"+
		"⏰ 更新时间: %s",
		weatherData.Location,
		weatherData.Temperature,
		weatherData.Description,
		weatherData.Humidity,
		weatherData.WindSpeed,
		weatherData.Timestamp,
	)

	return mcp.NewToolResultText(weatherText), nil
}

// handleGetWeatherForecast 处理获取天气预报请求
func handleGetWeatherForecast(ctx context.Context, request mcp.CallToolRequest, weatherClient *weather.WeatherClient, logger *logrus.Logger) (*mcp.CallToolResult, error) {
	logger.WithFields(logrus.Fields{
		"tool": "get_weather_forecast",
	}).Debug("Processing get_weather_forecast request")

	// 解析请求参数
	city, err := request.RequireString("city")
	if err != nil {
		logger.WithError(err).Error("Failed to parse city parameter")
		return mcp.NewToolResultError(fmt.Sprintf("参数解析失败: %v", err)), nil
	}

	if city == "" {
		return mcp.NewToolResultError("城市名称不能为空"), nil
	}

	// 解析天数参数（可选，默认为1天）
	days := request.GetInt("days", 1) // 默认1天
	if days <= 0 || days > 5 {
		days = 1 // 限制在1-5天范围内
	}

	// 获取天气预报数据
	forecastData, err := weatherClient.GetForecast(ctx, city, days)
	if err != nil {
		logger.WithError(err).Error("Failed to get weather forecast data")
		return mcp.NewToolResultError(fmt.Sprintf("获取天气预报失败: %v", err)), nil
	}

	// 格式化响应
	forecastText := fmt.Sprintf("📅 %s %d天天气预报:\n\n", city, len(forecastData))
	for i, forecast := range forecastData {
		forecastText += fmt.Sprintf("第%d天:\n", i+1)
		forecastText += fmt.Sprintf("🌡️ 温度: %.1f°C\n", forecast.Temperature)
		forecastText += fmt.Sprintf("☁️ 天气: %s\n", forecast.Description)
		forecastText += fmt.Sprintf("💧 湿度: %d%%\n", forecast.Humidity)
		forecastText += fmt.Sprintf("💨 风速: %.1f m/s\n", forecast.WindSpeed)
		if i < len(forecastData)-1 {
			forecastText += "\n"
		}
	}

	return mcp.NewToolResultText(forecastText), nil
}

// handleSearch 处理搜索请求
func handleSearch(ctx context.Context, request mcp.CallToolRequest, searchClient *search.TavilyClient, logger *logrus.Logger) (*mcp.CallToolResult, error) {
	logger.WithFields(logrus.Fields{
		"tool": "search",
	}).Debug("Processing search request")

	// 解析请求参数
	query, err := request.RequireString("query")
	if err != nil {
		logger.WithError(err).Error("Failed to parse query parameter")
		return mcp.NewToolResultError(fmt.Sprintf("参数解析失败: %v", err)), nil
	}

	if query == "" {
		return mcp.NewToolResultError("搜索查询不能为空"), nil
	}

	// 执行搜索
	searchResults, err := searchClient.Search(ctx, query)
	if err != nil {
		logger.WithError(err).Error("Failed to perform search")
		return mcp.NewToolResultError(fmt.Sprintf("搜索失败: %v", err)), nil
	}

	// 格式化搜索结果
	resultText := fmt.Sprintf("🔍 搜索结果 \"%s\":\n\n", query)
	for i, result := range searchResults.Results {
		resultText += fmt.Sprintf("%d. **%s**\n", i+1, result.Title)
		resultText += fmt.Sprintf("   📄 %s\n", result.Content)
		resultText += fmt.Sprintf("   🔗 %s\n", result.URL)
		if i < len(searchResults.Results)-1 {
			resultText += "\n"
		}
	}

	return mcp.NewToolResultText(resultText), nil
}
