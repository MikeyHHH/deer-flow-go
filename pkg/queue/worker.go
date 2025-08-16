package queue

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/sirupsen/logrus"
)

// Worker 工作协程
type Worker struct {
	id         int
	workerPool chan chan *RequestTask
	taskQueue  chan *RequestTask
	processor  RequestProcessor
	logger     *logrus.Logger
	running    int32
	quit       chan bool
}

// NewWorker 创建新的工作协程
func NewWorker(id int, workerPool chan chan *RequestTask, processor RequestProcessor, logger *logrus.Logger) *Worker {
	return &Worker{
		id:         id,
		workerPool: workerPool,
		taskQueue:  make(chan *RequestTask),
		processor:  processor,
		logger:     logger,
		quit:       make(chan bool),
	}
}

// Start 启动工作协程
func (w *Worker) Start() {
	if !atomic.CompareAndSwapInt32(&w.running, 0, 1) {
		return
	}

	w.logger.WithField("worker_id", w.id).Debug("Starting worker")

	go func() {
		defer func() {
			atomic.StoreInt32(&w.running, 0)
			w.logger.WithField("worker_id", w.id).Debug("Worker stopped")
		}()

		for {
			// 将自己的任务队列注册到工作池中
			select {
			case w.workerPool <- w.taskQueue:
				// 等待任务
				select {
				case task := <-w.taskQueue:
					w.processTask(task)
				case <-w.quit:
					return
				}
			case <-w.quit:
				return
			}
		}
	}()
}

// Stop 停止工作协程
func (w *Worker) Stop() {
	if atomic.LoadInt32(&w.running) == 0 {
		return
	}

	w.logger.WithField("worker_id", w.id).Debug("Stopping worker")
	close(w.quit)
}

// processTask 处理任务
func (w *Worker) processTask(task *RequestTask) {
	start := time.Now()
	w.logger.WithFields(logrus.Fields{
		"worker_id": w.id,
		"task_id":   task.ID,
		"query":     task.Query,
	}).Debug("Processing task")

	defer func() {
		if r := recover(); r != nil {
			w.logger.WithFields(logrus.Fields{
				"worker_id": w.id,
				"task_id":   task.ID,
				"panic":     r,
			}).Error("Worker panic during task processing")

			task.Response <- &TaskResult{
				Error: fmt.Errorf("internal error during task processing"),
			}
		}
	}()

	// 创建带超时的上下文
	ctx, cancel := context.WithTimeout(task.Context, 30*time.Second)
	defer cancel()

	// 处理请求
	response, err := w.processor.ProcessRequest(ctx, task.Query)

	duration := time.Since(start)
	w.logger.WithFields(logrus.Fields{
		"worker_id": w.id,
		"task_id":   task.ID,
		"duration":  duration,
		"success":   err == nil,
	}).Debug("Task processing completed")

	// 发送结果
	select {
	case task.Response <- &TaskResult{
		Response: response,
		Error:    err,
	}:
	case <-time.After(1 * time.Second):
		w.logger.WithFields(logrus.Fields{
			"worker_id": w.id,
			"task_id":   task.ID,
		}).Warn("Failed to send task result, response channel timeout")
	}
}

// IsRunning 检查工作协程是否运行中
func (w *Worker) IsRunning() bool {
	return atomic.LoadInt32(&w.running) == 1
}