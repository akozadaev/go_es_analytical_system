// Package handlers содержит HTTP обработчики для REST API рекомендательной системы.
package handlers

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/akozadaev/go_es_analytical_system/internal/models"
	"github.com/akozadaev/go_es_analytical_system/internal/storage"
	"github.com/gorilla/mux"
)

// Handlers содержит зависимости для обработки HTTP запросов.
// Использует Elasticsearch для поиска локаций и PostgreSQL для справочников.
type Handlers struct {
	esStorage *storage.ElasticsearchStorage // Хранилище для Elasticsearch/OpenSearch
	pgStorage *storage.PostgresStorage      // Хранилище для PostgreSQL
}

// NewHandlers создает новый экземпляр Handlers с заданными хранилищами.
func NewHandlers(esStorage *storage.ElasticsearchStorage, pgStorage *storage.PostgresStorage) *Handlers {
	return &Handlers{
		esStorage: esStorage,
		pgStorage: pgStorage,
	}
}

// RecommendLocations обрабатывает POST запрос на получение рекомендаций локаций.
// Принимает RecommendRequest в теле запроса и возвращает отсортированный список локаций.
// Эндпоинт: POST /locations/recommend
//
// @Summary      Получить рекомендации локаций
// @Description  Возвращает список рекомендованных локаций для указанного типа бизнеса в регионе. Локации ранжируются по релевантности с учетом traffic_score, competition_density и демографии.
// @Tags         locations
// @Accept       json
// @Produce      json
// @Param        request  body      models.RecommendRequest  true  "Запрос на рекомендации"
// @Success      200      {object}  models.RecommendResponse
// @Failure      400      {object}  map[string]string  "Неверный запрос"
// @Failure      500      {object}  map[string]string  "Внутренняя ошибка сервера"
// @Router       /locations/recommend [post]
func (h *Handlers) RecommendLocations(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req models.RecommendRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Region == "" || req.BusinessType == "" {
		http.Error(w, "Region and business_type are required", http.StatusBadRequest)
		return
	}

	if req.Limit == 0 {
		req.Limit = 20
	}

	locations, err := h.esStorage.RecommendLocations(r.Context(), &req)
	if err != nil {
		log.Printf("Error recommending locations: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Преобразуем указатели в значения для JSON
	locationValues := make([]models.Location, len(locations))
	for i, loc := range locations {
		locationValues[i] = *loc
	}

	response := models.RecommendResponse{
		Locations: locationValues,
		Total:     len(locationValues),
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Error encoding response: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
}

// GetLocation обрабатывает GET запрос на получение детальной информации о локации по ID.
// Эндпоинт: GET /locations/{id}
//
// @Summary      Получить детали локации
// @Description  Возвращает полную информацию о локации по её идентификатору
// @Tags         locations
// @Accept       json
// @Produce      json
// @Param        id   path      string  true  "Идентификатор локации"
// @Success      200  {object}  models.Location
// @Failure      404  {object}  map[string]string  "Локация не найдена"
// @Failure      500  {object}  map[string]string  "Внутренняя ошибка сервера"
// @Router       /locations/{id} [get]
func (h *Handlers) GetLocation(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	vars := mux.Vars(r)
	id := vars["id"]

	if id == "" {
		http.Error(w, "Location ID is required", http.StatusBadRequest)
		return
	}

	location, err := h.esStorage.GetLocation(r.Context(), id)
	if err != nil {
		if err.Error() == "location not found" {
			http.Error(w, "Location not found", http.StatusNotFound)
			return
		}
		log.Printf("Error getting location: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(location); err != nil {
		log.Printf("Error encoding response: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
}

// GetBusinessTypes обрабатывает GET запрос на получение списка всех типов бизнеса.
// Возвращает данные из справочника PostgreSQL.
// Эндпоинт: GET /business-types
//
// @Summary      Получить список типов бизнеса
// @Description  Возвращает все доступные типы бизнеса из справочника
// @Tags         business-types
// @Accept       json
// @Produce      json
// @Success      200  {array}   models.BusinessType
// @Failure      500  {object}  map[string]string  "Внутренняя ошибка сервера"
// @Router       /business-types [get]
func (h *Handlers) GetBusinessTypes(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	businessTypes, err := h.pgStorage.GetBusinessTypes(r.Context())
	if err != nil {
		log.Printf("Error getting business types: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Преобразуем указатели в значения для JSON
	btValues := make([]models.BusinessType, len(businessTypes))
	for i, bt := range businessTypes {
		btValues[i] = *bt
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(btValues); err != nil {
		log.Printf("Error encoding response: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
}

// GetRegions обрабатывает GET запрос на получение списка всех регионов.
// Возвращает данные из справочника PostgreSQL с поддержкой иерархии.
// Эндпоинт: GET /regions
//
// @Summary      Получить список регионов
// @Description  Возвращает все доступные регионы из справочника с поддержкой иерархии
// @Tags         regions
// @Accept       json
// @Produce      json
// @Success      200  {array}   models.Region
// @Failure      500  {object}  map[string]string  "Внутренняя ошибка сервера"
// @Router       /regions [get]
func (h *Handlers) GetRegions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	regions, err := h.pgStorage.GetRegions(r.Context())
	if err != nil {
		log.Printf("Error getting regions: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Преобразуем указатели в значения для JSON
	regionValues := make([]models.Region, len(regions))
	for i, r := range regions {
		regionValues[i] = *r
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(regionValues); err != nil {
		log.Printf("Error encoding response: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
}

// HealthCheck обрабатывает GET запрос на проверку работоспособности сервиса.
// Используется для мониторинга и проверки доступности API.
// Эндпоинт: GET /health
//
// @Summary      Проверка работоспособности сервиса
// @Description  Возвращает статус сервиса. Используется для мониторинга и проверки доступности.
// @Tags         health
// @Accept       json
// @Produce      json
// @Success      200  {object}  map[string]string
// @Router       /health [get]
func (h *Handlers) HealthCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "ok",
	})
}
