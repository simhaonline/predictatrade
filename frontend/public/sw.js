/*
 * Predict-A-Trade service worker (PWA installability + offline shell).
 *
 * Strategy (production-safe, self-invalidating — check.md 2026-08-30):
 *  - Never cache API/WS traffic (trading truth must be live).
 *  - Network-FIRST for EVERYTHING, with cache fallback only when the network
 *    is fully down. Cached JS chunks are therefore never served stale.
 *  - Build-scoped cache name: the backend /build-id endpoint returns the
 *    current build id; the SW compares to its own. On mismatch it purges ALL
 *    caches and reloads the page once (so clients always run the newest bundle
 *    after a deploy — no user action required).
 *  - Uses `clients.claim()` + `skipWaiting()` so the new SW takes over
 *    immediately instead of waiting for all tabs to close.
 */
const CACHE = 'pat-shell-v3';
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
  // Take control of all open tabs immediately so the new tactics apply at once.
  self.clients.claim();
});

self.addEventListener('fetch', (event) => {
  const url = new URL(event.request.url);

  // Trading/API truth is never intercepted.
  if (event.request.method !== 'GET' || url.pathname.startsWith('/api/')) return;

  // Never cache Next.js static JS/metadata chunks — these must always be
  // fresh from the network so a deploy reaches every client immediately.
  // (Cache-fallback for JS caused stale-UI reports on 2026-08-29/30.)
  if (url.pathname.startsWith('/_next/') && /\.(js|css|json|meta)$/i.test(url.pathname)) return;

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

// Build-mismatch purge: the page pings /api/build-id; if it differs from the
// build the SW has seen, purge EVERY cache and tell the page to reload once.
let KNOWN_BUILD = null;
self.addEventListener('message', (event) => {
  if (event.data?.type === 'BUILD_ID' && event.data.buildId !== KNOWN_BUILD) {
    if (KNOWN_BUILD !== null) {
      caches.keys().then((keys) => Promise.all(keys.map((k) => caches.delete(k))));
      event.source?.postMessage({ type: 'PURGED', stale: KNOWN_BUILD, fresh: event.data.buildId });
    }
    KNOWN_BUILD = event.data.buildId;
  }
});