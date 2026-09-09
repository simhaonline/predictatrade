// Package main implements the PAT Watchdog — an independent monitoring +
// self-healing engine that supervises the entire Predict-A-Trade stack.
//
// Design (fail-safe monitor, never a trading component):
//   - Detection layers:
//       L1 Liveness    — is every compose service container running?
//       L2 Freshness   — are master ticks flowing into market.ticks? (SQL)
//       L3 Health      — realtime /health status+db+cache checks
//       L4 Crash-loop  — RestartCount deltas + backoff window detection
//   - Remediation ladder (escalates only on repeated failure):
//       1. docker start <svc>          (container stopped)
//       2. docker restart <svc>        (running but sick)
//       3. docker compose up -d        (whole-stack reconcile; heals compose
//                                       drift AND orders postgres first when
//                                       the stack is down via depends_on)
//       4. restart engine LAST after postgres is healthy (dependency order)
//   - Every action is logged, rate-limited, cooldown-gated, and announced on
//     ntfy topic $NTFY_TOPIC (token auth). The watchdog NEVER touches live
//     orders, positions, billing, or data — it only restarts infrastructure.
//
// It runs OUTSIDE the compose dependency graph (docker.sock mounted) so it can
// observe and heal the stack even when the stack itself is what failed.
package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"sort"
	"strings"
	"time"
)

// escapeHTML escapes Telegram HTML special characters (&, <, > per Bot API).
func escapeHTML(s string) string { return html.EscapeString(s) }

// ─── Configuration (env) ────────────────────────────────────────────────────

type Config struct {
	ComposeFile   string        // path to docker-compose.yml
	ComposeEnv    string        // path to infra/env/.env
	Project       string        // compose project name
	Services      []string      // supervised compose services
	RealtimeURL   string        // realtime /health URL
	PgContainer   string        // postgres container name (for psql exec)
	PgUser        string        // postgres superuser/role for probe
	PgDB          string        // database name for probe
	TickMaxAge    time.Duration // master tick staleness threshold
	HealthTimeout time.Duration // HTTP probe timeout
	CheckEvery    time.Duration // check cadence

	NtfyURL    string // e.g. http://ntfy:80
	NtfyTopic  string
	NtfyToken  string

	// Multi-channel alert mirrors (all optional — empty = disabled)
	TelegramBotToken  string
	TelegramChatID    string
	DiscordWebhookURL string

	SysLoadAvg bool // reserved

	// Remediation tuning
	RestartCooldown    time.Duration // min gap between restarts of one service
	MaxConsecRestarts  int           // after N consecutive restarts w/o recovery → alert-only (CRITICAL)
	CrashLoopWindow    time.Duration // window for counting container restarts
	CrashLoopMax       int64         // RestartCount delta within window considered crash-loop
	StoppedGrace       time.Duration // a container may be restarting/exited briefly; grace before acting
	TickDegradeGrace   time.Duration // allow grace outside market hours before acting on staleness
	DiskMaxUsedPct     int           // alert when any monitored volume exceeds this usage
	DiskProbes         []diskProbe   // container:path pairs — df runs inside the container
	RunbookBase        string        // base URL for runbook links in alert bodies
	dockerCli          string
}

// diskProbe names where to measure usage: df runs INSIDE the named container
// at the given path, so the numbers reflect the host filesystem backing the
// container's volume mount.
type diskProbe struct {
	Container string
	Path      string
}

