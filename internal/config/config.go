package config

import "os"

// Config 汇总运行时所需的基础配置。
type Config struct {
	Environment string
	HTTP        HTTPConfig
}

// HTTPConfig 控制 HTTP 服务监听地址。
type HTTPConfig struct {
	Host string
	Port string
}

// Load 从环境变量中加载配置，并在缺省时使用安全默认值。
func Load() Config {
	return Config{
		Environment: envOrDefault("APP_ENV", "development"),
		HTTP: HTTPConfig{
			Host: envOrDefault("HTTP_HOST", "0.0.0.0"),
			Port: envOrDefault("HTTP_PORT", "8080"),
		},
	}
}

// Address 返回 HTTP 服务的监听地址。
func (h HTTPConfig) Address() string {
	return h.Host + ":" + h.Port
}

func envOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
