const CACHE_NAME = 'predict-a-trade-v14';
const APP_SHELL = [
  '/', '/index.html', '/manifest.json',
  '/assets/index-c4adef1e.js', '/assets/index-BJBN36Jh.css',
  '/cookie-consent.css', '/cookie-consent.js',
  '/media/predict-a-trade-static-hero.webp', '/media/pat-market-texture.webp',
  '/media/predict-a-trade_horizontal.webp', '/media/predict-a-trade_icon-only.webp',
  '/media/icon-192.webp', '/media/icon-512.webp'
];

self.addEventListener('install', (event) => {
  event.waitUntil(
    caches.open(CACHE_NAME)
      .then((cache) => Promise.allSettled(APP_SHELL.map((asset) => cache.add(asset))))
      .then(() => self.skipWaiting())
  );
});

self.addEventListener('activate', (event) => {
  event.waitUntil(
    caches.keys()
      .then((keys) => Promise.all(keys.filter((key) => key !== CACHE_NAME).map((key) => caches.delete(key))))
      .then(() => self.clients.claim())
  );
});

self.addEventListener('fetch', (event) => {
  if (event.request.method !== 'GET' || new URL(event.request.url).origin !== self.location.origin) return;
  event.respondWith(
    caches.match(event.request).then((cached) => {
      if (cached) return cached;
      return fetch(event.request).then((response) => {
        if (response.ok) {
          const copy = response.clone();
          caches.open(CACHE_NAME).then((cache) => cache.put(event.request, copy));
        }
        return response;
      }).catch(() => caches.match('/index.html'));
    })
  );
});
