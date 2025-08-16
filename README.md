# Deer Flow Go

一个基于Go语言的智能体对话系统，使用Gin作为Web框架，集成Azure OpenAI和Tavily搜索引擎，实现MCP协议的实时数据搜索功能。

## 功能特点

- 基于Go Gin框架的高性能API服务
- 集成Azure OpenAI大模型能力
- 支持Tavily搜索引擎进行实时数据检索
- 实现MCP协议标准的客户端-服务端通信
- 完整的智能体对话工作流程
- 灵活的配置管理

## 系统架构

系统由以下主要组件构成：

1. **API服务器**：基于Gin框架，提供HTTP接口
2. **LLM模块**：集成Azure OpenAI，处理自然语言理解和生成
3. **搜索模块**：集成Tavily搜索引擎，提供实时数据检索
4. **MCP协议**：实现标准化的客户端-服务端通信
5. **工作流引擎**：协调各组件，实现完整对话流程

## 工作流程

1. 前端发送用户问题到API服务器
2. 系统使用LLM将问题转换为MCP协议格式
3. MCP客户端调用Tavily搜索服务获取实时数据
4. 系统将搜索结果提供给LLM进行整合和润色
5. 将最终结果返回给前端展示

## 安装与配置

### 前置条件

- Go 1.21或更高版本
- Azure OpenAI API访问权限
- Tavily搜索API密钥

### 安装步骤

1. 克隆仓库

```bash
git clone https://github.com/yourusername/deer-flow-go.git
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
├── cmd/                # 命令行入口
│   └── main.go        # 主程序
├── pkg/               # 公共包
│   ├── config/        # 配置管理
│   ├── llm/           # 大模型集成
│   ├── search/        # 搜索引擎集成
│   ├── mcp/           # MCP协议实现
│   ├── models/        # 数据模型
│   └── handlers/      # API处理器
├── internal/          # 内部包
│   └── workflow/      # 工作流实现
├── .env               # 环境变量
└── README.md          # 项目文档
```

### 扩展指南

#### 添加新的搜索引擎

1. 在`pkg/search/`目录下创建新的搜索引擎实现
2. 实现与`TavilyClient`类似的接口
3. 在配置中添加相应的选项
4. 在工作流中集成新的搜索引擎

#### 修改提示词模板

可以在`pkg/llm/azure_openai.go`文件中修改以下方法中的提示词：

- `ParseQueryToMCP`: 用于将用户查询转换为MCP请求的提示词
- `FormatSearchResults`: 用于格式化搜索结果的提示词

## 许可证

MIT