package agent

import (
	"encoding/json"
	"fmt"
	"html"
	"log"
	"net/http"
	"os"
	"time"
)

// healthServer exposes a minimal local HTTP server so that operators (and the
// health-check.ps1 monitor) can see the agent's live CLIENT (EA/MT) and
// SERVER (backend) status without connecting to the backend or the terminal.
//
// The endpoint listens on 127.0.0.1:9000 by default. The port can be
// overridden via the PAT_HEALTH_PORT environment variable.
//
//   GET /                 → nice HTML status dashboard (auto-refreshing)
//   GET /status           → same HTML status dashboard
//   GET /health           → 200 OK JSON (for health-check.ps1 / NSSM)
//   GET /api/status       → full JSON status snapshot
//
// This is a strictly additive feature — it does not interfere with the
// existing WebSocket, pipe, or heartbeat subsystems.

const defaultHealthPort = "9000"

// healthServer wraps the net/http server so we can shut it down cleanly.
type healthServer struct {
	a         *Agent
	srv       *http.Server
	startTime time.Time
}

// newHealthServer creates a status endpoint bound to localhost only.
func newHealthServer(a *Agent) *healthServer {
	port := os.Getenv("PAT_HEALTH_PORT")
	if port == "" {
		port = defaultHealthPort
	}

	h := &healthServer{
		a:         a,
		startTime: time.Now(),
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/health", h.jsonHandler)
	mux.HandleFunc("/api/status", h.jsonHandler)
	mux.HandleFunc("/status", h.pageHandler)
	mux.HandleFunc("/", h.pageHandler)
	mux.HandleFunc("/api/update", h.updateHandler)

	h.srv = &http.Server{
		Addr:         fmt.Sprintf("127.0.0.1:%s", port),
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
	}
	return h
}

// start launches the status server in a background goroutine.
func (h *healthServer) start() {
	go func() {
		log.Printf("[health] Local status endpoint listening on http://%s/ (and /health)", h.srv.Addr)
		if err := h.srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("[health] WARNING: status endpoint failed: %v", err)
		}
	}()
}

// stop gracefully shuts down the status server.
func (h *healthServer) stop() {
	if h != nil && h.srv != nil {
		_ = h.srv.Close()
	}
}

func (h *healthServer) jsonHandler(w http.ResponseWriter, r *http.Request) {
	s := h.a.getStatus()
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(s)
}

func (h *healthServer) pageHandler(w http.ResponseWriter, r *http.Request) {
	s := h.a.getStatus()
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(renderStatusHTML(s)))
}

// updateHandler triggers an immediate background update check (dashboard button).
func (h *healthServer) updateHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	msg := h.a.RequestUpdate()
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok", "message": msg})
}

