#!/usr/bin/env python3
"""
pat-monitor — production watchdog for the Predict-A-Trade realtime engine.

Runs inside its own container and polls the engine's /health and /metrics
endpoints. It detects the exact failure modes operators reported:

  * Master (data) node offline  -> server-authoritative price/candles gone
  * Price feed stale            -> last_market_data_at too old (wrong prices possible)
  * Client agents all down      -> subscribers receive no signals
  * Signals delivered but UNACKed by client  (EXECUTION_ACK never arrived)
  * Signals acked but never filled
  * Engine HALTED / DB / cache down

Alerts are raised via ntfy only on STATE CHANGE (no spam): a "PROBLEM" message
when a check goes bad, and a "RECOVERED" message when it clears. Everything is
also logged to stdout for the container log / Loki.

Designed to run forever; crashes self-restart via `restart: always`.
"""

import json
import os
import re
import sys
import time
import urllib.error
import urllib.request

# ---- config (all optional; sane defaults for the docker-compose network) ----
ENGINE_HEALTH_URL = os.environ.get("ENGINE_HEALTH_URL", "http://realtime:13081/health")
ENGINE_METRICS_URL = os.environ.get("ENGINE_METRICS_URL", "http://realtime:13081/metrics")
NTFY_URL = os.environ.get("NTFY_URL", "http://ntfy:80")
NTFY_TOPIC = os.environ.get("NTFY_TOPIC", "predictatrade-alerts")
NTFY_TOKEN = os.environ.get("NTFY_TOKEN", "")
CHECK_INTERVAL = int(os.environ.get("CHECK_INTERVAL_SEC", "30"))
PRICE_STALE_SEC = int(os.environ.get("PRICE_STALE_SEC", "120"))
DATA_NODE_MIN = int(os.environ.get("DATA_NODE_MIN", "1"))
AGENTS_MIN = int(os.environ.get("AGENTS_MIN", "1"))
ACK_TIMEOUT_ALERT = os.environ.get("ACK_TIMEOUT_ALERT", "true").lower() == "true"
FILL_TIMEOUT_ALERT = os.environ.get("FILL_TIMEOUT_ALERT", "true").lower() == "true"


def log(msg):
    ts = time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime())
    print(f"[{ts}] {msg}", flush=True)


def http_get_json(url):
    req = urllib.request.Request(url, headers={"Accept": "application/json"})
    with urllib.request.urlopen(req, timeout=10) as resp:
        return json.loads(resp.read().decode())


def http_get_text(url):
    req = urllib.request.Request(url)
    with urllib.request.urlopen(req, timeout=10) as resp:
        return resp.read().decode()


def parse_gauge(metrics_text, name):
    """Return the first numeric value for a Prometheus gauge (ignores labels)."""
    # pattern: name{...} value  OR  name value
    pat = re.compile(r"^" + re.escape(name) + r"(?:\{[^}]*\})?\s+([0-9eE+\-.]+)", re.MULTILINE)
    m = pat.search(metrics_text)
    if not m:
        return None
    try:
        return float(m.group(1))
    except ValueError:
        return None


def send_alert(title, message, priority="5", tags="warning"):
    url = f"{NTFY_URL.rstrip('/')}/{NTFY_TOPIC}"
    headers = {
        "Title": title,
        "Tags": tags,
        "Priority": priority,
        "Content-Type": "text/plain",
    }
    if NTFY_TOKEN:
        headers["Authorization"] = f"Bearer {NTFY_TOKEN}"
    try:
        req = urllib.request.Request(url, data=message.encode("utf-8"), headers=headers, method="POST")
        urllib.request.urlopen(req, timeout=10)
        log(f"ntfy sent: {title}")
    except Exception as e:  # never let alerting failures crash the monitor
        log(f"WARN: ntfy send failed: {e}")


class Checker:
    """Tracks per-check alert state so we only alert on transitions."""

    def __init__(self):
        self.states = {}  # name -> bool (True = currently alerting)

    def evaluate(self, name, is_bad, problem_msg, recover_msg=None, priority="5", tags="warning"):
        prev = self.states.get(name, False)
        if is_bad and not prev:
            send_alert(f"[PAT] {name}", problem_msg, priority=priority, tags=tags)
            self.states[name] = True
        elif not is_bad and prev:
            send_alert(f"[PAT] {name} RECOVERED", recover_msg or f"{name} is healthy again.",
                       priority="3", tags="white_check_mark")
            self.states[name] = False
        elif is_bad:
            # stays bad; re-send every N cycles? Keep quiet to avoid spam.
            pass


def is_market_open_now():
    """Cheap UTC weekday check: XAUUSD trades Sun 22:00 UTC -> Fri 22:00 UTC.
    Used only to silence 'price stale' alerts during the weekend dead window."""
    now = time.gmtime()
    wd = now.tm_wday  # Mon=0 .. Sun=6
    # Fri 22:00 UTC close -> Sun 22:00 UTC open
    if wd == 4:  # Friday
        return now.tm_hour < 22
    if wd == 5:  # Saturday -> closed
        return False
    if wd == 6:  # Sunday
        return now.tm_hour >= 22
    return True


