package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"deer-flow-go/internal/workflow"
	"deer-flow-go/pkg/config"
	"deer-flow-go/pkg/handlers"
)

func main() {
	// 加载配置
	cfg, err := config.LoadConfig()
	if err != nil {
		fmt.Printf("Failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	// 设置日志
	logger := logrus.New()
	logLevel, err := logrus.ParseLevel(cfg.LogLevel)
	if err != nil {
		logLevel = logrus.InfoLevel
	}
	logger.SetLevel(logLevel)
	logger.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
	})

	logger.Info("Starting deer-flow-go agent dialogue system")

	// 设置Gin模式
	if logLevel == logrus.DebugLevel {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	// 创建工作流
	agentWorkflow := workflow.NewAgentWorkflow(cfg, logger)

	// 验证工作流配置
	ctx := context.Background()
	if err := agentWorkflow.ValidateWorkflow(ctx); err != nil {
		logger.WithError(err).Warn("Workflow validation failed, but continuing startup")
	}

	// 创建路由器
	router := gin.Default()

	// 设置API处理器
	apiHandler := handlers.NewAPIHandler(agentWorkflow, logger)
	apiHandler.SetupRoutes(router)

	// 启动服务器
	serverAddr := fmt.Sprintf(":%s", cfg.Port)
	logger.WithField("addr", serverAddr).Info("Starting HTTP server")

	// 在goroutine中启动服务器
	go func() {
		if err := router.Run(serverAddr); err != nil {
			logger.WithError(err).Fatal("Failed to start HTTP server")
		}
	}()

	// 等待中断信号
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server...")
}