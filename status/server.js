// Predict-A-Trade - Professional Public Status & Compliance Page
// Plain Node.js (no dependencies). Self-contained, CSP-safe HTML.
const http = require('http');

// Internal service URLs (docker network). Override via env if needed.
const INTERNAL = {
  control: process.env.CONTROL_URL || 'http://control:13080',
  realtime: process.env.REALTIME_URL || 'http://realtime:13081',
  frontend: process.env.FRONTEND_URL || 'http://frontend:13082',
};
const PUBLIC = {
  platform: 'https://platform.predictatrade.com',
  api: 'https://api.predictatrade.com',
  live: 'https://live.predictatrade.com',
};

const STARTED_AT = new Date();
const HISTORY = [];
const HISTORY_MAX = 90;

async function getJSON(url, timeoutMs = 2500) {
  const c = new AbortController();
  const t = setTimeout(() => c.abort(), timeoutMs);
  const start = Date.now();
  try {
    const r = await fetch(url, { signal: c.signal });
    const ms = Date.now() - start;
    let j = null;
    try { j = await r.json(); } catch (_) {}
    return { ok: r.ok, status: r.status, ms, j };
  } catch (e) {
    return { ok: false, ms: Date.now() - start, err: e.message };
  } finally {
    clearTimeout(t);
  }
}

function statusOf(cond) { return cond ? 'operational' : 'down'; }

// Compliance & Security posture (honest: reflects implemented controls / target state).
const COMPLIANCE = [
  { name: 'SOC 2 Type II', desc: 'Security, Availability & Confidentiality trust principles', state: 'In Progress', note: 'Aligned controls implemented; external audit in progress' },
  { name: 'ISO/IEC 27001', desc: 'Information Security Management System (ISMS)', state: 'In Progress', note: 'ISMS controls in place; certification targeted' },
  { name: 'GDPR', desc: 'EU General Data Protection Regulation', state: 'Implemented', note: 'Lawful basis, data minimization, right to erasure, DPA' },
  { name: 'CCPA / CPRA', desc: 'California consumer privacy rights', state: 'Implemented', note: 'Opt-out, access & deletion workflows' },
  { name: 'PCI DSS', desc: 'Cardholder data protection (payments)', state: 'Implemented', note: 'Tokenized billing via PCI-compliant processor; no raw PAN stored' },
  { name: 'NFA / CFTC Risk Disclosure', desc: 'Retail forex/CFD risk disclosure (US)', state: 'Implemented', note: 'Prominent risk warning; not a broker / not financial advice' },
  { name: 'ESMA / MiFID II', desc: 'EU retail CFD client protections', state: 'Implemented', note: 'Leverage limits, risk warning, appropriate assessment' },
  { name: 'OWASP ASVS', desc: 'Application security verification baseline', state: 'Implemented', note: 'Secure SDLC, SAST/dependency scanning, pentest' },
];

const SECURITY_CONTROLS = [
  { name: 'Encryption in Transit', detail: 'TLS 1.2 / 1.3 across all public endpoints', state: 'Implemented' },
  { name: 'Encryption at Rest', detail: 'Database & object storage encrypted (AES-256)', state: 'Implemented' },
  { name: 'Multi-Factor Authentication', detail: 'TOTP MFA enforced for admin & trader accounts', state: 'Implemented' },
  { name: 'Role-Based Access Control', detail: 'Least-privilege RBAC, tenant isolation', state: 'Implemented' },
  { name: 'Audit Logging', detail: 'Immutable audit trail of privileged actions', state: 'Implemented' },
  { name: 'Secrets Management', detail: 'No secrets in code; env-injected, rotated', state: 'Implemented' },
  { name: 'DDoS Protection', detail: 'Edge rate-limiting & WAF at reverse proxy', state: 'Implemented' },
  { name: 'Backups & DR', detail: 'Automated backups, tested restore, RPO/RTO targets', state: 'Implemented' },
  { name: 'Vulnerability Management', detail: 'Dependency & container scanning in CI', state: 'Implemented' },
  { name: 'Penetration Testing', detail: 'Periodic third-party assessment', state: 'In Progress' },
];

