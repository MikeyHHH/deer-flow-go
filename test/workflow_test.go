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
)

// TestAgentWorkflow 测试智能体工作流
func TestAgentWorkflow(t *testing.T) {
	// 加载配置
	cfg, err := config.LoadConfig()
	require.NoError(t, err, "Failed to load config")

	// 创建日志器
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	// 创建智能体工作流
	agent := workflow.NewAgentWorkflow(cfg, logger)
	require.NotNil(t, agent, "Failed to create agent workflow")

	// 验证工作流配置
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err = agent.ValidateWorkflow(ctx)
	require.NoError(t, err, "Workflow validation failed")

	// 测试工作流状态
	t.Run("Workflow Status", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		status, err := agent.GetWorkflowStatus(ctx)
		require.NoError(t, err, "Failed to get workflow status")
		assert.NotNil(t, status, "Status should not be nil")
		assert.Equal(t, "ready", status.Step, "Workflow should be in ready state")
		assert.NotNil(t, status.SearchData, "Search data should not be nil")

		// 验证搜索数据存在
		if status.SearchData != nil {
			// 尝试类型断言为map
			if searchDataMap, ok := status.SearchData.(map[string]interface{}); ok {
				// 验证MCP健康状态
				if mcpHealthy, exists := searchDataMap["mcp_healthy"]; exists {
					if healthy, ok := mcpHealthy.(bool); ok {
						assert.True(t, healthy, "MCP should be healthy")
					}
				}

				// 验证MCP能力
				if capabilities, exists := searchDataMap["capabilities"]; exists {
					if capMap, ok := capabilities.(map[string]interface{}); ok {
						assert.NotEmpty(t, capMap, "Capabilities should not be empty")
					}
				}
			}
		}

		t.Logf("Workflow status: %+v", status)
	})

	// 测试处理需要搜索的查询
	t.Run("Process Search Query", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()

		query := "Go语言的并发特性有哪些？"
		response, err := agent.ProcessQuery(ctx, query)
		require.NoError(t, err, "Failed to process search query")
		assert.NotNil(t, response, "Response should not be nil")
		assert.True(t, response.Success, "Response should be successful")
		assert.NotEmpty(t, response.Response, "Response content should not be empty")
		assert.Empty(t, response.Error, "Response should not have error")

		t.Logf("Search query processed successfully")
		t.Logf("Response: %s", response.Response)
	})

	// 测试处理不需要搜索的查询
	t.Run("Process Direct Query", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		query := "1+1等于几？"
		response, err := agent.ProcessQuery(ctx, query)
		require.NoError(t, err, "Failed to process direct query")
		assert.NotNil(t, response, "Response should not be nil")
		assert.True(t, response.Success, "Response should be successful")
		assert.NotEmpty(t, response.Response, "Response content should not be empty")
		assert.Empty(t, response.Error, "Response should not have error")

		t.Logf("Direct query processed successfully")
		t.Logf("Response: %s", response.Response)
	})

	// 测试处理多轮对话
	t.Run("Process Multi-turn Conversation", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()

		// 第一轮对话
		query1 := "什么是人工智能？"
		response1, err := agent.ProcessQuery(ctx, query1)
		require.NoError(t, err, "Failed to process first query")
		assert.True(t, response1.Success, "First response should be successful")
		assert.NotEmpty(t, response1.Response, "First response content should not be empty")

		t.Logf("First query processed successfully")
		t.Logf("First response: %s", response1.Response)

		// 第二轮对话（基于第一轮）
		query2 := "它有哪些应用领域？"
		response2, err := agent.ProcessQuery(ctx, query2)
		require.NoError(t, err, "Failed to process second query")
		assert.True(t, response2.Success, "Second response should be successful")
		assert.NotEmpty(t, response2.Response, "Second response content should not be empty")

		t.Logf("Second query processed successfully")
		t.Logf("Second response: %s", response2.Response)
	})
}

// TestAgentWorkflowErrorHandling 测试智能体工作流错误处理
func TestAgentWorkflowErrorHandling(t *testing.T) {
	// 加载配置
	cfg, err := config.LoadConfig()
	require.NoError(t, err, "Failed to load config")

	// 创建日志器
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	// 创建智能体工作流
	agent := workflow.NewAgentWorkflow(cfg, logger)
	require.NotNil(t, agent, "Failed to create agent workflow")

	// 测试空查询处理
	t.Run("Empty Query", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		query := ""
		response, err := agent.ProcessQuery(ctx, query)
		require.NoError(t, err, "Should not return error for empty query")
		assert.NotNil(t, response, "Response should not be nil")
		// 注意：根据实际实现，空查询可能成功也可能失败，这里我们只验证有响应

		t.Logf("Empty query handled: %v", response.Success)
		t.Logf("Response: %s", response.Response)
		if response.Error != "" {
			t.Logf("Error: %s", response.Error)
		}
	})

	// 测试超长查询处理
	t.Run("Long Query", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		// 创建一个超长查询
		longQuery := ""
		for i := 0; i < 100; i++ {
			longQuery += "这是一个非常长的查询，用于测试系统对超长输入的处理能力。"
		}

		response, err := agent.ProcessQuery(ctx, longQuery)
		require.NoError(t, err, "Should not return error for long query")
		assert.NotNil(t, response, "Response should not be nil")

		t.Logf("Long query handled: %v", response.Success)
		t.Logf("Response: %s", response.Response)
		if response.Error != "" {
			t.Logf("Error: %s", response.Error)
		}
	})

	// 测试超时处理
	t.Run("Timeout Handling", func(t *testing.T) {
		// 创建一个短超时上下文
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
		defer cancel()

		// 等待超时触发
		time.Sleep(5 * time.Millisecond)

		query := "这个查询应该超时"
		response, err := agent.ProcessQuery(ctx, query)

		// 验证超时处理
		if err != nil {
			// 如果返回错误，应该是上下文超时错误
			assert.Contains(t, err.Error(), "context", "Error should be related to context")
			t.Logf("Timeout correctly returned error: %v", err)
		} else {
			// 如果没有返回错误，响应应该包含错误信息
			assert.NotNil(t, response, "Response should not be nil")
			assert.False(t, response.Success, "Response should not be successful")
			assert.NotEmpty(t, response.Error, "Response should have error message")
			t.Logf("Timeout handled in response: %s", response.Error)
		}
	})
}

// TestAgentWorkflowPerformance 测试智能体工作流性能
func TestAgentWorkflowPerformance(t *testing.T) {
	// 跳过长时间运行的性能测试，除非明确启用
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
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

	// 准备测试查询
	queries := []string{
		"Go语言的特点",
		"Python与Go的区别",
		"云计算的定义",
		"微服务架构的优缺点",
		"Docker与Kubernetes的关系",
	}

	// 测量处理时间
	totalTime := time.Duration(0)
	successCount := 0

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	for i, query := range queries {
		t.Logf("Processing query %d: %s", i+1, query)

		startTime := time.Now()
		response, err := agent.ProcessQuery(ctx, query)
		processTime := time.Since(startTime)

		if err == nil && response.Success {
			successCount++
			totalTime += processTime
			t.Logf("Query %d processed in %v", i+1, processTime)
		} else {
			t.Logf("Query %d failed: %v, %s", i+1, err, response.Error)
		}
	}

	// 计算平均处理时间
	if successCount > 0 {
		avgTime := totalTime / time.Duration(successCount)
		t.Logf("Average processing time: %v for %d successful queries", avgTime, successCount)
		assert.True(t, successCount > 0, "Should have some successful queries")
	} else {
		t.Error("No successful queries processed")
	}
}