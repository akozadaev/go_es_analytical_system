// @title           Location Recommendation System API
// @version         1.0
// @description     REST API для рекомендательной системы локаций для бизнеса. Система предоставляет рекомендации локаций на основе анализа трафика, конкуренции и демографических данных.
// @termsOfService  http://swagger.io/terms/

// @contact.name   API Support
// @contact.email  akozadaev@inbox.ru
// @contact.url    https://github.com/akozadaev/go_es_analytical_system

// @license.name  MIT
// @license.url   https://opensource.org/licenses/MIT

// @host      localhost:8080
// @BasePath  /

// @schemes   http https
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	_ "github.com/akozadaev/go_es_analytical_system/docs" // swagger docs
	"github.com/akozadaev/go_es_analytical_system/internal/config"
	"github.com/akozadaev/go_es_analytical_system/internal/handlers"
	"github.com/akozadaev/go_es_analytical_system/internal/storage"
	"github.com/elastic/go-elasticsearch/v8"
	"github.com/gorilla/mux"
	httpSwagger "github.com/swaggo/http-swagger"
)

func main() {
	cfg := config.Load()

	// Инициализация Elasticsearch клиента
	// Используем кастомный транспорт для обхода проверки типа сервера
	esCfg := elasticsearch.Config{
		Addresses:         []string{cfg.ElasticsearchURL},
		DisableMetaHeader: true,
	}

	esClient, err := elasticsearch.NewClient(esCfg)
	if err != nil {
		log.Fatalf("Error creating Elasticsearch client: %v", err)
	}

	// Простая проверка доступности через прямой HTTP запрос
	// (клиент go-elasticsearch проверяет тип сервера, поэтому пропускаем стандартные методы)
	log.Println("Elasticsearch/OpenSearch client initialized")

	// Создание индекса с маппингом
	esStorage := storage.NewElasticsearchStorageWithURL(esClient, "locations", cfg.ElasticsearchURL)

	// Пытаемся найти файл маппинга в разных местах
	mappingPaths := []string{
		"migrations/elasticsearch_mapping.json",
		"../migrations/elasticsearch_mapping.json",
		filepath.Join(filepath.Dir(os.Args[0]), "../migrations/elasticsearch_mapping.json"),
	}

	var mappingData []byte
	for _, path := range mappingPaths {
		var readErr error
		mappingData, readErr = os.ReadFile(path)
		if readErr == nil {
			break
		}
	}

	if len(mappingData) > 0 {
		if err := esStorage.CreateIndex(context.Background(), string(mappingData)); err != nil {
			log.Printf("Warning: could not create index: %v", err)
		} else {
			log.Println("Elasticsearch index created/verified")
		}
	} else {
		log.Printf("Warning: could not read mapping file from any location")
	}

	// Инициализация PostgreSQL клиента
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		cfg.PostgresHost,
		cfg.PostgresPort,
		cfg.PostgresUser,
		cfg.PostgresPassword,
		cfg.PostgresDB,
	)

	pgStorage, err := storage.NewPostgresStorage(dsn)
	if err != nil {
		log.Fatalf("Error creating PostgreSQL client: %v", err)
	}
	defer pgStorage.Close()
	log.Println("Connected to PostgreSQL")

	// Инициализация handlers
	h := handlers.NewHandlers(esStorage, pgStorage)

	// Настройка роутера
	router := mux.NewRouter()
	router.HandleFunc("/health", h.HealthCheck).Methods("GET")
	router.HandleFunc("/locations/recommend", h.RecommendLocations).Methods("POST")
	router.HandleFunc("/locations/{id}", h.GetLocation).Methods("GET")
	router.HandleFunc("/business-types", h.GetBusinessTypes).Methods("GET")
	router.HandleFunc("/regions", h.GetRegions).Methods("GET")

	// Swagger UI
	router.PathPrefix("/swagger/").Handler(httpSwagger.Handler(
		httpSwagger.URL("http://localhost:8080/swagger/doc.json"),
		httpSwagger.DeepLinking(true),
		httpSwagger.DocExpansion("none"),
		httpSwagger.DomID("swagger-ui"),
	))

	// Настройка CORS
	router.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}
			next.ServeHTTP(w, r)
		})
	})

	// Настройка сервера
	srv := &http.Server{
		Addr:         ":" + cfg.AppPort,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Graceful shutdown
	go func() {
		log.Printf("Server starting on port %s", cfg.AppPort)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed to start: %v", err)
		}
	}()

	// Ожидание сигнала для graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited")
}
