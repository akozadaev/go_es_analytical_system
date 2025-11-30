-- Создание таблицы типов бизнеса
CREATE TABLE IF NOT EXISTS business_types (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL UNIQUE,
    description TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Создание таблицы регионов
CREATE TABLE IF NOT EXISTS regions (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    parent_region_id INTEGER REFERENCES regions(id) ON DELETE SET NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(name, parent_region_id)
);

-- Создание индексов для оптимизации запросов
CREATE INDEX IF NOT EXISTS idx_regions_parent ON regions(parent_region_id);
CREATE INDEX IF NOT EXISTS idx_business_types_name ON business_types(name);

-- Вставка начальных данных для типов бизнеса
INSERT INTO business_types (name, description) VALUES
    ('cafe', 'Кафе'),
    ('repair_shop', 'Ремонт техники'),
    ('tailoring', 'Пошив одежды'),
    ('beauty_salon', 'Салон красоты'),
    ('barbershop', 'Барбершоп'),
    ('laundry', 'Прачечная'),
    ('restaurant', 'Ресторан'),
    ('gym', 'Спортивный зал'),
    ('pharmacy', 'Аптека'),
    ('grocery_store', 'Продуктовый магазин')
ON CONFLICT (name) DO NOTHING;

-- Вставка начальных данных для регионов (пример для России)
INSERT INTO regions (name, parent_region_id) VALUES
    ('Россия', NULL),
    ('Ленинградская область', (SELECT id FROM regions WHERE name = 'Россия')),
    ('Санкт-Петербург', (SELECT id FROM regions WHERE name = 'Ленинградская область')),
    ('Московская область', (SELECT id FROM regions WHERE name = 'Московская область')),
    ('Москва', (SELECT id FROM regions WHERE name = 'Россия')),
    ('Тамбовский муниципальный округ', (SELECT id FROM regions WHERE name = 'Россия')),
    ('Тамбов', (SELECT id FROM regions WHERE name = 'Тамбовский муниципальный округ'))
ON CONFLICT (name, parent_region_id) DO NOTHING;

