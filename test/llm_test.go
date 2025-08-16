package test

import (
	"context"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"deer-flow-go/pkg/config"
	"deer-flow-go/pkg/llm"
	"deer-flow-go/pkg/models"
)

// TestAzureOpenAIChatCompletion 测试Azure OpenAI聊天完成功能
func TestAzureOpenAIChatCompletion(t *testing.T) {
	// 加载配置
	cfg, err := config.LoadConfig()
	require.NoError(t, err, "Failed to load config")

	// 创建日志器
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	// 创建Azure OpenAI客户端
	client := llm.NewAzureOpenAIClient(&cfg.AzureOpenAI, logger)
	require.NotNil(t, client, "Failed to create Azure OpenAI client")

	// 测试简单聊天
	t.Run("Simple Chat", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		messages := []models.ChatMessage{
			{Role: "user", Content: "你好，请简单介绍一下自己"},
		}

		systemPrompt := "你是一个友好的AI助手，请用中文回答问题。"

		response, err := client.ChatCompletion(ctx, messages, systemPrompt)
		require.NoError(t, err, "Chat completion failed")
		assert.NotEmpty(t, response, "Response should not be empty")
		assert.Contains(t, response, "AI", "Response should mention AI")

		t.Logf("Chat response: %s", response)
	})

	// 测试多轮对话
	t.Run("Multi-turn Conversation", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		messages := []models.ChatMessage{
			{Role: "user", Content: "我想了解Go语言"},
			{Role: "assistant", Content: "Go是Google开发的编程语言，具有简洁、高效的特点。"},
			{Role: "user", Content: "Go语言有什么优势？"},
		}

		systemPrompt := "你是一个编程专家，请详细回答关于编程语言的问题。"

		response, err := client.ChatCompletion(ctx, messages, systemPrompt)
		require.NoError(t, err, "Multi-turn chat failed")
		assert.NotEmpty(t, response, "Response should not be empty")
		assert.Contains(t, response, "Go", "Response should mention Go")

		t.Logf("Multi-turn response: %s", response)
	})

	// 测试查询解析为MCP请求
	t.Run("Parse Query to MCP", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		query := "今天北京的天气怎么样？"

		mcpRequest, err := client.ParseQueryToMCP(ctx, query)
		require.NoError(t, err, "Failed to parse query to MCP")
		assert.NotNil(t, mcpRequest, "MCP request should not be nil")
		assert.Equal(t, "search", mcpRequest.Method, "Method should be search")
		assert.NotNil(t, mcpRequest.Params, "Params should not be nil")

		t.Logf("MCP Request: %+v", mcpRequest)
	})
}

// TestAzureOpenAIFormatSearchResults 测试搜索结果格式化功能
func TestAzureOpenAIFormatSearchResults(t *testing.T) {
	// 加载配置
	cfg, err := config.LoadConfig()
	require.NoError(t, err, "Failed to load config")

	// 创建日志器
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	// 创建Azure OpenAI客户端
	client := llm.NewAzureOpenAIClient(&cfg.AzureOpenAI, logger)
	require.NotNil(t, client, "Failed to create Azure OpenAI client")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 模拟搜索结果
	searchResults := &models.SearchResponse{
		Query:  "Go语言特点",
		Answer: "Go语言是一种开源编程语言",
		Results: []models.SearchResult{
			{
				Title:   "Go语言官方文档",
				URL:     "https://golang.org",
				Content: "Go是Google开发的开源编程语言，具有简洁、快速、安全的特点。",
				Score:   0.95,
			},
			{
				Title:   "Go语言教程",
				URL:     "https://tour.golang.org",
				Content: "Go语言支持并发编程，有垃圾回收机制，编译速度快。",
				Score:   0.88,
			},
		},
	}

	query := "Go语言有什么特点？"
	formattedResponse, err := client.FormatSearchResults(ctx, query, searchResults)
	require.NoError(t, err, "Failed to format search results")
	assert.NotEmpty(t, formattedResponse, "Formatted response should not be empty")
	assert.Contains(t, formattedResponse, "Go", "Response should mention Go")

	t.Logf("Formatted response: %s", formattedResponse)
}