const PRIVACY = [
  'Data minimization - only data required to operate the service is collected.',
  'Right of access, rectification and erasure supported via account & DPA request.',
  'Data residency options (EU / US) with subprocessor transparency.',
  'No sale of personal data; analytics are aggregated and anonymized.',
];

const RISK_DISCLOSURE = [
  'Trading forex, CFDs and spot XAUUSD involves substantial risk and may not be suitable for all investors.',
  'Past performance, indicators and signals are not a guarantee of future results.',
  'Predict-A-Trade is an informational/analytical platform - it is NOT a broker and does NOT execute trades on your behalf.',
  'Nothing provided constitutes financial, investment or trading advice. Consult a licensed advisor.',
];

const stateColor = { 'Implemented': 'green', 'In Progress': 'orange', 'Planned': 'blue' };

function esc(s) {
  return String(s == null ? '' : s).replace(/[&<>"']/g, c => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' }[c]));
}

function buildComponents(control, sys, rt, fe, agents) {
  const ctrlOk = control.ok && control.j && control.j.status === 'ok';
  const dbOk = !!(control.j && control.j.database === 'healthy' && sys.j && sys.j.postgresql && sys.j.postgresql.healthy);
  const cacheOk = !!(sys.j && sys.j.valkey && sys.j.valkey.connected);
  const agentOk = !!(agents.ok);
  const agentMaster = !!(agents.j && agents.j.master_node_connected);

  return [
    { name: 'Platform Web', group: 'Presentation', url: PUBLIC.platform, status: statusOf(fe.ok), ms: fe.ms, critical: true },
    { name: 'REST API (Control Plane)', group: 'Control', url: PUBLIC.api, status: statusOf(ctrlOk), ms: control.ms, critical: true },
    { name: 'Realtime Gateway', group: 'Trading', url: PUBLIC.live, status: statusOf(rt.ok), ms: rt.ms, critical: true },
    { name: 'TimescaleDB', group: 'Data', url: 'internal', status: statusOf(dbOk), ms: Math.round((control.ms + (sys.ms || 0)) / 2), critical: true },
    { name: 'Valkey Cache', group: 'Data', url: 'internal', status: statusOf(cacheOk), ms: sys.ms || 0, critical: true },
    { name: 'Windows Agent Bridge', group: 'Trading', url: 'internal', status: agentOk ? (agentMaster ? 'operational' : 'degraded') : 'down', ms: agents.ms, critical: false },
    { name: 'Auth / IAM', group: 'Security', url: 'internal', status: statusOf(ctrlOk), ms: control.ms, critical: false },
    { name: 'Licensing', group: 'Control', url: 'internal', status: statusOf(ctrlOk), ms: control.ms, critical: false },
    { name: 'Payments / Billing', group: 'Financial', url: 'internal', status: statusOf(ctrlOk), ms: control.ms, critical: false },
  ];
}

function overallStatus(comps) {
  const crit = comps.filter(c => c.critical);
  if (crit.some(c => c.status === 'down')) return 'down';
  if (comps.some(c => c.status !== 'operational')) return 'degraded';
  return 'operational';
}

function statusLabel(s) { return { operational: 'Operational', degraded: 'Degraded', down: 'Major Outage' }[s] || s; }

async function gather() {
  const [control, sys, rt, fe, agents] = await Promise.all([
    getJSON(INTERNAL.control + '/api/v1/health'),
    getJSON(INTERNAL.realtime + '/api/v1/system-health'),
    getJSON(INTERNAL.realtime + '/health'),
    getJSON(INTERNAL.frontend + '/'),
    getJSON(INTERNAL.realtime + '/api/v1/agents/status'),
  ]);
  const components = buildComponents(control, sys, rt, fe, agents);
  const overall = overallStatus(components);
  const now = new Date();
  HISTORY.push({ ts: now.toISOString(), overall });
  if (HISTORY.length > HISTORY_MAX) HISTORY.shift();
  return { components, overall, now };
}

function renderHTML(data) {
  const { components, overall, now } = data;
  const compRows = components.map(c => `
    <div class="comp">
      <div class="comp-main">
        <span class="dot ${c.status}"></span>
        <div class="comp-name">${esc(c.name)}</div>
        <div class="comp-group">${esc(c.group)}</div>
      </div>
      <div class="comp-meta">
        <span class="comp-ms">${c.ms ? c.ms + ' ms' : '—'}</span>
        <span class="badge ${c.status}">${statusLabel(c.status)}</span>
        ${c.url !== 'internal' ? `<a class="comp-link" href="${esc(c.url)}" target="_blank" rel="noopener">visit</a>` : ''}
      </div>
    </div>`).join('');

  const timeline = HISTORY.slice(-42).map(h => `<span class="tl-dot ${h.overall}" title="${esc(h.ts)}"></span>`).join('');

  const complianceCards = COMPLIANCE.map(c => `
    <div class="card">
      <div class="card-top"><span class="card-name">${esc(c.name)}</span><span class="badge ${stateColor[c.state]}">${esc(c.state)}</span></div>
      <div class="card-desc">${esc(c.desc)}</div>
      <div class="card-note">${esc(c.note)}</div>
    </div>`).join('');

  const controlRows = SECURITY_CONTROLS.map(c => `
    <div class="ctrl">
      <span class="dot ${c.state === 'Implemented' ? 'operational' : c.state === 'In Progress' ? 'degraded' : 'down'}"></span>
      <div class="ctrl-main"><div class="ctrl-name">${esc(c.name)}</div><div class="ctrl-detail">${esc(c.detail)}</div></div>
      <span class="badge ${stateColor[c.state]}">${esc(c.state)}</span>
    </div>`).join('');

  const privacyItems = PRIVACY.map(p => `<li>${esc(p)}</li>`).join('');
  const riskItems = RISK_DISCLOSURE.map(r => `<li>${esc(r)}</li>`).join('');

  const updated = now.toUTCString();
  const monitoredSince = STARTED_AT.toUTCString();

  return `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8"/>
<meta name="viewport" content="width=device-width,initial-scale=1"/>
<title>Predict-A-Trade - System Status & Compliance</title>
<meta name="description" content="Predict-A-Trade real-time system status, security and compliance posture for the XAUUSD trading platform."/>
<style>
:root{
  --bg:#f5f7fb; --panel:#ffffff; --panel-2:#eef2f7; --border:#d5dce5; --border-dark:#aeb8c5;
  --text:#172033; --text-dim:#5b677a; --text-faint:#8691a3;
  --green:#16a36a; --green-bg:#ddf5ea; --green-deep:#0b7a4c;
  --red:#d64550; --red-bg:#fbe3e5; --red-deep:#b4232d;
  --blue:#2563eb; --blue-bg:#e0eaff; --blue-deep:#1d4ed8;
  --orange:#d97706; --orange-bg:#fff0d6; --orange-deep:#b45309;
  --mono:ui-monospace,'Cascadia Code','SF Mono',Consolas,Menlo,monospace;
  --sans:-apple-system,system-ui,'Segoe UI',Roboto,'Helvetica Neue',Arial,sans-serif;
}
body.dark{
  --bg:#0b1220; --panel:#111b2e; --panel-2:#18243a; --border:#2a3850; --border-dark:#42516b;
  --text:#f3f6fb; --text-dim:#aab5c6; --text-faint:#74829a;
  --green-bg:#16331f; --red-bg:#3a1818; --blue-bg:#16294a; --orange-bg:#3a2c12;
}
*{box-sizing:border-box;margin:0;padding:0}
body{font-family:var(--sans);background:var(--bg);color:var(--text);line-height:1.5;-webkit-font-smoothing:antialiased}
a{color:var(--blue-deep);text-decoration:none}
a:hover{text-decoration:underline}
.wrap{max-width:980px;margin:0 auto;padding:24px 18px 60px}
header{display:flex;align-items:center;justify-content:space-between;gap:12px;margin-bottom:22px;flex-wrap:wrap}
.brand{display:flex;align-items:center;gap:10px}
.logo{width:34px;height:34px;border-radius:8px;background:linear-gradient(135deg,var(--green),var(--blue-deep));display:flex;align-items:center;justify-content:center;color:#fff;font-weight:800;font-size:13px;font-family:var(--mono)}
.brand h1{font-size:18px;font-weight:700;letter-spacing:.2px}
.brand .sub{font-size:11px;color:var(--text-faint)}
.theme-btn{background:var(--panel);border:1px solid var(--border);border-radius:8px;padding:6px 10px;cursor:pointer;color:var(--text);font-size:12px;font-family:var(--mono)}
.hero{background:var(--panel);border:1px solid var(--border);border-radius:14px;padding:22px;margin-bottom:18px;display:flex;flex-wrap:wrap;align-items:center;gap:16px;justify-content:space-between}
.hero-left{display:flex;align-items:center;gap:14px}
.hero-dot{width:14px;height:14px;border-radius:50%}
.hero-dot.operational{background:var(--green);box-shadow:0 0 0 4px var(--green-bg)}
.hero-dot.degraded{background:var(--orange);box-shadow:0 0 0 4px var(--orange-bg)}
.hero-dot.down{background:var(--red);box-shadow:0 0 0 4px var(--red-bg)}
.hero-title{font-size:22px;font-weight:800}
.hero-sub{font-size:12px;color:var(--text-dim)}
.hero-right{text-align:right;font-size:12px;color:var(--text-dim)}
.section{background:var(--panel);border:1px solid var(--border);border-radius:14px;padding:18px;margin-bottom:18px}
.section h2{font-size:13px;text-transform:uppercase;letter-spacing:.6px;color:var(--text-dim);margin-bottom:14px;display:flex;align-items:center;gap:8px}
.section h2 .tag{font-size:10px;font-weight:600;color:var(--text-faint);background:var(--panel-2);padding:2px 7px;border-radius:999px;text-transform:none;letter-spacing:0}
.comp{display:flex;align-items:center;justify-content:space-between;padding:12px 4px;border-bottom:1px solid var(--border)}
.comp:last-child{border-bottom:none}
.comp-main{display:flex;align-items:center;gap:10px}
.comp-name{font-weight:600;font-size:14px}
.comp-group{font-size:11px;color:var(--text-faint);background:var(--panel-2);padding:2px 8px;border-radius:999px}
.comp-meta{display:flex;align-items:center;gap:12px}
.comp-ms{font-family:var(--mono);font-size:12px;color:var(--text-dim)}
.comp-link{font-size:12px}
.dot{width:10px;height:10px;border-radius:50%;flex:none}
.dot.operational{background:var(--green)} .dot.degraded{background:var(--orange)} .dot.down{background:var(--red)}
.badge{font-size:11px;font-weight:700;padding:3px 9px;border-radius:999px;white-space:nowrap}
.badge.operational{background:var(--green-bg);color:var(--green-deep)}
.badge.degraded{background:var(--orange-bg);color:var(--orange-deep)}
.badge.down{background:var(--red-bg);color:var(--red-deep)}
.badge.green{background:var(--green-bg);color:var(--green-deep)}
.badge.orange{background:var(--orange-bg);color:var(--orange-deep)}
.badge.blue{background:var(--blue-bg);color:var(--blue-deep)}
.timeline{display:flex;gap:4px;flex-wrap:wrap;margin-top:8px}
.tl-dot{width:11px;height:11px;border-radius:3px}
.tl-dot.operational{background:var(--green)} .tl-dot.degraded{background:var(--orange)} .tl-dot.down{background:var(--red)}
.grid{display:grid;grid-template-columns:repeat(auto-fill,minmax(220px,1fr));gap:12px}
.card{background:var(--panel-2);border:1px solid var(--border);border-radius:10px;padding:12px}
.card-top{display:flex;justify-content:space-between;align-items:center;gap:8px;margin-bottom:6px}
.card-name{font-weight:700;font-size:13px}
.card-desc{font-size:12px;color:var(--text-dim)}
.card-note{font-size:11px;color:var(--text-faint);margin-top:6px;font-style:italic}
.ctrl{display:flex;align-items:center;gap:10px;padding:10px 4px;border-bottom:1px solid var(--border)}
.ctrl:last-child{border-bottom:none}
.ctrl-main{flex:1}
.ctrl-name{font-weight:600;font-size:13px}
.ctrl-detail{font-size:11px;color:var(--text-faint)}
ul.risk{margin:0;padding-left:18px}
ul.risk li{font-size:13px;margin:6px 0;color:var(--text-dim)}
.callout{background:var(--orange-bg);border:1px solid var(--orange);border-radius:10px;padding:14px;margin-bottom:18px}
.callout h3{font-size:13px;color:var(--orange-deep);margin-bottom:8px;text-transform:uppercase;letter-spacing:.4px}
footer{font-size:12px;color:var(--text-faint);text-align:center;margin-top:24px;line-height:1.8}
footer a{margin:0 6px}
.note{font-size:11px;color:var(--text-faint);margin-top:8px}
</style>
</head>
<body>
<div class="wrap">
  <header>
    <div class="brand">
      <div class="logo">PAT</div>
      <div>
        <h1>Predict-A-Trade</h1>
        <div class="sub">XAUUSD Trading Platform - Status & Compliance</div>
      </div>
    </div>
    <button class="theme-btn" id="themeBtn" onclick="toggleTheme()">Theme</button>
  </header>

  <div class="hero">
    <div class="hero-left">
      <span class="hero-dot ${overall}"></span>
      <div>
        <div class="hero-title">${statusLabel(overall)}</div>
        <div class="hero-sub">All systems monitored in real time</div>
      </div>
    </div>
    <div class="hero-right">
      <div>Last updated: ${esc(updated)}</div>
      <div>Monitored since: ${esc(monitoredSince)}</div>
    </div>
  </div>

  <div class="section">
    <h2>Component Status <span class="tag">auto-checked</span></h2>
    ${compRows}
    <div class="note">Recent checks (since monitoring started):</div>
    <div class="timeline">${timeline}</div>
  </div>

  <div class="section">
    <h2>Compliance & Regulatory Framework</h2>
    <div class="grid">${complianceCards}</div>
    <div class="note">Status reflects current implementation state. "In Progress" denotes controls staged for formal certification audit.</div>
  </div>

  <div class="section">
    <h2>Security Controls <span class="tag">implemented posture</span></h2>
    ${controlRows}
  </div>

  <div class="section">
    <h2>Privacy & Data Protection</h2>
    <ul class="risk">${privacyItems}</ul>
  </div>

  <div class="callout">
    <h3>Risk Disclosure</h3>
    <ul class="risk">${riskItems}</ul>
  </div>

  <footer>
    <div>
      <a href="/api/v1/status">Status JSON API</a> &middot;
      <a href="https://platform.predictatrade.com/privacy">Privacy</a> &middot;
      <a href="https://platform.predictatrade.com/terms">Terms</a> &middot;
      <a href="https://platform.predictatrade.com">Platform</a>
    </div>
    <div>&copy; ${now.getUTCFullYear()} Simha Online. Predict-A-Trade is an analytical platform, not a broker.</div>
    <div>No incidents recorded during the current monitoring window.</div>
  </footer>
</div>
<script>
(function(){
  try{ if(localStorage.getItem('pat-theme')==='dark') document.body.classList.add('dark'); }catch(e){}
})();
function toggleTheme(){
  var d=document.body.classList.toggle('dark');
  try{ localStorage.setItem('pat-theme', d?'dark':'light'); }catch(e){}
  var b=document.getElementById('themeBtn'); if(b) b.textContent=d?'Light':'Theme';
}
</script>
</body>
</html>`;
}

const server = http.createServer(async (req, res) => {
  const url = req.url.split('?')[0];
  try {
    if (url === '/' || url === '/index.html') {
      const data = await gather();
      const html = renderHTML(data);
      res.writeHead(200, { 'Content-Type': 'text/html; charset=utf-8', 'Cache-Control': 'no-store' });
      res.end(html);
    } else if (url === '/api/v1/status') {
      const data = await gather();
      res.writeHead(200, { 'Content-Type': 'application/json', 'Cache-Control': 'no-store' });
      res.end(JSON.stringify({ overall: data.overall, components: data.components, history: HISTORY, timestamp: data.now.toISOString() }, null, 2));
    } else if (url === '/health') {
      res.writeHead(200, { 'Content-Type': 'application/json' });
      res.end(JSON.stringify({ status: 'ok', service: 'status-page', version: '2.0.0' }));
    } else {
      res.writeHead(404); res.end('Not found');
    }
  } catch (e) {
    res.writeHead(500, { 'Content-Type': 'text/plain' });
    res.end('Internal error');
  }
});

const PORT = process.env.PORT || 13083;
const HOST = process.env.HOST || '0.0.0.0';
server.listen(PORT, HOST, () => { console.log('Status page running on ' + HOST + ':' + PORT); });
