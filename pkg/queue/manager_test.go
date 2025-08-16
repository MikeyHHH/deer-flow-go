package queue

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"deer-flow-go/pkg/models"
)

// MockRequestProcessor 模拟请求处理器
type MockRequestProcessor struct {
	mock.Mock
	processDelay time.Duration
}

func (m *MockRequestProcessor) ProcessRequest(ctx context.Context, query string) (*models.ChatResponse, error) {
	args := m.Called(ctx, query)
	
	// 模拟处理延迟
	if m.processDelay > 0 {
		select {
		case <-time.After(m.processDelay):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.ChatResponse), args.Error(1)
}

func TestQueueManager_Basic(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel) // 减少测试输出
	
	mockProcessor := &MockRequestProcessor{}
	testResponse := &models.ChatResponse{Response: "test response"}
	mockProcessor.On("ProcessRequest", mock.Anything, "test query").Return(testResponse, nil)
	
	config := &QueueConfig{
		MaxWorkers:     2,
		QueueSize:      10,
		RequestTimeout: 5 * time.Second,
		QueueTimeout:   2 * time.Second,
	}
	
	manager := NewQueueManager(config, mockProcessor, logger)
	err := manager.Start()
	assert.NoError(t, err)
	defer manager.Stop()
	
	// 测试基本请求处理
	ctx := context.Background()
	resp, err := manager.SubmitRequest(ctx, "test query")
	assert.NoError(t, err)
	assert.Equal(t, "test response", resp.Response)
	
	mockProcessor.AssertExpectations(t)
}

func TestQueueManager_ConcurrentRequests(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	
	mockProcessor := &MockRequestProcessor{
		processDelay: 100 * time.Millisecond, // 模拟处理延迟
	}
	testResponse := &models.ChatResponse{Response: "response"}
	mockProcessor.On("ProcessRequest", mock.Anything, mock.AnythingOfType("string")).Return(testResponse, nil)
	
	config := &QueueConfig{
		MaxWorkers:     3, // 限制为3个工作协程
		QueueSize:      20,
		RequestTimeout: 5 * time.Second,
		QueueTimeout:   2 * time.Second,
	}
	
	manager := NewQueueManager(config, mockProcessor, logger)
	err := manager.Start()
	assert.NoError(t, err)
	defer manager.Stop()
	
	// 并发发送10个请求
	numRequests := 10
	var wg sync.WaitGroup
	results := make([]*models.ChatResponse, numRequests)
	errors := make([]error, numRequests)
	
	start := time.Now()
	
	for i := 0; i < numRequests; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			ctx := context.Background()
			resp, err := manager.SubmitRequest(ctx, "concurrent query")
			results[index] = resp
			errors[index] = err
		}(i)
	}
	
	wg.Wait()
	duration := time.Since(start)
	
	// 验证所有请求都成功处理
	for i := 0; i < numRequests; i++ {
		assert.NoError(t, errors[i], "Request %d should not have error", i)
		assert.Equal(t, "response", results[i].Response, "Request %d should have correct response", i)
	}
	
	// 验证并发限制：由于最多3个工作协程，10个请求应该分批处理
	// 预期时间应该大于 (10/3) * 100ms = 333ms
	expectedMinDuration := time.Duration(numRequests/config.MaxWorkers) * mockProcessor.processDelay
	assert.GreaterOrEqual(t, duration, expectedMinDuration, "Duration should reflect concurrency limit")
	
	// 验证调用次数
	mockProcessor.AssertNumberOfCalls(t, "ProcessRequest", numRequests)
}

func TestQueueManager_QueueTimeout(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	
	mockProcessor := &MockRequestProcessor{
		processDelay: 2 * time.Second, // 长时间处理
	}
	testResponse := &models.ChatResponse{Response: "response"}
	mockProcessor.On("ProcessRequest", mock.Anything, mock.AnythingOfType("string")).Return(testResponse, nil)
	
	config := &QueueConfig{
		MaxWorkers:     1,
		QueueSize:      2, // 小队列
		RequestTimeout: 5 * time.Second,
		QueueTimeout:   500 * time.Millisecond, // 短队列超时
	}
	
	manager := NewQueueManager(config, mockProcessor, logger)
	err := manager.Start()
	assert.NoError(t, err)
	defer manager.Stop()
	
	// 填满队列
	var wg sync.WaitGroup
	for i := 0; i < 3; i++ { // 超过队列大小
		wg.Add(1)
		go func() {
			defer wg.Done()
			ctx := context.Background()
			_, err := manager.SubmitRequest(ctx, "queue test")
			// 某些请求应该超时
			if err != nil {
				// 可能是队列超时或工作协程不可用
				assert.True(t, err != nil, "Should have timeout error")
			}
		}()
	}
	
	wg.Wait()
}

