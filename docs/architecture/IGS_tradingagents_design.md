# ADR: Institutional Gold Signal (IGS) + TradingAgents Research Bridge

Status: IMPLEMENTED (shadow) · Version 1.0.0 · 2026-08-30
Source documents: this ADR is self-contained. The legacy `check.md` (Institutional Gold Intelligence) was consolidated into this document and removed from the repo root; `link.md` (TradingAgents) remains the research-bridge reference.

## 1. Purpose

Two upgrades, one chassis:

1. **IGS** — the deterministic Institutional Gold Signal from the consolidated IGS design (this ADR), implemented as a
   composite driver on the existing crossmarket engine pattern.
2. **TradingAgents** — the open-source multi-agent LLM framework, wired as an
   *optional research-plane job* that produces a daily XAUUSD institutional
   bias report. Its output feeds exactly one low-weight IGS component.

## 2. What was reused (per AGENTS.md: reuse before replace)

| Existing | Classification | Role |
|---|---|---|
| `realtime/internal/crossmarket` (Engine, normalizers, persister) | REUSE (pattern) | IGS mirrors its bounded-adjustment/shadow-mode discipline |
| `marketdata.NewTwelveDataProvider` | REUSE | ETF provider uses the same key, HTTP conventions |
| COT + DXY + FRED providers | REUSE | Fan-in supplies 3 of 8 IGS components with zero new I/O |
| `trading.cross_market_*` migrations | REUSE (convention) | 099 migration follows 065 style: hypertables, provenance, quality |
| Ollama sentiment path (0.05 weight) | UNCHANGED | untouched |
| `research/` plane | EXTEND | new `patresearch.ai_research` package |

## 3. IGS design (deterministic core)

`realtime/internal/igs/engine.go`

Components (IGS tier hierarchy → weights):

| Tier | Component | Key | Base weight | Feed status |
|---|---|---|-----|-----|
| S | USD regime (DXY) | `usd_regime` | 20 | ✅ fan-in from crossmarket DXY |
| S | Real yield regime | `real_yield_regime` | 20 | ✅ fan-in from FRED DFII10 |
| S | Central-bank flow | `central_bank_flow` | 15 | ❌ **UNAVAILABLE** — no authoritative feed; never fabricated |
| A | Managed-money positioning (COT) | `cot_positioning` | 12 | ✅ fan-in from FMP COT |
| A | ETF flows (GLD/IAU proxy) | `etf_flows` | 15 | ✅ opt-in (`IGS_ETF_ENABLED=true`), capped 40 impact / 0.4 confidence |
| A | Options/dealer gamma | `options_gamma` | 8 | ❌ unavailable — requires COMEX OI by strike |
| B | Institutional research (LLM) | `institutional_research` | 6 | ✅ opt-in via TradingAgents adapter |
| B | Physical demand (CN/IN) | `physical_demand` | 4 | ❌ unavailable — no feed |

Output: `Composite` with score (−100..+100), classification bands from
this ADR (EXTREME_INSTITUTIONAL_BULLISH … EXTREME_INSTITUTIONAL_BEARISH,
INSUFFICIENT_DATA when <2 components), agreement/conflict, freshness decay,
quality. Missing components are surfaced in `missing_components` — they are
never zero-filled silently.

## 4. TradingAgents integration (Python research plane)

`research/src/patresearch/ai_research/tradingagents_adapter.py`

- **Optional dependency**: guarded import; CI passes without the package
  (tests use stub decision graphs and monkeypatching — no network, no keys).
- **Adapter contract**: `TradingAgentsGraph.propagate()` decision →
  `ResearchReport` (bias, confidence, theses, risks, key drivers) → persisted
  to `trading.ai_research_reports` via the IGS persister.
- **Defensive mapping**: unknown decision shapes degrade to NEUTRAL/REJECTED
  rather than guessing. Failures never raise into the research runner.
- **Honesty**: report provenance marks `deterministic: false`; LLM output is
  `GENERATED` quality, never AUTHORITATIVE; confidence is self-reported and
  explicitly NOT calibrated probability (SOW: raw score ≠ probability).

Runtime (operator opt-in):
```bash
pip install "tradingagents"   # heavy optional extra (LangGraph etc.)
# IGS_AI_RESEARCH_ENABLED=true in realtime env
python -m patresearch.ai_research.run_daily   # scheduled daily job (cron/docker)
```

## 5. Production safety envelope

- IGS default `IGS_ENABLED=false` + `Mode=shadow` → `ScoreAdjustment` is
  hard-zero in every non-active mode (unit-tested).
- In `active` mode the adjustment is bounded (MaxBonus 10 / MaxPenalty −15)
  and applied as an *evidence adjustment* on the existing score — never a
  trade trigger, never a gate override.
- Hot path untouched: components come from background refresh loops;
  `Evaluate()` reads cached state only (same contract as crossmarket).
- LLM bias gets Tier-B weight (6/134) and a 3-day freshness TTL — a stale or
  absent AI report cannot dominate the composite.
- `NO-TRADE` precedence, hard-gate authority, and all SOW safety ordering are
  unchanged; IGS only refines confidence of already-validated candidates.

## 6. Migration

`database/migrations/099_igs_institutional_gold_signal.sql`

- `trading.igs_results` (hypertable) — IGS composites, provenance/quality columns
- `trading.ai_research_reports` — LLM research artifact (UNIQUE run_date+symbol+framework)
- `trading.igs_weight_versions` — auditable weight versioning (like strategy_config_versions)

## 7. Rollback

Set `IGS_ENABLED=false` (or remove) and restart realtime — IGS contributes
nothing when disabled; tables are additive and never read by the signal path.
No data backfill or destructive migration involved.

## 8. Deliberately NOT adopted from TradingAgents

- Its per-trade execution decisioning (LLM approves orders) — SOW forbids LLM
  in the execution authority chain.
- Its equity-ticker data scrapers (yfinance/StockTwits/Reddit) — we keep our
  own provenance-qualified providers.
- Any subscriber-facing probability derived from LLM output — IGS research
  bias is directional context only, calibration remains in `calibration.*`.

## 9. Test evidence

- Go: `go test ./internal/igs/ ./internal/crossmarket/ ./internal/marketdata/ ./internal/strategy/...` — all pass
  (shadow-zero-adjustment, missing-feed honesty, staleness decay, bounded
  adjustment, fan-in mapping, ETF provider).
- Python: `pytest research/tests/ai_research/` → 14 passed, 1 skipped
  (optional-dependency guard works as designed).
- `go build ./... && go vet ./...` clean.