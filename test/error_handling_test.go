package test

import (
	"context"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"deer-flow-go/internal/workflow"
	"deer-flow-go/pkg/config"
	"deer-flow-go/pkg/models"
)

// TestAgentWorkflowErrors 测试智能体工作流错误处理
func TestAgentWorkflowErrors(t *testing.T) {
	// 加载配置
	cfg, err := config.LoadConfig()
	require.NoError(t, err, "Failed to load config")

	// 创建日志器
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	// 创建智能体工作流
	agent := workflow.NewAgentWorkflow(cfg, logger)
	require.NotNil(t, agent, "Failed to create agent workflow")

	// 测试无效查询场景
	t.Run("Invalid Query", func(t *testing.T) {
		// 创建上下文
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		// 使用特殊字符和无意义内容的查询
		query := "@#$%^&*()_+<>?:{}|~`"
		response, err := agent.ProcessQuery(ctx, query)

		// 验证结果 - 系统应该能处理特殊字符
		require.NoError(t, err, "Should not return error for special characters")
		assert.NotNil(t, response, "Response should not be nil")
		t.Logf("Special character query handled: %v", response.Success)
		t.Logf("Response: %s", response.Response)
		if response.Error != "" {
			t.Logf("Error: %s", response.Error)
		}
	})

	// 测试非ASCII字符查询
	t.Run("Non-ASCII Query", func(t *testing.T) {
		// 创建上下文
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		// 使用包含各种非ASCII字符的查询
		query := "你好世界！こんにちは世界！안녕하세요 세계！Привет, мир!"
		response, err := agent.ProcessQuery(ctx, query)

		// 验证结果 - 系统应该能处理多语言字符
		require.NoError(t, err, "Should not return error for non-ASCII characters")
		assert.NotNil(t, response, "Response should not be nil")
		t.Logf("Multi-language query handled: %v", response.Success)
		t.Logf("Response: %s", response.Response)
		if response.Error != "" {
			t.Logf("Error: %s", response.Error)
		}
	})

	// 测试极短查询
	t.Run("Very Short Query", func(t *testing.T) {
		// 创建上下文
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		// 使用极短的查询
		query := "a"
		response, err := agent.ProcessQuery(ctx, query)

		// 验证结果 - 系统应该能处理极短查询
		require.NoError(t, err, "Should not return error for very short query")
		assert.NotNil(t, response, "Response should not be nil")
		t.Logf("Very short query handled: %v", response.Success)
		t.Logf("Response: %s", response.Response)
		if response.Error != "" {
			t.Logf("Error: %s", response.Error)
		}
	})

	// 测试JSON格式查询
	t.Run("JSON Format Query", func(t *testing.T) {
		// 创建上下文
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		// 使用JSON格式的查询
		query := `{"question": "什么是人工智能?", "format": "detailed"}`
		response, err := agent.ProcessQuery(ctx, query)

		// 验证结果 - 系统应该能处理JSON格式查询
		require.NoError(t, err, "Should not return error for JSON format query")
		assert.NotNil(t, response, "Response should not be nil")
		t.Logf("JSON format query handled: %v", response.Success)
		t.Logf("Response: %s", response.Response)
		if response.Error != "" {
			t.Logf("Error: %s", response.Error)
		}
	})
}

// TestAgentWorkflowCancellation 测试上下文取消场景
func TestAgentWorkflowCancellation(t *testing.T) {
	// 加载配置
	cfg, err := config.LoadConfig()
	require.NoError(t, err, "Failed to load config")

	// 创建日志器
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	// 创建智能体工作流
	agent := workflow.NewAgentWorkflow(cfg, logger)
	require.NotNil(t, agent, "Failed to create agent workflow")

	// 测试上下文取消
	t.Run("Context Cancellation", func(t *testing.T) {
		// 创建可取消的上下文
		ctx, cancel := context.WithCancel(context.Background())

		// 启动一个goroutine在短时间后取消上下文
		go func() {
			time.Sleep(100 * time.Millisecond)
			cancel()
		}()

		query := "这个查询应该被取消"
		response, err := agent.ProcessQuery(ctx, query)

		// 验证取消处理
		if err != nil {
			// 如果返回错误，应该是上下文取消错误
			assert.Contains(t, err.Error(), "context", "Error should be related to context")
			t.Logf("Cancellation correctly returned error: %v", err)
		} else {
			// 如果没有返回错误，响应应该包含错误信息
			assert.NotNil(t, response, "Response should not be nil")
			assert.False(t, response.Success, "Response should not be successful")
			assert.NotEmpty(t, response.Error, "Response should have error message")
			assert.Contains(t, response.Error, "cancel", "Error should mention cancellation")
			t.Logf("Cancellation handled in response: %s", response.Error)
		}
	})
}

// TestAgentWorkflowRateLimiting 测试速率限制场景
func TestAgentWorkflowRateLimiting(t *testing.T) {
	// 跳过长时间运行的测试，除非明确启用
	if testing.Short() {
		t.Skip("Skipping rate limiting test in short mode")
	}

	// 加载配置
	cfg, err := config.LoadConfig()
	require.NoError(t, err, "Failed to load config")

	// 创建日志器
	logger := logrus.New()
	logger.SetLevel(logrus.InfoLevel) // 使用INFO级别减少日志输出

	// 创建智能体工作流
	agent := workflow.NewAgentWorkflow(cfg, logger)
	require.NotNil(t, agent, "Failed to create agent workflow")

	// 测试并发请求
	t.Run("Concurrent Requests", func(t *testing.T) {
		// 创建上下文
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()

		// 准备测试查询
		queries := []string{
			"查询1",
			"查询2",
			"查询3",
			"查询4",
			"查询5",
		}

		// 并发处理查询
		resultCh := make(chan struct {
			index    int
			response *models.ChatResponse
			err      error
		}, len(queries))

		for i, query := range queries {
			go func(idx int, q string) {
				response, err := agent.ProcessQuery(ctx, q)
				resultCh <- struct {
					index    int
					response *models.ChatResponse
					err      error
				}{idx, response, err}
			}(i, query)
		}

		// 收集结果
		successCount := 0
		rateLimit := 0

		for i := 0; i < len(queries); i++ {
			result := <-resultCh
			t.Logf("Query %d result: success=%v, error=%v", 
				result.index+1, result.response.Success, result.err)

			if result.err == nil && result.response.Success {
				successCount++
			} else if result.err != nil && (result.err.Error() == "rate limit exceeded" || 
				result.response.Error == "rate limit exceeded") {
				rateLimit++
			}
		}

		// 验证结果
		t.Logf("Concurrent requests: %d successful, %d rate limited", 
			successCount, rateLimit)
		assert.True(t, successCount > 0, "Some requests should succeed")
	})
}