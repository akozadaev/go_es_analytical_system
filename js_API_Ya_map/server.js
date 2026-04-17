const express = require('express');
const fs = require('fs/promises');
const path = require('path');

const app = express();
const port = process.env.PORT || 3001;
const exportsDir = path.join(__dirname, 'exports');

app.use(express.json({ limit: '5mb' }));
app.use(express.static(path.join(__dirname, 'public')));

function normalizeEsUrl(esUrl) {
  const url = (esUrl || '').trim();
  if (!url) {
    throw new Error('Не указан URL Elasticsearch/OpenSearch');
  }

  return url.replace(/\/$/, '');
}

function createHeaders(username, password, contentType = 'application/json') {
  const headers = {
    'Content-Type': contentType
  };

  if (username && password) {
    const token = Buffer.from(`${username}:${password}`).toString('base64');
    headers.Authorization = `Basic ${token}`;
  }

  return headers;
}

async function readMappingFile(mappingPath) {
  const fallbackPath = path.resolve(__dirname, '..', 'migrations', 'elasticsearch_mapping.json');
  const filePath = mappingPath
    ? path.resolve(__dirname, mappingPath)
    : fallbackPath;

  const content = await fs.readFile(filePath, 'utf-8');
  return { filePath, body: JSON.parse(content) };
}

