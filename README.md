# Deer Flow Go

一个基于Go语言的智能体对话系统，使用Gin作为Web框架，集成Azure OpenAI和Tavily搜索引擎，支持天气服务查询，实现MCP协议的实时数据搜索和服务调用功能。

## 功能特点

- 基于Go Gin框架的高性能API服务
- 集成Azure OpenAI大模型能力
- 支持Tavily搜索引擎进行实时数据检索
- 集成OpenWeatherMap天气服务，支持实时天气查询和预报
- 实现MCP协议标准的客户端-服务端通信
- 完整的智能体对话工作流程
- 灵活的配置管理
- 完善的错误处理和健康检查机制
- 全面的单元测试覆盖

## 系统架构

系统由以下主要组件构成：

1. **API服务器**：基于Gin框架，提供HTTP接口
2. **LLM模块**：集成Azure OpenAI，处理自然语言理解和生成
3. **搜索模块**：集成Tavily搜索引擎，提供实时数据检索
4. **天气服务**：集成OpenWeatherMap API，提供天气查询和预报功能
5. **MCP协议**：实现标准化的客户端-服务端通信
6. **工作流引擎**：协调各组件，实现完整对话流程

## 工作流程

1. 前端发送用户问题到API服务器
2. 系统使用LLM将问题转换为MCP协议格式
3. MCP客户端根据问题类型调用相应服务：
   - 搜索相关问题：调用Tavily搜索服务获取实时数据
   - 天气相关问题：调用OpenWeatherMap获取天气信息
4. 系统将服务结果提供给LLM进行整合和润色
5. 将最终结果返回给前端展示

## 安装与配置

### 前置条件

- Go 1.21或更高版本
- Azure OpenAI API访问权限
- Tavily搜索API密钥
- OpenWeatherMap API密钥（用于天气服务）

### 安装步骤

1. 克隆仓库

```bash
git clone https://github.com/MikeyHHH/deer-flow-go.git
cd deer-flow-go
```

2. 安装依赖

```bash
go mod tidy
```

3. 配置环境变量

创建`.env`文件并设置以下变量：

```
# 服务器配置
PORT=8080
LOG_LEVEL=info

# Azure OpenAI配置
AZURE_OPENAI_ENDPOINT=https://your-endpoint.openai.azure.com
AZURE_OPENAI_API_KEY=your-api-key
AZURE_OPENAI_DEPLOYMENT=your-deployment-name
AZURE_OPENAI_API_VERSION=2023-08-01-preview
AZURE_OPENAI_TEMPERATURE=0.0

# Tavily搜索配置
TAVILY_API_KEY=your-tavily-api-key
TAVILY_MAX_RESULTS=5
TAVILY_SEARCH_DEPTH=advanced

# 天气服务配置
WEATHER_API_KEY=your-openweathermap-api-key
WEATHER_BASE_URL=https://api.openweathermap.org/data/2.5
WEATHER_TIMEOUT=10

# MCP配置
MCP_ENABLED=true
MCP_TIMEOUT=30
```

4. 编译项目

```bash
go build -o bin/deer-flow-go cmd/main.go
```

5. 运行服务

```bash
./bin/deer-flow-go
```

## API接口

### 聊天接口

- **URL**: `/api/chat`
- **方法**: POST
- **请求体**:

```json
{
  "messages": [
    {"role": "user", "content": "历史消息1"},
    {"role": "assistant", "content": "历史回复1"}
  ],
  "query": "用户当前问题"
}
```

- **响应**:

```json
{
  "response": "助手回复内容",
  "timestamp": "2023-08-01T12:34:56Z",
  "success": true
}
```

### MCP服务接口

系统支持以下MCP方法：

#### 搜索服务
- `search`: 使用Tavily进行网络搜索
- `direct_response`: 直接响应用户查询

#### 天气服务
- `get_weather`: 获取指定城市的当前天气信息
- `get_weather_forecast`: 获取指定城市的天气预报（1-5天）

**天气查询示例**:
```json
{
  "method": "get_weather",
  "params": {
    "city": "北京"
  }
}
```

**天气预报示例**:
```json
{
  "method": "get_weather_forecast",
  "params": {
    "city": "上海",
    "days": 3
  }
}
```

