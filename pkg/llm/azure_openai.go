package llm

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/sashabaranov/go-openai"
	"github.com/sirupsen/logrus"

	"deer-flow-go/pkg/config"
	"deer-flow-go/pkg/models"
)

// AzureOpenAIClient Azure OpenAI 客户端
type AzureOpenAIClient struct {
	client *openai.Client
	config *config.AzureOpenAIConfig
	logger *logrus.Logger
}

// NewAzureOpenAIClient 创建新的 Azure OpenAI 客户端
func NewAzureOpenAIClient(cfg *config.AzureOpenAIConfig, logger *logrus.Logger) *AzureOpenAIClient {
	clientConfig := openai.DefaultAzureConfig(cfg.APIKey, cfg.Endpoint)
	clientConfig.APIVersion = cfg.APIVersion

	client := openai.NewClientWithConfig(clientConfig)

	return &AzureOpenAIClient{
		client: client,
		config: cfg,
		logger: logger,
	}
}

// ChatCompletion 调用聊天完成API
func (c *AzureOpenAIClient) ChatCompletion(ctx context.Context, messages []models.ChatMessage, systemPrompt string) (string, error) {
	// 构建OpenAI消息格式
	openaiMessages := make([]openai.ChatCompletionMessage, 0, len(messages)+1)

	// 添加系统提示词
	if systemPrompt != "" {
		openaiMessages = append(openaiMessages, openai.ChatCompletionMessage{
			Role:    openai.ChatMessageRoleSystem,
			Content: systemPrompt,
		})
	}

	// 添加用户消息
	for _, msg := range messages {
		role := openai.ChatMessageRoleUser
		switch msg.Role {
		case "system":
			role = openai.ChatMessageRoleSystem
		case "assistant":
			role = openai.ChatMessageRoleAssistant
		case "user":
			role = openai.ChatMessageRoleUser
		}

		openaiMessages = append(openaiMessages, openai.ChatCompletionMessage{
			Role:    role,
			Content: msg.Content,
		})
	}

	// 创建请求
	req := openai.ChatCompletionRequest{
		Model:       c.config.Deployment,
		Messages:    openaiMessages,
		Temperature: c.config.Temperature,
		Stream:      false,
	}

	c.logger.WithFields(logrus.Fields{
		"deployment": c.config.Deployment,
		"messages":   len(openaiMessages),
	}).Debug("Calling Azure OpenAI API")

	// 调用API
	resp, err := c.client.CreateChatCompletion(ctx, req)
	if err != nil {
		c.logger.WithError(err).Error("Failed to call Azure OpenAI API")
		return "", fmt.Errorf("Azure OpenAI API call failed: %w", err)
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("no response choices returned from Azure OpenAI")
	}

	result := resp.Choices[0].Message.Content
	c.logger.WithFields(logrus.Fields{
		"response_length": len(result),
		"usage_tokens":    resp.Usage.TotalTokens,
	}).Debug("Azure OpenAI API response received")

	return result, nil
}

