// Predict-A-Trade Service Worker
// Caches static assets for offline use, network-first for API/WS

var CACHE_NAME = 'pat-dashboard-v8';
var STATIC_ASSETS = [
  '/',
  '/index.html',
  '/echarts.min.js',
  '/manifest.json',
  '/favicon.svg',
  '/favicon.ico',
  '/favicon-16.png',
  '/favicon-32.png',
  '/favicon-48.png',
  '/logo-light.svg',
  '/logo-dark.svg',
  '/icon-64.png',
  '/icon-128.png',
  '/icon-256.png',
  '/icon-512.png',
  '/icon-1024.png',
  '/icon.svg'
];

// Install — cache static assets
self.addEventListener('install', function(e) {
  e.waitUntil(
    caches.open(CACHE_NAME).then(function(cache) {
      return cache.addAll(STATIC_ASSETS.map(function(url) {
        return new Request(url, {cache: 'no-store'});
      })).catch(function() {
        // If some assets fail, continue anyway
        return;
      });
    }).then(function() {
      return self.skipWaiting();
    })
  );
});

// Activate — clean old caches
self.addEventListener('activate', function(e) {
  e.waitUntil(
    caches.keys().then(function(keys) {
      return Promise.all(
        keys.filter(function(key) {
          return key !== CACHE_NAME;
        }).map(function(key) {
          return caches.delete(key);
        })
      );
    }).then(function() {
      return self.clients.claim();
    })
  );
});

// Fetch — network-first for API/WS, cache-first for static assets
self.addEventListener('fetch', function(e) {
  var url = new URL(e.request.url);
  
  // Never intercept WebSocket upgrades
  if (e.request.url.indexOf('ws://') === 0 || e.request.url.indexOf('wss://') === 0) {
    return;
  }
  
  // Never intercept API calls (always network)
  if (url.pathname.indexOf('/api/') === 0) {
    return;
  }
  
  // For navigation requests — network first, fallback to cache
  if (e.request.mode === 'navigate') {
    e.respondWith(
      fetch(e.request).catch(function() {
        return caches.match('/index.html');
      })
    );
    return;
  }
  
  // For static assets — cache first, then network
  if (STATIC_ASSETS.indexOf(url.pathname) >= 0 || 
      /\.(js|css|svg|ico|png|jpg|woff|woff2)$/.test(url.pathname)) {
    e.respondWith(
      caches.match(e.request).then(function(cached) {
        if (cached) return cached;
        return fetch(e.request).then(function(resp) {
          if (resp.ok) {
            var clone = resp.clone();
            caches.open(CACHE_NAME).then(function(cache) {
              cache.put(e.request, clone);
            });
          }
          return resp;
        }).catch(function() {
          return caches.match(e.request);
        });
      })
    );
    return;
  }
  
  // Default — try network, fallback to cache
  e.respondWith(
    fetch(e.request).catch(function() {
      return caches.match(e.request);
    })
  );
});
