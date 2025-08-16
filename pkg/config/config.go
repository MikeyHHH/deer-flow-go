package config

import (
	"os"
	"strconv"

	"github.com/joho/godotenv"
	"github.com/sirupsen/logrus"
)

// Config 应用配置结构
type Config struct {
	// 服务器配置
	Port string `yaml:"port"`

	// Azure OpenAI 配置
	AzureOpenAI AzureOpenAIConfig `yaml:"azure_openai"`

	// Tavily 搜索配置
	Tavily TavilyConfig `yaml:"tavily"`

	// MCP 配置
	MCP MCPConfig `yaml:"mcp"`

	// 天气服务配置
	Weather WeatherConfig `yaml:"weather"`

	// 队列管理配置
	Queue QueueConfig `yaml:"queue"`

	// 日志配置
	LogLevel string `yaml:"log_level"`
}

// AzureOpenAIConfig Azure OpenAI 配置
type AzureOpenAIConfig struct {
	Endpoint    string  `yaml:"endpoint"`
	APIKey      string  `yaml:"api_key"`
	Deployment  string  `yaml:"deployment"`
	APIVersion  string  `yaml:"api_version"`
	Temperature float32 `yaml:"temperature"`
}

// TavilyConfig Tavily 搜索配置
type TavilyConfig struct {
	APIKey      string `yaml:"api_key"`
	MaxResults  int    `yaml:"max_results"`
	SearchDepth string `yaml:"search_depth"`
}

// MCPConfig MCP 配置
type MCPConfig struct {
	Enabled bool `yaml:"enabled"`
	Timeout int  `yaml:"timeout"`
}

// WeatherConfig 天气服务配置
type WeatherConfig struct {
	APIKey  string `yaml:"api_key"`
	BaseURL string `yaml:"base_url"`
	Timeout int    `yaml:"timeout"`
}

// QueueConfig 队列管理配置
type QueueConfig struct {
	MaxWorkers     int `yaml:"max_workers"`     // 最大工作协程数
	QueueSize      int `yaml:"queue_size"`      // 队列大小
	RequestTimeout int `yaml:"request_timeout"` // 请求超时时间(秒)
	QueueTimeout   int `yaml:"queue_timeout"`   // 队列等待超时时间(秒)
}

// LoadConfig 加载配置
func LoadConfig() (*Config, error) {
	// 加载 .env 文件
	if err := godotenv.Load(); err != nil {
		logrus.Warn("No .env file found")
	}

	config := &Config{
		Port:     getEnv("PORT", "8080"),
		LogLevel: getEnv("LOG_LEVEL", "info"),

		AzureOpenAI: AzureOpenAIConfig{
			Endpoint:    getEnv("AZURE_OPENAI_ENDPOINT", "https://dajia-it-openai-japaneast.openai.azure.com"),
			APIKey:      getEnv("AZURE_OPENAI_API_KEY", "**********************"),
			Deployment:  getEnv("AZURE_OPENAI_DEPLOYMENT", "dajia-it-openai-JapanEast-gpt-4"),
			APIVersion:  getEnv("AZURE_OPENAI_API_VERSION", "2023-08-01-preview"),
			Temperature: getEnvFloat32("AZURE_OPENAI_TEMPERATURE", 0.0),
		},

		Tavily: TavilyConfig{
			APIKey:      getEnv("TAVILY_API_KEY", "***************"),
			MaxResults:  getEnvInt("TAVILY_MAX_RESULTS", 5),
			SearchDepth: getEnv("TAVILY_SEARCH_DEPTH", "advanced"),
		},

		MCP: MCPConfig{
			Enabled: getEnvBool("MCP_ENABLED", true),
			Timeout: getEnvInt("MCP_TIMEOUT", 60),
		},

		Weather: WeatherConfig{
			APIKey:  getEnv("WEATHER_API_KEY", "***********"),
			BaseURL: getEnv("WEATHER_BASE_URL", "https://api.openweathermap.org/data/2.5"),
			Timeout: getEnvInt("WEATHER_TIMEOUT", 10),
		},

		Queue: QueueConfig{
			MaxWorkers:     getEnvInt("QUEUE_MAX_WORKERS", 3),
			QueueSize:      getEnvInt("QUEUE_SIZE", 100),
			RequestTimeout: getEnvInt("QUEUE_REQUEST_TIMEOUT", 30),
			QueueTimeout:   getEnvInt("QUEUE_TIMEOUT", 10),
		},
	}

	return config, nil
}

// getEnv 获取环境变量，如果不存在则返回默认值
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvInt 获取整数类型环境变量
func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

// getEnvFloat32 获取浮点数类型环境变量
func getEnvFloat32(key string, defaultValue float32) float32 {
	if value := os.Getenv(key); value != "" {
		if floatValue, err := strconv.ParseFloat(value, 32); err == nil {
			return float32(floatValue)
		}
	}
	return defaultValue
}

// getEnvBool 获取布尔类型环境变量
func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}
