(function () {
  function loadApp() {
    var s = document.createElement('script');
    s.src = './app.js';
    document.body.appendChild(s);
  }

  // Локальный server.js (порт 3001): нет /api/me — работаем без OAuth-шлюза.
  if (!window.location.pathname.startsWith('/map-indexer')) {
    loadApp();
    return;
  }

  function deny() {
    window.location.replace('/app/#forbidden-map');
  }

  var token = sessionStorage.getItem('oauth_access_token');
  if (!token) {
    deny();
    return;
  }

  fetch('/api/me', { headers: { Authorization: 'Bearer ' + token } })
    .then(function (res) {
      if (!res.ok) {
        throw new Error('me');
      }
      return res.json();
    })
    .then(function (me) {
      if (me.oauth2_api_auth === false) {
        loadApp();
        return;
      }
      if (!me.map_indexer) {
        deny();
        return;
      }
      loadApp();
    })
    .catch(function () {
      deny();
    });
})();
