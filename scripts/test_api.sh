#!/bin/bash

# Примеры использования API рекомендательной системы

BASE_URL="http://localhost:8080"

echo "=== Проверка здоровья сервиса ==="
curl -s "$BASE_URL/health" | jq .
echo -e "\n"

echo "=== Получение списка типов бизнеса ==="
curl -s "$BASE_URL/business-types" | jq .
echo -e "\n"

echo "=== Получение списка регионов ==="
curl -s "$BASE_URL/regions" | jq .
echo -e "\n"

echo "=== Получение рекомендаций локаций для кафе в Москве ==="
curl -s -X POST "$BASE_URL/locations/recommend" \
  -H "Content-Type: application/json" \
  -d '{
    "region": "Москва",
    "city": "Москва",
    "business_type": "cafe",
    "limit": 5
  }' | jq .
echo -e "\n"

echo "=== Получение рекомендаций для барбершопа в Санкт-Петербурге ==="
curl -s -X POST "$BASE_URL/locations/recommend" \
  -H "Content-Type: application/json" \
  -d '{
    "region": "Санкт-Петербург",
    "city": "Санкт-Петербург",
    "business_type": "barbershop",
    "limit": 3
  }' | jq .
echo -e "\n"

echo "=== Получение деталей локации (пример) ==="
LOCATION_ID="loc_1"
curl -s "$BASE_URL/locations/$LOCATION_ID" | jq .
echo -e "\n"

