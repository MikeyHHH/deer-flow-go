package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/sirupsen/logrus"

	"deer-flow-go/pkg/models"
)

// Client MCP协议客户端
type Client struct {
	cmd       *exec.Cmd
	stdin     io.WriteCloser
	stdout    io.ReadCloser
	scanner   *bufio.Scanner
	logger    *logrus.Logger
	mutex     sync.Mutex
	requestID int
	running   bool
}

// MCPJSONRPCMessage MCP JSON-RPC 2.0 消息
type MCPJSONRPCMessage struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      int         `json:"id"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params,omitempty"`
	Result  interface{} `json:"result,omitempty"`
	Error   interface{} `json:"error,omitempty"`
}

// InitializeParams MCP初始化参数
type InitializeParams struct {
	ProtocolVersion string                 `json:"protocolVersion"`
	Capabilities    map[string]interface{} `json:"capabilities"`
	ClientInfo      ClientInfo             `json:"clientInfo"`
}

type ClientInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// CallToolParams 工具调用参数
type CallToolParams struct {
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments"`
}

// NewClient 创建MCP客户端
func NewClient(logger *logrus.Logger) *Client {
	return &Client{
		logger:    logger,
		requestID: 0,
		running:   false,
	}
}

// Start 启动MCP服务器进程并建立连接
func (c *Client) Start(ctx context.Context) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if c.running {
		return nil
	}

	c.logger.Info("Starting MCP server process...")

	// 启动MCP服务器进程
	c.cmd = exec.CommandContext(ctx, "go", "run", "cmd/server/main.go")

	// 创建管道
	stdin, err := c.cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdin pipe: %w", err)
	}
	c.stdin = stdin

	stdout, err := c.cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}
	c.stdout = stdout
	c.scanner = bufio.NewScanner(stdout)

	// 启动进程
	if err := c.cmd.Start(); err != nil {
		return fmt.Errorf("failed to start MCP server: %w", err)
	}

	// 等待服务器启动
	time.Sleep(500 * time.Millisecond)

	// 发送初始化消息
	if err := c.initialize(); err != nil {
		return fmt.Errorf("failed to initialize MCP connection: %w", err)
	}

	c.running = true
	c.logger.Info("MCP server process started and initialized")
	return nil
}

// Stop 停止MCP服务器进程
func (c *Client) Stop() error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if !c.running {
		return nil
	}

	c.logger.Info("Stopping MCP server process...")

	if c.stdin != nil {
		c.stdin.Close()
	}
	if c.stdout != nil {
		c.stdout.Close()
	}
	if c.cmd != nil && c.cmd.Process != nil {
		c.cmd.Process.Kill()
		c.cmd.Wait()
	}

	c.running = false
	c.logger.Info("MCP server process stopped")
	return nil
}

// initialize 发送MCP初始化消息
func (c *Client) initialize() error {
	initMsg := MCPJSONRPCMessage{
		JSONRPC: "2.0",
		ID:      c.getNextRequestID(),
		Method:  "initialize",
		Params: InitializeParams{
			ProtocolVersion: "2024-11-05",
			Capabilities:    map[string]interface{}{"tools": map[string]interface{}{}},
			ClientInfo: ClientInfo{
				Name:    "deer-flow-api-client",
				Version: "1.0.0",
			},
		},
	}

	// 发送消息
	if err := c.sendMessage(initMsg); err != nil {
		return err
	}

	// 读取响应
	_, err := c.readResponse()
	return err
}

