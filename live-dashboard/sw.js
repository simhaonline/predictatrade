// Predict-A-Trade Service Worker v12
var CACHE_NAME = 'pat-dashboard-v29';
var STATIC_ASSETS = ['/','/index.html','/echarts.min.js','/manifest.json','/favicon.svg','/favicon.ico'];
self.addEventListener('install',function(e){e.waitUntil(caches.open(CACHE_NAME).then(c=>c.addAll(STATIC_ASSETS.map(u=>new Request(u,{cache:'no-store'}))).catch(()=>{})).then(()=>self.skipWaiting()));});
self.addEventListener('activate',function(e){e.waitUntil(caches.keys().then(keys=>Promise.all(keys.filter(k=>k!==CACHE_NAME).map(k=>caches.delete(k)))).then(()=>self.clients.claim()));});
self.addEventListener('fetch',function(e){
  var url=new URL(e.request.url);
  if(e.request.url.indexOf('ws://')===0||e.request.url.indexOf('wss://')===0)return;
  if(url.pathname.indexOf('/api/')===0)return;
  if(e.request.mode==='navigate'){e.respondWith(fetch(e.request).catch(()=>caches.match('/index.html')));return;}
  e.respondWith(fetch(e.request).catch(()=>caches.match(e.request)));
});