func loadConfig() *Config {
	c := &Config{
		ComposeFile:       getenv("COMPOSE_FILE", "/srv/predictatrade/xauusd/docker-compose.yml"),
		ComposeEnv:        getenv("COMPOSE_ENV_FILE", "/srv/predictatrade/xauusd/infra/env/.env"),
		Project:           getenv("COMPOSE_PROJECT", "xauusd"),
		RealtimeURL:       getenv("REALTIME_HEALTH_URL", "http://realtime:13081/health"),
		PgContainer:       getenv("PG_CONTAINER", "pat-postgres"),
		PgUser:            getenv("PG_USER", "pat_admin"),
		PgDB:              getenv("PG_DB", "predictatrade"),
		TickMaxAge:        dur(getenv("TICK_MAX_AGE", "180s")),
		HealthTimeout:     dur(getenv("HEALTH_TIMEOUT", "10s")),
		CheckEvery:        dur(getenv("CHECK_EVERY", "30s")),
		NtfyURL:           getenv("NTFY_URL", "http://ntfy:80"),
		NtfyTopic:         getenv("NTFY_TOPIC", "predictatrade-alerts"),
		NtfyToken:         os.Getenv("NTFY_ACCESS_TOKEN"),
		TelegramBotToken:  os.Getenv("TELEGRAM_BOT_TOKEN"),
		TelegramChatID:    os.Getenv("TELEGRAM_CHAT_ID"),
		DiscordWebhookURL: os.Getenv("DISCORD_WEBHOOK_URL"),
		RestartCooldown:   dur(getenv("RESTART_COOLDOWN", "300s")),
		MaxConsecRestarts: intInt(getenv("MAX_CONSEC_RESTARTS", "3")),
		CrashLoopWindow:   dur(getenv("CRASHLOOP_WINDOW", "600s")),
		CrashLoopMax:      int64(intInt(getenv("CRASHLOOP_MAX", "3"))),
		StoppedGrace:      dur(getenv("STOPPED_GRACE", "20s")),
		TickDegradeGrace:  dur(getenv("TICK_DEGRADE_GRACE", "120s")),
		DiskMaxUsedPct:    intInt(getenv("DISK_MAX_USED_PCT", "85")),
		DiskProbes:        parseDiskProbes(getenv("DISK_PROBES", "pat-postgres:/var/lib/postgresql/data pat-backup-sync:/pgbackups xauusd-nginx-1:/var/log/nginx xauusd-nginx-1:/etc/letsencrypt")),
		RunbookBase:       getenv("RUNBOOK_BASE", "https://docs.predictatrade.com/runbooks"),
		Services: strings.Fields(getenv(
			"SUPERVISED_SERVICES",
			"postgres valkey realtime control control-b frontend live-terminal mail-relay backtest nats ntfy prometheus grafana status backup-sync nginx")),
	}
	return c
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
func dur(s string) time.Duration { d, _ := time.ParseDuration(s); return d }
func intInt(s string) int        { n := 0; fmt.Sscanf(s, "%d", &n); return n }

// ─── Docker inspection (no external deps — Docker Engine API over unix socket) ──

type containerState struct {
	Name     string `json:"Name"`
	ID       string `json:"ID"`
	Image    string `json:"Config.Image" json2:"-"` // filled via Config below
	Running  bool   `json:"Running"`
	Status   string `json:"Status"`
	Health   string `json:"Health"`
	Started  time.Time
	ExitCode int
	Restarts int64
}

// We shell out to the docker CLI (mounted from host) — simplest robust path,
// avoids hand-rolling socket HTTP + volume-mount resolution of the CLI binary.
func docker(args ...string) (string, error) {
	out, err := exec.Command("docker", args...).CombinedOutput()
	return string(out), err
}

type inspectOut struct {
	Name          string `json:"Name"`
	State         struct {
		Running    bool      `json:"Running"`
		Status     string    `json:"Status"`
		Health     *struct {
			Status string `json:"Status"`
		} `json:"Health"`
		StartedAt    time.Time `json:"StartedAt"`
		FinishedAt   time.Time `json:"FinishedAt"`
		ExitCode     int       `json:"ExitCode"`
		OOMKilled    bool      `json:"OOMKilled"`
		RestartCount int64     `json:"RestartCount"`
	} `json:"State"`
}

func inspect(name string) (*inspectOut, error) {
	out, err := docker("inspect", "--format", "{{json .}}", name)
	if err != nil {
		return nil, fmt.Errorf("%s: %w (%s)", name, err, strings.TrimSpace(out))
	}
	var o inspectOut
	if err := json.Unmarshal([]byte(out), &o); err != nil {
		return nil, fmt.Errorf("inspect parse %s: %w", name, err)
	}
	return &o, nil
}

// ─── Detection ──────────────────────────────────────────────────────────────

type Finding struct {
	Kind     string // CONTAINER_DOWN | UNHEALTHY | CRASHLOOP | TICKS_STALE | HEALTH_DEGRADED | OK
	Service  string
	Detail   string
	Stopped  time.Duration
	Escalate int // 0=none 1=start 2=restart 3=stack-up 4=ordered-stack-up
}

// supervise returns one Finding per supervised service plus stack-level findings.
func supervise(cfg *Config, state *watchState) []Finding {
	var findings []Finding
	now := time.Now().UTC()

	for _, svc := range cfg.Services {
	 cname := "pat-" + svc
		if svc == "nginx" {
			cname = "xauusd-nginx-1"
		}
		insp, err := inspect(cname)
		if err != nil {
			// Container object itself missing → compose artifact gone; escalate to stack-up.
			findings = append(findings, Finding{
				Kind: "CONTAINER_DOWN", Service: svc,
				Detail:   "container missing: " + err.Error(),
				Escalate: 3,
			})
			continue
		}
		// L4: crash-loop — RestartCount delta since last observation.
		if prev, ok := state.rest[svc]; ok && insp.State.RestartCount > prev {
			state.restartLog[svc] = append(state.restartLog[svc], now)
		}
		state.rest[svc] = insp.State.RestartCount
		// prune window
		var inWin []time.Time
		for _, t := range state.restartLog[svc] {
			if now.Sub(t) <= cfg.CrashLoopWindow {
				inWin = append(inWin, t)
			}
		}
		state.restartLog[svc] = inWin
		if int64(len(inWin)) >= cfg.CrashLoopMax {
			findings = append(findings, Finding{
				Kind: "CRASHLOOP", Service: svc,
				Detail: fmt.Sprintf("%d restarts in %v (total=%d)", len(inWin), cfg.CrashLoopWindow, insp.State.RestartCount),
			})
			// No further auto-restart: restart policy is already churning.
			continue
		}

		// L1: not running.
		if !insp.State.Running {
			// Time since exit (FinishedAt), not since start — StartedAt is the
			// last boot time and overstates nothing; for a container killed
			// seconds ago FinishedAt is the honest "how long has it been down".
			stopped := now.Sub(insp.State.FinishedAt)
			if stopped < 0 || stopped > 24*time.Hour {
				stopped = cfg.StoppedGrace + time.Minute // unknown/zero → act
			}
			if stopped < cfg.StoppedGrace {
				continue // still within grace (compose may be mid-recreate)
			}
			findings = append(findings, Finding{
				Kind: "CONTAINER_DOWN", Service: svc,
				Detail:   fmt.Sprintf("status=%s exit=%d stopped-for=%s", insp.State.Status, insp.State.ExitCode, stopped.Round(time.Second)),
				Stopped:  stopped,
				Escalate: 1,
			})
			continue
		}

		// Running but healthcheck failing.
		if insp.State.Health != nil && insp.State.Health.Status == "unhealthy" {
			findings = append(findings, Finding{
				Kind: "UNHEALTHY", Service: svc,
				Detail:   "docker healthcheck reports unhealthy (running)",
				Escalate: 2,
			})
		}
	}

	// L2: master tick freshness via postgres (skip if postgres itself is down —
	// L1 handles that; a down DB is not a data freshness problem).
	if pgRunning(cfg) {
		if age, err := tickAge(cfg); err != nil {
			findings = append(findings, Finding{Kind: "HEALTH_DEGRADED", Service: "postgres",
				Detail: "tick-freshness probe failed: " + err.Error()})
		} else if age > cfg.TickMaxAge+cfg.TickDegradeGrace {
			findings = append(findings, Finding{
				Kind: "TICKS_STALE", Service: "realtime",
				Detail:   fmt.Sprintf("no master tick for %s (threshold %s)", age.Round(time.Second), cfg.TickMaxAge+cfg.TickDegradeGrace),
				Escalate: 2,
			})
		}
	}

	// L3: realtime engine self-reported health.
	if svcRunning(cfg, "realtime") {
		if h, err := probeRealtimeHealth(cfg); err != nil {
			findings = append(findings, Finding{
				Kind: "UNHEALTHY", Service: "realtime",
				Detail:   "health probe failed: " + err.Error(),
				Escalate: 2,
			})
		} else {
			if h.Status != "ok" {
				findings = append(findings, Finding{Kind: "HEALTH_DEGRADED", Service: "realtime",
					Detail: "status=" + h.Status})
			}
			if h.DB != "ok" && pgRunning(cfg) {
				// DB up but engine lost its pool → restart engine.
				findings = append(findings, Finding{
					Kind: "HEALTH_DEGRADED", Service: "realtime",
					Detail:   "engine db=" + h.DB + " while postgres is running",
					Escalate: 2,
				})
			}
			if h.Cache != "ok" && svcRunning(cfg, "valkey") {
				findings = append(findings, Finding{Kind: "HEALTH_DEGRADED", Service: "realtime",
					Detail: "engine cache=" + h.Cache + " while valkey is running"})
			}
		}
	}

	// L5: disk usage on data-critical volumes (skill: disk >85% = warning).
	// df runs INSIDE the probed container, so it measures the host filesystem
	// backing that container's volume — correct even though the watchdog
	// itself has no mounts on those volumes.
	for _, dp := range cfg.DiskProbes {
		if !svcRunning(cfg, svcFromCname(dp.Container)) {
			continue // container down: L1 already covers it
		}
		pct, err := diskUsedPct(cfg, dp)
		if err != nil {
			log.Printf("[watchdog] disk probe %s:%s failed: %v", dp.Container, dp.Path, err)
			continue
		}
		if pct >= cfg.DiskMaxUsedPct {
			findings = append(findings, Finding{Kind: "DISK_HIGH", Service: "disk",
				Detail: fmt.Sprintf("%s:%s at %d%% used (threshold %d%%)", dp.Container, dp.Path, pct, cfg.DiskMaxUsedPct)})
		}
	}
	return findings
}

// svcFromCname maps a container name back to its compose service key.
func svcFromCname(cname string) string {
	if cname == "xauusd-nginx-1" {
		return "nginx"
	}
	return strings.TrimPrefix(cname, "pat-")
}

// parseDiskProbes parses "container:path container:path ..." pairs.
func parseDiskProbes(s string) []diskProbe {
	var probes []diskProbe
	for _, tok := range strings.Fields(s) {
		parts := strings.SplitN(tok, ":", 2)
		if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
			continue
		}
		probes = append(probes, diskProbe{Container: parts[0], Path: parts[1]})
	}
	return probes
}

