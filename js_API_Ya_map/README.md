# js_API_Ya_map

Проект индексации локаций через **JavaScript API Яндекс Карт** вместо `cmd/indexer/main.go`.

Используется API-ключ:
`6bf8d3e3-967a-4e76-ab37-e238bb4dd0c2`

## Что делает

- Показывает карту Яндекса в браузере.
- По клику добавляет точку и делает reverse geocode (адрес/город/регион).
- Формирует документы в формате модели `Location`.
- Создает индекс `locations` (или указанный вами).
- Делает bulk-индексацию в Elasticsearch/OpenSearch.
- Сохраняет XML-файл в `js_API_Ya_map/exports`, совместимый с `loadLocationsFromFile`.

## Запуск

```bash
cd js_API_Ya_map
npm install
npm start
```

Откройте:
`http://localhost:3001`

## Как использовать

1. Укажите `URL Elasticsearch/OpenSearch` (например, `http://localhost:9200`).
2. Нажмите `Создать индекс`.
3. Кликните по карте для добавления точек.
4. Нажмите `Индексировать точки`.
5. Нажмите `Сохранить XML` для сохранения файла (например, `locations.xml`).

## Примечание

По умолчанию маппинг берется из:
`../migrations/elasticsearch_mapping.json`

Сохраненный XML можно загружать индексатором, например:
`loadLocationsFromFile("js_API_Ya_map/exports/locations.xml")`
