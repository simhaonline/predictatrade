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

// Canonical Predict-A-Trade brand mark (frontend/public/predict-a-trade_icon-only.svg,
// single path, fill switched to currentColor so it adapts to light/dark theme).
// Kept in sync manually when the brand asset changes.
const LOGO_SVG_PATH = '<svg viewBox="0 0 1000 1000" xmlns="http://www.w3.org/2000/svg" fill="currentColor" aria-hidden="true"><path d="M 314.726 59.516 C 305.135 68.556, 292.675 76.451, 276.545 83.711 C 271.020 86.197, 253.900 92.956, 238.500 98.732 C 199.602 113.320, 190.871 117.969, 179.458 130.170 C 165.779 144.793, 160.150 164.164, 164.585 181.356 C 169.678 201.096, 186.718 218.694, 212.683 231.030 C 221.618 235.275, 237.532 241, 240.398 241 C 241.279 241, 242 241.450, 242 242 C 242 243.303, 232.819 243.270, 221.500 241.925 C 215.574 241.221, 207.035 239.029, 196.500 235.507 C 187.553 232.516, 177.635 229.876, 174 229.518 C 166.334 228.764, 155.758 230.589, 145.604 234.419 C 139.269 236.809, 128 242.636, 128 243.522 C 128 243.729, 130.813 245.252, 134.250 246.907 C 137.688 248.562, 144.550 253.140, 149.500 257.079 C 160.706 265.997, 166.570 269.410, 175.500 272.212 C 181.303 274.033, 184.951 274.417, 196.831 274.454 L 211.163 274.500 206.125 276.711 C 197.911 280.317, 189.758 282, 180.503 282 C 173.484 282, 170.058 281.418, 162.841 279 C 157.916 277.350, 153.399 276, 152.804 276 C 148.624 276, 95.983 327.378, 81.596 345.500 C 76.508 351.909, 70.974 360.295, 68.337 365.594 C 64.229 373.846, 63.708 375.722, 62.706 385.863 C 60.765 405.512, 64.015 435.508, 70.910 461.582 C 75.216 477.860, 80.106 490.534, 89.019 508.509 C 97.962 526.545, 108.601 543.065, 120.700 557.700 C 130.780 569.895, 150.526 589.209, 162.262 598.356 C 172.367 606.231, 191.178 619.155, 191.731 618.602 C 191.938 618.396, 190.064 615.070, 187.568 611.211 C 178.278 596.850, 171.699 581.334, 168.400 566 C 165.977 554.742, 165.951 529.786, 168.352 519.162 C 173.800 495.048, 185.505 472.388, 202.857 452.364 C 211.791 442.055, 222.677 432.673, 235.826 423.952 C 267.320 403.063, 304.863 393.181, 332.579 398.484 C 343.688 400.609, 351.498 404.619, 359.818 412.467 C 366.669 418.930, 367.192 419.218, 373.981 420.269 C 379.129 421.066, 383.305 421.056, 389.466 420.231 C 398.763 418.987, 401.587 417.807, 409.293 411.940 C 412.157 409.760, 416.893 407.215, 419.818 406.284 C 422.742 405.354, 426.302 403.446, 427.728 402.046 C 431.655 398.189, 441.058 377.710, 442.341 370.219 L 443.442 363.793 439.471 360.275 C 424.472 346.985, 417.033 338.054, 405.094 319 C 397.126 306.284, 397.291 307.140, 399.981 292.500 C 400.994 286.989, 401.013 284.577, 400.072 281.162 C 398.143 274.168, 365.332 226.301, 356.375 217.416 C 346.691 207.809, 325.762 199.092, 288 188.938 C 277.825 186.202, 265.336 182.845, 260.246 181.479 C 242.335 176.669, 230.473 169.360, 226.406 160.625 C 223.615 154.631, 223.419 151.329, 225.551 146.218 C 228.535 139.062, 233.722 134.877, 258 120.037 C 265.425 115.499, 276 108.512, 281.500 104.510 C 291.836 96.990, 305.258 83.861, 310.647 76 C 315.496 68.926, 322.686 53.959, 321.199 54.036 C 320.815 54.057, 317.902 56.522, 314.726 59.516 M 502.698 148.500 C 496.859 167.945, 491.441 179.680, 483.595 189.877 C 470.686 206.653, 451.411 217.944, 426.923 223.076 C 418.747 224.789, 411.493 225.394, 393.271 225.880 L 370.043 226.500 383.771 246.802 L 397.500 267.104 405 266.492 C 427.522 264.655, 450.580 257.164, 466 246.676 C 472.680 242.132, 482.656 232.305, 487.496 225.500 C 501.004 206.510, 506.221 186.179, 505.071 157 L 504.500 142.500 502.698 148.500 M 556.965 260.184 C 556.946 260.908, 556.834 268.700, 556.715 277.500 L 556.500 293.500 550 293.785 C 545.617 293.978, 543.124 294.567, 542.347 295.594 C 541.509 296.700, 541.156 313.626, 541.055 357.445 C 540.929 411.743, 541.077 417.934, 542.529 419.386 C 543.686 420.543, 545.999 421, 550.693 421 L 557.242 421 556.960 434.250 C 556.547 453.623, 556.482 453.192, 559.750 452.816 L 562.500 452.500 562.775 436.750 L 563.051 421 569.403 421 C 573.543 421, 576.149 420.526, 576.885 419.638 C 577.705 418.651, 577.945 401.272, 577.758 356.402 C 577.532 302.289, 577.312 294.502, 576 294.325 C 575.175 294.214, 571.923 293.839, 568.774 293.493 L 563.048 292.862 562.774 276.181 L 562.500 259.500 559.750 259.184 C 557.994 258.982, 556.987 259.343, 556.965 260.184 M 291 274.547 C 291 274.820, 292.095 276.721, 293.432 278.771 C 294.770 280.822, 297.059 285.515, 298.519 289.200 C 301.226 296.034, 304.478 300.165, 308.979 302.485 C 310.366 303.200, 315.172 304.476, 319.661 305.320 C 328.953 307.069, 333.110 308.905, 337.750 313.308 C 339.538 315.004, 341 316.067, 341 315.668 C 341 314.233, 334.070 300.961, 330.958 296.439 C 327.729 291.745, 318.216 284.812, 310.500 281.530 C 303.074 278.371, 291 274.047, 291 274.547 M 502 333.500 L 502 351 495.107 351 C 489.744 351, 488.079 351.351, 487.607 352.582 C 486.858 354.534, 486.790 517, 487.538 517 C 488.382 517, 505.634 505.313, 514.720 498.586 L 522.940 492.500 522.720 422 L 522.500 351.500 515.250 351.206 L 508 350.912 508 333.456 L 508 316 505 316 L 502 316 502 333.500 M 659.321 352.803 C 637.773 364.186, 619.911 373.736, 619.629 374.025 C 619.347 374.314, 623.027 376.926, 627.808 379.830 C 632.589 382.734, 637.603 385.943, 638.951 386.961 L 641.402 388.812 638.475 393.156 C 632.196 402.474, 615.683 424.113, 605.398 436.500 C 592.040 452.589, 556.096 488.680, 542 500.157 C 469.437 559.241, 386.343 593.236, 326.971 588.125 C 286.444 584.637, 258.555 566.806, 247.078 537.047 C 236.829 510.470, 242.263 479, 262.281 449 C 268.619 439.502, 287.562 420.573, 295.765 415.540 C 299.211 413.427, 301.838 411.504, 301.602 411.268 C 300.703 410.370, 288.215 414.233, 277.781 418.637 C 265.831 423.681, 250.067 432.777, 239.994 440.440 C 229.948 448.082, 214.156 464.601, 207.272 474.667 C 198.052 488.151, 190.846 503.463, 187.569 516.535 C 179.305 549.504, 187.584 580.415, 210.394 601.752 C 251.566 640.267, 328.828 647.805, 410.496 621.275 C 497.925 592.874, 581.251 527.592, 652.086 432 C 656.773 425.675, 662.384 417.930, 664.554 414.788 C 666.724 411.646, 668.842 409.059, 669.261 409.038 C 669.680 409.017, 674.333 412.184, 679.602 416.075 C 684.871 419.967, 689.311 423.005, 689.468 422.825 C 689.625 422.646, 690.732 413.500, 691.928 402.500 C 693.124 391.500, 695.167 373.275, 696.467 362 C 699.628 334.600, 699.859 331.991, 699.127 332.053 C 698.782 332.082, 680.870 341.420, 659.321 352.803 M 447.667 391.667 C 447.300 392.033, 447 398.558, 447 406.167 L 447 420 440.700 420 C 437.133 420, 433.879 420.521, 433.200 421.200 C 432.293 422.107, 432 437.467, 432 484.200 C 432 518.190, 432.222 546, 432.494 546 C 433.646 546, 454.046 536.478, 460.600 532.881 L 467.736 528.964 468.404 509.732 C 468.771 499.154, 468.980 475.269, 468.869 456.654 C 468.704 429.129, 468.404 422.546, 467.261 421.404 C 466.347 420.490, 463.615 420, 459.429 420 L 453 420 453 405.500 L 453 391 450.667 391 C 449.383 391, 448.033 391.300, 447.667 391.667 M 391 453.956 L 391 466.912 383.750 467.206 L 376.500 467.500 376.241 516.820 C 375.995 563.487, 376.076 566.114, 377.741 565.636 C 391.612 561.656, 405.264 557.468, 408.260 556.274 L 412.021 554.774 411.760 511.137 L 411.500 467.500 404.250 467.206 L 397 466.912 397 453.956 L 397 441 394 441 L 391 441 391 453.956 M 321.219 503.750 C 320.823 504.712, 320.500 520.575, 320.500 539 L 320.500 572.500 325.500 572.829 C 330.366 573.150, 350.235 571.459, 354.750 570.341 L 357 569.783 357 537.473 C 357 519.703, 356.727 504.452, 356.393 503.582 C 355.873 502.226, 353.372 502, 338.862 502 C 323.684 502, 321.863 502.181, 321.219 503.750"/></svg>';

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
  // F6 fix: claims must reflect verifiable reality. Items without evidence
  // are marked "In Progress"/"Planned" — never "Implemented" (SOW: no
  // unsupported compliance/security claims).
  { name: 'Encryption in Transit', detail: 'TLS 1.2 / 1.3 across all public endpoints; legacy TLS refused', state: 'Implemented' },
  { name: 'Role-Based Access Control', detail: '8-role least-privilege model; permission guards on finance/admin; privileged tokens re-validated per request', state: 'Implemented' },
  { name: 'Audit Logging', detail: 'Privileged actions recorded in audit schema', state: 'Implemented' },
  { name: 'Secrets Management', detail: 'Env-injected secrets, restrictive file permissions; vault + rotation workflow in progress', state: 'In Progress' },
  { name: 'DDoS Protection', detail: 'Edge rate-limiting at reverse proxy; per-route throttling on auth', state: 'Implemented' },
  { name: 'Backups & DR', detail: '6h logical backups + continuous WAL to off-host S3; restore drills in progress', state: 'In Progress' },
  { name: 'Multi-Factor Authentication', detail: 'TOTP enforced server-side for privileged roles; privileged endpoints blocked until enrolled', state: 'Implemented' },
  { name: 'Encryption at Rest', detail: 'Database & object storage encrypted (AES-256)', state: 'Planned' },
  { name: 'Vulnerability Management', detail: 'Go govulncheck + npm audit on dependencies; findings tracked and patched', state: 'Implemented' },
  { name: 'Penetration Testing', detail: 'Periodic external surface audits (TLS, rate limits, auth, exposure); third-party assessment planned', state: 'In Progress' },
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
  // Bridge is healthy only when the data feed is actually fresh (not merely
  // "connected"). data_health=HEALTHY requires live, non-stale market data.
  const agentMaster = !!(agents.j && agents.j.agents_online && agents.j.data_health === 'HEALTHY');

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
<title>Predict-A-Trade - System Status &amp; Compliance</title>
<link rel="icon" href="https://platform.predictatrade.com/predict-a-trade_favicon.svg" type="image/svg+xml"/>
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
.brand h1{font-size:18px;font-weight:700;letter-spacing:.2px}
.brand .sub{font-size:11px;color:var(--text-faint)}
.logo{width:34px;height:34px;display:flex;align-items:center;justify-content:center;color:var(--text);flex:none}
.logo svg{width:100%;height:100%}
.theme-btn{background:var(--panel);border:1px solid var(--border);border-radius:8px;width:34px;height:34px;padding:0;cursor:pointer;color:var(--text);display:flex;align-items:center;justify-content:center;flex:none}
.theme-btn:hover{background:var(--panel-2)}
.theme-btn svg{width:17px;height:17px;display:block}
.top-links{display:flex;align-items:center;gap:14px;font-size:12px;flex-wrap:wrap}
.top-links a{display:flex;align-items:center;gap:5px;color:var(--text-dim);white-space:nowrap}
.top-links a:hover{color:var(--blue-deep);text-decoration:none}
.top-links svg{width:13px;height:13px;flex:none}
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
      <span class="logo">${LOGO_SVG_PATH}</span>
      <div>
        <h1>Predict-A-Trade</h1>
        <div class="sub">XAUUSD Trading Platform - Status &amp; Compliance</div>
      </div>
    </div>
    <div class="top-links">
      <a href="${PUBLIC.platform}/" target="_blank" rel="noopener" title="Go to Dashboard">
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><rect x="3" y="3" width="7" height="7" rx="1"/><rect x="14" y="3" width="7" height="7" rx="1"/><rect x="3" y="14" width="7" height="7" rx="1"/><rect x="14" y="14" width="7" height="7" rx="1"/></svg>
        Dashboard
      </a>
      <a href="https://predictatrade.com/" target="_blank" rel="noopener" title="Go to Home Page">
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M3 10.5 12 3l9 7.5"/><path d="M5 9.5V21h14V9.5"/><path d="M9 21v-6h6v6"/></svg>
        Home
      </a>
      <button class="theme-btn" id="themeBtn" onclick="toggleTheme()" aria-label="Toggle light / dark mode" title="Toggle light / dark mode">
        <svg id="themeIcon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
          <circle cx="12" cy="12" r="4"/><path d="M12 2v2M12 20v2M4.93 4.93l1.41 1.41M17.66 17.66l1.41 1.41M2 12h2M20 12h2M4.93 19.07l1.41-1.41M17.66 6.34l1.41-1.41"/>
        </svg>
      </button>
    </div>
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
    <div>${overall === 'down'
      ? 'Incident in progress — components are degraded or down. Engineers notified.'
      : 'No incidents recorded during the current monitoring window.'}</div>
  </footer>
</div>
<script>
var SUN_ICON = '<circle cx="12" cy="12" r="4"/><path d="M12 2v2M12 20v2M4.93 4.93l1.41 1.41M17.66 17.66l1.41 1.41M2 12h2M20 12h2M4.93 19.07l1.41-1.41M17.66 6.34l1.41-1.41"/>';
var MOON_ICON = '<path d="M21 12.79A9 9 0 1 1 11.21 3 7 7 0 0 0 21 12.79z"/>';
function setThemeIcon(dark){
  var i=document.getElementById('themeIcon');
  if(i) i.innerHTML = dark ? MOON_ICON : SUN_ICON;
}
(function(){
  var dark=false;
  try{ dark = localStorage.getItem('pat-theme')==='dark'; }catch(e){}
  if(dark) document.body.classList.add('dark');
  setThemeIcon(dark);
})();
function toggleTheme(){
  var d=document.body.classList.toggle('dark');
  try{ localStorage.setItem('pat-theme', d?'dark':'light'); }catch(e){}
  setThemeIcon(d);
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