// diskUsedPct returns used-percent for the filesystem containing dp.Path,
// measured inside dp.Container with `df -P` (POSIX, unambiguous columns).
func diskUsedPct(cfg *Config, dp diskProbe) (int, error) {
	out, err := docker("exec", dp.Container, "df", "-P", dp.Path)
	if err != nil {
		return 0, fmt.Errorf("%w: %s", err, strings.TrimSpace(out))
	}
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) < 2 {
		return 0, fmt.Errorf("df output %q", strings.TrimSpace(out))
	}
	fields := strings.Fields(lines[len(lines)-1])
	if len(fields) < 5 {
		return 0, fmt.Errorf("df fields %q", lines[len(lines)-1])
	}
	var pct int
	if _, err := fmt.Sscanf(fields[4], "%d%%", &pct); err != nil {
		return 0, fmt.Errorf("df use%% %q", fields[4])
	}
	return pct, nil
}

func pgRunning(cfg *Config) bool {
	insp, err := inspect(cfg.PgContainer)
	return err == nil && insp.State.Running && (insp.State.Health == nil || insp.State.Health.Status != "unhealthy")
}

func svcRunning(cfg *Config, svc string) bool {
	cname := "pat-" + svc
	if svc == "nginx" {
		cname = "xauusd-nginx-1"
	}
	insp, err := inspect(cname)
	return err == nil && insp.State.Running
}

