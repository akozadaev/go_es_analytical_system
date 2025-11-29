package config

import (
	"os"
)

type Config struct {
	ElasticsearchURL string
	PostgresHost     string
	PostgresPort     string
	PostgresUser     string
	PostgresPassword string
	PostgresDB       string
	AppPort          string
}

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