function escapeXml(value) {
  return String(value ?? '')
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
    .replace(/'/g, '&apos;');
}

function buildLocationsXml(locations) {
  const lines = ['<?xml version="1.0" encoding="UTF-8"?>', '<locations>'];

  for (const location of locations) {
    const businessTypes = Array.isArray(location.business_types_suitable)
      ? location.business_types_suitable
      : [];
    const interests = Array.isArray(location.demographics?.interests)
      ? location.demographics.interests
      : [];
    const embedding = Array.isArray(location.embedding) ? location.embedding : [];

    lines.push('  <location>');
    lines.push(`    <id>${escapeXml(location.id)}</id>`);
    lines.push(`    <name>${escapeXml(location.name)}</name>`);
    lines.push(`    <address>${escapeXml(location.address)}</address>`);
    lines.push('    <coordinates>');
    lines.push(`      <lat>${Number(location.coordinates?.lat || 0)}</lat>`);
    lines.push(`      <lon>${Number(location.coordinates?.lon || 0)}</lon>`);
    lines.push('    </coordinates>');
    lines.push(`    <region>${escapeXml(location.region)}</region>`);
    lines.push(`    <city>${escapeXml(location.city)}</city>`);
    lines.push(`    <description>${escapeXml(location.description)}</description>`);
    lines.push('    <business_types_suitable>');
    for (const businessType of businessTypes) {
      lines.push(`      <business_type>${escapeXml(businessType)}</business_type>`);
    }
    lines.push('    </business_types_suitable>');
    lines.push(`    <traffic_score>${Number(location.traffic_score || 0)}</traffic_score>`);
    lines.push(`    <competition_density>${Number(location.competition_density || 0)}</competition_density>`);
    lines.push('    <demographics>');
    lines.push(`      <age_group>${escapeXml(location.demographics?.age_group || '')}</age_group>`);
    lines.push(`      <average_income>${Number(location.demographics?.average_income || 0)}</average_income>`);
    lines.push('      <interests>');
    for (const interest of interests) {
      lines.push(`        <interest>${escapeXml(interest)}</interest>`);
    }
    lines.push('      </interests>');
    lines.push(`      <population_density>${Number(location.demographics?.population_density || 0)}</population_density>`);
    lines.push('    </demographics>');
    lines.push('    <embedding>');
    for (const value of embedding) {
      lines.push(`      <value>${Number(value)}</value>`);
    }
    lines.push('    </embedding>');
    lines.push(`    <created_at>${escapeXml(location.created_at)}</created_at>`);
    lines.push(`    <updated_at>${escapeXml(location.updated_at)}</updated_at>`);
    lines.push('  </location>');
  }

  lines.push('</locations>');
  return lines.join('\n');
}

app.post('/api/create-index', async (req, res) => {
  try {
    const {
      esUrl,
      indexName = 'locations',
      mappingPath,
      username,
      password
    } = req.body;

    const normalizedEsUrl = normalizeEsUrl(esUrl);

    const existsResponse = await fetch(`${normalizedEsUrl}/${indexName}`, {
      method: 'HEAD',
      headers: createHeaders(username, password)
    });

    if (existsResponse.status === 200) {
      return res.json({
        ok: true,
        message: `Индекс ${indexName} уже существует`
      });
    }

    if (existsResponse.status !== 404) {
      const text = await existsResponse.text();
      return res.status(400).json({
        ok: false,
        message: `Не удалось проверить индекс: ${existsResponse.status}`,
        details: text
      });
    }

    const mapping = await readMappingFile(mappingPath);

    const createResponse = await fetch(`${normalizedEsUrl}/${indexName}`, {
      method: 'PUT',
      headers: createHeaders(username, password),
      body: JSON.stringify(mapping.body)
    });

    const text = await createResponse.text();

    if (!createResponse.ok) {
      return res.status(400).json({
        ok: false,
        message: `Ошибка создания индекса: ${createResponse.status}`,
        details: text
      });
    }

    return res.json({
      ok: true,
      message: `Индекс ${indexName} создан`,
      mappingPath: mapping.filePath,
      details: text
    });
  } catch (error) {
    return res.status(500).json({
      ok: false,
      message: error.message
    });
  }
});

app.post('/api/bulk-index', async (req, res) => {
  try {
    const {
      esUrl,
      indexName = 'locations',
      locations,
      username,
      password
    } = req.body;

    if (!Array.isArray(locations) || locations.length === 0) {
      return res.status(400).json({
        ok: false,
        message: 'Список locations пуст'
      });
    }

    const normalizedEsUrl = normalizeEsUrl(esUrl);
    const headers = createHeaders(username, password, 'application/x-ndjson');

    let payload = '';
    for (const location of locations) {
      const id = location.id || `loc_${Date.now()}_${Math.random().toString(16).slice(2)}`;
      const meta = { index: { _index: indexName, _id: id } };
      payload += `${JSON.stringify(meta)}\n${JSON.stringify({ ...location, id })}\n`;
    }

    const bulkResponse = await fetch(`${normalizedEsUrl}/_bulk?refresh=true`, {
      method: 'POST',
      headers,
      body: payload
    });

    const result = await bulkResponse.json().catch(() => ({}));

    if (!bulkResponse.ok) {
      return res.status(400).json({
        ok: false,
        message: `Ошибка bulk индексации: ${bulkResponse.status}`,
        details: result
      });
    }

    const hasErrors = Boolean(result.errors);
    const failedItems = hasErrors
      ? (result.items || []).filter((item) => item.index && item.index.error)
      : [];

    return res.json({
      ok: !hasErrors,
      message: hasErrors
        ? `Частичная индексация: ошибок ${failedItems.length}`
        : `Успешно проиндексировано ${locations.length} локаций`,
      failedItems
    });
  } catch (error) {
    return res.status(500).json({
      ok: false,
      message: error.message
    });
  }
});

app.post('/api/save-xml', async (req, res) => {
  try {
    const { locations, filename } = req.body;

    if (!Array.isArray(locations) || locations.length === 0) {
      return res.status(400).json({
        ok: false,
        message: 'Список locations пуст'
      });
    }

    const safeName = String(filename || 'locations.xml')
      .replace(/[^a-zA-Z0-9._-]/g, '_')
      .replace(/_+/g, '_');
    const finalFileName = safeName.toLowerCase().endsWith('.xml') ? safeName : `${safeName}.xml`;

    const xmlContent = buildLocationsXml(locations);
    await fs.mkdir(exportsDir, { recursive: true });
    const filePath = path.join(exportsDir, finalFileName);
    await fs.writeFile(filePath, xmlContent, 'utf-8');

    return res.json({
      ok: true,
      message: `XML сохранен: ${filePath}`,
      filePath
    });
  } catch (error) {
    return res.status(500).json({
      ok: false,
      message: error.message
    });
  }
});

app.listen(port, () => {
  console.log(`Yandex Maps indexer started: http://localhost:${port}`);
});
