# 天气服务 MCP 集成

本项目集成了天气服务功能到 MCP (Model Context Protocol) 框架中，提供了完整的天气查询和预报功能。

## 功能特性

- **实时天气查询**: 获取指定城市的当前天气信息
- **天气预报**: 获取未来1-5天的天气预报
- **MCP协议支持**: 完全兼容MCP协议规范
- **错误处理**: 完善的参数验证和错误处理机制
- **健康检查**: 内置服务健康检查功能
- **配置管理**: 灵活的配置管理系统

## 快速开始

### 1. 环境准备

#### 获取 OpenWeatherMap API 密钥

1. 访问 [OpenWeatherMap](https://openweathermap.org/api) 注册账号
2. 获取免费的 API 密钥
3. 将 API 密钥添加到环境变量或配置文件中

#### 设置环境变量

```bash
export WEATHER_API_KEY="your_openweathermap_api_key"
export WEATHER_BASE_URL="https://api.openweathermap.org/data/2.5"
export WEATHER_TIMEOUT=30
```

### 2. 配置文件

在 `config/config.yaml` 中添加天气服务配置：

```yaml
weather:
  api_key: "${WEATHER_API_KEY}"
  base_url: "${WEATHER_BASE_URL}"
  timeout: 30

mcp:
  enabled: true
  timeout: 30
  max_retries: 3
```

### 3. 运行示例

```bash
# 运行天气服务示例
go run examples/weather_example.go

# 运行测试
go test ./test/ -v -run TestWeather
```

## API 使用方法

### 获取当前天气

```go
request := &models.MCPRequest{
    Method: "get_weather",
    Params: map[string]interface{}{
        "city": "北京",
    },
}

response, err := mcpClient.ProcessRequest(ctx, request)
```

### 获取天气预报

```go
request := &models.MCPRequest{
    Method: "get_weather_forecast",
    Params: map[string]interface{}{
        "city": "上海",
        "days": float64(3), // 1-5天
    },
}

response, err := mcpClient.ProcessRequest(ctx, request)
```

## MCP 方法说明

### get_weather

获取指定城市的当前天气信息。

**参数:**
- `city` (string, 必需): 城市名称，支持中文和英文

**返回:**
```json
{
  "location": "Beijing",
  "temperature": 25.6,
  "description": "晴天",
  "humidity": 45,
  "wind_speed": 3.2,
  "timestamp": "2024-01-15 14:30:00"
}
```

### get_weather_forecast

获取指定城市的天气预报。

**参数:**
- `city` (string, 必需): 城市名称
- `days` (number, 必需): 预报天数，范围1-5

**返回:**
```json
[
  {
    "location": "Beijing",
    "temperature": 28.1,
    "description": "多云",
    "humidity": 52,
    "wind_speed": 2.8,
    "timestamp": "2024-01-16 12:00:00"
  }
]
```

## 错误处理

### 常见错误代码

- `-32602`: 无效参数
  - 缺少必需的 `city` 参数
  - `days` 参数超出范围 (1-5)
- `-32603`: 内部错误
  - API 请求失败
  - 网络连接问题
- `-32601`: 方法不存在
  - 调用了不支持的方法

### 错误响应示例

```json
{
  "error": {
    "code": -32602,
    "message": "Missing required parameter: city"
  }
}
```

## 项目结构

```
├── pkg/
│   ├── weather/           # 天气服务核心实现
│   │   └── weather.go
│   ├── mcp/              # MCP 客户端实现
│   │   └── client.go
│   ├── config/           # 配置管理
│   └── models/           # 数据模型
├── test/                 # 测试文件
│   ├── weather_mcp_test.go
│   └── mcp_test.go
├── examples/             # 使用示例
│   └── weather_example.go
├── docs/                 # 文档
│   └── weather-service.md
└── config/               # 配置文件
    └── config.yaml
```

## 测试

### 运行所有测试

```bash
go test ./test/ -v
```

### 运行天气相关测试

```bash
go test ./test/ -v -run TestWeather
```

### 运行 MCP 相关测试

```bash
go test ./test/ -v -run TestMCP
```

## 性能优化

### 缓存策略

- 天气数据缓存时间：10分钟
- 预报数据缓存时间：1小时
- 使用内存缓存减少API调用

### 请求限制

- 单个IP每分钟最多60次请求
- 建议在生产环境中实现请求队列

### 超时设置

- 默认请求超时：30秒
- 健康检查超时：15秒
- 可通过配置文件调整

## 故障排除

### 常见问题

1. **API密钥无效**
   - 检查环境变量 `WEATHER_API_KEY` 是否正确设置
   - 确认API密钥在OpenWeatherMap控制台中有效

2. **网络连接失败**
   - 检查网络连接
   - 确认防火墙设置允许访问 `api.openweathermap.org`

3. **城市名称无法识别**
   - 尝试使用英文城市名称
   - 检查城市名称拼写是否正确

### 调试模式

启用详细日志输出：

```bash
export LOG_LEVEL=debug
go run examples/weather_example.go
```

## 贡献指南

1. Fork 本项目
2. 创建功能分支 (`git checkout -b feature/amazing-feature`)
3. 提交更改 (`git commit -m 'Add some amazing feature'`)
4. 推送到分支 (`git push origin feature/amazing-feature`)
5. 创建 Pull Request

## 许可证

本项目采用 MIT 许可证 - 查看 [LICENSE](LICENSE) 文件了解详情。

## 支持

如果您遇到问题或有建议，请：

1. 查看 [文档](docs/weather-service.md)
2. 搜索现有的 [Issues](../../issues)
3. 创建新的 Issue 描述问题

## 更新日志

### v1.0.0 (2024-01-15)

- ✨ 新增天气服务MCP集成
- ✨ 支持实时天气查询
- ✨ 支持天气预报功能
- ✨ 完善的错误处理机制
- ✨ 健康检查功能
- 📝 完整的文档和示例
- 🧪 全面的测试覆盖