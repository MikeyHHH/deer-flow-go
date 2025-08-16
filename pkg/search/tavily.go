package search

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/sirupsen/logrus"

	"deer-flow-go/pkg/config"
	"deer-flow-go/pkg/models"
)

// TavilyClient Tavily搜索客户端
type TavilyClient struct {
	config     *config.TavilyConfig
	httpClient *http.Client
	logger     *logrus.Logger
}

// TavilySearchRequest Tavily API请求结构
type TavilySearchRequest struct {
	APIKey            string   `json:"api_key"`
	Query             string   `json:"query"`
	SearchDepth       string   `json:"search_depth,omitempty"`
	IncludeAnswer     bool     `json:"include_answer,omitempty"`
	IncludeImages     bool     `json:"include_images,omitempty"`
	IncludeRawContent bool     `json:"include_raw_content,omitempty"`
	MaxResults        int      `json:"max_results,omitempty"`
	IncludeDomains    []string `json:"include_domains,omitempty"`
	ExcludeDomains    []string `json:"exclude_domains,omitempty"`
}

// TavilySearchResponse Tavily API响应结构
type TavilySearchResponse struct {
	Answer            string         `json:"answer"`
	Query             string         `json:"query"`
	ResponseTime      float64        `json:"response_time"`
	Images            []TavilyImage  `json:"images"`
	Results           []TavilyResult `json:"results"`
	FollowUpQuestions []string       `json:"follow_up_questions"`
}

// TavilyResult Tavily搜索结果
type TavilyResult struct {
	Title         string  `json:"title"`
	URL           string  `json:"url"`
	Content       string  `json:"content"`
	RawContent    string  `json:"raw_content"`
	Score         float64 `json:"score"`
	PublishedDate string  `json:"published_date"`
}

// TavilyImage Tavily图片结果
type TavilyImage struct {
	URL         string `json:"url"`
	Description string `json:"description"`
}

// NewTavilyClient 创建新的Tavily客户端
func NewTavilyClient(cfg *config.TavilyConfig, logger *logrus.Logger) *TavilyClient {
	return &TavilyClient{
		config: cfg,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		logger: logger,
	}
}

// Search 执行搜索
func (c *TavilyClient) Search(ctx context.Context, query string) (*models.SearchResponse, error) {
	// 构建请求
	req := TavilySearchRequest{
		APIKey:            c.config.APIKey,
		Query:             query,
		SearchDepth:       c.config.SearchDepth,
		IncludeAnswer:     true,
		IncludeImages:     false,
		IncludeRawContent: false,
		MaxResults:        c.config.MaxResults,
	}

	// 序列化请求
	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	c.logger.WithFields(logrus.Fields{
		"query":        query,
		"search_depth": c.config.SearchDepth,
		"max_results":  c.config.MaxResults,
	}).Debug("Sending Tavily search request")
	// 创建HTTP请求
	httpReq, err := http.NewRequestWithContext(ctx, "POST", "https://api.tavily.com/search", bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	// 发送请求
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send HTTP request: %w", err)
	}
	defer resp.Body.Close()

	// 读取响应
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		c.logger.WithFields(logrus.Fields{
			"status_code": resp.StatusCode,
			"response":    string(respBody),
		}).Error("Tavily API returned error")
		return nil, fmt.Errorf("Tavily API error: status %d, body: %s", resp.StatusCode, string(respBody))
	}

	// 解析响应
	var tavilyResp TavilySearchResponse
	if err := json.Unmarshal(respBody, &tavilyResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	c.logger.WithFields(logrus.Fields{
		"results_count": len(tavilyResp.Results),
		"response_time": tavilyResp.ResponseTime,
		"has_answer":    tavilyResp.Answer != "",
	}).Debug("Tavily search completed")

	// 转换为标准格式
	searchResp := &models.SearchResponse{
		Query:   query,
		Answer:  tavilyResp.Answer,
		Results: make([]models.SearchResult, len(tavilyResp.Results)),
	}

	for i, result := range tavilyResp.Results {
		searchResp.Results[i] = models.SearchResult{
			Title:   result.Title,
			URL:     result.URL,
			Content: result.Content,
			Score:   result.Score,
		}
	}

	return searchResp, nil
}

// CleanResults 清理和优化搜索结果
func (c *TavilyClient) CleanResults(results *models.SearchResponse) *models.SearchResponse {
	if results == nil {
		return nil
	}

	// 过滤和清理结果
	cleanedResults := make([]models.SearchResult, 0, len(results.Results))
	for _, result := range results.Results {
		// 跳过空内容或评分过低的结果
		if result.Content == "" || result.Score < 0.1 {
			continue
		}

		// 限制内容长度
		if len(result.Content) > 1000 {
			result.Content = result.Content[:1000] + "..."
		}

		cleanedResults = append(cleanedResults, result)
	}

	results.Results = cleanedResults
	c.logger.WithFields(logrus.Fields{
		"original_count": len(results.Results),
		"cleaned_count":  len(cleanedResults),
	}).Debug("Search results cleaned")

	return results
}
