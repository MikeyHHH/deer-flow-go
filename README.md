# Deer Flow Go - MCP智能代理系统

一个基于Model Context Protocol (MCP)的智能代理系统，提供天气查询和搜索功能的RESTful API服务。

## 🏗️ 系统架构

### 整体架构图

```
┌─────────────────┐    HTTP API     ┌─────────────────┐
│   客户端应用     │ ──────────────► │   API 服务器     │
│  (Web/Mobile)   │                 │  (cmd/main.go)  │
└─────────────────┘                 └─────────────────┘
                                             │
                                             │ 进程间通信
                                             │ (JSON-RPC 2.0)
                                             ▼
                                    ┌─────────────────┐
                                    │   MCP 服务器     │
                                    │ (cmd/server)    │
                                    └─────────────────┘
                                             │
                                             │ 外部API调用
                                             ▼
                            ┌─────────────────┬─────────────────┐
                            │   天气服务API    │   搜索服务API    │
                            │  (Weather API)  │  (Tavily API)   │
                            └─────────────────┴─────────────────┘
```

### 核心组件

1. **API服务器** (`cmd/main.go`): 提供HTTP RESTful接口
2. **MCP客户端** (`pkg/mcp/mcp_client.go`): 负责与MCP服务器通信
3. **MCP服务器** (`cmd/server/main.go`): 实现MCP协议，处理工具调用
4. **工作流引擎** (`internal/workflow/agent.go`): 协调LLM和MCP调用
5. **外部服务集成**: 天气API和搜索API

## 📊 数据流程详解

### 完整请求处理流程

```
1. HTTP请求接收
   │
   ▼
2. 请求解析与验证
   │
   ▼
3. LLM查询解析
   │ (将自然语言转换为结构化MCP请求)
   ▼
4. MCP请求构建
   │ (JSON-RPC 2.0格式)
   ▼
5. 进程间通信
   │ (通过stdin/stdout管道)
   ▼
6. MCP服务器处理
   │ (工具调用: 天气/搜索)
   ▼
7. 外部API调用
   │ (Weather API / Tavily Search API)
   ▼
8. 响应数据处理
   │ (格式化和解析)
   ▼
9. JSON-RPC响应
   │
   ▼
10. HTTP响应返回
```

### 数据传输格式

#### HTTP API请求
```json
{
  "query": "北京今天的天气怎么样？",
  "user_id": "user123"
}
```

#### MCP JSON-RPC请求
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "tools/call",
  "params": {
    "name": "get_weather",
    "arguments": {
      "location": "北京",
      "date": "today"
    }
  }
}
```

#### MCP JSON-RPC响应
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "content": [
      {
        "type": "text",
        "text": "🌤️ 北京今天多云，温度15-25°C，湿度60%"
      }
    ]
  }
}
```

## 🔧 核心模块详解

### 1. MCP客户端 (`pkg/mcp/mcp_client.go`)

**主要功能:**
- 管理MCP服务器进程生命周期
- 处理JSON-RPC 2.0协议通信
- 提供线程安全的请求/响应处理

**核心方法:**
```go
// 启动MCP服务器进程
func (c *Client) Start(ctx context.Context) error

// 处理MCP请求
func (c *Client) ProcessRequest(ctx context.Context, req *models.MCPRequest) (*models.MCPResponse, error)

// 停止MCP服务器进程
func (c *Client) Stop() error
```

**实现细节:**
- 使用`exec.CommandContext`启动子进程
- 通过stdin/stdout建立管道通信
- 使用`sync.Mutex`保证线程安全
- 支持动态请求ID生成

### 2. MCP服务器 (`cmd/server/main.go`)

**主要功能:**
- 实现标准MCP协议
- 注册和管理工具(天气、搜索)
- 处理工具调用请求

**支持的工具:**

#### 天气工具
```go
// 工具定义
{
    "name": "get_weather",
    "description": "获取指定地点的天气信息",
    "inputSchema": {
        "type": "object",
        "properties": {
            "location": {"type": "string"},
            "date": {"type": "string"}
        }
    }
}
```

#### 搜索工具
```go
// 工具定义
{
    "name": "search",
    "description": "搜索最新信息",
    "inputSchema": {
        "type": "object",
        "properties": {
            "query": {"type": "string"},
            "max_results": {"type": "integer"},
            "search_depth": {"type": "string"}
        }
    }
}
```

### 3. 工作流引擎 (`internal/workflow/agent.go`)

**主要功能:**
- 协调LLM和MCP客户端
- 处理自然语言查询解析
- 管理请求生命周期

**核心流程:**
```go
func (w *AgentWorkflow) ProcessQuery(ctx context.Context, query string, userID string) (*models.WorkflowResponse, error) {
    // 1. LLM解析查询
    mcpRequest := w.parseQueryWithLLM(ctx, query)
    
    // 2. 调用MCP客户端
    mcpResponse := w.mcpClient.ProcessRequest(ctx, mcpRequest)
    
    // 3. 格式化响应
    return w.formatResponse(mcpResponse)
}
```