// renderStatusHTML builds a clean, self-contained status dashboard.
// The displayed cards depend on the agent role:
//   - data (Master Node): MASTER NODE connection + live CANDLE DELIVERY status.
//   - exec (Client):      CLIENT connection + live SIGNAL DELIVERY status.
// A standalone "SERVER (Backend)" card is intentionally omitted; the backend
// link is shown only as the relevant delivery channel for the role.
func renderStatusHTML(s AgentStatus) string {
	uptime := time.Duration(s.UptimeSeconds) * time.Second
	uptimeStr := uptime.Truncate(time.Second).String()

	isMaster := s.Mode == "data"

	serverBadge := badge(s.BackendConnected, "CONNECTED", "OFFLINE")
	mt4 := badge(s.MT4Connected, "MT4 CONNECTED", "MT4 OFFLINE")
	mt5 := badge(s.MT5Connected, "MT5 CONNECTED", "MT5 OFFLINE")
	terminalLink := fmt.Sprintf("%s %s", mt4, mt5)

	licClass := "bad"
	licText := "UNKNOWN"
	if s.LicenseStatus != "" {
		switch s.LicenseStatus {
		case "ACTIVE", "VALID", "OK":
			licClass = "ok"
			licText = s.LicenseStatus
		case "INVALID", "EXPIRED", "BANNED", "SUSPENDED":
			licClass = "bad"
			licText = s.LicenseStatus
		default:
			licClass = "warn"
			licText = s.LicenseStatus
		}
	} else if !isMaster {
		// No terminal connected → no live license verdict to show (it would be stale).
		licClass = "warn"
		licText = "NO TERMINAL"
	}
	// A Master Node (data role) is purely a market-data source and requires NO
	// trading license, so never show it as "LICENSE PENDING".
	if isMaster {
		licClass = "warn"
		licText = "N/A · DATA NODE"
	}
	licBadge := fmt.Sprintf(`<span class="badge %s">LICENSE %s</span>`, licClass, html.EscapeString(licText))
	planText := html.EscapeString(s.LicensePlan)
	if planText == "" {
		planText = "—"
	}

	drift := s.ClockDriftMs
	if drift < 0 {
		drift = -drift
	}
	driftClass := "ok"
	if drift > 120000 {
		driftClass = "bad"
	} else if drift > 30000 {
		driftClass = "warn"
	}
	driftStr := fmt.Sprintf("%d ms", s.ClockDriftMs)

	deviceShort := s.DeviceID
	if len(deviceShort) > 8 {
		deviceShort = deviceShort[:8]
	}

	// Role-specific cards.
	var cards string
	if isMaster {
		roleTitle := "Master Node (Data · Broker TF)"
		// A Master Node is a pure market-data source — it has NO trading license
		// and must not display license/plan rows at all.
		connCard := cardHTML("MASTER NODE (Terminal)",
			rowHTML("Terminal link", terminalLink))
		deliveryCard := cardHTML("CANDLE DELIVERY → Engine",
			rowHTML("Backend URL", html.EscapeString(s.BackendURL))+
				rowHTML("Backend data WS", serverBadge)+
				rowHTML("Candles delivered", fmt.Sprintf("%d", s.CandlesDelivered))+
				rowHTML("Last candle", html.EscapeString(s.LastCandleDelivered))+
				rowHTML("Clock drift", fmt.Sprintf(`<span class="badge %s">%s</span>`, driftClass, driftStr)))
		cards = connCard + deliveryCard
		_ = roleTitle
	} else {
		roleTitle := "Client (Execution)"
		connCard := cardHTML("CLIENT (EA / MetaTrader)",
			rowHTML("Terminal link", terminalLink)+
				rowHTML("License", licBadge)+
				rowHTML("Plan", planText))
		deliveryCard := cardHTML("SIGNAL DELIVERY → EA",
			rowHTML("Backend URL", html.EscapeString(s.BackendURL))+
				rowHTML("Backend exec WS", serverBadge)+
				rowHTML("Signals delivered", fmt.Sprintf("%d", s.SignalsDelivered))+
				rowHTML("Last signal", html.EscapeString(s.LastSignalDelivered)))
		cards = connCard + deliveryCard
		_ = roleTitle
	}

	return fmt.Sprintf(statusPageTemplate,
		html.EscapeString(s.Version),
		html.EscapeString(deviceShort),
		uptimeStr,
		html.EscapeString(s.Mode),
		cards,
		html.EscapeString(s.GeneratedAt),
	)
}

// cardHTML renders a titled card containing pre-built row HTML.
func cardHTML(title, rows string) string {
	return fmt.Sprintf(`<div class="card"><h2>%s</h2>%s</div>`, title, rows)
}

// rowHTML renders a label/value row.
func rowHTML(label, value string) string {
	return fmt.Sprintf(`<div class="row"><span class="label">%s</span><span class="value">%s</span></div>`, label, value)
}

func badge(ok bool, yes, no string) string {
	if ok {
		return `<span class="badge ok">` + yes + `</span>`
	}
	return `<span class="badge bad">` + no + `</span>`
}

const statusPageTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta http-equiv="refresh" content="5">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>Predict-A-Trade XAUUSD — Agent Status</title>
<style>
  :root { --bg:#0b1020; --card:#141b2e; --fg:#e6ecff; --muted:#8aa0c8; --ok:#1fbf75; --warn:#f5b942; --bad:#ff5d6c; --accent:#4c8dff; }
  * { box-sizing:border-box; }
  body { margin:0; background:var(--bg); color:var(--fg); font-family:"Segoe UI",system-ui,Roboto,Helvetica,Arial,sans-serif; }
  .wrap { max-width:920px; margin:0 auto; padding:28px 18px 40px; }
  header { display:flex; align-items:baseline; justify-content:space-between; flex-wrap:wrap; gap:8px; border-bottom:1px solid #223; padding-bottom:14px; margin-bottom:18px; }
  h1 { font-size:20px; margin:0; letter-spacing:.3px; }
  h1 .sub { color:var(--muted); font-weight:400; font-size:14px; }
  .meta { color:var(--muted); font-size:13px; }
  .grid { display:grid; grid-template-columns:1fr 1fr; gap:16px; }
  @media (max-width:680px){ .grid { grid-template-columns:1fr; } }
  .card { background:var(--card); border:1px solid #223; border-radius:14px; padding:18px; }
  .card h2 { margin:0 0 12px; font-size:14px; text-transform:uppercase; letter-spacing:1.2px; color:var(--accent); }
  .row { display:flex; align-items:center; justify-content:space-between; padding:9px 0; border-bottom:1px dashed #223; }
  .row:last-child { border-bottom:0; }
  .label { color:var(--muted); font-size:13px; }
  .value { font-size:14px; font-weight:600; text-align:right; }
  .badges { display:flex; flex-wrap:wrap; gap:8px; margin-top:6px; }
  .badge { display:inline-block; padding:5px 10px; border-radius:999px; font-size:12px; font-weight:700; letter-spacing:.4px; }
  .badge.ok { background:rgba(31,191,117,.15); color:var(--ok); border:1px solid rgba(31,191,117,.4); }
  .badge.warn { background:rgba(245,185,66,.15); color:var(--warn); border:1px solid rgba(245,185,66,.4); }
  .badge.bad { background:rgba(255,93,108,.15); color:var(--bad); border:1px solid rgba(255,93,108,.4); }
  .legend { margin-top:18px; color:var(--muted); font-size:12px; text-align:center; }
  .actions { display:flex; align-items:center; gap:12px; margin-top:16px; }
  .btn { background:var(--accent); color:#fff; border:0; border-radius:10px; padding:10px 16px; font-size:14px; font-weight:700; cursor:pointer; }
  .btn:hover { filter:brightness(1.08); }
  .upd-msg { color:var(--muted); font-size:13px; }
  code { background:#0e1530; padding:1px 6px; border-radius:6px; color:#cfe0ff; }
</style>
</head>
<body>
<div class="wrap">
  <header>
    <h1>Predict-A-Trade <span class="sub">XAUUSD · Windows Agent</span></h1>
    <div class="meta">v%s · device <code>%s</code> · uptime %s · role <code>%s</code></div>
  </header>

  <div class="grid">
    %s
  </div>

  <div class="actions">
    <button class="btn" onclick="triggerUpdate()">Check for Update</button>
    <span id="upd-msg" class="upd-msg"></span>
  </div>

  <div class="legend">Auto-refreshing every 5s · page generated %s · open <code>/health</code> for JSON</div>
</div>
<script>
function triggerUpdate(){
  var el=document.getElementById('upd-msg');
  el.textContent='checking…';
  fetch('/api/update',{method:'POST'}).then(function(r){return r.json();}).then(function(d){
    el.textContent=d.message||'done';
  }).catch(function(e){el.textContent='Error: '+e;});
}
</script>
</body>
</html>`
