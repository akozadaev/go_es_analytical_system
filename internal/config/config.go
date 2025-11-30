// Package config предоставляет загрузку конфигурации приложения из переменных окружения.
package config

import (
	"os"
)

// Config содержит все параметры конфигурации приложения.
// Значения загружаются из переменных окружения с fallback на значения по умолчанию.
type Config struct {
	ElasticsearchURL string // URL для подключения к Elasticsearch/OpenSearch
	PostgresHost     string // Хост PostgreSQL
	PostgresPort     string // Порт PostgreSQL
	PostgresUser     string // Пользователь PostgreSQL
	PostgresPassword string // Пароль PostgreSQL
	PostgresDB       string // Имя базы данных PostgreSQL
	AppPort          string // Порт для HTTP сервера
}

// Load загружает конфигурацию из переменных окружения.
// Если переменная не установлена, используется значение по умолчанию.
func Load() *Config {
	return &Config{
		ElasticsearchURL: getEnv("ELASTICSEARCH_URL", "http://localhost:9200"),
		PostgresHost:     getEnv("POSTGRES_HOST", "localhost"),
		PostgresPort:     getEnv("POSTGRES_PORT", "5432"),
		PostgresUser:     getEnv("POSTGRES_USER", "analytical_user"),
		PostgresPassword: getEnv("POSTGRES_PASSWORD", "analytical_pass"),
		PostgresDB:       getEnv("POSTGRES_DB", "analytical_db"),
		AppPort:          getEnv("APP_PORT", "8080"),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
