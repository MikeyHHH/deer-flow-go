# 天气服务 MCP 集成

本文档介绍如何在 deer-flow-go 项目中使用天气服务 MCP 功能。

## 概述

天气服务 MCP 集成为 deer-flow-go 项目提供了天气查询和预报功能，支持通过 MCP (Model Context Protocol) 协议进行调用。

## 功能特性

- **实时天气查询**: 获取指定城市的当前天气信息
- **天气预报**: 获取1-5天的天气预报数据
- **多语言支持**: 支持中文天气信息显示
- **错误处理**: 完善的参数验证和错误处理机制
- **健康检查**: 内置服务健康检查功能

## 配置要求

### 1. 环境变量配置

在 `.env` 文件中添加以下配置：

```env
# OpenWeatherMap API 配置
WEATHER_API_KEY=your_openweathermap_api_key
WEATHER_BASE_URL=https://api.openweathermap.org/data/2.5
WEATHER_TIMEOUT=10
```

### 2. 获取 API 密钥

1. 访问 [OpenWeatherMap](https://openweathermap.org/api) 官网
2. 注册账户并获取免费的 API 密钥
3. 将 API 密钥配置到环境变量中

## MCP 方法

### 1. get_weather

获取指定城市的当前天气信息。

**参数:**
- `city` (string, 必需): 城市名称，支持中英文

**示例请求:**
```json
{
  "method": "get_weather",
  "params": {
    "city": "北京"
  }
}
```

**响应格式:**
```json
{
  "result": {
    "city": "北京",
    "temperature": 25.5,
    "feels_like": 27.2,
    "humidity": 65,
    "pressure": 1013,
    "description": "晴朗",
    "wind_speed": 3.2,
    "timestamp": "2024-01-15T10:30:00Z"
  }
}
```

### 2. get_weather_forecast

获取指定城市的天气预报信息。

**参数:**
- `city` (string, 必需): 城市名称，支持中英文
- `days` (number, 可选): 预报天数，范围1-5天，默认为1天

**示例请求:**
```json
{
  "method": "get_weather_forecast",
  "params": {
    "city": "上海",
    "days": 3
  }
}
```

**响应格式:**
```json
{
  "result": [
    {
      "date": "2024-01-15",
      "temperature": 22.1,
      "feels_like": 23.5,
      "humidity": 70,
      "pressure": 1015,
      "description": "多云",
      "wind_speed": 2.8
    },
    {
      "date": "2024-01-16",
      "temperature": 24.3,
      "feels_like": 25.1,
      "humidity": 68,
      "pressure": 1012,
      "description": "晴朗",
      "wind_speed": 3.5
    }
  ]
}
```

## 错误处理

### 常见错误码

- `-32602`: 参数错误
  - 缺少必需的 `city` 参数
  - `days` 参数超出范围（必须在1-5之间）
  - 参数格式不正确

- `-32603`: 内部错误
  - API 密钥无效或过期
  - 网络连接问题
  - 服务暂时不可用

### 错误响应示例

```json
{
  "error": {
    "code": -32602,
    "message": "Missing or invalid city parameter"
  }
}
```

## 使用示例

### 1. 启动天气 MCP 服务器

```bash
# 构建项目
go build ./cmd/weather-mcp/

# 运行天气 MCP 服务器
./weather-mcp
```

### 2. 在代码中使用

```go
package main

import (
    "context"
    "fmt"
    "log"
    
    "deer-flow-go/pkg/config"
    "deer-flow-go/pkg/mcp"
    "deer-flow-go/pkg/models"
    "deer-flow-go/pkg/search"
    "deer-flow-go/pkg/weather"
    "github.com/sirupsen/logrus"
)

func main() {
    // 加载配置
    cfg, err := config.LoadConfig()
    if err != nil {
        log.Fatal("Failed to load config:", err)
    }
    
    // 创建日志器
    logger := logrus.New()
    
    // 创建客户端
    tavilyClient := search.NewTavilyClient(&cfg.Tavily, logger)
    weatherClient := weather.NewWeatherClient(&weather.WeatherConfig{
        APIKey:  cfg.Weather.APIKey,
        BaseURL: cfg.Weather.BaseURL,
        Timeout: cfg.Weather.Timeout,
    }, logger)
    
    // 创建 MCP 客户端
    mcpClient := mcp.NewMCPClient(&cfg.MCP, tavilyClient, weatherClient, logger)
    
    // 查询天气
    ctx := context.Background()
    request := &models.MCPRequest{
        Method: "get_weather",
        Params: map[string]interface{}{
            "city": "北京",
        },
    }
    
    response, err := mcpClient.ProcessRequest(ctx, request)
    if err != nil {
        log.Fatal("Request failed:", err)
    }
    
    if response.Error != nil {
        fmt.Printf("MCP Error: %s\n", response.Error.Message)
    } else {
        fmt.Printf("Weather data: %+v\n", response.Result)
    }
}
```

## 测试

### 运行测试

```bash
# 运行所有天气相关测试
go test ./test/ -v -run TestWeather

# 运行 MCP 集成测试
go test ./test/ -v -run TestWeatherMCPIntegration

# 运行能力测试
go test ./test/ -v -run TestWeatherMCPCapabilities
```

### 健康检查

```bash
# 测试天气客户端健康状态
go test ./test/ -v -run TestWeatherClientHealthCheck
```

## 故障排除

### 1. API 密钥问题

**症状**: 收到 401 错误
**解决方案**: 
- 检查 `.env` 文件中的 `WEATHER_API_KEY` 是否正确
- 确认 API 密钥未过期
- 验证 API 密钥的使用限制

### 2. 网络连接问题

**症状**: 请求超时或连接失败
**解决方案**:
- 检查网络连接
- 增加 `WEATHER_TIMEOUT` 配置值
- 确认防火墙设置允许外部 API 访问

### 3. 城市名称问题

**症状**: 找不到城市或返回错误的天气数据
**解决方案**:
- 使用标准的城市名称（中英文均可）
- 对于有歧义的城市名，可以加上国家或地区信息
- 参考 OpenWeatherMap 的城市列表

## 性能优化

### 1. 缓存策略

考虑实现天气数据缓存以减少 API 调用：
- 当前天气数据缓存 10-15 分钟
- 天气预报数据缓存 1-2 小时

### 2. 并发控制

- 限制同时进行的 API 请求数量
- 实现请求队列和重试机制

### 3. 监控和日志

- 监控 API 调用频率和响应时间
- 记录错误和异常情况
- 设置告警机制

## 扩展功能

### 未来可能的增强功能

1. **多天气服务支持**: 集成其他天气 API 服务
2. **地理位置支持**: 基于经纬度查询天气
3. **天气告警**: 恶劣天气预警功能
4. **历史天气**: 查询历史天气数据
5. **天气图表**: 生成天气趋势图表

## 许可证

本项目遵循项目根目录下的许可证条款。

## 贡献

欢迎提交 Issue 和 Pull Request 来改进天气服务功能。