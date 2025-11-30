// Package models содержит модели данных для рекомендательной системы локаций.
package models

import "time"

// Location представляет локацию в Elasticsearch.
// Содержит информацию о географическом положении, подходящих типах бизнеса,
// оценках трафика и конкуренции, а также демографических данных.
type Location struct {
	ID                    string       `json:"id"`
	Name                  string       `json:"name"`
	Address               string       `json:"address"`
	Coordinates           GeoPoint     `json:"coordinates"`
	Region                string       `json:"region"`
	City                  string       `json:"city"`
	Description           string       `json:"description"`
	BusinessTypesSuitable []string     `json:"business_types_suitable"`
	TrafficScore          float64      `json:"traffic_score"`
	CompetitionDensity    float64      `json:"competition_density"`
	Demographics          Demographics `json:"demographics"`
	Embedding             []float64    `json:"embedding,omitempty"`
	CreatedAt             time.Time    `json:"created_at"`
	UpdatedAt             time.Time    `json:"updated_at"`
	Score                 float64      `json:"score,omitempty"` // Для ранжирования
}

// GeoPoint представляет географические координаты точки на карте.
// Используется для геопространственных запросов в Elasticsearch.
type GeoPoint struct {
	Lat float64 `json:"lat"` // Широта (latitude)
	Lon float64 `json:"lon"` // Долгота (longitude)
}

// Demographics представляет демографические данные района локации.
// Используется для анализа целевой аудитории и соответствия типу бизнеса.
type Demographics struct {
	AgeGroup          string   `json:"age_group"`
	AverageIncome     float64  `json:"average_income"`
	Interests         []string `json:"interests"`
	PopulationDensity float64  `json:"population_density"`
}

// BusinessType представляет тип бизнеса из справочника PostgreSQL.
// Используется для фильтрации и рекомендаций локаций.
type BusinessType struct {
	ID          int       `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Region представляет регион из справочника PostgreSQL.
// Поддерживает иерархическую структуру через ParentRegionID.
type Region struct {
	ID             int       `json:"id"`
	Name           string    `json:"name"`
	ParentRegionID *int      `json:"parent_region_id,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// RecommendRequest представляет запрос на получение рекомендаций локаций.
// Все поля, кроме City, являются обязательными.
type RecommendRequest struct {
	Region       string `json:"region"`        // Регион для поиска (обязательно)
	City         string `json:"city,omitempty"` // Город для фильтрации (опционально)
	BusinessType string `json:"business_type"`  // Тип бизнеса (обязательно)
	Limit        int    `json:"limit,omitempty"` // Максимальное количество результатов (по умолчанию 20)
}

// RecommendResponse представляет ответ с рекомендованными локациями.
// Содержит отсортированный список локаций и общее количество найденных результатов.
type RecommendResponse struct {
	Locations []Location `json:"locations"`
	Total     int        `json:"total"`
}
