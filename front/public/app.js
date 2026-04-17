const API = '';

const authPanel = document.getElementById('authPanel');
const workspace = document.getElementById('workspace');
const bannerSlot = document.getElementById('bannerSlot');
const wsBanner = document.getElementById('wsBanner');
const authStatus = document.getElementById('authStatus');
const wsStatus = document.getElementById('wsStatus');
const tabLogin = document.getElementById('tabLogin');
const tabRegister = document.getElementById('tabRegister');
const loginForm = document.getElementById('loginForm');
const registerForm = document.getElementById('registerForm');
const mapLinkWrap = document.getElementById('mapLinkWrap');

const TOKEN_KEY = 'oauth_access_token';

function setAuthStatus(msg, isError) {
  authStatus.textContent = msg;
  authStatus.style.color = isError ? '#b91c1c' : '#166534';
}

function setWsStatus(msg, isError) {
  wsStatus.textContent = msg;
  wsStatus.style.color = isError ? '#b91c1c' : '#166534';
}

function authHeaders() {
  const t = sessionStorage.getItem(TOKEN_KEY);
  const h = { Accept: 'application/json' };
  if (t) {
    h.Authorization = 'Bearer ' + t;
  }
  return h;
}

function hasMapIndexerAccess(me) {
  const roles = Array.isArray(me && me.roles) ? me.roles : [];
  if (me && me.map_indexer === true) {
    return true;
  }
  const allowed = new Set([
    'ROLE_SUPER_ADMIN',
    'ROLE_ADMIN',
    'ROLE_EDITOR',
    'super_admin_role',
    'admin_role',
    'editor_role'
  ]);
  return roles.some((r) => allowed.has(r));
}

async function fetchMe() {
  const meRes = await fetch(API + '/api/me', { headers: authHeaders() });
  const me = await meRes.json().catch(() => ({}));
  return { meRes, me };
}

function redirectAfterLogin(me) {
  if (hasMapIndexerAccess(me)) {
    window.location.replace('/map-indexer/');
    return true;
  }
  return false;
}

function showBanner(el, text, kind) {
  el.innerHTML = '';
  const d = document.createElement('div');
  d.className = 'banner ' + (kind || 'info');
  d.textContent = text;
  el.appendChild(d);
}

function setTab(which) {
  const login = which === 'login';
  tabLogin.classList.toggle('active', login);
  tabRegister.classList.toggle('active', !login);
  loginForm.classList.toggle('hidden', !login);
  registerForm.classList.toggle('hidden', login);
}

tabLogin.addEventListener('click', () => setTab('login'));
tabRegister.addEventListener('click', () => setTab('register'));

loginForm.addEventListener('submit', async (e) => {
  e.preventDefault();
  const username = document.getElementById('loginUsername').value.trim();
  const password = document.getElementById('loginPassword').value;
  setAuthStatus('Вход…');
  try {
    const res = await fetch(API + '/api/auth/login', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ username, password })
    });
    const data = await res.json().catch(() => ({}));
    if (!res.ok) {
      const msg = data.error_description || data.error || data.message || JSON.stringify(data);
      throw new Error(msg || 'Ошибка входа');
    }
    if (!data.access_token) {
      throw new Error('Сервер не вернул access_token');
    }
    sessionStorage.setItem(TOKEN_KEY, data.access_token);
    setAuthStatus('Успешный вход');
    const { meRes, me } = await fetchMe();
    if (!meRes.ok) {
      const msg = me.error_description || me.error || me.message || meRes.status;
      throw new Error('Не удалось получить /api/me: ' + msg);
    }
    if (!me.authenticated) {
      throw new Error('Пользователь не аутентифицирован');
    }
    if (!redirectAfterLogin(me)) {
      await openWorkspace(meRes, me);
    }
  } catch (err) {
    setAuthStatus(err.message || String(err), true);
  }
});

