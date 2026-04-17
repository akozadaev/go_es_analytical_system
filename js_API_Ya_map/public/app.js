const statusEl = document.getElementById('status');
const pointsListEl = document.getElementById('pointsList');
const countEl = document.getElementById('count');

const fields = {
  esUrl: document.getElementById('esUrl'),
  indexName: document.getElementById('indexName'),
  username: document.getElementById('username'),
  password: document.getElementById('password'),
  xmlFileName: document.getElementById('xmlFileName'),
  name: document.getElementById('name'),
  description: document.getElementById('description'),
  businessTypes: document.getElementById('businessTypes'),
  trafficScore: document.getElementById('trafficScore'),
  competitionDensity: document.getElementById('competitionDensity'),
  ageGroup: document.getElementById('ageGroup'),
  averageIncome: document.getElementById('averageIncome'),
  interests: document.getElementById('interests'),
  populationDensity: document.getElementById('populationDensity')
};

const createIndexBtn = document.getElementById('createIndexBtn');
const bulkIndexBtn = document.getElementById('bulkIndexBtn');
const saveXmlBtn = document.getElementById('saveXmlBtn');

const locations = [];
let locationCounter = 1;
let map;

function setStatus(message, isError = false) {
  statusEl.textContent = message;
  statusEl.style.color = isError ? '#b91c1c' : '#166534';
}

async function readJSONResponse(response) {
  const text = await response.text();
  if (!text.trim()) {
    return {};
  }
  try {
    return JSON.parse(text);
  } catch {
    throw new Error(`Сервер вернул не-JSON ответ (${response.status}): ${text.slice(0, 180)}`);
  }
}

function getBusinessTypes() {
  return fields.businessTypes.value
    .split(',')
    .map((item) => item.trim())
    .filter(Boolean);
}

function getInterests() {
  return fields.interests.value
    .split(',')
    .map((item) => item.trim())
    .filter(Boolean);
}

function formatCoord(value) {
  return Number(value).toFixed(6);
}

function updatePointsList() {
  pointsListEl.innerHTML = '';
  for (const location of locations) {
    const item = document.createElement('li');
    item.textContent =
      `${location.name} | ${location.city || '-'} | ${formatCoord(location.coordinates.lat)}, ${formatCoord(location.coordinates.lon)}`;
    pointsListEl.appendChild(item);
  }

  countEl.textContent = String(locations.length);
}

function toLocationPayload(coords, geocodeMeta = {}) {
  const now = new Date().toISOString();
  const id = `loc_${Date.now()}_${locationCounter++}`;

  return {
    id,
    name: fields.name.value || `Локация ${locationCounter}`,
    address: geocodeMeta.address || 'Не определен',
    coordinates: {
      lat: coords[0],
      lon: coords[1]
    },
    region: geocodeMeta.region || 'Не определен',
    city: geocodeMeta.city || 'Не определен',
    description: fields.description.value || 'Добавлено через интерфейс Yandex Maps JS API',
    business_types_suitable: getBusinessTypes(),
    traffic_score: Number(fields.trafficScore.value) || 0,
    competition_density: Number(fields.competitionDensity.value) || 0,
    demographics: {
      age_group: fields.ageGroup.value || '26-35',
      average_income: Number(fields.averageIncome.value) || 0,
      interests: getInterests(),
      population_density: Number(fields.populationDensity.value) || 0
    },
    created_at: now,
    updated_at: now
  };
}

function addPlacemark(coords, locationName) {
  const placemark = new ymaps.Placemark(
    coords,
    {
      balloonContent: locationName
    },
    {
      preset: 'islands#blueDotIcon'
    }
  );

  map.geoObjects.add(placemark);
}

function parseAddressMeta(geoObject) {
  const properties = geoObject.properties;
  const fullAddress = properties.get('text') || '';
  const components = properties.get('metaDataProperty.GeocoderMetaData.Address.Components') || [];

  let city = '';
  let region = '';
  for (const component of components) {
    if (!city && component.kind === 'locality') {
      city = component.name;
    }
    if (!region && component.kind === 'province') {
      region = component.name;
    }
  }

  return {
    address: fullAddress,
    city: city || '',
    region: region || ''
  };
}

function onMapClick(coords) {
  ymaps
    .geocode(coords, { results: 1 })
    .then((result) => {
      const firstGeoObject = result.geoObjects.get(0);
      const meta = firstGeoObject ? parseAddressMeta(firstGeoObject) : {};
      const location = toLocationPayload(coords, meta);

      locations.push(location);
      addPlacemark(coords, location.name);
      updatePointsList();

      setStatus(`Добавлена точка: ${location.name} (${location.address})`);
    })
    .catch(() => {
      const location = toLocationPayload(coords);
      locations.push(location);
      addPlacemark(coords, location.name);
      updatePointsList();
      setStatus(`Добавлена точка: ${location.name} (без geocode-адреса)`);
    });
}

function initMap() {
  map = new ymaps.Map('map', {
    center: [55.751244, 37.618423],
    zoom: 10,
    controls: ['zoomControl', 'searchControl']
  });

  map.events.add('click', (event) => {
    const coords = event.get('coords');
    onMapClick(coords);
  });
}

async function createIndex() {
  try {
    setStatus('Создаю индекс...');

    const response = await fetch('/map-indexer/api/create-index', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        esUrl: fields.esUrl.value,
        indexName: fields.indexName.value,
        username: fields.username.value || undefined,
        password: fields.password.value || undefined
      })
    });

    const result = await readJSONResponse(response);
    if (!response.ok || !result.ok) {
      throw new Error(result.message || 'Не удалось создать индекс');
    }

    setStatus(result.message);
  } catch (error) {
    setStatus(error.message, true);
  }
}

async function bulkIndex() {
  try {
    if (locations.length === 0) {
      setStatus('Нет точек для индексации', true);
      return;
    }

    setStatus(`Индексирую ${locations.length} точек...`);

    const response = await fetch('/map-indexer/api/bulk-index', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        esUrl: fields.esUrl.value,
        indexName: fields.indexName.value,
        username: fields.username.value || undefined,
        password: fields.password.value || undefined,
        locations
      })
    });

    const result = await readJSONResponse(response);
    if (!response.ok || !result.ok) {
      const msg = result.message || 'Ошибка индексации';
      throw new Error(msg);
    }

    setStatus(result.message);
  } catch (error) {
    setStatus(error.message, true);
  }
}

async function saveXml() {
  try {
    if (locations.length === 0) {
      setStatus('Нет точек для сохранения XML', true);
      return;
    }

    setStatus(`Сохраняю XML для ${locations.length} точек...`);
    const response = await fetch('/map-indexer/api/save-xml', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        filename: fields.xmlFileName.value || 'locations.xml',
        locations
      })
    });

    const result = await readJSONResponse(response);
    if (!response.ok || !result.ok) {
      throw new Error(result.message || 'Ошибка сохранения XML');
    }

    setStatus(result.message);
  } catch (error) {
    setStatus(error.message, true);
  }
}

createIndexBtn.addEventListener('click', createIndex);
bulkIndexBtn.addEventListener('click', bulkIndex);
saveXmlBtn.addEventListener('click', saveXml);

ymaps.ready(initMap);
