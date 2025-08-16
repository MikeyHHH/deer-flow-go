package weather

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/sirupsen/logrus"
)

// WeatherConfig 天气服务配置
type WeatherConfig struct {
	APIKey  string `yaml:"api_key"`
	BaseURL string `yaml:"base_url"`
	Timeout int    `yaml:"timeout"`
}

// WeatherClient 天气服务客户端
type WeatherClient struct {
	config     *WeatherConfig
	httpClient *http.Client
	logger     *logrus.Logger
}

// WeatherData 天气数据结构
type WeatherData struct {
	Location    string  `json:"location"`
	Temperature float64 `json:"temperature"`
	Description string  `json:"description"`
	Humidity    int     `json:"humidity"`
	WindSpeed   float64 `json:"wind_speed"`
	Timestamp   string  `json:"timestamp"`
}

// WeatherAPIResponse OpenWeatherMap API响应结构
type WeatherAPIResponse struct {
	Name string `json:"name"`
	Main struct {
		Temp     float64 `json:"temp"`
		Humidity int     `json:"humidity"`
	} `json:"main"`
	Weather []struct {
		Description string `json:"description"`
	} `json:"weather"`
	Wind struct {
		Speed float64 `json:"speed"`
	} `json:"wind"`
}

// NewWeatherClient 创建新的天气客户端
func NewWeatherClient(config *WeatherConfig, logger *logrus.Logger) *WeatherClient {
	return &WeatherClient{
		config: config,
		httpClient: &http.Client{
			Timeout: time.Duration(config.Timeout) * time.Second,
		},
		logger: logger,
	}
}

// GetWeather 获取指定城市的天气信息
func (w *WeatherClient) GetWeather(ctx context.Context, city string) (*WeatherData, error) {
	w.logger.WithFields(logrus.Fields{
		"city": city,
	}).Debug("Fetching weather data")

	// 构建请求URL
	params := url.Values{}
	params.Add("q", city)
	params.Add("appid", w.config.APIKey)
	params.Add("units", "metric") // 使用摄氏度
	params.Add("lang", "zh_cn")   // 中文描述

	requestURL := fmt.Sprintf("%s/weather?%s", w.config.BaseURL, params.Encode())

	// 创建HTTP请求
	req, err := http.NewRequestWithContext(ctx, "GET", requestURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// 发送请求
	resp, err := w.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API request failed with status: %d", resp.StatusCode)
	}

	// 解析响应
	var apiResp WeatherAPIResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// 转换为内部数据结构
	weatherData := &WeatherData{
		Location:    apiResp.Name,
		Temperature: apiResp.Main.Temp,
		Humidity:    apiResp.Main.Humidity,
		WindSpeed:   apiResp.Wind.Speed,
		Timestamp:   time.Now().Format(time.RFC3339),
	}

	if len(apiResp.Weather) > 0 {
		weatherData.Description = apiResp.Weather[0].Description
	}

	w.logger.WithFields(logrus.Fields{
		"location":    weatherData.Location,
		"temperature": weatherData.Temperature,
		"description": weatherData.Description,
	}).Debug("Weather data fetched successfully")

	return weatherData, nil
}

// GetForecast 获取天气预报
func (c *WeatherClient) GetForecast(ctx context.Context, city string, days int) ([]WeatherData, error) {
	c.logger.WithFields(logrus.Fields{
		"city": city,
		"days": days,
	}).Debug("Getting weather forecast")

	if city == "" {
		return nil, fmt.Errorf("city name cannot be empty")
	}

	if days <= 0 || days > 5 {
		days = 1 // 限制预报天数在1-5天之间
	}

	// 构建API请求URL (使用5天预报API)
	url := fmt.Sprintf("%s/forecast?q=%s&appid=%s&units=metric&lang=zh_cn&cnt=%d",
		c.config.BaseURL, city, c.config.APIKey, days*8) // 每天8个时间点

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		c.logger.WithError(err).Error("Failed to create forecast request")
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.logger.WithError(err).Error("Failed to execute forecast request")
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		c.logger.WithField("status_code", resp.StatusCode).Error("API returned non-200 status")
		return nil, fmt.Errorf("API request failed with status: %d", resp.StatusCode)
	}

	var forecastResp struct {
		List []struct {
			Main struct {
				Temp     float64 `json:"temp"`
				Humidity int     `json:"humidity"`
			} `json:"main"`
			Weather []struct {
				Description string `json:"description"`
			} `json:"weather"`
			Wind struct {
				Speed float64 `json:"speed"`
			} `json:"wind"`
			DtTxt string `json:"dt_txt"`
		} `json:"list"`
		City struct {
			Name string `json:"name"`
		} `json:"city"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&forecastResp); err != nil {
		c.logger.WithError(err).Error("Failed to decode forecast response")
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if len(forecastResp.List) == 0 {
		return nil, fmt.Errorf("no forecast data available")
	}

	// 处理预报数据，每天取中午12点的数据作为代表
	var forecasts []WeatherData
	processedDays := make(map[string]bool)

	for _, item := range forecastResp.List {
		// 提取日期部分
		date := item.DtTxt[:10] // YYYY-MM-DD
		if processedDays[date] {
			continue // 跳过已处理的日期
		}

		// 只处理中午12点的数据，或者如果没有12点数据则取第一个
		if len(item.DtTxt) >= 13 && item.DtTxt[11:13] != "12" && len(forecasts) < days {
			// 如果不是12点且还没有这一天的数据，先跳过
			continue
		}

		description := "未知"
		if len(item.Weather) > 0 {
			description = item.Weather[0].Description
		}

		weatherData := WeatherData{
			Location:    forecastResp.City.Name,
			Temperature: item.Main.Temp,
			Description: description,
			Humidity:    item.Main.Humidity,
			WindSpeed:   item.Wind.Speed,
			Timestamp:   item.DtTxt,
		}

		forecasts = append(forecasts, weatherData)
		processedDays[date] = true

		if len(forecasts) >= days {
			break
		}
	}

	c.logger.WithFields(logrus.Fields{
		"city":          city,
		"forecast_days": len(forecasts),
	}).Info("Successfully retrieved weather forecast")

	return forecasts, nil
}

// HealthCheck 健康检查
func (w *WeatherClient) HealthCheck(ctx context.Context) error {
	// 测试获取北京天气
	_, err := w.GetWeather(ctx, "Beijing")
	if err != nil {
		return fmt.Errorf("weather service health check failed: %w", err)
	}
	return nil
}