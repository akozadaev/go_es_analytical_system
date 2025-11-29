package models

import "time"

// Location представляет локацию в Elasticsearch
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

// GeoPoint представляет географические координаты
type GeoPoint struct {
	Lat float64 `json:"lat"`
	Lon float64 `json:"lon"`
}

// Demographics представляет демографические данные
type Demographics struct {
	AgeGroup          string   `json:"age_group"`
	AverageIncome     float64  `json:"average_income"`
	Interests         []string `json:"interests"`
	PopulationDensity float64  `json:"population_density"`
}

// BusinessType представляет тип бизнеса в PostgreSQL
type BusinessType struct {
	ID          int       `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Region представляет регион в PostgreSQL
type Region struct {
	ID             int       `json:"id"`
	Name           string    `json:"name"`
	ParentRegionID *int      `json:"parent_region_id,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// RecommendRequest представляет запрос на рекомендацию
type RecommendRequest struct {
	Region       string `json:"region"`
	City         string `json:"city,omitempty"`
	BusinessType string `json:"business_type"`
	Limit        int    `json:"limit,omitempty"`
}

// RecommendResponse представляет ответ с рекомендациями
type RecommendResponse struct {
	Locations []Location `json:"locations"`
	Total     int        `json:"total"`
}
