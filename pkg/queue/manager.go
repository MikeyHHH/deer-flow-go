package queue

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/sirupsen/logrus"

	"deer-flow-go/pkg/models"
)

// RequestTask 请求任务
type RequestTask struct {
	ID       string
	Query    string
	Context  context.Context
	Response chan *TaskResult
	Created  time.Time
}

// TaskResult 任务结果
type TaskResult struct {
	Response *models.ChatResponse
	Error    error
}

// QueueConfig 队列配置
type QueueConfig struct {
	MaxWorkers     int           // 最大工作协程数
	QueueSize      int           // 队列大小
	RequestTimeout time.Duration // 请求超时时间
	QueueTimeout   time.Duration // 队列等待超时时间
}

// QueueManager 队列管理器
type QueueManager struct {
	config      *QueueConfig
	taskQueue   chan *RequestTask
	workerPool  chan chan *RequestTask
	workers     []*Worker
	logger      *logrus.Logger
	processor   RequestProcessor
	running     int32
	mu          sync.RWMutex

	// 统计信息
	totalRequests   int64
	processedCount  int64
	failedCount     int64
	queuedCount     int64
}

// RequestProcessor 请求处理器接口
type RequestProcessor interface {
	ProcessRequest(ctx context.Context, query string) (*models.ChatResponse, error)
}

// NewQueueManager 创建新的队列管理器
func NewQueueManager(config *QueueConfig, processor RequestProcessor, logger *logrus.Logger) *QueueManager {
	if config.MaxWorkers <= 0 {
		config.MaxWorkers = 3 // 默认3个工作协程
	}
	if config.QueueSize <= 0 {
		config.QueueSize = 100 // 默认队列大小100
	}
	if config.RequestTimeout <= 0 {
		config.RequestTimeout = 30 * time.Second // 默认30秒超时
	}
	if config.QueueTimeout <= 0 {
		config.QueueTimeout = 10 * time.Second // 默认10秒队列等待超时
	}

	qm := &QueueManager{
		config:     config,
		taskQueue:  make(chan *RequestTask, config.QueueSize),
		workerPool: make(chan chan *RequestTask, config.MaxWorkers),
		workers:    make([]*Worker, config.MaxWorkers),
		logger:     logger,
		processor:  processor,
	}

	// 创建工作协程
	for i := 0; i < config.MaxWorkers; i++ {
		worker := NewWorker(i+1, qm.workerPool, processor, logger)
		qm.workers[i] = worker
	}

	return qm
}

// Start 启动队列管理器
func (qm *QueueManager) Start() error {
	if !atomic.CompareAndSwapInt32(&qm.running, 0, 1) {
		return fmt.Errorf("queue manager is already running")
	}

	qm.logger.WithFields(logrus.Fields{
		"max_workers": qm.config.MaxWorkers,
		"queue_size":  qm.config.QueueSize,
	}).Info("Starting queue manager")

	// 启动所有工作协程
	for _, worker := range qm.workers {
		worker.Start()
	}

	// 启动调度器
	go qm.dispatcher()

	return nil
}

// Stop 停止队列管理器
func (qm *QueueManager) Stop() {
	if !atomic.CompareAndSwapInt32(&qm.running, 1, 0) {
		return
	}

	qm.logger.Info("Stopping queue manager")

	// 关闭任务队列
	close(qm.taskQueue)

	// 停止所有工作协程
	for _, worker := range qm.workers {
		worker.Stop()
	}

	qm.logger.Info("Queue manager stopped")
}

// SubmitRequest 提交请求到队列
func (qm *QueueManager) SubmitRequest(ctx context.Context, query string) (*models.ChatResponse, error) {
	if atomic.LoadInt32(&qm.running) == 0 {
		return nil, fmt.Errorf("queue manager is not running")
	}

	// 创建任务
	task := &RequestTask{
		ID:       fmt.Sprintf("task_%d_%d", time.Now().UnixNano(), atomic.AddInt64(&qm.totalRequests, 1)),
		Query:    query,
		Context:  ctx,
		Response: make(chan *TaskResult, 1),
		Created:  time.Now(),
	}

	qm.logger.WithFields(logrus.Fields{
		"task_id": task.ID,
		"query":   query,
	}).Debug("Submitting request to queue")

	// 尝试将任务加入队列
	select {
	case qm.taskQueue <- task:
		atomic.AddInt64(&qm.queuedCount, 1)
	case <-time.After(qm.config.QueueTimeout):
		atomic.AddInt64(&qm.failedCount, 1)
		return nil, fmt.Errorf("request queue is full, timeout after %v", qm.config.QueueTimeout)
	case <-ctx.Done():
		atomic.AddInt64(&qm.failedCount, 1)
		return nil, ctx.Err()
	}

	// 等待结果
	select {
	case result := <-task.Response:
		if result.Error != nil {
			atomic.AddInt64(&qm.failedCount, 1)
			return nil, result.Error
		}
		atomic.AddInt64(&qm.processedCount, 1)
		return result.Response, nil
	case <-time.After(qm.config.RequestTimeout):
		atomic.AddInt64(&qm.failedCount, 1)
		return nil, fmt.Errorf("request timeout after %v", qm.config.RequestTimeout)
	case <-ctx.Done():
		atomic.AddInt64(&qm.failedCount, 1)
		return nil, ctx.Err()
	}
}

// dispatcher 调度器，将任务分发给工作协程
func (qm *QueueManager) dispatcher() {
	qm.logger.Info("Queue dispatcher started")
	defer qm.logger.Info("Queue dispatcher stopped")

	for {
		select {
		case task, ok := <-qm.taskQueue:
			if !ok {
				return // 队列已关闭
			}

			// 获取可用的工作协程
			select {
			case workerTaskQueue := <-qm.workerPool:
				// 将任务分发给工作协程
				select {
				case workerTaskQueue <- task:
					atomic.AddInt64(&qm.queuedCount, -1)
				case <-time.After(1 * time.Second):
					// 工作协程超时，返回错误
					task.Response <- &TaskResult{
						Error: fmt.Errorf("worker assignment timeout"),
					}
					atomic.AddInt64(&qm.queuedCount, -1)
				}
			case <-time.After(qm.config.QueueTimeout):
				// 没有可用的工作协程
				task.Response <- &TaskResult{
					Error: fmt.Errorf("no available workers, timeout after %v", qm.config.QueueTimeout),
				}
				atomic.AddInt64(&qm.queuedCount, -1)
			}
		}
	}
}

// GetStats 获取队列统计信息
func (qm *QueueManager) GetStats() map[string]interface{} {
	qm.mu.RLock()
	defer qm.mu.RUnlock()

	return map[string]interface{}{
		"running":         atomic.LoadInt32(&qm.running) == 1,
		"max_workers":     qm.config.MaxWorkers,
		"queue_size":      qm.config.QueueSize,
		"queued_count":    atomic.LoadInt64(&qm.queuedCount),
		"total_requests":  atomic.LoadInt64(&qm.totalRequests),
		"processed_count": atomic.LoadInt64(&qm.processedCount),
		"failed_count":    atomic.LoadInt64(&qm.failedCount),
		"queue_length":    len(qm.taskQueue),
		"available_workers": len(qm.workerPool),
	}
}

// IsHealthy 检查队列管理器健康状态
func (qm *QueueManager) IsHealthy() bool {
	return atomic.LoadInt32(&qm.running) == 1
}