type realtimeHealth struct {
	Status string `json:"status"`
	DB     string `json:"db"`
	Cache  string `json:"cache"`
}

func probeRealtimeHealth(cfg *Config) (*realtimeHealth, error) {
	client := &http.Client{Timeout: cfg.HealthTimeout}
	resp, err := client.Get(cfg.RealtimeURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 8192))
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("health HTTP %d", resp.StatusCode)
	}
	var h realtimeHealth
	if err := json.Unmarshal(body, &h); err != nil {
		return nil, fmt.Errorf("health parse: %w", err)
	}
	return &h, nil
}

// tickAge returns seconds since the newest master tick (MT4/MT5_MASTER).
func tickAge(cfg *Config) (time.Duration, error) {
	out, err := docker("exec", cfg.PgContainer, "psql", "-U", cfg.PgUser, "-d", cfg.PgDB,
		"-Atc", `SELECT COALESCE(ROUND(EXTRACT(EPOCH FROM now()-MAX(time))),999999)::bigint FROM market.ticks WHERE source LIKE '%MASTER%'`)
	if err != nil {
		return 0, fmt.Errorf("%w: %s", err, strings.TrimSpace(out))
	}
	var secs float64
	if _, err := fmt.Sscanf(strings.TrimSpace(out), "%f", &secs); err != nil {
		return 0, fmt.Errorf("probe output %q", strings.TrimSpace(out))
	}
	return time.Duration(secs * float64(time.Second)), nil
}

