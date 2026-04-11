package main

import (
	"bytes"
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
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
		Addresses:         []string{cfg.ElasticsearchURL},
		DisableMetaHeader: true, // Для поддержки OpenSearch
	}

	esClient, err := elasticsearch.NewClient(esCfg)
	if err != nil {
		log.Fatalf("Error creating Elasticsearch client: %v", err)
	}

	esStorage := storage.NewElasticsearchStorageWithURL(esClient, "locations", cfg.ElasticsearchURL)

	// Генерация тестовых данных
	//locations := generateSampleLocations(100)
	locations, _ := loadLocationsFromFile("100")

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
		fmt.Print("===========\n")
		fmt.Print(location)
	}

	return locations
}

type xmlLocations struct {
	XMLName   xml.Name      `xml:"locations"`
	Locations []xmlLocation `xml:"location"`
}

type xmlLocation struct {
	ID                    string          `xml:"id"`
	Name                  string          `xml:"name"`
	Address               string          `xml:"address"`
	Coordinates           xmlCoordinates  `xml:"coordinates"`
	Region                string          `xml:"region"`
	City                  string          `xml:"city"`
	Description           string          `xml:"description"`
	BusinessTypesSuitable []string        `xml:"business_types_suitable>business_type"`
	TrafficScore          float64         `xml:"traffic_score"`
	CompetitionDensity    float64         `xml:"competition_density"`
	Demographics          xmlDemographics `xml:"demographics"`
	Embedding             []float64       `xml:"embedding>value"`
	CreatedAt             time.Time       `xml:"created_at"`
	UpdatedAt             time.Time       `xml:"updated_at"`
}

type xmlCoordinates struct {
	Lat float64 `xml:"lat"`
	Lon float64 `xml:"lon"`
}

type xmlDemographics struct {
	AgeGroup          string   `xml:"age_group"`
	AverageIncome     float64  `xml:"average_income"`
	Interests         []string `xml:"interests>interest"`
	PopulationDensity float64  `xml:"population_density"`
}

// loadLocationsFromFile загружает локации из JSON файла
func loadLocationsFromFile(filename string) ([]*models.Location, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	isXMLByExt := strings.EqualFold(filepath.Ext(filename), ".xml")
	isXMLByContent := bytes.HasPrefix(bytes.TrimSpace(data), []byte("<"))
	if isXMLByExt || isXMLByContent {
		return unmarshalLocationsXML(data)
	}

	var locations []*models.Location
	if err := json.Unmarshal(data, &locations); err != nil {
		return nil, err
	}

	return locations, nil
}

func unmarshalLocationsXML(data []byte) ([]*models.Location, error) {
	var root xmlLocations
	if err := xml.Unmarshal(data, &root); err != nil {
		return nil, fmt.Errorf("failed to unmarshal XML locations: %w", err)
	}

	locations := make([]*models.Location, 0, len(root.Locations))
	for _, loc := range root.Locations {
		modelLocation := &models.Location{
			ID:      loc.ID,
			Name:    loc.Name,
			Address: loc.Address,
			Coordinates: models.GeoPoint{
				Lat: loc.Coordinates.Lat,
				Lon: loc.Coordinates.Lon,
			},
			Region:                loc.Region,
			City:                  loc.City,
			Description:           loc.Description,
			BusinessTypesSuitable: loc.BusinessTypesSuitable,
			TrafficScore:          loc.TrafficScore,
			CompetitionDensity:    loc.CompetitionDensity,
			Demographics: models.Demographics{
				AgeGroup:          loc.Demographics.AgeGroup,
				AverageIncome:     loc.Demographics.AverageIncome,
				Interests:         loc.Demographics.Interests,
				PopulationDensity: loc.Demographics.PopulationDensity,
			},
			Embedding: loc.Embedding,
			CreatedAt: loc.CreatedAt,
			UpdatedAt: loc.UpdatedAt,
		}

		locations = append(locations, modelLocation)
	}

	return locations, nil
}
