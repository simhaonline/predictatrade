// Predict-A-Trade Status Page — safe public health aggregation
const http = require('http');

const SERVICES = [
  { name: 'API', url: 'http://127.0.0.1:13080/api/v1/health', public: 'https://api.predictatrade.com' },
  { name: 'Realtime Gateway', url: 'http://127.0.0.1:13081/health', public: 'https://live.predictatrade.com' },
  { name: 'Platform', url: 'http://127.0.0.1:13082/', public: 'https://platform.predictatrade.com' },
  { name: 'Database', url: 'http://127.0.0.1:13080/api/v1/health', public: 'internal' },
];

async function checkService(svc) {
  try {
    const controller = new AbortController();
    const timeout = setTimeout(() => controller.abort(), 3000);
    const res = await fetch(svc.url, { signal: controller.signal });
    clearTimeout(timeout);
    if (res.ok) return { name: svc.name, status: 'operational', url: svc.public };
    return { name: svc.name, status: 'degraded', url: svc.public };
  } catch {
    return { name: svc.name, status: 'down', url: svc.public };
  }
}

const server = http.createServer(async (req, res) => {
  if (req.url === '/' || req.url === '/index.html') {
    const results = await Promise.all(SERVICES.map(checkService));
    const allOperational = results.every(r => r.status === 'operational');
    const overall = allOperational ? 'operational' : 'degraded';
    const html = `<!DOCTYPE html>
<html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1">
<title>Predict-A-Trade — System Status</title>
<style>
body{font-family:Inter,-apple-system,sans-serif;background:#121B2B;color:#F9F9FA;margin:0;padding:40px}
.container{max-width:800px;margin:0 auto}
h1{color:#E9A265;font-size:28px;margin-bottom:8px}
.subtitle{color:#B7BBC4;margin-bottom:32px}
.status-badge{display:inline-block;padding:4px 12px;border-radius:9999px;font-size:14px;font-weight:500;margin-bottom:24px}
.operational{background:rgba(6,214,111,0.15);color:#06D66F}
.degraded{background:rgba(233,162,101,0.15);color:#E9A265}
.down{background:rgba(232,97,97,0.15);color:#E86161}
.service{display:flex;justify-content:space-between;align-items:center;padding:16px;background:#172030;border:1px solid #2B3547;border-radius:8px;margin-bottom:12px}
.service-name{font-size:16px;font-weight:500}
.service-status{font-size:14px;font-weight:500}
.updated{color:#9297A0;font-size:12px;margin-top:24px}
a{color:#4F98EC;text-decoration:none}
</style></head><body><div class="container">
<h1>Predict-A-Trade</h1><div class="subtitle">System Status</div>
<div class="status-badge ${overall}">● ${overall.charAt(0).toUpperCase()+overall.slice(1)}</div>
${results.map(r=>`<div class="service"><span class="service-name">${r.name}</span><span class="service-status ${r.status}">● ${r.status.charAt(0).toUpperCase()+r.status.slice(1)}</span></div>`).join('')}
<div class="updated">Last updated: ${new Date().toISOString()} · <a href="https://platform.predictatrade.com">platform.predictatrade.com</a></div>
</div></body></html>`;
    res.writeHead(200, { 'Content-Type': 'text/html', 'Cache-Control': 'no-store' });
    res.end(html);
  } else if (req.url === '/api/v1/status') {
    const results = await Promise.all(SERVICES.map(checkService));
    res.writeHead(200, { 'Content-Type': 'application/json', 'Cache-Control': 'no-store' });
    res.end(JSON.stringify({ services: results, timestamp: new Date().toISOString() }));
  } else if (req.url === '/health') {
    res.writeHead(200, { 'Content-Type': 'application/json' });
    res.end(JSON.stringify({ status: 'ok', service: 'status-page', version: '1.0.0' }));
  } else {
    res.writeHead(404); res.end('Not found');
  }
});

const PORT = process.env.STATUS_PORT || 13083;
const HOST = process.env.STATUS_HOST || '127.0.0.1';
server.listen(PORT, HOST, () => { console.log(`Status page running on ${HOST}:${PORT}`); });