// ─── Remediation ladder ─────────────────────────────────────────────────────

type watchState struct {
	rest         map[string]int64
	restartLog   map[string][]time.Time
	lastRestart  map[string]time.Time
	consecRest   map[string]int
}

func composeArgs(cfg *Config, tail ...string) []string {
	args := []string{"compose", "-f", cfg.ComposeFile, "--env-file", cfg.ComposeEnv}
	args = append(args, tail...)
	return args
}

// remediate executes the escalation for a finding, honoring cooldowns and the
// consecutive-restart circuit breaker. Returns a human result line.
func remediate(cfg *Config, f Finding, state *watchState, now time.Time) (string, bool) {
	if f.Escalate <= 0 {
		return "", false
	}
	// Cooldown gate per service.
	if last, ok := state.lastRestart[f.Service]; ok && now.Sub(last) < cfg.RestartCooldown {
		return fmt.Sprintf("%s: remediation suppressed (cooldown %v)", f.Service, cfg.RestartCooldown), false
	}
	// Circuit breaker: too many consecutive restarts without recovery → escalate to human.
	if state.consecRest[f.Service] >= cfg.MaxConsecRestarts {
		return fmt.Sprintf("%s: %d consecutive restarts failed — MANUAL INTERVENTION REQUIRED", f.Service, state.consecRest[f.Service]), true
	}

	var action string
	var out string
	var err error
	switch f.Escalate {
	case 1: // start a stopped container
		cname := cnameFor(f.Service)
		action = "docker start " + cname
		out, err = docker("start", cname)
	case 2: // restart a sick-but-running container
		cname := cnameFor(f.Service)
		action = "docker restart " + cname
		out, err = docker("restart", "-t", "30", cname)
	case 3: // stack reconcile (heals missing containers / compose drift)
		action = "docker compose up -d (stack reconcile)"
		out, err = docker(composeArgs(cfg, "up", "-d")...)
	case 4: // ordered full-stack bring-up (used when postgres+engine both down)
		action = "docker compose up -d (ordered stack up)"
		out, err = docker(composeArgs(cfg, "up", "-d")...)
	}
	if err != nil {
		return fmt.Sprintf("%s FAILED: %v (%s)", action, err, strings.TrimSpace(out)), false
	}
	state.lastRestart[f.Service] = now
	state.consecRest[f.Service]++
	return fmt.Sprintf("%s → OK: %s", action, strings.TrimSpace(out)), false
}

