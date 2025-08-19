package search

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/sirupsen/logrus"
)

// SearchMCPServer MCP服务器实现
type SearchMCPServer struct {
	searchClient *TavilyClient
	logger       *logrus.Logger
	server       *server.MCPServer
}

// NewSearchMCPServer 创建新的搜索MCP服务器
func NewSearchMCPServer(searchClient *TavilyClient, logger *logrus.Logger) *SearchMCPServer {
	s := &SearchMCPServer{
		searchClient: searchClient,
		logger:       logger,
	}

	// 创建MCP服务器
	s.server = server.NewMCPServer("search-server", "1.0.0")

	// 注册工具
	s.registerTools()

	return s
}

// registerTools 注册搜索工具
func (s *SearchMCPServer) registerTools() {
	// 注册搜索工具
	searchTool := mcp.NewTool("search",
		mcp.WithDescription("搜索互联网信息，返回相关的搜索结果"),
		mcp.WithString("query",
			mcp.Required(),
			mcp.Description("搜索查询关键词"),
		),
		mcp.WithNumber("max_results",
			mcp.Description("最大返回结果数量，默认为5"),
		),
	)

	s.server.AddTool(searchTool, s.handleSearch)
}



// handleSearch 处理搜索请求
func (s *SearchMCPServer) handleSearch(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	s.logger.WithFields(logrus.Fields{
		"tool": "search",
	}).Debug("Processing search request")

	// 解析请求参数
	query, err := request.RequireString("query")
	if err != nil {
		s.logger.WithError(err).Error("Failed to parse query parameter")
		return mcp.NewToolResultError(fmt.Sprintf("参数解析失败: %v", err)), nil
	}

	if query == "" {
		return mcp.NewToolResultError("搜索查询不能为空"), nil
	}

	// 执行搜索
	searchResults, err := s.searchClient.Search(ctx, query)
	if err != nil {
		s.logger.WithError(err).Error("Failed to perform search")
		return mcp.NewToolResultError(fmt.Sprintf("搜索失败: %v", err)), nil
	}

	// 格式化搜索结果
	resultText := fmt.Sprintf("🔍 搜索结果 \"%s\":\n\n", query)
	for i, result := range searchResults.Results {
		resultText += fmt.Sprintf("%d. **%s**\n", i+1, result.Title)
		resultText += fmt.Sprintf("   📄 %s\n", result.Content)
		resultText += fmt.Sprintf("   🔗 %s\n", result.URL)
		if i < len(searchResults.Results)-1 {
			resultText += "\n"
		}
	}

	return mcp.NewToolResultText(resultText), nil
}

// GetServer 获取MCP服务器实例
func (s *SearchMCPServer) GetServer() *server.MCPServer {
	return s.server
}

// Start 启动MCP服务器
func (s *SearchMCPServer) Start(ctx context.Context) error {
	return server.ServeStdio(s.server)
}

// GetCapabilities 获取服务器能力
func (s *SearchMCPServer) GetCapabilities() map[string]interface{} {
	return map[string]interface{}{
		"tools": []string{"search"},
		"description": "互联网搜索服务，提供实时搜索功能",
	}
}