// ParseQueryToMCP 将用户查询解析为MCP请求格式
func (c *AzureOpenAIClient) ParseQueryToMCP(ctx context.Context, query string) (*models.MCPRequest, error) {
	systemPrompt := `你是一个专门将用户查询转换为MCP协议格式的助手。

你的任务是：
1. 分析用户的查询内容
2. 判断查询类型并选择合适的处理方法
3. 将查询转换为标准的MCP请求格式

判断规则：
- 如果查询涉及天气信息（如天气、气温、降雨、预报等），使用get_weather或get_weather_forecast方法
- 如果查询涉及其他实时信息（如新闻、股价等），使用search方法
- 如果查询是一般知识问题、问候语、数学计算等，使用direct_response方法

城市名处理规则：
- 对于天气查询，必须将中文城市名转换为对应的英文城市名
- 常见转换：北京→Beijing, 上海→Shanghai, 广州→Guangzhou, 深圳→Shenzhen, 杭州→Hangzhou, 南京→Nanjing, 成都→Chengdu, 西安→Xi'an, 重庆→Chongqing, 天津→Tianjin, 武汉→Wuhan, 苏州→Suzhou, 青岛→Qingdao, 大连→Dalian, 厦门→Xiamen, 长沙→Changsha, 哈尔滨→Harbin, 沈阳→Shenyang, 郑州→Zhengzhou, 济南→Jinan
- 如果是其他中文城市名，请转换为对应的英文拼音形式

请严格按照以下JSON格式返回：
对于天气查询：
{
  "method": "get_weather",
  "params": {
    "city": "英文城市名称（如Beijing、Shanghai等）"
  }
}

对于天气预报查询（包含"预报"、"未来"、"明天"等关键词）：
{
  "method": "get_weather_forecast",
  "params": {
    "city": "英文城市名称（如Beijing、Shanghai等）",
    "days": 3
  }
}

对于需要搜索的查询：
{
  "method": "search",
  "params": {
    "query": "优化后的搜索关键词",
    "max_results": 5,
    "search_depth": "advanced"
  }
}

对于不需要搜索的查询：
{
  "method": "direct_response",
  "params": {
    "message": "直接回复内容"
  }
}

只返回JSON格式，不要添加任何其他文字说明。`

	messages := []models.ChatMessage{
		{Role: "user", Content: query},
	}

	response, err := c.ChatCompletion(ctx, messages, systemPrompt)
	if err != nil {
		return nil, fmt.Errorf("failed to parse query to MCP: %w", err)
	}
	fmt.Println("!!!!!!!!!!!!!!",response)
	c.logger.WithFields(logrus.Fields{
		"llm_response": response,
	}).Debug("LLM response for MCP parsing")

	// 解析JSON响应
	var mcpRequest models.MCPRequest
	err = json.Unmarshal([]byte(response), &mcpRequest)
	if err != nil {
		c.logger.WithError(err).Warn("Failed to parse LLM response as JSON, falling back to search")
		// 如果解析失败，默认使用搜索
		mcpRequest = models.MCPRequest{
			Method: "search",
			Params: map[string]interface{}{
				"query":        query,
				"max_results":  5,
				"search_depth": "advanced",
			},
		}
	}

	c.logger.WithFields(logrus.Fields{
		"original_query": query,
		"mcp_method":     mcpRequest.Method,
	}).Debug("Query parsed to MCP request")

	return &mcpRequest, nil
}

// FormatSearchResults 格式化搜索结果
func (c *AzureOpenAIClient) FormatSearchResults(ctx context.Context, query string, searchResults *models.SearchResponse) (string, error) {
	systemPrompt := `你是一个专业的信息整理助手。你的任务是：

1. 分析用户的原始问题
2. 整理和总结搜索到的信息
3. 提供准确、有用、结构化的回答
4. 确保信息的时效性和准确性

请遵循以下原则：
- 直接回答用户的问题
- 使用清晰的结构组织信息
- 引用相关的数据和事实
- 保持客观和中立
- 如果信息不足，请明确说明

请用中文回答，格式要清晰易读。`

	// 构建包含搜索结果的用户消息
	userContent := fmt.Sprintf("原始问题：%s\n\n搜索结果：\n", query)
	for i, result := range searchResults.Results {
		userContent += fmt.Sprintf("%d. 标题：%s\n   链接：%s\n   内容：%s\n\n",
			i+1, result.Title, result.URL, result.Content)
	}

	messages := []models.ChatMessage{
		{Role: "user", Content: userContent},
	}

	response, err := c.ChatCompletion(ctx, messages, systemPrompt)
	if err != nil {
		return "", fmt.Errorf("failed to format search results: %w", err)
	}

	c.logger.WithFields(logrus.Fields{
		"original_query":  query,
		"search_results":  len(searchResults.Results),
		"response_length": len(response),
	}).Debug("Search results formatted")

	return response, nil
}