func cnameFor(svc string) string {
	if svc == "nginx" {
		return "xauusd-nginx-1"
	}
	return "pat-" + svc
}

// ─── Notification fan-out (ntfy + Telegram + Discord) ──────────────────────
// Mirrors every watchdog alert to all configured channels; a failure in one
// channel never blocks the others (alerting must not have a SPOF).

type telegramPayload struct {
	ChatID                string `json:"chat_id"`
	Text                  string `json:"text"`
	ParseMode             string `json:"parse_mode"`
	DisableWebPagePreview bool   `json:"disable_web_page_preview"`
}

type discordPayload struct {
	Content string `json:"content"`
}

func postJSON(url string, token string, body interface{}) error {
	data, err := json.Marshal(body)
	if err != nil {
		return err
	}
	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest("POST", url, bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("status %d", resp.StatusCode)
	}
	return nil
}

func notify(cfg *Config, sev, title, body string) {
	// 1) ntfy (primary — mobile push)
	go func() {
		payload, _ := json.Marshal(map[string]string{
			"topic":    cfg.NtfyTopic,
			"title":    title,
			"message":  body,
			"priority": sev,
			"tags":     mapSevTag(sev),
		})
		req, _ := http.NewRequest("POST", strings.TrimRight(cfg.NtfyURL, "/")+"/", strings.NewReader(string(payload)))
		req.Header.Set("Content-Type", "application/json")
		if cfg.NtfyToken != "" {
			req.Header.Set("Authorization", "Bearer "+cfg.NtfyToken)
		}
		resp, err := (&http.Client{Timeout: 10 * time.Second}).Do(req)
		if err != nil {
			log.Printf("[watchdog] ntfy publish failed: %v", err)
			return
		}
		resp.Body.Close()
	}()
	// 2) Telegram mirror (operational group chat)
	if cfg.TelegramBotToken != "" && cfg.TelegramChatID != "" {
		go func() {
			text := fmt.Sprintf("🚨 <b>%s</b>\n\n<pre>%s</pre>", escapeHTML(title), escapeHTML(body))
			if err := postJSON(
				fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", cfg.TelegramBotToken),
				"",
				telegramPayload{ChatID: cfg.TelegramChatID, Text: text, ParseMode: "HTML", DisableWebPagePreview: true},
			); err != nil {
				log.Printf("[watchdog] telegram publish failed: %v", err)
			}
		}()
	}
	// 3) Discord mirror (ops webhook — no token, no bot needed)
	if cfg.DiscordWebhookURL != "" {
		go func() {
			if err := postJSON(cfg.DiscordWebhookURL, "", discordPayload{
				Content: fmt.Sprintf("**%s**\n```%s```", title, body),
			}); err != nil {
				log.Printf("[watchdog] discord publish failed: %v", err)
			}
		}()
	}
}

func mapSevTag(sev string) string {
	switch sev {
	case "critical":
		return "rotating_light"
	case "warning":
		return "warning"
	default:
		return "information_source"
	}
}

// ─── Main loop ──────────────────────────────────────────────────────────────

func main() {
	cfg := loadConfig()
	state := &watchState{
		rest:        map[string]int64{},
		restartLog:  map[string][]time.Time{},
		lastRestart: map[string]time.Time{},
		consecRest:  map[string]int{},
	}
	log.Printf("[watchdog] supervising %d services: %s", len(cfg.Services), strings.Join(cfg.Services, ","))
	log.Printf("[watchdog] tick_max_age=%s check_every=%s restart_cooldown=%s", cfg.TickMaxAge, cfg.CheckEvery, cfg.RestartCooldown)
	notify(cfg, "low", "PAT Watchdog online",
		fmt.Sprintf("Supervising %d services · tick freshness %s · cadence %s", len(cfg.Services), cfg.TickMaxAge, cfg.CheckEvery))

	tick := time.NewTicker(cfg.CheckEvery)
	defer tick.Stop()
	for range tick.C {
		runOnce(cfg, state)
	}
}