## 🚀 快速开始

### 环境要求

- Go 1.21+
- 有效的Azure OpenAI API密钥
- 有效的天气API密钥
- 有效的Tavily搜索API密钥

### 安装步骤

1. **克隆项目**
```bash
git clone <repository-url>
cd deer-flow-go
```

2. **安装依赖**
```bash
go mod download
```

3. **配置环境变量**
```bash
# 复制配置模板
cp .env.example .env

# 编辑配置文件
vim .env
```

**必需的环境变量:**
```bash
# Azure OpenAI配置
AZURE_OPENAI_ENDPOINT=https://your-resource.openai.azure.com/
AZURE_OPENAI_API_KEY=your-api-key
AZURE_OPENAI_DEPLOYMENT_NAME=your-deployment-name
AZURE_OPENAI_API_VERSION=2024-02-15-preview

# 天气API配置
WEATHER_API_KEY=your-weather-api-key
WEATHER_API_BASE_URL=https://api.weatherapi.com/v1

# 搜索API配置
TAVILY_API_KEY=your-tavily-api-key

# 服务器配置
SERVER_PORT=8080
SERVER_HOST=localhost
```

4. **启动服务**
```bash
# 开发模式启动
go run cmd/main.go

# 或编译后启动
go build -o bin/deer-flow ./cmd/main.go
./bin/deer-flow
```

### API使用示例

#### 天气查询
```bash
curl -X POST http://localhost:8080/api/query \
  -H "Content-Type: application/json" \
  -d '{
    "query": "北京今天的天气怎么样？",
    "user_id": "user123"
  }'
```

**响应示例:**
```json
{
  "success": true,
  "data": {
    "response": "🌤️ 北京今天多云，温度15-25°C，湿度60%，风速10km/h",
    "tool_used": "get_weather",
    "processing_time": "1.25s"
  },
  "user_id": "user123",
  "timestamp": "2024-01-15T10:30:00Z"
}
```

#### 搜索查询
```bash
curl -X POST http://localhost:8080/api/query \
  -H "Content-Type: application/json" \
  -d '{
    "query": "最新的人工智能发展趋势",
    "user_id": "user123"
  }'
```

**响应示例:**
```json
{
  "success": true,
  "data": {
    "response": "🔍 根据最新搜索结果，人工智能发展趋势包括：\n1. 大语言模型持续优化...\n2. 多模态AI应用普及...\n3. AI安全和伦理关注度提升...",
    "tool_used": "search",
    "processing_time": "2.51s"
  },
  "user_id": "user123",
  "timestamp": "2024-01-15T10:30:00Z"
}
```

## 🔍 技术实现细节

### MCP协议实现

**协议标准:** JSON-RPC 2.0 over stdin/stdout

**消息格式:**
```go
type MCPJSONRPCMessage struct {
    JSONRPC string                 `json:"jsonrpc"`
    ID      int                    `json:"id,omitempty"`
    Method  string                 `json:"method,omitempty"`
    Params  map[string]interface{} `json:"params,omitempty"`
    Result  interface{}            `json:"result,omitempty"`
    Error   *MCPError             `json:"error,omitempty"`
}
```

**错误处理:**
```go
type MCPError struct {
    Code    int    `json:"code"`
    Message string `json:"message"`
    Data    interface{} `json:"data,omitempty"`
}
```

### 进程管理

**启动流程:**
1. 创建子进程: `go run cmd/server/main.go`
2. 建立stdin/stdout管道
3. 发送初始化消息
4. 等待服务器就绪确认

**通信机制:**
- **发送**: JSON序列化 → 写入stdin → 添加换行符
- **接收**: 从stdout读取 → 按行扫描 → JSON反序列化

**生命周期管理:**
```go
// 启动时
func (c *Client) Start(ctx context.Context) error {
    c.cmd = exec.CommandContext(ctx, "go", "run", "cmd/server/main.go")
    c.stdin, _ = c.cmd.StdinPipe()
    c.stdout, _ = c.cmd.StdoutPipe()
    c.cmd.Start()
    return c.initialize()
}

// 停止时
func (c *Client) Stop() error {
    c.stdin.Close()
    return c.cmd.Wait()
}
```

### 并发安全

**线程安全措施:**
- 使用`sync.Mutex`保护共享状态
- 原子操作管理请求ID
- 进程状态标志保护

```go
type Client struct {
    mutex     sync.Mutex  // 保护并发访问
    running   bool        // 进程状态
    requestID int         // 请求ID计数器
    // ... 其他字段
}
```

### 错误处理策略

