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
	"github.com/akozadaev/go_es_analytical_system/internal/auth"
	"github.com/akozadaev/go_es_analytical_system/internal/config"
	"github.com/akozadaev/go_es_analytical_system/internal/handlers"
	"github.com/akozadaev/go_es_analytical_system/internal/storage"
	"github.com/elastic/go-elasticsearch/v8"
	"github.com/gorilla/mux"
	httpSwagger "github.com/swaggo/http-swagger"
	"ollamaclient"
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

	// Пытаемся найти файл маппинга в разных местах (см. ES_INDEX_MAPPING в config).
	rel := cfg.ElasticsearchMappingPath
	if rel == "" {
		rel = "migrations/elasticsearch_mapping.json"
	}
	exeDir := filepath.Dir(os.Args[0])
	mappingPaths := []string{
		rel,
		filepath.Join("..", rel),
		filepath.Join(exeDir, rel),
		filepath.Join(exeDir, "..", rel),
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

	ollamaClient := ollamaclient.NewClient(cfg.OllamaClientConfig())

	// Инициализация handlers
	h := handlers.NewHandlers(esStorage, pgStorage, cfg, ollamaClient)

	if cfg.OAuth2AuthEnabled() {
		log.Println("OAuth2 bearer auth enabled for API routes (go_oauth2_server-compatible)")
	}
	if cfg.OAuth2BrowserProxyEnabled() {
		log.Println("OAuth2 browser proxy enabled: POST /api/auth/login, /api/auth/register → go_oauth2_server")
	}

	frontDir := resolveStaticDir("front/public")
	mapDir := resolveStaticDir("js_API_Ya_map/public")
	log.Printf("Static front: %s", frontDir)
	log.Printf("Static map indexer: %s", mapDir)

	// Настройка роутера
	router := mux.NewRouter()
	router.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusOK)
				return
			}
			next.ServeHTTP(w, r)
		})
	})
	router.Use(auth.Middleware(cfg))

	router.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		http.Redirect(w, r, "/app/", http.StatusFound)
	}).Methods(http.MethodGet)
	router.HandleFunc("/app", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/app/", http.StatusFound)
	}).Methods(http.MethodGet)

	router.PathPrefix("/app/").Handler(http.StripPrefix("/app/", http.FileServer(http.Dir(frontDir))))
	router.HandleFunc("/map-indexer", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/map-indexer/", http.StatusFound)
	}).Methods(http.MethodGet)
	router.PathPrefix("/map-indexer/").Handler(http.StripPrefix("/map-indexer/", http.FileServer(http.Dir(mapDir))))

	router.HandleFunc("/health", h.HealthCheck).Methods("GET")
	router.HandleFunc("/locations/recommend", h.RecommendLocations).Methods("POST")
	router.HandleFunc("/locations/{id}", h.GetLocation).Methods("GET")
	router.HandleFunc("/business-types", h.GetBusinessTypes).Methods("GET")
	router.HandleFunc("/regions", h.GetRegions).Methods("GET")
	router.HandleFunc("/readiness", h.GetReadiness).Methods("GET")
	router.HandleFunc("/ollama/chat", h.OllamaChat).Methods("POST")
	router.HandleFunc("/ollama/autocomplete", h.OllamaAutocomplete).Methods("POST")

	router.HandleFunc("/api/auth/register", h.OAuthRegister).Methods("POST", "OPTIONS")
	router.HandleFunc("/api/auth/login", h.OAuthLogin).Methods("POST", "OPTIONS")
	router.HandleFunc("/api/me", h.GetMe).Methods("GET", "OPTIONS")

	// Swagger UI
	router.PathPrefix("/swagger/").Handler(httpSwagger.Handler(
		httpSwagger.URL("http://localhost:8080/swagger/doc.json"),
		httpSwagger.DeepLinking(true),
		httpSwagger.DocExpansion("none"),
		httpSwagger.DomID("swagger-ui"),
	))

	// Настройка сервера
	srv := &http.Server{
		Addr:         ":" + cfg.AppPort,
		Handler:      router,
		ReadTimeout:  120 * time.Second,
		WriteTimeout: 120 * time.Second,
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

// resolveStaticDir ищет каталог относительно cwd и каталога бинарника (как путь к маппингу ES).
func resolveStaticDir(rel string) string {
	cwd, err := os.Getwd()
	if err != nil {
		cwd = "."
	}
	exePath, err := os.Executable()
	exeDir := cwd
	if err == nil {
		exeDir = filepath.Dir(exePath)
	}
	candidates := []string{
		rel,
		filepath.Join(cwd, rel),
		filepath.Join(exeDir, rel),
		filepath.Join(exeDir, "..", rel),
	}
	for _, p := range candidates {
		if fi, e := os.Stat(p); e == nil && fi.IsDir() {
			abs, _ := filepath.Abs(p)
			if abs != "" {
				return abs
			}
			return p
		}
	}
	abs, _ := filepath.Abs(filepath.Join(cwd, rel))
	if abs != "" {
		return abs
	}
	return filepath.Join(cwd, rel)
}
