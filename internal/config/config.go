package config

import (
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

// Config содержит всю конфигурацию приложения
type Config struct {
	Port          string
	DatabaseDSN   string
	RedisAddr     string
	RedisPassword string
	BaseURL       string
	EnableRedis   bool
	RedisTTL      time.Duration
}

// Load загружает конфигурацию из переменных окружения и .env файла
// Приоритет: переменные окружения > .env файл > значения по умолчанию
func Load() (*Config, error) {
	// Загружаем .env файл (если существует, игнорируем ошибку)
	_ = godotenv.Load()

	cfg := &Config{
		Port:          getEnv("PORT", "8080"),
		DatabaseDSN:   getEnv("DATABASE_DSN", "postgres://user:password@localhost/shortener?sslmode=disable"),
		RedisAddr:     getEnv("REDIS_ADDR", "localhost:6379"),
		RedisPassword: getEnv("REDIS_PASSWORD", ""),
		BaseURL:       getEnv("BASE_URL", "http://localhost:8080"),
		EnableRedis:   getEnvBool("ENABLE_REDIS", false),
		RedisTTL:      getEnvDuration("REDIS_TTL", 30*time.Minute),
	}

	return cfg, nil
}

// getEnv получает переменную окружения или возвращает значение по умолчанию
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvBool получает булеву переменную окружения или возвращает значение по умолчанию
func getEnvBool(key string, defaultValue bool) bool {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	result, err := strconv.ParseBool(value)
	if err != nil {
		return defaultValue
	}
	return result
}

// getEnvDuration получает переменную окружения как duration или возвращает значение по умолчанию
func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	duration, err := time.ParseDuration(value)
	if err != nil {
		return defaultValue
	}
	return duration
}