def parse_rfc3339(s):
    if not s:
        return None
    s = s.replace("Z", "+00:00")
    try:
        import datetime
        return datetime.datetime.fromisoformat(s)
    except Exception:
        return None


def run_once(checker):
    # 1) Engine reachable?
    try:
        health = http_get_json(ENGINE_HEALTH_URL)
    except Exception as e:
        checker.evaluate(
            "ENGINE_UNREACHABLE", True,
            f"Monitor cannot reach engine /health at {ENGINE_HEALTH_URL}: {e}\n"
            f"All signal/price delivery may be DOWN.",
            priority="5", tags="rotating_light")
        return
    checker.evaluate("ENGINE_UNREACHABLE", False, "Engine reachable again.")

    # 2) HALTED?
    halt = (health.get("emergency_halt") or {}).get("active", False)
    checker.evaluate(
        "ENGINE_HALTED", halt,
        f"Engine EMERGENCY HALT is ACTIVE (reason={ (health.get('emergency_halt') or {}).get('reason') }).\n"
        f"Signal generation/delivery is suspended. Investigate before clearing.",
        priority="5", tags="stop_sign")

    # 3) DB / cache
    checker.evaluate("DB_DOWN", health.get("db") in (None, "down", "not_configured", ""),
                     f"Database health degraded: {health.get('db')}")
    checker.evaluate("CACHE_DOWN", health.get("cache") in ("down",),
                     f"Valkey/cache health degraded: {health.get('cache')}")

    # 4) Master (data) node presence + price freshness
    data_nodes = int(health.get("data_node_count", 0) or 0)
    checker.evaluate(
        "MASTER_DATA_NODE_DOWN", data_nodes < DATA_NODE_MIN,
        f"Master (data) node count={data_nodes} (< {DATA_NODE_MIN}).\n"
        f"Server-authoritative price/candle feed is OFFLINE. Signals will be stale or wrong.",
        priority="5", tags="rotating_light")

    last_data = parse_rfc3339(health.get("last_market_data_at", ""))
    if last_data is not None and is_market_open_now():
        age = (time.time() - last_data.timestamp())
        checker.evaluate(
            "PRICE_FEED_STALE", age > PRICE_STALE_SEC,
            f"Price feed STALE: last market data {int(age)}s ago (> {PRICE_STALE_SEC}s).\n"
            f"Master node may have disconnected. Clients may receive wrong Entry/SL/TP.",
            priority="5", tags="warning")
    else:
        checker.evaluate("PRICE_FEED_STALE", False, "Price feed fresh again.")

    # 5) Client agents
    agents = int(health.get("agents", 0) or 0)
    checker.evaluate(
        "AGENTS_DOWN", agents < AGENTS_MIN,
        f"Connected client agents={agents} (< {AGENTS_MIN}).\n"
        f"Subscribers are receiving NO signals for paid plans.",
        priority="4", tags="warning")

    # 6) Reconciliation gauges from /metrics
    try:
        metrics = http_get_text(ENGINE_METRICS_URL)
        if ACK_TIMEOUT_ALERT:
            acks = parse_gauge(metrics, "pat_reconciliation_acks_timeout")
            if acks is not None:
                checker.evaluate(
                    "SIGNAL_ACK_TIMEOUT", acks > 0,
                    f"{int(acks)} signal(s) delivered to client agent but NOT acknowledged "
                    f"(EXECUTION_ACK missing) for >5m.\n"
                    f"Client EA may not be executing; subscribers lose paid trades.",
                    priority="5", tags="rotating_light")
        if FILL_TIMEOUT_ALERT:
            fills = parse_gauge(metrics, "pat_reconciliation_fills_timeout")
            if fills is not None:
                checker.evaluate(
                    "SIGNAL_FILL_TIMEOUT", fills > 0,
                    f"{int(fills)} signal(s) acknowledged but NO fill reported for >15m.\n"
                    f"Check client MT terminal connection / execution permissions.",
                    priority="4", tags="warning")
    except Exception as e:
        log(f"WARN: metrics fetch failed: {e}")


def main():
    log("pat-monitor starting; engine=%s ntfy=%s/%s" % (ENGINE_HEALTH_URL, NTFY_URL, NTFY_TOPIC))
    checker = Checker()
    # Prime states without spamming: do an initial silent evaluation by emitting
    # recoveries for anything already healthy. We simply run; first bad -> alert.
    while True:
        try:
            run_once(checker)
        except Exception as e:
            log(f"ERROR in run_once: {e}")
        time.sleep(CHECK_INTERVAL)


if __name__ == "__main__":
    try:
        main()
    except KeyboardInterrupt:
        sys.exit(0)
