// Predict-A-Trade Service Worker v14 — cache-busting fix
var CACHE_NAME = 'pat-dashboard-v36';
var STATIC_ASSETS = [
  '/echarts.min.js',
  '/manifest.json',
  '/favicon.svg',
  '/favicon.ico',
  '/predict-a-trade_favicon.svg',
  '/predict-a-trade_favicon.ico',
  '/apple-touch-icon.png',
  '/icon-192.png',
  '/icon-512.png'
];

// Install: cache static assets only (NOT index.html)
self.addEventListener('install', function(e) {
  e.waitUntil(
    caches.open(CACHE_NAME)
      .then(function(c) {
        return c.addAll(STATIC_ASSETS.map(function(u) {
          return new Request(u, { cache: 'no-store' });
        })).catch(function() {});
      })
      .then(function() { return self.skipWaiting(); })
  );
});

// Activate: purge ALL old caches
self.addEventListener('activate', function(e) {
  e.waitUntil(
    caches.keys()
      .then(function(keys) {
        return Promise.all(
          keys.filter(function(k) { return k !== CACHE_NAME; })
              .map(function(k) { return caches.delete(k); })
        );
      })
      .then(function() { return self.clients.claim(); })
  );
});

// Fetch: network-first for ALL requests, cache is fallback only
self.addEventListener('fetch', function(e) {
  var url = new URL(e.request.url);
  
  // Skip WebSocket
  if (e.request.url.indexOf('ws://') === 0 || e.request.url.indexOf('wss://') === 0) return;
  
  // Skip API requests — always go to network
  if (url.pathname.indexOf('/api/') === 0) return;
  
  // Navigation requests: ALWAYS network-first, NEVER cache HTML
  if (e.request.mode === 'navigate') {
    e.respondWith(
      fetch(e.request, { cache: 'no-store', redirect: 'follow' })
        .then(function(response) {
          return response;
        })
        .catch(function() {
          // Only fall back to cached index.html if network completely fails
          return caches.match('/index.html');
        })
    );
    return;
  }
  
  // Static assets: cache-first (fast load), fallback to network
  e.respondWith(
    caches.match(e.request)
      .then(function(cached) {
        if (cached) return cached;
        return fetch(e.request)
          .then(function(response) {
            // Cache successful responses for static assets
            if (response.ok && url.pathname.match(/\.(js|css|png|svg|ico|json)$/)) {
              var clone = response.clone();
              caches.open(CACHE_NAME).then(function(c) { c.put(e.request, clone); });
            }
            return response;
          });
      })
  );
});
