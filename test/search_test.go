package test

import (
	"context"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"deer-flow-go/pkg/config"
	"deer-flow-go/pkg/search"
)

// TestTavilySearch 测试Tavily搜索功能
func TestTavilySearch(t *testing.T) {
	// 加载配置
	cfg, err := config.LoadConfig()
	require.NoError(t, err, "Failed to load config")

	// 创建日志器
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	// 创建Tavily客户端
	client := search.NewTavilyClient(&cfg.Tavily, logger)
	require.NotNil(t, client, "Failed to create Tavily client")

	// 测试基本搜索功能
	t.Run("Basic Search", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		query := "Go语言特点"
		result, err := client.Search(ctx, query)
		require.NoError(t, err, "Search failed")
		assert.NotNil(t, result, "Search result should not be nil")
		assert.Equal(t, query, result.Query, "Query should match")
		assert.NotEmpty(t, result.Results, "Results should not be empty")

		// 验证搜索结果结构
		for i, searchResult := range result.Results {
			assert.NotEmpty(t, searchResult.Title, "Result %d title should not be empty", i)
			assert.NotEmpty(t, searchResult.URL, "Result %d URL should not be empty", i)
			assert.NotEmpty(t, searchResult.Content, "Result %d content should not be empty", i)
			assert.GreaterOrEqual(t, searchResult.Score, 0.0, "Result %d score should be non-negative", i)
		}

		t.Logf("Search completed: %d results found", len(result.Results))
		if result.Answer != "" {
			t.Logf("Answer: %s", result.Answer)
		}
	})

	// 测试实时信息搜索
	t.Run("Real-time Search", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		query := "今天的新闻"
		result, err := client.Search(ctx, query)
		require.NoError(t, err, "Real-time search failed")
		assert.NotNil(t, result, "Search result should not be nil")
		assert.Equal(t, query, result.Query, "Query should match")
		assert.NotEmpty(t, result.Results, "Results should not be empty")

		t.Logf("Real-time search completed: %d results found", len(result.Results))
		for i, searchResult := range result.Results {
			t.Logf("Result %d: %s - %s", i+1, searchResult.Title, searchResult.URL)
		}
	})

	// 测试技术相关搜索
	t.Run("Technical Search", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		query := "Kubernetes最新版本特性"
		result, err := client.Search(ctx, query)
		require.NoError(t, err, "Technical search failed")
		assert.NotNil(t, result, "Search result should not be nil")
		assert.Equal(t, query, result.Query, "Query should match")
		assert.NotEmpty(t, result.Results, "Results should not be empty")

		// 验证技术搜索结果的质量
		found := false
		for _, searchResult := range result.Results {
			if searchResult.Score > 0.7 {
				found = true
				break
			}
		}
		assert.True(t, found, "Should have at least one high-quality result")

		t.Logf("Technical search completed: %d results found", len(result.Results))
	})
}

// TestTavilyCleanResults 测试搜索结果清理功能
func TestTavilyCleanResults(t *testing.T) {
	// 加载配置
	cfg, err := config.LoadConfig()
	require.NoError(t, err, "Failed to load config")

	// 创建日志器
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	// 创建Tavily客户端
	client := search.NewTavilyClient(&cfg.Tavily, logger)
	require.NotNil(t, client, "Failed to create Tavily client")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 执行搜索获取原始结果
	query := "人工智能发展趋势"
	originalResult, err := client.Search(ctx, query)
	require.NoError(t, err, "Search failed")
	require.NotNil(t, originalResult, "Original result should not be nil")

	// 清理结果
	cleanedResult := client.CleanResults(originalResult)
	require.NotNil(t, cleanedResult, "Cleaned result should not be nil")

	// 验证清理效果
	assert.Equal(t, originalResult.Query, cleanedResult.Query, "Query should remain the same")
	assert.Equal(t, originalResult.Answer, cleanedResult.Answer, "Answer should remain the same")

	// 验证结果质量提升
	for _, result := range cleanedResult.Results {
		assert.NotEmpty(t, result.Content, "Cleaned result content should not be empty")
		assert.GreaterOrEqual(t, result.Score, 0.1, "Cleaned result should have decent score")
		assert.LessOrEqual(t, len(result.Content), 1003, "Content should be truncated if too long") // 1000 + "..."
	}

	t.Logf("Original results: %d, Cleaned results: %d", len(originalResult.Results), len(cleanedResult.Results))
}

// TestTavilySearchError 测试搜索错误处理
func TestTavilySearchError(t *testing.T) {
	// 创建带有无效API密钥的配置
	invalidConfig := &config.TavilyConfig{
		APIKey:      "invalid-api-key",
		MaxResults:  5,
		SearchDepth: "advanced",
	}

	// 创建日志器
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	// 创建Tavily客户端
	client := search.NewTavilyClient(invalidConfig, logger)
	require.NotNil(t, client, "Failed to create Tavily client")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 测试无效API密钥的错误处理
	query := "test query"
	result, err := client.Search(ctx, query)
	assert.Error(t, err, "Should return error for invalid API key")
	assert.Nil(t, result, "Result should be nil on error")
	assert.Contains(t, err.Error(), "Tavily API error", "Error should mention Tavily API")

	t.Logf("Expected error occurred: %v", err)
}