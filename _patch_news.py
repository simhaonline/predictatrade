import re

P = "realtime/internal/gateway/http.go"
s = open(P, encoding="utf-8").read()

# 1) import news package if missing
if "realtime/pkg/news" not in s:
    s = s.replace(
        'import (\n\t"context"',
        'import (\n\t"context"\n\t"github.com/predictatrade/realtime/pkg/news"',
        1,
    )

# 2) add newsEngine field to struct
s = s.replace(
    "\tcrossMarketEngine *crossmarket.Engine\n\tmux",
    "\tcrossMarketEngine *crossmarket.Engine\n NEWSFIELD\n\tmux",
    1,
)
s = s.replace(" NEWSFIELD\n", "\tnewsEngine *news.RiskEngine\n", 1)

# 3) add param to NewHTTPServer signature
s = s.replace(
    "xmEngine *crossmarket.Engine) *HTTPServer {",
    "xmEngine *crossmarket.Engine, newsEngine *news.RiskEngine) *HTTPServer {",
    1,
)
s = s.replace(
    "\t\tcrossMa",
    "\t\tcrossMa",
    1,
)
# assign newsEngine in the struct literal (find the crossMarketEngine assignment line)
s = s.replace(
    "\t\tcrossMarketEngine: xmEngine,\n",
    "\t\tcrossMarketEngine: xmEngine,\n\t\tnewsEngine:        newsEngine,\n",
    1,
)

# 4) add route
s = s.replace(
    'h.mux.HandleFunc("/api/v1/agents/status", h.handleAgentsStatus)',
    'h.mux.HandleFunc("/api/v1/agents/status", h.handleAgentsStatus)\n\th.mux.HandleFunc("/api/v1/news", h.handleNews)',
    1,
)

# 5) add handler method (append before closing of file is fine; insert after handleAgentsStatus)
handler = '''

// handleNews exposes the current news risk level and upcoming economic events
// (sourced from the configured provider, e.g. FMP). This powers the live news
// feed on the trading terminal.
func (h *HTTPServer) handleNews(w http.ResponseWriter, r *http.Request) {
\tw.Header().Set("Content-Type", "application/json")
\tif h.newsEngine == nil {
\t\tjson.NewEncoder(w).Encode(map[string]interface{}{"risk": "NONE", "events": []interface{}{}, "configured": false})
\t\treturn
\t}
\trisk := h.newsEngine.ComputeRisk(time.Now())
\tevents := h.newsEngine.GetEvents()
\tout := map[string]interface{}{
\t\t"risk":       string(risk.Level),
\t\t"reason":     risk.ReasonCode,
\t\t"evidence":   risk.Evidence,
\t\t"computedAt": risk.ComputedAt,
\t\t"configured": h.newsEngine.HasProvider(),
\t\t"events":     events,
\t}
\tjson.NewEncoder(w).Encode(out)
}
'''
i = s.find("func (h *HTTPServer) handleAgentsStatus")
insert_at = s.find("\n}", i)  # end of that function
s = s[:insert_at+2] + handler + s[insert_at+2:]

open(P, "w", encoding="utf-8").write(s)
print("http.go patched for /api/v1/news")

# 6) wire in main.go
M = "realtime/cmd/realtime-engine/main.go"
m = open(M, encoding="utf-8").read()
m = m.replace(
    "httpServer := gateway.NewHTTPServer(wsHub, persister, stateMgr, agentHub, agentProvider, valkeyCache, xmEngine)",
    "httpServer := gateway.NewHTTPServer(wsHub, persister, stateMgr, agentHub, agentProvider, valkeyCache, xmEngine, newsRiskEngine)",
    1,
)
open(M, "w", encoding="utf-8").write(m)
print("main.go wired newsRiskEngine")