**分层错误处理:**
1. **网络层**: 连接错误、超时处理
2. **协议层**: JSON-RPC错误码处理
3. **业务层**: 工具调用失败处理
4. **应用层**: 用户友好错误消息

**错误恢复机制:**
- 自动重试机制
- 优雅降级处理
- 详细错误日志记录

## 📁 项目结构

```
deer-flow-go/
├── cmd/                    # 可执行文件
│   ├── main.go            # API服务器主程序
│   └── server/            # MCP服务器
│       └── main.go        # MCP服务器主程序
├── internal/              # 内部包
│   └── workflow/          # 工作流引擎
│       └── agent.go       # 智能代理实现
├── pkg/                   # 公共包
│   ├── config/            # 配置管理
│   │   └── config.go
│   ├── handlers/          # HTTP处理器
│   │   └── api.go
│   ├── llm/              # LLM客户端
│   │   └── azure_openai.go
│   ├── mcp/              # MCP协议实现
│   │   ├── client.go     # MCP接口定义
│   │   └── mcp_client.go # MCP客户端实现
│   ├── models/           # 数据模型
│   │   └── models.go
│   ├── queue/            # 队列管理
│   │   ├── manager.go
│   │   └── worker.go
│   ├── search/           # 搜索服务
│   │   ├── search_mcp.go
│   │   └── tavily.go
│   └── weather/          # 天气服务
│       ├── weather.go
│       └── weather_mcp.go
├── test/                 # 测试文件
├── docs/                 # 文档
│   └── mcp-architecture.svg
├── go.mod               # Go模块定义
├── go.sum               # 依赖校验
└── README.md           # 项目说明
```

## 🧪 测试

### 运行测试
```bash
# 运行所有测试
go test ./...

# 运行特定模块测试
go test ./pkg/mcp/
go test ./internal/workflow/

# 运行测试并显示覆盖率
go test -cover ./...
```

### 集成测试
```bash
# 启动服务后进行集成测试
go run cmd/main.go &

# 测试天气API
curl -X POST http://localhost:8080/api/query \
  -H "Content-Type: application/json" \
  -d '{"query": "上海明天天气", "user_id": "test"}'

# 测试搜索API
curl -X POST http://localhost:8080/api/query \
  -H "Content-Type: application/json" \
  -d '{"query": "最新科技新闻", "user_id": "test"}'
```

## 🔧 开发指南

### 添加新工具

1. **在MCP服务器中注册工具**
```go
// cmd/server/main.go
server.RegisterTool(mcp.Tool{
    Name: "your_tool",
    Description: "工具描述",
    InputSchema: map[string]interface{}{
        "type": "object",
        "properties": map[string]interface{}{
            "param1": map[string]interface{}{"type": "string"},
        },
    },
})
```

2. **实现工具处理函数**
```go
func handleYourTool(params map[string]interface{}) (*mcp.ToolResult, error) {
    // 实现工具逻辑
    return mcp.NewToolResultText("结果"), nil
}
```

3. **更新路由**
```go
server.SetToolHandler("your_tool", handleYourTool)
```

### 性能优化建议

1. **连接池**: 对于频繁的外部API调用，使用连接池
2. **缓存**: 实现响应缓存减少重复请求
3. **异步处理**: 对于耗时操作使用异步处理
4. **监控**: 添加性能监控和日志记录

## 📈 监控和日志

### 日志级别
- **DEBUG**: 详细调试信息
- **INFO**: 一般信息记录
- **WARN**: 警告信息
- **ERROR**: 错误信息
- **FATAL**: 致命错误

### 关键指标监控
- API响应时间
- MCP调用成功率
- 外部API调用延迟
- 系统资源使用情况

## 🤝 贡献指南

1. Fork项目
2. 创建特性分支 (`git checkout -b feature/AmazingFeature`)
3. 提交更改 (`git commit -m 'Add some AmazingFeature'`)
4. 推送到分支 (`git push origin feature/AmazingFeature`)
5. 打开Pull Request

## 📄 许可证

本项目采用MIT许可证 - 查看 [LICENSE](LICENSE) 文件了解详情。

## 🆘 故障排除

### 常见问题

**Q: MCP服务器启动失败**
A: 检查Go环境和依赖是否正确安装，确认端口未被占用

**Q: API调用返回超时**
A: 检查外部API密钥配置，确认网络连接正常

**Q: 天气查询无结果**
A: 验证天气API密钥有效性，检查地点名称格式

**Q: 搜索功能异常**
A: 确认Tavily API密钥配置正确，检查搜索参数格式

### 调试模式

启用详细日志:
```bash
LOG_LEVEL=debug go run cmd/main.go
```

查看MCP通信日志:
```bash
MCP_DEBUG=true go run cmd/main.go
```

---

**项目维护者**: [Your Name]
**最后更新**: 2024年1月
**版本**: v1.0.0