registerForm.addEventListener('submit', async (e) => {
  e.preventDefault();
  const username = document.getElementById('regUsername').value.trim();
  const password = document.getElementById('regPassword').value;
  setAuthStatus('Регистрация…');
  try {
    const res = await fetch(API + '/api/auth/register', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ username, password })
    });
    const text = await res.text();
    let data;
    try {
      data = JSON.parse(text);
    } catch {
      data = { raw: text };
    }
    if (!res.ok) {
      const msg = data.message || data.error || text || 'Ошибка регистрации';
      throw new Error(typeof msg === 'string' ? msg : JSON.stringify(msg));
    }
    setAuthStatus('Аккаунт создан. Войдите с этими данными.');
    setTab('login');
    document.getElementById('loginUsername').value = username;
    document.getElementById('loginPassword').value = '';
    document.getElementById('loginPassword').focus();
  } catch (err) {
    setAuthStatus(err.message || String(err), true);
  }
});

document.getElementById('logoutBtn').addEventListener('click', () => {
  sessionStorage.removeItem(TOKEN_KEY);
  workspace.classList.add('hidden');
  authPanel.classList.remove('hidden');
  wsBanner.innerHTML = '';
  window.location.hash = '';
});

document.getElementById('recommendBtn').addEventListener('click', async () => {
  const region = document.getElementById('regionSelect').value;
  const businessType = document.getElementById('businessSelect').value;
  const limit = Number(document.getElementById('limitInput').value) || 10;
  setWsStatus('Запрос…');
  document.getElementById('resultPre').textContent = '';
  try {
    const res = await fetch(API + '/locations/recommend', {
      method: 'POST',
      headers: { ...authHeaders(), 'Content-Type': 'application/json' },
      body: JSON.stringify({ region, business_type: businessType, limit })
    });
    const data = await res.json().catch(() => ({}));
    if (!res.ok) {
      const msg = data.error_description || data.error || data.message || res.statusText;
      throw new Error(msg || 'Ошибка API');
    }
    document.getElementById('resultPre').textContent = JSON.stringify(data, null, 2);
    setWsStatus('Готово');
  } catch (err) {
    setWsStatus(err.message || String(err), true);
    document.getElementById('resultPre').textContent = '{}';
  }
});

async function loadRefs() {
  const [rRes, bRes] = await Promise.all([
    fetch(API + '/regions', { headers: authHeaders() }),
    fetch(API + '/business-types', { headers: authHeaders() })
  ]);
  const regions = rRes.ok ? await rRes.json() : [];
  const business = bRes.ok ? await bRes.json() : [];
  const rs = document.getElementById('regionSelect');
  const bs = document.getElementById('businessSelect');
  rs.innerHTML = '';
  bs.innerHTML = '';
  for (const reg of regions) {
    const o = document.createElement('option');
    o.value = reg.name || '';
    o.textContent = reg.name || String(reg.id || '');
    rs.appendChild(o);
  }
  for (const bt of business) {
    const o = document.createElement('option');
    o.value = bt.name || '';
    o.textContent = bt.name || String(bt.id || '');
    bs.appendChild(o);
  }
}

async function openWorkspace(meRes, me) {
  if (me.oauth2_api_auth === false) {
    showBanner(wsBanner, 'OAuth2 для API отключён: запросы к /locations/* без токена.', 'warn');
    mapLinkWrap.classList.remove('hidden');
  } else {
    wsBanner.innerHTML = '';
    if (!meRes.ok) {
      showBanner(wsBanner, 'Не удалось получить /api/me: ' + (me.error_description || me.error || meRes.status), 'warn');
    } else if (me.map_indexer) {
      showBanner(wsBanner, 'Роли: ' + (me.roles || []).join(', ') + '. Доступна индексация карт.', 'info');
      mapLinkWrap.classList.remove('hidden');
    } else {
      showBanner(wsBanner, 'Роли: ' + (me.roles || []).join(', ') + '. Раздел карт только для администраторов.', 'info');
      mapLinkWrap.classList.add('hidden');
    }
  }

  await loadRefs();
  authPanel.classList.add('hidden');
  workspace.classList.remove('hidden');
}

async function boot() {
  if (window.location.hash === '#forbidden-map') {
    showBanner(bannerSlot, 'Недостаточно прав для инструмента карт (нужны admin / super_admin / editor).', 'warn');
    window.location.hash = '';
  }

  const token = sessionStorage.getItem(TOKEN_KEY);
  if (!token) {
    return;
  }

  const { meRes, me } = await fetchMe();
  if (meRes.ok && me.authenticated) {
    if (!redirectAfterLogin(me)) {
      await openWorkspace(meRes, me);
    }
  }
}

boot();
