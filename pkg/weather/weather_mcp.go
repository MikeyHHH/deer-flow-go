package weather

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/sirupsen/logrus"
)

// WeatherMCPServer MCP天气服务器
type WeatherMCPServer struct {
	weatherClient *WeatherClient
	logger        *logrus.Logger
	server        *server.MCPServer
}



// NewWeatherMCPServer 创建新的MCP天气服务器
func NewWeatherMCPServer(weatherClient *WeatherClient, logger *logrus.Logger) *WeatherMCPServer {
	mcpServer := server.NewMCPServer(
		"weather-server",
		"1.0.0",
	)

	weatherMCP := &WeatherMCPServer{
		weatherClient: weatherClient,
		logger:        logger,
		server:        mcpServer,
	}

	// 注册天气工具
	weatherMCP.registerTools()

	return weatherMCP
}

// registerTools 注册MCP工具
func (w *WeatherMCPServer) registerTools() {
	// 注册获取当前天气工具
	getWeatherTool := mcp.NewTool("get_weather",
		mcp.WithDescription("获取指定城市的当前天气信息"),
		mcp.WithString("city",
			mcp.Required(),
			mcp.Description("城市名称，例如：北京、上海、New York"),
		),
	)
	w.server.AddTool(getWeatherTool, w.handleGetWeather)

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
	w.server.AddTool(getForecastTool, w.handleGetWeatherForecast)
}

// handleGetWeather 处理获取当前天气请求
func (w *WeatherMCPServer) handleGetWeather(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	w.logger.WithFields(logrus.Fields{
		"tool": "get_weather",
	}).Debug("Processing get_weather request")

	// 解析请求参数
	city, err := request.RequireString("city")
	if err != nil {
		w.logger.WithError(err).Error("Failed to parse city parameter")
		return mcp.NewToolResultError(fmt.Sprintf("参数解析失败: %v", err)), nil
	}

	if city == "" {
		return mcp.NewToolResultError("城市名称不能为空"), nil
	}

	// 获取天气数据
	weatherData, err := w.weatherClient.GetWeather(ctx, city)
	if err != nil {
		w.logger.WithError(err).Error("Failed to get weather data")
		return mcp.NewToolResultError(fmt.Sprintf("获取天气信息失败: %v", err)), nil
	}

	// 格式化响应
	weatherText := fmt.Sprintf("🌤️ %s 当前天气:\n" +
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

	return mcp.NewToolResultText(weatherText), nil
}

// handleGetWeatherForecast 处理获取天气预报请求
func (w *WeatherMCPServer) handleGetWeatherForecast(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	w.logger.WithFields(logrus.Fields{
		"tool": "get_weather_forecast",
	}).Debug("Processing get_weather_forecast request")

	// 解析请求参数
	city, err := request.RequireString("city")
	if err != nil {
		w.logger.WithError(err).Error("Failed to parse city parameter")
		return mcp.NewToolResultError(fmt.Sprintf("参数解析失败: %v", err)), nil
	}

	if city == "" {
		return mcp.NewToolResultError("城市名称不能为空"), nil
	}

	days := request.GetInt("days", 1) // 默认1天
	if days <= 0 {
		days = 1
	}

	// 获取天气预报数据
	forecastData, err := w.weatherClient.GetForecast(ctx, city, days)
	if err != nil {
		w.logger.WithError(err).Error("Failed to get weather forecast data")
		return mcp.NewToolResultError(fmt.Sprintf("获取天气预报失败: %v", err)), nil
	}

	// 格式化预报响应
	forecastText := fmt.Sprintf("📅 %s %d天天气预报:\n\n", city, days)
	for i, weather := range forecastData {
		forecastText += fmt.Sprintf("第%d天:\n" +
			"🌡️ 温度: %.1f°C\n" +
			"☁️ 天气: %s\n" +
			"💧 湿度: %d%%\n" +
			"💨 风速: %.1f m/s\n",
			i+1,
			weather.Temperature,
			weather.Description,
			weather.Humidity,
			weather.WindSpeed)
		if i < len(forecastData)-1 {
			forecastText += "\n"
		}
	}

	return mcp.NewToolResultText(forecastText), nil
}

// GetServer 获取MCP服务器实例
func (w *WeatherMCPServer) GetServer() *server.MCPServer {
	return w.server
}

// Start 启动MCP服务器
func (w *WeatherMCPServer) Start(ctx context.Context) error {
	w.logger.Info("Starting Weather MCP Server")
	return server.ServeStdio(w.server)
}

// GetCapabilities 获取天气服务能力
func (w *WeatherMCPServer) GetCapabilities() map[string]interface{} {
	return map[string]interface{}{
		"tools": []string{"get_weather", "get_weather_forecast"},
		"description": "天气服务MCP工具，支持获取当前天气和天气预报",
		"version": "1.0.0",
	}
}