package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"os"
	"time"

	"github.com/akozadaev/go_es_analytical_system/internal/config"
	"github.com/akozadaev/go_es_analytical_system/internal/models"
	"github.com/akozadaev/go_es_analytical_system/internal/storage"
	"github.com/elastic/go-elasticsearch/v8"
)

func main() {
	cfg := config.Load()

	// Инициализация Elasticsearch клиента
	esCfg := elasticsearch.Config{
		Addresses: []string{cfg.ElasticsearchURL},
	}

	esClient, err := elasticsearch.NewClient(esCfg)
	if err != nil {
		log.Fatalf("Error creating Elasticsearch client: %v", err)
	}

	esStorage := storage.NewElasticsearchStorageWithURL(esClient, "locations", cfg.ElasticsearchURL)

	// Генерация тестовых данных
	locations := generateSampleLocations(100)

	log.Printf("Indexing %d locations...", len(locations))

	// Индексация данных
	if err := esStorage.BulkIndexLocations(context.Background(), locations); err != nil {
		log.Fatalf("Error indexing locations: %v", err)
	}

	log.Println("Indexing completed successfully!")
}

// generateSampleLocations генерирует тестовые данные локаций
func generateSampleLocations(count int) []*models.Location {
	cities := []string{"Москва", "Санкт-Петербург", "Новосибирск", "Екатеринбург", "Казань", "Тамбов"}
	regions := []string{"Москва", "Санкт-Петербург", "Новосибирская область", "Свердловская область", "Республика Татарстан", "Тамбовский муниципальный округ"}
	businessTypes := []string{"cafe", "repair_shop", "tailoring", "beauty_salon", "barbershop", "laundry", "restaurant", "gym", "pharmacy", "grocery_store"}
	ageGroups := []string{"18-25", "26-35", "36-45", "46-55", "55+"}
	interests := []string{"technology", "sports", "food", "fashion", "health", "entertainment"}

	locations := make([]*models.Location, 0, count)

	for i := 0; i < count; i++ {
		city := cities[rand.Intn(len(cities))]
		region := regions[rand.Intn(len(regions))]

		// Генерируем случайные координаты для России
		lat := 55.0 + rand.Float64()*10.0 // Примерно 55-65 градусов северной широты
		lon := 30.0 + rand.Float64()*50.0 // Примерно 30-80 градусов восточной долготы

		// Выбираем 2-4 подходящих типа бизнеса
		numTypes := 2 + rand.Intn(3)
		suitableTypes := make([]string, numTypes)
		used := make(map[string]bool)
		for j := 0; j < numTypes; j++ {
			bt := businessTypes[rand.Intn(len(businessTypes))]
			for used[bt] {
				bt = businessTypes[rand.Intn(len(businessTypes))]
			}
			used[bt] = true
			suitableTypes[j] = bt
		}

		// Генерируем случайные интересы
		numInterests := 2 + rand.Intn(3)
		locationInterests := make([]string, numInterests)
		usedInterests := make(map[string]bool)
		for j := 0; j < numInterests; j++ {
			interest := interests[rand.Intn(len(interests))]
			for usedInterests[interest] {
				interest = interests[rand.Intn(len(interests))]
			}
			usedInterests[interest] = true
			locationInterests[j] = interest
		}

		// Генерируем embedding (128 измерений)
		embedding := make([]float64, 128)
		for j := range embedding {
			embedding[j] = rand.Float64()*2 - 1 // Значения от -1 до 1
		}

		location := &models.Location{
			ID:      fmt.Sprintf("loc_%d", i+1),
			Name:    fmt.Sprintf("Локация %d", i+1),
			Address: fmt.Sprintf("ул. Примерная, д. %d, %s", rand.Intn(100)+1, city),
			Coordinates: models.GeoPoint{
				Lat: lat,
				Lon: lon,
			},
			Region:                region,
			City:                  city,
			Description:           fmt.Sprintf("Описание локации %d в городе %s", i+1, city),
			BusinessTypesSuitable: suitableTypes,
			TrafficScore:          rand.Float64() * 10, // 0-10
			CompetitionDensity:    rand.Float64() * 10, // 0-10
			Demographics: models.Demographics{
				AgeGroup:          ageGroups[rand.Intn(len(ageGroups))],
				AverageIncome:     float64(rand.Intn(100000) + 20000), // 20k-120k
				Interests:         locationInterests,
				PopulationDensity: rand.Float64() * 10000, // 0-10000 чел/км²
			},
			Embedding: embedding,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		locations = append(locations, location)
	}

	return locations
}

// loadLocationsFromFile загружает локации из JSON файла
func loadLocationsFromFile(filename string) ([]*models.Location, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var locations []*models.Location
	if err := json.Unmarshal(data, &locations); err != nil {
		return nil, err
	}

	return locations, nil
}