### 工作流状态接口

- **URL**: `/api/workflow/status`
- **方法**: GET
- **响应**:

```json
{
  "step": "ready",
  "search_data": {
    "mcp_healthy": true,
    "capabilities": {
      "enabled": true,
      "methods": ["search", "direct_response"],
      "search_engine": "tavily",
      "timeout_seconds": 30
    }
  }
}
```

## 开发指南

### 项目结构

```
/
├── cmd/                    # 命令行入口
│   └── main.go            # 主程序
├── pkg/                   # 公共包
│   ├── config/            # 配置管理
│   ├── llm/               # 大模型集成
│   ├── search/            # 搜索引擎集成
│   ├── weather/           # 天气服务集成
│   ├── mcp/               # MCP协议实现
│   ├── models/            # 数据模型
│   └── handlers/          # API处理器
├── internal/              # 内部包
│   └── workflow/          # 工作流实现
├── test/                  # 测试文件
│   ├── llm_test.go        # LLM模块测试
│   ├── search_test.go     # 搜索模块测试
│   ├── weather_mcp_test.go # 天气服务测试
│   ├── mcp_test.go        # MCP协议测试
│   ├── workflow_test.go   # 工作流测试
│   └── error_handling_test.go # 错误处理测试
├── examples/              # 使用示例
│   └── weather_example.go # 天气服务示例
├── docs/                  # 文档目录
│   └── weather-service.md # 天气服务文档
├── .env                   # 环境变量
├── .gitignore            # Git忽略文件
├── go.mod                # Go模块文件
├── go.sum                # Go依赖校验
├── README.md             # 项目文档
└── README_WEATHER.md     # 天气服务专项文档
```

## 测试

项目包含完整的单元测试覆盖：

```bash
# 运行所有测试
go test ./test/ -v

# 运行特定测试
go test ./test/ -v -run TestWeather
go test ./test/ -v -run TestMCP
go test ./test/ -v -run TestLLM

# 运行测试并查看覆盖率
go test ./test/ -v -cover
```

## 使用示例

### 天气服务示例

```bash
# 运行天气服务示例
go run examples/weather_example.go
```

该示例演示了：
- 配置加载和客户端初始化
- 健康检查
- 当前天气查询
- 天气预报查询
- 错误处理机制

### 扩展指南

#### 添加新的搜索引擎

1. 在`pkg/search/`目录下创建新的搜索引擎实现
2. 实现与`TavilyClient`类似的接口
3. 在配置中添加相应的选项
4. 在工作流中集成新的搜索引擎

#### 添加新的服务

1. 在`pkg/`目录下创建新的服务包
2. 实现服务客户端和相关方法
3. 在`pkg/mcp/client.go`中集成新服务
4. 添加相应的MCP方法支持
5. 编写单元测试

#### 修改提示词模板

可以在`pkg/llm/azure_openai.go`文件中修改以下方法中的提示词：

- `ParseQueryToMCP`: 用于将用户查询转换为MCP请求的提示词
- `FormatSearchResults`: 用于格式化搜索结果的提示词

## 依赖项

主要依赖包括：

- **Web框架**: `github.com/gin-gonic/gin` - HTTP Web框架
- **OpenAI客户端**: `github.com/sashabaranov/go-openai` - OpenAI API客户端
- **MCP协议**: `github.com/mark3labs/mcp-go` - MCP协议实现
- **配置管理**: `github.com/joho/godotenv` - 环境变量加载
- **日志**: `github.com/sirupsen/logrus` - 结构化日志
- **测试**: `github.com/stretchr/testify` - 测试工具包

## 贡献指南

1. Fork 本仓库
2. 创建特性分支 (`git checkout -b feature/AmazingFeature`)
3. 提交更改 (`git commit -m 'Add some AmazingFeature'`)
4. 推送到分支 (`git push origin feature/AmazingFeature`)
5. 打开 Pull Request

## 更多文档

- [天气服务详细文档](README_WEATHER.md)
- [天气服务技术文档](docs/weather-service.md)
- [使用示例](examples/weather_example.go)

## 许可证

MIT