# Рекомендательная Система Локаций для Бизнеса

Система для поиска и рекомендации локаций для различных типов бизнеса на основе анализа трафика, конкуренции и демографических данных.

## Технологический стек

- **Backend**: Go 1.25+
- **Поисковая система**: Elasticsearch 8.11
- **База данных**: PostgreSQL 16
- **Веб-интерфейс**: Kibana (для мониторинга ES)
- **Контейнеризация**: Docker & Docker Compose
- **LLM (опционально)**: [Ollama](https://ollama.com) на localhost для эндпоинтов `/ollama/*`

## Архитектура

Система состоит из следующих компонентов:

1. **Go REST API** - основной сервер приложения
2. **Elasticsearch** - хранилище и поиск по локациям
3. **PostgreSQL** - справочники (типы бизнеса, регионы)
4. **Kibana** - мониторинг и отладка Elasticsearch

## Структура проекта

```
.
├── docs/
│   ├── docs.go          # Go-пакет для встраивания OpenAPI-спецификации
│   ├── swagger.json     # OpenAPI 3.0 спецификация (JSON)
│   └── swagger.yaml     # OpenAPI 3.0 спецификация (YAML)
├── cmd/
│   ├── server/          # Основной сервер приложения
│   └── indexer/         # Утилита для индексации данных
├── internal/
│   ├── config/          # Конфигурация приложения
│   ├── handlers/        # HTTP handlers
│   ├── models/          # Модели данных
│   └── storage/         # Клиенты для ES и PostgreSQL
├── migrations/
│   ├── 001_init_schema.sql           # SQL миграции для PostgreSQL
│   ├── elasticsearch_mapping.json     # Схема данных для индекса Elasticsearch
│   └── opensearch_mapping.json        # Схема данных для индекса OpenSearch
├── docker-compose.yml
├── Dockerfile
└── README.md
```

## Файлы маппинга индексов

Проект поддерживает два варианта поисковых движков: Elasticsearch и OpenSearch. Для каждого из них предусмотрены отдельные файлы схемы данных (mapping):

### `migrations/elasticsearch_mapping.json`
Определяет структуру индекса для Elasticsearch 8.x:
- Содержит настройки индекса (1 шард, 0 реплик)
- Определяет типы полей для документов локаций
- Использует `dense_vector` для поля `embedding` (векторные представления)

### `migrations/opensearch_mapping.json`
Аналогичная схема для OpenSearch 2.x:
- Включает настройку `"knn": true` для поддержки векторного поиска
- Использует `knn_vector` вместо `dense_vector`
- Полностью совместим с Elasticsearch API

Оба файла описывают одинаковые поля документов:
- `id`, `name`, `address` - идентификация локации
- `coordinates` - географические координаты (lat/lon)
- `region`, `city` - административное деление
- `business_types_suitable` - подходящие типы бизнеса
- `traffic_score`, `competition_density` - аналитические метрики
- `demographics` - демографические данные
- `embedding` - векторное представление для семантического поиска

## Быстрый старт

### Предварительные требования

- Docker и Docker Compose
- Go 1.25+ (для локальной разработки)

### Запуск через Docker Compose

1. Клонируйте репозиторий:
```bash
git clone <repository-url>
cd go_es_analytical_system
```

2. Запустите все сервисы:
```bash
docker-compose up -d
```

**Примечание:** Если возникают проблемы с доступом к образам Elasticsearch (ошибка 403), используйте один из альтернативных вариантов:

**Вариант 1:** Использовать образы из Docker Hub (уже настроено в docker-compose.yml):
```bash
docker-compose up -d
```

**Вариант 2:** Использовать OpenSearch (совместим с Elasticsearch API):
```bash
docker-compose -f docker-compose.opensearch.yml up -d
```

**Вариант 3:** Использовать локальные образы (если они уже загружены):
```bash
# Сначала загрузите образы через VPN или прокси
docker pull elasticsearch:8.10.0
docker pull kibana:8.10.0
# Затем запустите
docker-compose up -d
```

Это запустит:
- Elasticsearch/OpenSearch на порту 9200
- Kibana/OpenSearch Dashboards на порту 5601
- PostgreSQL на порту 5432
- Go приложение на порту 8080

3. Дождитесь готовности сервисов (около 30-60 секунд)

4. Индексируйте тестовые данные:
```bash
docker-compose exec app ./indexer
```

Или если используете OpenSearch вариант:
```bash
docker-compose -f docker-compose.opensearch.yml exec app ./indexer
```

Или если запускаете локально:
```bash
go run cmd/indexer/main.go
```

### Локальная разработка

1. Убедитесь, что Elasticsearch и PostgreSQL запущены:
```bash
docker-compose up -d elasticsearch postgres
```

2. Установите зависимости:
```bash
go mod download
```

3. Запустите сервер:
```bash
go run cmd/server/main.go
```

4. Индексируйте данные:
```bash
go run cmd/indexer/main.go
```

## API Endpoints

### 1. Получить рекомендации локаций

**POST** `/locations/recommend`

Запрос:
```json
{
  "region": "Москва",
  "city": "Москва",
  "business_type": "cafe",
  "limit": 20
}
```

Ответ:
```json
{
  "locations": [
    {
      "id": "loc_1",
      "name": "Локация 1",
      "address": "ул. Примерная, д. 10, Москва",
      "coordinates": {
        "lat": 55.7558,
        "lon": 37.6173
      },
      "region": "Москва",
      "city": "Москва",
      "description": "Описание локации",
      "business_types_suitable": ["cafe", "restaurant"],
      "traffic_score": 8.5,
      "competition_density": 2.3,
      "demographics": {
        "age_group": "26-35",
        "average_income": 75000,
        "interests": ["food", "technology"],
        "population_density": 5000
      },
      "score": 0.95
    }
  ],
  "total": 1
}
```

### 2. Получить детали локации

**GET** `/locations/{id}`

Ответ:
```json
{
  "id": "loc_1",
  "name": "Локация 1",
  ...
}
```

### 3. Получить список типов бизнеса

**GET** `/business-types`

Ответ:
```json
[
  {
    "id": 1,
    "name": "cafe",
    "description": "Кафе"
  },
  ...
]
```

### 4. Получить список регионов

**GET** `/regions`

Ответ:
```json
[
  {
    "id": 1,
    "name": "Москва",
    "parent_region_id": 1
  },
  ...
]
```

### 5. Проверка здоровья сервиса

**GET** `/health`

Ответ:
```json
{
  "status": "ok"
}
```

## Алгоритм рекомендаций

Система использует комбинированный подход для ранжирования локаций:

1. **Фильтрация**:
   - По региону и городу
   - По типу бизнеса (должен быть в списке `business_types_suitable`)

2. **Ранжирование**:
   - **Traffic Score** (выше = лучше): Бустинг для локаций с score >= 7.0
   - **Competition Density** (ниже = лучше): Бустинг для локаций с density <= 3.0
   - **Демография**: Соответствие целевой аудитории (планируется расширение)

3. **Сортировка**:
   - По релевантности (score)
   - По traffic_score (по убыванию)
   - По competition_density (по возрастанию)

## Конфигурация

Переменные окружения:

- `ELASTICSEARCH_URL` - URL Elasticsearch (по умолчанию: http://localhost:9200)
- `ES_INDEX_MAPPING` - путь к JSON создания индекса `locations` (settings + mappings). По умолчанию `migrations/elasticsearch_mapping.json` (**Elasticsearch 8**: поле `embedding` как **`dense_vector`**, 128 измерений, `index: true`, `similarity: cosinus` — для kNN в ES). Для **OpenSearch** в `docker-compose.opensearch.yml` задано `migrations/opensearch_mapping.json` (**`knn_vector`**, `dimension: 128`). После смены маппинга удалите индекс `locations` и создайте его заново, затем переиндексируйте данные (`./indexer` / `make docker-index-opensearch`).
- `POSTGRES_HOST` - Хост PostgreSQL (по умолчанию: localhost)
- `POSTGRES_PORT` - Порт PostgreSQL (по умолчанию: 5432)
- `POSTGRES_USER` - Пользователь PostgreSQL (по умолчанию: analytical_user)
- `POSTGRES_PASSWORD` - Пароль PostgreSQL (по умолчанию: analytical_pass)
- `POSTGRES_DB` - Имя базы данных (по умолчанию: analytical_db)
- `APP_PORT` - Порт приложения (по умолчанию: 8080)
- `OLLAMA_BASE_URL` - Базовый URL OpenAI-совместимого API Ollama (по умолчанию в клиенте: `http://localhost:11434/v1`; в Docker Compose для сервиса `app` по умолчанию используется `http://host.docker.internal:11434/v1`, но его можно переопределить переменной окружения)
- `OLLAMA_CHAT_MODEL` - Модель для `POST /ollama/chat` (пусто — значение по умолчанию из библиотеки [go_ollama_client](https://github.com/akozadaev/go_ollama_client))
- `OLLAMA_AUTOCOMPLETE_MODEL` - Модель для `POST /ollama/autocomplete` (пусто — дефолт клиента)
- `OLLAMA_EMBED_MODEL` - Модель эмбеддингов (зарезервировано под будущее использование; пусто — дефолт клиента)

## Ollama

Эндпоинты **`POST /ollama/chat`** и **`POST /ollama/autocomplete`** проксируют запросы в локальную Ollama через зависимость [go_ollama_client](https://github.com/akozadaev/go_ollama_client). Нужны установленная Ollama и скачанные модели (см. README того репозитория).

### Доступ из Docker: `OLLAMA_HOST=0.0.0.0:11434`

По умолчанию Ollama часто слушает только **`127.0.0.1:11434`**. Контейнер `app` ходит на хост по **`host.docker.internal`** (на Linux это обычно адрес вроде **`172.17.0.1`**). Пока Ollama привязана только к loopback, запросы из контейнера дают **`connection refused`**.

**Запуск вручную (одна сессия):**

```bash
OLLAMA_HOST=0.0.0.0:11434 ollama serve
```

**Постоянно через systemd** (если Ollama установлена как сервис):

```bash
sudo systemctl edit ollama
```

В открывшемся фрагменте укажите:

```ini
[Service]
Environment="OLLAMA_HOST=0.0.0.0:11434"
```

Затем примените изменения и перезапустите сервис:

```bash
sudo systemctl daemon-reload
sudo systemctl restart ollama
```

### Проверка с хоста

После настройки `OLLAMA_HOST` оба запроса должны успешно отвечать (второй имитирует доступ «с Docker-моста» к тому же хосту):

```bash
curl -sS http://127.0.0.1:11434/api/version
curl -sS http://172.17.0.1:11434/api/version
```

Если второй адрес у вас другой (не `172.17.0.1`), подставьте IP интерфейса **`docker0`** на машине: `ip -brief addr show docker0`.

### Если `host.docker.internal` не подходит (WSL/кастомный bridge)

Переопределите URL Ollama перед запуском compose. Например, если Ollama доступна по `172.31.224.1:11434`:

```bash
OLLAMA_BASE_URL=http://172.31.224.1:11434/v1 make docker-up-opensearch
```

Альтернатива для постоянной настройки: положите в `.env` в корне проекта:

```dotenv
OLLAMA_BASE_URL=http://172.31.224.1:11434/v1
```

### Безопасность

При `0.0.0.0:11434` API Ollama доступен на всех привязанных к хосту интерфейсах (в том числе в LAN). При необходимости ограничьте доступ фаерволом, например только подсетью Docker.

### Примеры вызова API приложения

```bash
curl -sS -X POST 'http://localhost:8080/ollama/chat' \
  -H 'Content-Type: application/json' \
  -d '{"prompt":"Кратко опиши REST."}'

curl -sS -X POST 'http://localhost:8080/ollama/autocomplete' \
  -H 'Content-Type: application/json' \
  -d '{"prefix":"func main() {\n\t"}'
```

## Структура данных

### Elasticsearch Index: `locations`

- `id` (keyword) - Уникальный идентификатор
- `name` (text, keyword) - Название локации
- `address` (text, keyword) - Адрес
- `coordinates` (geo_point) - Географические координаты
- `region` (keyword) - Регион
- `city` (keyword) - Город
- `description` (text) - Описание
- `business_types_suitable` (keyword[]) - Подходящие типы бизнеса
- `traffic_score` (float) - Оценка трафика (0-10)
- `competition_density` (float) - Плотность конкурентов (0-10)
- `demographics` (object) - Демографические данные
- `embedding` — **`dense_vector`** (Elasticsearch) или **`knn_vector`** (OpenSearch), **128** измерений; маппинг задаётся файлами `migrations/elasticsearch_mapping.json` и `migrations/opensearch_mapping.json`

### PostgreSQL Tables

- `business_types` - Справочник типов бизнеса
- `regions` - Справочник регионов

## Документация API

### Swagger/OpenAPI (автоматическая генерация)

Проект использует [swaggo/swag](https://github.com/swaggo/swag) для автоматической генерации OpenAPI документации из комментариев в коде.

**Генерация документации:**
```bash
make swagger
# или
swag init -g cmd/server/main.go -o ./docs --parseDependency --parseInternal
```

**Просмотр Swagger UI:**
1. Запустите сервер: `go run cmd/server/main.go`
2. Откройте в браузере: http://localhost:8080/swagger/index.html

**Дополнительная документация:**
- `SWAGGER.md` - подробное руководство по использованию Swagger
- `api/openapi.yaml` - ручная OpenAPI спецификация (альтернатива)

### Godoc документация

Для просмотра документации Go кода:
```bash
# Локально
go doc ./...

# Или через веб-интерфейс
godoc -http=:6060
# Откройте http://localhost:6060
```

## Разработка

### Добавление новых данных

1. Подготовьте данные в формате JSON или используйте утилиту `indexer`
2. Запустите индексацию:
```bash
go run cmd/indexer/main.go
```

### Тестирование

```bash
# Проверка здоровья
curl http://localhost:8080/health

# Получение типов бизнеса
curl http://localhost:8080/business-types

# Получение регионов
curl http://localhost:8080/regions

# Рекомендация локаций
curl -X POST http://localhost:8080/locations/recommend \
  -H "Content-Type: application/json" \
  -d '{
    "region": "Москва",
    "business_type": "cafe",
    "limit": 10
  }'
```

## Мониторинг

- **Kibana/OpenSearch Dashboards**: http://localhost:5601
- **Elasticsearch/OpenSearch API**: http://localhost:9200
- **Health Check**: http://localhost:8080/health

## Лицензия

MIT License

## Автор

Alexey Kozadaev
