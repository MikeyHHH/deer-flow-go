package models

import "time"

// ChatMessage 聊天消息结构
type ChatMessage struct {
	Role    string `json:"role"`    // system, user, assistant
	Content string `json:"content"` // 消息内容
}

// ChatRequest 聊天请求结构
type ChatRequest struct {
	Messages []ChatMessage `json:"messages"`
	Query    string        `json:"query"` // 用户输入的问题
}

// ChatResponse 聊天响应结构
type ChatResponse struct {
	Response  string    `json:"response"`
	Timestamp time.Time `json:"timestamp"`
	Success   bool      `json:"success"`
	Error     string    `json:"error,omitempty"`
}

// MCPRequest MCP协议请求结构
type MCPRequest struct {
	Method string      `json:"method"`
	Params interface{} `json:"params"`
}

// MCPResponse MCP协议响应结构
type MCPResponse struct {
	Result interface{} `json:"result,omitempty"`
	Error  *MCPError   `json:"error,omitempty"`
}

// MCPError MCP错误结构
type MCPError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// SearchRequest 搜索请求结构
type SearchRequest struct {
	Query       string `json:"query"`
	MaxResults  int    `json:"max_results,omitempty"`
	SearchDepth string `json:"search_depth,omitempty"`
}

// SearchResult 搜索结果结构
type SearchResult struct {
	Title   string  `json:"title"`
	URL     string  `json:"url"`
	Content string  `json:"content"`
	Score   float64 `json:"score"`
}

// SearchResponse 搜索响应结构
type SearchResponse struct {
	Results []SearchResult `json:"results"`
	Query   string         `json:"query"`
	Answer  string         `json:"answer,omitempty"`
}

// WorkflowState 工作流状态
type WorkflowState struct {
	Step        string      `json:"step"`        // 当前步骤
	Query       string      `json:"query"`       // 原始查询
	MCPRequest  *MCPRequest `json:"mcp_request"` // MCP请求
	SearchData  interface{} `json:"search_data"` // 搜索数据
	FinalResult string      `json:"final_result"` // 最终结果
}

// PromptTemplate 提示词模板
type PromptTemplate struct {
	Name     string `json:"name"`
	Template string `json:"template"`
	Type     string `json:"type"` // query_parser, result_formatter
}