package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/sirupsen/logrus"

	"deer-flow-go/pkg/config"
	"deer-flow-go/pkg/mcp"
	"deer-flow-go/pkg/models"
	"deer-flow-go/pkg/search"
	"deer-flow-go/pkg/weather"
)

// WeatherExample 天气服务使用示例
func main() {
	fmt.Println("=== 天气服务 MCP 集成示例 ===")

	// 1. 加载配置
	fmt.Println("\n1. 加载配置...")
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatal("Failed to load config:", err)
	}
	fmt.Println("✓ 配置加载成功")

	// 2. 创建日志器
	logger := logrus.New()
	logger.SetLevel(logrus.InfoLevel)

	// 3. 创建客户端
	fmt.Println("\n2. 初始化客户端...")
	tavilyClient := search.NewTavilyClient(&cfg.Tavily, logger)

	// 创建天气客户端配置
	weatherConfig := &weather.WeatherConfig{
		APIKey:  cfg.Weather.APIKey,
		BaseURL: cfg.Weather.BaseURL,
		Timeout: cfg.Weather.Timeout,
	}
	weatherClient := weather.NewWeatherClient(weatherConfig, logger)

	// 创建 MCP 客户端
	mcpClient := mcp.NewMCPClient(&cfg.MCP, tavilyClient, weatherClient, logger)
	fmt.Println("✓ 客户端初始化完成")

	// 4. 测试服务能力
	fmt.Println("\n3. 检查服务能力...")
	capabilities := mcpClient.GetCapabilities()
	fmt.Printf("✓ 支持的方法: %v\n", capabilities["methods"])

	// 5. 健康检查
	fmt.Println("\n4. 执行健康检查...")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err = mcpClient.HealthCheck(ctx)
	if err != nil {
		fmt.Printf("⚠ 健康检查失败: %v\n", err)
	} else {
		fmt.Println("✓ 健康检查通过")
	}

	// 6. 天气查询示例
	fmt.Println("\n5. 天气查询示例...")
	cities := []string{"北京", "上海", "广州", "深圳"}

	for _, city := range cities {
		fmt.Printf("\n查询 %s 的天气信息:\n", city)
		err := queryWeather(mcpClient, city)
		if err != nil {
			fmt.Printf("❌ 查询失败: %v\n", err)
		}
		time.Sleep(1 * time.Second) // 避免请求过于频繁
	}

	// 7. 天气预报示例
	fmt.Println("\n6. 天气预报示例...")
	err = queryWeatherForecast(mcpClient, "北京", 3)
	if err != nil {
		fmt.Printf("❌ 预报查询失败: %v\n", err)
	}

	// 8. 错误处理示例
	fmt.Println("\n7. 错误处理示例...")
	demonstateErrorHandling(mcpClient)

	fmt.Println("\n=== 示例完成 ===")
}

// queryWeather 查询指定城市的当前天气
func queryWeather(mcpClient *mcp.MCPClient, city string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	request := &models.MCPRequest{
		Method: "get_weather",
		Params: map[string]interface{}{
			"city": city,
		},
	}

	response, err := mcpClient.ProcessRequest(ctx, request)
	if err != nil {
		return fmt.Errorf("请求处理失败: %w", err)
	}

	if response.Error != nil {
		return fmt.Errorf("MCP 错误: %s", response.Error.Message)
	}

	// 解析天气数据
	if weatherData, ok := response.Result.(*weather.WeatherData); ok {
		fmt.Printf("  城市: %s\n", weatherData.Location)
		fmt.Printf("  温度: %.1f°C\n", weatherData.Temperature)
		fmt.Printf("  湿度: %d%%\n", weatherData.Humidity)
		fmt.Printf("  天气: %s\n", weatherData.Description)
		fmt.Printf("  风速: %.1f m/s\n", weatherData.WindSpeed)
		fmt.Printf("  时间: %s\n", weatherData.Timestamp)
		fmt.Println("  ✓ 查询成功")
	} else {
		fmt.Printf("  ✓ 响应数据: %+v\n", response.Result)
	}

	return nil
}

// queryWeatherForecast 查询指定城市的天气预报
func queryWeatherForecast(mcpClient *mcp.MCPClient, city string, days int) error {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	request := &models.MCPRequest{
		Method: "get_weather_forecast",
		Params: map[string]interface{}{
			"city": city,
			"days": float64(days),
		},
	}

	response, err := mcpClient.ProcessRequest(ctx, request)
	if err != nil {
		return fmt.Errorf("请求处理失败: %w", err)
	}

	if response.Error != nil {
		return fmt.Errorf("MCP 错误: %s", response.Error.Message)
	}

	fmt.Printf("\n%s 未来 %d 天天气预报:\n", city, days)

	// 解析预报数据
	if forecastData, ok := response.Result.([]*weather.WeatherData); ok {
		for i, data := range forecastData {
			fmt.Printf("  第 %d 天 (%s):\n", i+1, data.Timestamp[:10])
			fmt.Printf("    温度: %.1f°C\n", data.Temperature)
			fmt.Printf("    湿度: %d%%\n", data.Humidity)
			fmt.Printf("    天气: %s, 风速: %.1f m/s\n", data.Description, data.WindSpeed)
		}
		fmt.Println("  ✓ 预报查询成功")
	} else {
		fmt.Printf("  ✓ 响应数据: %+v\n", response.Result)
	}

	return nil
}

// demonstateErrorHandling 演示错误处理
func demonstateErrorHandling(mcpClient *mcp.MCPClient) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 测试缺少城市参数
	fmt.Println("\n测试缺少城市参数:")
	request := &models.MCPRequest{
		Method: "get_weather",
		Params: map[string]interface{}{},
	}

	response, err := mcpClient.ProcessRequest(ctx, request)
	if err != nil {
		fmt.Printf("  ❌ 请求错误: %v\n", err)
	} else if response.Error != nil {
		fmt.Printf("  ✓ 正确处理参数错误: %s\n", response.Error.Message)
	}

	// 测试无效的天数参数
	fmt.Println("\n测试无效的天数参数:")
	request = &models.MCPRequest{
		Method: "get_weather_forecast",
		Params: map[string]interface{}{
			"city": "北京",
			"days": float64(10), // 超出范围
		},
	}

	response, err = mcpClient.ProcessRequest(ctx, request)
	if err != nil {
		fmt.Printf("  ❌ 请求错误: %v\n", err)
	} else if response.Error != nil {
		fmt.Printf("  ✓ 正确处理参数错误: %s\n", response.Error.Message)
	}

	// 测试无效的方法
	fmt.Println("\n测试无效的方法:")
	request = &models.MCPRequest{
		Method: "invalid_method",
		Params: map[string]interface{}{},
	}

	response, err = mcpClient.ProcessRequest(ctx, request)
	if err != nil {
		fmt.Printf("  ❌ 请求错误: %v\n", err)
	} else if response.Error != nil {
		fmt.Printf("  ✓ 正确处理方法错误: %s\n", response.Error.Message)
	}
}