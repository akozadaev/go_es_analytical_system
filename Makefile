.PHONY: build run test clean docker-up docker-down docker-logs index help

help: ## Показать справку
	@echo "Доступные команды:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-15s %s\n", $$1, $$2}'

build: ## Собрать приложение
	go build -o bin/server ./cmd/server
	go build -o bin/indexer ./cmd/indexer

run: ## Запустить сервер локально
	go run cmd/server/main.go

index: ## Индексировать тестовые данные
	go run cmd/indexer/main.go

test: ## Запустить тесты
	go test ./...

clean: ## Очистить собранные файлы
	rm -rf bin/

docker-up: ## Запустить все сервисы через Docker Compose
	docker rm -f go_analytical_app 2>/dev/null || true
	docker-compose up -d

docker-up-opensearch: ## Запустить все сервисы с OpenSearch (альтернатива ES)
	docker rm -f go_analytical_app 2>/dev/null || true
	docker-compose -f docker-compose.opensearch.yml up -d

docker-down: ## Остановить все сервисы
	docker-compose down

docker-down-opensearch: ## Остановить сервисы OpenSearch
	docker-compose -f docker-compose.opensearch.yml down

docker-logs: ## Показать логи Docker Compose
	docker-compose logs -f

docker-restart: ## Перезапустить сервисы
	docker-compose restart

docker-rebuild: ## Пересобрать образ приложения и перезапустить контейнер
	@echo "Пересборка образа приложения..."
	@docker rm -f go_analytical_app 2>/dev/null || true
	@docker-compose -f docker-compose.opensearch.yml rm -f app 2>/dev/null || true
	docker-compose -f docker-compose.opensearch.yml build app
	docker-compose -f docker-compose.opensearch.yml up -d app
	@echo "✓ Образ пересобран и контейнер перезапущен"

docker-rebuild-opensearch: ## Пересобрать образ и перезапустить все сервисы (OpenSearch)
	@echo "Пересборка образа приложения..."
	@docker rm -f go_analytical_app 2>/dev/null || true
	@docker-compose -f docker-compose.opensearch.yml rm -f app 2>/dev/null || true
	docker-compose -f docker-compose.opensearch.yml build app
	docker-compose -f docker-compose.opensearch.yml up -d
	@echo "✓ Образ пересобран и все сервисы перезапущены"

docker-rebuild-all: ## Пересобрать образ приложения без кеша и перезапустить
	@echo "Пересборка образа приложения (без кеша)..."
	@docker rm -f go_analytical_app 2>/dev/null || true
	@docker-compose -f docker-compose.opensearch.yml rm -f app 2>/dev/null || true
	docker-compose -f docker-compose.opensearch.yml build --no-cache app
	docker-compose -f docker-compose.opensearch.yml up -d app
	@echo "✓ Образ пересобран (без кеша) и контейнер перезапущен"

docker-index: ## Индексировать данные в Docker контейнере
	docker-compose exec app ./indexer

docker-index-opensearch: ## Индексировать данные в Docker контейнере (OpenSearch)
	docker-compose -f docker-compose.opensearch.yml exec app ./indexer

docker-shell: ## Открыть shell в контейнере приложения
	docker-compose exec app sh

docker-clean: ## Удалить конфликтующие контейнеры (если были созданы вручную)
	docker rm -f go_analytical_app 2>/dev/null || true

install-deps: ## Установить зависимости
	go mod download
	go mod tidy

swagger: ## Сгенерировать Swagger/OpenAPI документацию
	@echo "Генерация Swagger документации..."
	swag init -g cmd/server/main.go -o ./docs --parseDependency --parseInternal
	@echo "✓ Swagger документация сгенерирована в docs/"

swagger-serve: swagger ## Сгенерировать и показать Swagger UI (требует запущенный сервер)
	@echo "Swagger UI доступен по адресу: http://localhost:8080/swagger/index.html"

db-reset: ## Очистить и перезаполнить данные PostgreSQL из миграций
	@echo "Очистка и перезаполнение данных PostgreSQL..."
	@if docker ps | grep -q postgres_analytical_system; then \
		cat migrations/002_reset_data.sql | docker exec -i postgres_analytical_system psql -U analytical_user -d analytical_db; \
	elif docker-compose -f docker-compose.opensearch.yml ps postgres 2>/dev/null | grep -q "Up"; then \
		cat migrations/002_reset_data.sql | docker-compose -f docker-compose.opensearch.yml exec -T postgres psql -U analytical_user -d analytical_db; \
	else \
		echo "Ошибка: PostgreSQL не запущен. Запустите: make docker-up-opensearch"; \
		exit 1; \
	fi
	@echo "✓ Данные PostgreSQL перезаполнены"

db-reset-local: ## Очистить и перезаполнить данные PostgreSQL (локально, без Docker)
	@echo "Очистка и перезаполнение данных PostgreSQL (локально)..."
	@PGPASSWORD=analytical_pass psql -h localhost -U analytical_user -d analytical_db -f migrations/002_reset_data.sql || \
	(echo "Ошибка: убедитесь, что PostgreSQL запущен и доступен на localhost:5432" && exit 1)
	@echo "✓ Данные PostgreSQL перезаполнены"

db-clean: ## Очистить все данные из таблиц (без перезаполнения)
	@echo "Очистка данных PostgreSQL..."
	@if docker ps | grep -q postgres_analytical_system; then \
		docker exec postgres_analytical_system psql -U analytical_user -d analytical_db -c "TRUNCATE TABLE regions CASCADE; TRUNCATE TABLE business_types CASCADE;"; \
	elif docker-compose -f docker-compose.opensearch.yml ps postgres 2>/dev/null | grep -q "Up"; then \
		docker-compose -f docker-compose.opensearch.yml exec postgres psql -U analytical_user -d analytical_db -c "TRUNCATE TABLE regions CASCADE; TRUNCATE TABLE business_types CASCADE;"; \
	else \
		echo "Ошибка: PostgreSQL не запущен"; \
		exit 1; \
	fi
	@echo "✓ Данные очищены"

db-recreate: ## Полностью пересоздать базу данных (удалить volumes и создать заново)
	@echo "⚠️  ВНИМАНИЕ: Это удалит все данные и volumes!"
	@read -p "Продолжить? [y/N] " -n 1 -r; \
	echo; \
	if [[ $$REPLY =~ ^[Yy]$$ ]]; then \
		docker-compose -f docker-compose.opensearch.yml stop postgres 2>/dev/null || docker-compose stop postgres 2>/dev/null; \
		docker-compose -f docker-compose.opensearch.yml rm -f postgres 2>/dev/null || docker-compose rm -f postgres 2>/dev/null; \
		docker volume rm go_es_analytical_system_postgres_data 2>/dev/null || true; \
		docker-compose -f docker-compose.opensearch.yml up -d postgres || docker-compose up -d postgres; \
		echo "Ожидание готовности PostgreSQL..."; \
		sleep 10; \
		echo "✓ База данных пересоздана"; \
	else \
		echo "Отменено"; \
	fi

