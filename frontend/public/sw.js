/*
 * Predict-A-Trade service worker (PWA installability + offline shell).
 *
 * Strategy:
 *  - Never cache API or WebSocket traffic (trading truth must be live).
 *  - Network-first for navigations/static with cache fallback so the shell
 *    still opens when the network is down. Cached pages are labeled stale.
 */
const CACHE = 'pat-shell-v2'; // bump to invalidate old cached chunks
const SHELL_ASSETS = ['/', '/offline'];

self.addEventListener('install', (event) => {
  event.waitUntil(
    caches.open(CACHE).then((cache) => cache.addAll(SHELL_ASSETS)).catch(() => undefined),
  );
  self.skipWaiting();
});

self.addEventListener('activate', (event) => {
  event.waitUntil(
    caches.keys().then((keys) =>
      Promise.all(keys.filter((k) => k !== CACHE).map((k) => caches.delete(k))),
    ),
  );
  self.clients.claim();
});

self.addEventListener('fetch', (event) => {
  const url = new URL(event.request.url);

  // Trading/API truth is never served from cache.
  if (event.request.method !== 'GET' || url.pathname.startsWith('/api/')) return;

  event.respondWith(
    fetch(event.request)
      .then((response) => {
        if (response.ok && url.origin === self.location.origin) {
          const clone = response.clone();
          caches.open(CACHE).then((cache) => cache.put(event.request, clone));
        }
        return response;
      })
      .catch(() =>
        caches.match(event.request).then(
          (cached) =>
            cached ||
            caches.match('/offline').then((offline) => offline || Response.error()),
        ),
      ),
  );
});
