// Package config предоставляет загрузку конфигурации приложения из переменных окружения.
package config

import (
	"os"
	"strconv"
	"strings"

	"ollamaclient"
)

// Config содержит все параметры конфигурации приложения.
// Значения загружаются из переменных окружения с fallback на значения по умолчанию.
type Config struct {
	ElasticsearchURL string // URL для подключения к Elasticsearch/OpenSearch
	// ElasticsearchMappingPath — JSON тела создания индекса (settings + mappings).
	// Elasticsearch: migrations/elasticsearch_mapping.json (dense_vector).
	// OpenSearch kNN: migrations/opensearch_mapping.json (knn_vector). Переменная окружения: ES_INDEX_MAPPING.
	ElasticsearchMappingPath string
	PostgresHost             string // Хост PostgreSQL
	PostgresPort             string // Порт PostgreSQL
	PostgresUser             string // Пользователь PostgreSQL
	PostgresPassword         string // Пароль PostgreSQL
	PostgresDB               string // Имя базы данных PostgreSQL
	AppPort                  string // Порт для HTTP сервера

	// Readiness check parameters
	ReadinessDBTimeoutSec  int    // Таймаут для проверки БД в секундах
	ReadinessDiskPath      string // Путь для проверки диска
	ReadinessDiskMinFreeMB int    // Минимальное свободное место на диске в MB
	ReadinessRAMMinFreeMB  int    // Минимальная свободная оперативная память в MB
	BuildVersion           string // Версия сборки
	GitCommit              string // Хэш коммита Git

	// Ollama (OpenAI-совместимый API через go_ollama_client)
	OllamaBaseURL           string // Базовый URL, например http://localhost:11434/v1
	OllamaChatModel         string // Модель для чата
	OllamaAutocompleteModel string // Модель для автодополнения кода
	OllamaEmbedModel        string // Модель для эмбеддингов (на будущее)

	// OAuth2 (go_oauth2_server): защита API Bearer access token.
	// Либо OAUTH2_INTROSPECT_URL (POST JSON как у сервера), либо OAUTH2_JWT_SECRET (локальная проверка HS256).
	OAuth2JWTSecret     string
	OAuth2IntrospectURL string

	// OAuth2 browser proxy: POST /users и password grant /token на стороне go_oauth2_server (секрет клиента не в браузере).
	OAuth2ServerURL    string
	OAuth2ClientID     string
	OAuth2ClientSecret string
	OAuth2Scope        string
}

// Load загружает конфигурацию из переменных окружения.
// Если переменная не установлена, используется значение по умолчанию.
func Load() *Config {
	return &Config{
		ElasticsearchURL:         getEnv("ELASTICSEARCH_URL", "http://localhost:9200"),
		ElasticsearchMappingPath: getEnv("ES_INDEX_MAPPING", "migrations/elasticsearch_mapping.json"),
		PostgresHost:             getEnv("POSTGRES_HOST", "localhost"),
		PostgresPort:             getEnv("POSTGRES_PORT", "5432"),
		PostgresUser:             getEnv("POSTGRES_USER", "analytical_user"),
		PostgresPassword:         getEnv("POSTGRES_PASSWORD", "analytical_pass"),
		PostgresDB:               getEnv("POSTGRES_DB", "analytical_db"),
		AppPort:                  getEnv("APP_PORT", "8080"),

		ReadinessDBTimeoutSec:  getEnvInt("READINESS_DB_TIMEOUT_SEC", 5),
		ReadinessDiskPath:      getEnv("READINESS_DISK_PATH", "."),
		ReadinessDiskMinFreeMB: getEnvInt("READINESS_DISK_MIN_FREE_MB", 100),
		ReadinessRAMMinFreeMB:  getEnvInt("READINESS_RAM_MIN_FREE_MB", 50),
		BuildVersion:           getEnv("BUILD_VERSION", "dev"),
		GitCommit:              getEnv("GIT_COMMIT", ""),

		OllamaBaseURL:           getEnv("OLLAMA_BASE_URL", ""),
		OllamaChatModel:         getEnv("OLLAMA_CHAT_MODEL", ""),
		OllamaAutocompleteModel: getEnv("OLLAMA_AUTOCOMPLETE_MODEL", ""),
		OllamaEmbedModel:        getEnv("OLLAMA_EMBED_MODEL", ""),

		OAuth2JWTSecret:     getEnv("OAUTH2_JWT_SECRET", ""),
		OAuth2IntrospectURL: getEnv("OAUTH2_INTROSPECT_URL", ""),

		OAuth2ServerURL:    getEnv("OAUTH2_SERVER_URL", ""),
		OAuth2ClientID:     getEnv("OAUTH2_CLIENT_ID", ""),
		OAuth2ClientSecret: getEnv("OAUTH2_CLIENT_SECRET", ""),
		OAuth2Scope:        getEnv("OAUTH2_SCOPE", "read"),
	}
}

// OAuth2AuthEnabled — true, если задан секрет для локальной проверки JWT или URL интроспекции.
func (c *Config) OAuth2AuthEnabled() bool {
	if c == nil {
		return false
	}
	return strings.TrimSpace(c.OAuth2JWTSecret) != "" || strings.TrimSpace(c.OAuth2IntrospectURL) != ""
}

// OAuth2BrowserProxyEnabled — прокси регистрации и password grant для фронта ./front.
func (c *Config) OAuth2BrowserProxyEnabled() bool {
	if c == nil {
		return false
	}
	return strings.TrimSpace(c.OAuth2ServerURL) != "" &&
		strings.TrimSpace(c.OAuth2ClientID) != "" &&
		strings.TrimSpace(c.OAuth2ClientSecret) != ""
}

// OllamaClientConfig собирает конфигурацию для ollamaclient: значения из .env через ollamaclient.DefaultConfig().
func (c *Config) OllamaClientConfig() ollamaclient.OllamaConfig {
	dc := ollamaclient.DefaultConfig()
	if c.OllamaBaseURL != "" {
		dc.BaseURL = c.OllamaBaseURL
	}
	if c.OllamaChatModel != "" {
		dc.ChatModel = c.OllamaChatModel
	}
	if c.OllamaAutocompleteModel != "" {
		dc.AutocompleteModel = c.OllamaAutocompleteModel
	}
	if c.OllamaEmbedModel != "" {
		dc.EmbedModel = c.OllamaEmbedModel
	}
	return dc
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return defaultValue
}