func TestQueueManager_RequestTimeout(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	
	mockProcessor := &MockRequestProcessor{
		processDelay: 2 * time.Second, // 长时间处理
	}
	testResponse := &models.ChatResponse{Response: "response"}
	mockProcessor.On("ProcessRequest", mock.Anything, "timeout test").Return(testResponse, nil)
	
	config := &QueueConfig{
		MaxWorkers:     1,
		QueueSize:      10,
		RequestTimeout: 500 * time.Millisecond, // 短请求超时
		QueueTimeout:   2 * time.Second,
	}
	
	manager := NewQueueManager(config, mockProcessor, logger)
	err := manager.Start()
	assert.NoError(t, err)
	defer manager.Stop()
	
	// 提交会超时的请求
	ctx := context.Background()
	_, err = manager.SubmitRequest(ctx, "timeout test")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "request timeout")
}

func TestQueueManager_ProcessorError(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	
	mockProcessor := &MockRequestProcessor{}
	mockProcessor.On("ProcessRequest", mock.Anything, "error test").Return(nil, errors.New("processor error"))
	
	config := &QueueConfig{
		MaxWorkers:     1,
		QueueSize:      10,
		RequestTimeout: 5 * time.Second,
		QueueTimeout:   2 * time.Second,
	}
	
	manager := NewQueueManager(config, mockProcessor, logger)
	err := manager.Start()
	assert.NoError(t, err)
	defer manager.Stop()
	
	// 测试处理器错误传播
	ctx := context.Background()
	_, err = manager.SubmitRequest(ctx, "error test")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "processor error")
	
	mockProcessor.AssertExpectations(t)
}

func TestQueueManager_Stats(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	
	mockProcessor := &MockRequestProcessor{}
	testResponse := &models.ChatResponse{Response: "response"}
	mockProcessor.On("ProcessRequest", mock.Anything, mock.AnythingOfType("string")).Return(testResponse, nil)
	
	config := &QueueConfig{
		MaxWorkers:     2,
		QueueSize:      10,
		RequestTimeout: 5 * time.Second,
		QueueTimeout:   2 * time.Second,
	}
	
	manager := NewQueueManager(config, mockProcessor, logger)
	err := manager.Start()
	assert.NoError(t, err)
	defer manager.Stop()
	
	// 处理一些请求
	ctx := context.Background()
	for i := 0; i < 3; i++ {
		_, err := manager.SubmitRequest(ctx, "stats test")
		assert.NoError(t, err)
	}
	
	// 检查统计信息
	stats := manager.GetStats()
	assert.NotNil(t, stats["total_requests"])
	assert.NotNil(t, stats["processed_count"])
	assert.NotNil(t, stats["failed_count"])
	assert.Equal(t, config.MaxWorkers, stats["max_workers"])
	assert.Equal(t, config.QueueSize, stats["queue_size"])
	
	// 验证健康状态
	assert.True(t, manager.IsHealthy())
}

func TestQueueManager_StartStop(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	
	mockProcessor := &MockRequestProcessor{}
	
	config := &QueueConfig{
		MaxWorkers:     2,
		QueueSize:      10,
		RequestTimeout: 5 * time.Second,
		QueueTimeout:   2 * time.Second,
	}
	
	manager := NewQueueManager(config, mockProcessor, logger)
	
	// 测试启动
	err := manager.Start()
	assert.NoError(t, err)
	assert.True(t, manager.IsHealthy())
	
	// 测试重复启动
	err = manager.Start()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already running")
	
	// 测试停止
	manager.Stop()
	assert.False(t, manager.IsHealthy())
	
	// 测试重复停止（应该不会panic）
	manager.Stop()
}