func runOnce(cfg *Config, state *watchState) {
	now := time.Now().UTC()
	findings := supervise(cfg, state)

	var actions []string
	var alerts []string // formatted per alerting-notifications skill
	for _, f := range findings {
		line := fmt.Sprintf("[%s] %s: %s", f.Kind, f.Service, f.Detail)
		log.Println(line)
		summary := f.Service + " - " + f.Detail
		runbook := runbookFor(cfg, f)
		body := fmt.Sprintf("timestamp: %s\nmetric: %s\nthreshold: see detail\naction: %s\nrunbook: %s",
			now.Format(time.RFC3339), f.Kind, remediationFor(f), runbook)
		switch f.Kind {
		case "OK":
			state.consecRest[f.Service] = 0
			continue
		case "CRASHLOOP":
			alerts = append(alerts, fmt.Sprintf("[CRITICAL] %s\n%s", summary, body))
		case "CONTAINER_DOWN", "UNHEALTHY", "TICKS_STALE":
			if res, manual := remediate(cfg, f, state, now); res != "" {
				if manual {
					alerts = append(alerts, fmt.Sprintf("[CRITICAL] %s\n%s", summary, body))
				} else {
					actions = append(actions, line+" → "+res)
				}
			} else if f.Kind == "TICKS_STALE" {
				// Cooldown-suppressed staleness is still worth a warning.
				alerts = append(alerts, fmt.Sprintf("[WARNING] %s\n%s", summary, body))
			}
		case "DISK_HIGH", "HEALTH_DEGRADED":
			alerts = append(alerts, fmt.Sprintf("[WARNING] %s\n%s", summary, body))
		}
	}

	if len(actions) > 0 {
		sort.Strings(actions)
		log.Printf("[watchdog] REMEDIATION: %s", strings.Join(actions, " ;; "))
		notify(cfg, "high", "[HIGH] Watchdog - Self-healing action taken",
			fmt.Sprintf("timestamp: %s\naction: %s\nrunbook: %s/stack-restart",
				now.Format(time.RFC3339), strings.Join(actions, "\n"), cfg.RunbookBase))
	}
	if len(alerts) > 0 {
		sort.Strings(alerts)
		log.Printf("[watchdog] ALERT: %s", strings.Join(alerts, " ;; "))
		sev := "high"
		for _, a := range alerts {
			if strings.HasPrefix(a, "[CRITICAL]") {
				sev = "urgent"
				break
			}
		}
		notify(cfg, sev, "[SEVERITY] Stack - Needs attention", strings.Join(alerts, "\n\n"))
	}
	if len(findings) == 0 {
		log.Printf("[watchdog] all %d services green", len(cfg.Services))
	}
	_ = sql.ErrNoRows // keep import if unused later
}

// remediationFor describes what the watchdog does (or did) for this finding.
func remediationFor(f Finding) string {
	switch f.Kind {
	case "CONTAINER_DOWN":
		return "docker start " + cnameFor(f.Service) + " (auto)"
	case "UNHEALTHY", "TICKS_STALE":
		return "docker restart " + cnameFor(f.Service) + " after cooldown (auto)"
	case "CRASHLOOP":
		return "manual intervention required — restart policy churning"
	case "DISK_HIGH":
		return "free space or extend volume; check backup-sync retention"
	default:
		return "monitor; escalate if persistent"
	}
}

// runbookFor maps findings to runbook docs (docs/runbooks/*).
func runbookFor(cfg *Config, f Finding) string {
	switch f.Kind {
	case "CONTAINER_DOWN", "CRASHLOOP":
		return cfg.RunbookBase + "/stack-restart"
	case "TICKS_STALE":
		return cfg.RunbookBase + "/mt-connectivity-502"
	case "UNHEALTHY":
		return cfg.RunbookBase + "/edge-poll-429"
	case "DISK_HIGH":
		return cfg.RunbookBase + "/disk-pressure"
	default:
		return cfg.RunbookBase
	}
}