// ProcessRequest 处理MCP请求（真正的协议调用）
func (c *Client) ProcessRequest(ctx context.Context, req *models.MCPRequest) (*models.MCPResponse, error) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if !c.running {
		return nil, fmt.Errorf("MCP client is not running")
	}

	c.logger.WithFields(logrus.Fields{
		"method": req.Method,
	}).Debug("Processing MCP request via JSON-RPC")

	// 构造JSON-RPC消息
	var rpcMsg MCPJSONRPCMessage

	switch req.Method {
	case "get_weather", "get_weather_forecast":
		// 调用工具
		params, ok := req.Params.(map[string]interface{})
		if !ok {
			return &models.MCPResponse{
				Error: &models.MCPError{
					Code:    -32602,
					Message: "Invalid params format",
				},
			}, nil
		}

		rpcMsg = MCPJSONRPCMessage{
			JSONRPC: "2.0",
			ID:      c.getNextRequestID(),
			Method:  "tools/call",
			Params: CallToolParams{
				Name:      req.Method,
				Arguments: params,
			},
		}

	case "search":
		// 调用搜索工具
		params, ok := req.Params.(map[string]interface{})
		if !ok {
			return &models.MCPResponse{
				Error: &models.MCPError{
					Code:    -32602,
					Message: "Invalid params format",
				},
			}, nil
		}

		rpcMsg = MCPJSONRPCMessage{
			JSONRPC: "2.0",
			ID:      c.getNextRequestID(),
			Method:  "tools/call",
			Params: CallToolParams{
				Name:      req.Method,
				Arguments: params,
			},
		}

	case "direct_response":
		// 直接响应不需要MCP调用
		params, ok := req.Params.(map[string]interface{})
		if !ok {
			return &models.MCPResponse{
				Error: &models.MCPError{
					Code:    -32602,
					Message: "Invalid params format",
				},
			}, nil
		}

		response, ok := params["response"].(string)
		if !ok {
			return &models.MCPResponse{
				Error: &models.MCPError{
					Code:    -32602,
					Message: "Missing response parameter",
				},
			}, nil
		}

		return &models.MCPResponse{
			Result: map[string]interface{}{
				"content": response,
				"type":    "direct",
			},
		}, nil

	default:
		return &models.MCPResponse{
			Error: &models.MCPError{
				Code:    -32601,
				Message: fmt.Sprintf("Method not found: %s", req.Method),
			},
		}, nil
	}

	// 发送JSON-RPC消息
	if err := c.sendMessage(rpcMsg); err != nil {
		return nil, fmt.Errorf("failed to send MCP message: %w", err)
	}

	// 读取响应
	response, err := c.readResponse()
	if err != nil {
		return nil, fmt.Errorf("failed to read MCP response: %w", err)
	}

	// 解析响应
	return c.parseResponse(response)
}

// sendMessage 发送JSON-RPC消息
func (c *Client) sendMessage(msg MCPJSONRPCMessage) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	c.logger.WithFields(logrus.Fields{
		"message": string(data),
	}).Debug("Sending MCP message")

	if _, err := c.stdin.Write(append(data, '\n')); err != nil {
		return fmt.Errorf("failed to write message: %w", err)
	}

	return nil
}

// readResponse 读取JSON-RPC响应
func (c *Client) readResponse() (*MCPJSONRPCMessage, error) {
	if !c.scanner.Scan() {
		if err := c.scanner.Err(); err != nil {
			return nil, fmt.Errorf("failed to read response: %w", err)
		}
		return nil, fmt.Errorf("no response received")
	}

	data := c.scanner.Bytes()
	c.logger.WithFields(logrus.Fields{
		"response": string(data),
	}).Debug("Received MCP response")

	var response MCPJSONRPCMessage
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &response, nil
}

// parseResponse 解析MCP响应为标准格式
func (c *Client) parseResponse(rpcResponse *MCPJSONRPCMessage) (*models.MCPResponse, error) {
	if rpcResponse.Error != nil {
		return &models.MCPResponse{
			Error: &models.MCPError{
				Code:    -1,
				Message: fmt.Sprintf("MCP server error: %v", rpcResponse.Error),
			},
		}, nil
	}

	if rpcResponse.Result == nil {
		return &models.MCPResponse{
			Error: &models.MCPError{
				Code:    -1,
				Message: "No result in MCP response",
			},
		}, nil
	}

	// 解析工具调用结果
	result, ok := rpcResponse.Result.(map[string]interface{})
	if !ok {
		return &models.MCPResponse{
			Error: &models.MCPError{
				Code:    -1,
				Message: "Invalid result format",
			},
		}, nil
	}

	// 检查是否有content字段（工具调用结果）
	if content, exists := result["content"]; exists {
		contentList, ok := content.([]interface{})
		if ok && len(contentList) > 0 {
			if contentItem, ok := contentList[0].(map[string]interface{}); ok {
				if text, ok := contentItem["text"].(string); ok {
					// 根据内容判断工具类型
					toolType := "unknown"
					if strings.Contains(text, "🌤️") || strings.Contains(text, "温度") {
						toolType = "weather"
					} else if strings.Contains(text, "🔍") || strings.Contains(text, "搜索结果") {
						toolType = "search"
					}

					return &models.MCPResponse{
						Result: map[string]interface{}{
							"content": text,
							"type":    toolType,
						},
					}, nil
				}
			}
		}
	}

	return &models.MCPResponse{
		Result: result,
	}, nil
}

// getNextRequestID 获取下一个请求ID
func (c *Client) getNextRequestID() int {
	c.requestID++
	return c.requestID
}

// HealthCheck 健康检查
func (c *Client) HealthCheck(ctx context.Context) error {
	if !c.running {
		return fmt.Errorf("MCP client is not running")
	}
	return nil
}

// GetCapabilities 获取能力信息
func (c *Client) GetCapabilities() map[string]interface{} {
	return map[string]interface{}{
		"tools":       []string{"get_weather", "get_weather_forecast"},
		"description": "Real MCP client with JSON-RPC 2.0 protocol",
		"version":     "1.0.0",
		"protocol":    "MCP 2024-11-05",
	}
}
