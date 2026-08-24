# Cross-Market Confluence — Forensic Audit Report

## 1. Existing Capabilities Discovered

| Capability | Location | Status | Wired to Scoring? |
|---|---|---|---|
| DXY Provider | `marketdata/dxy_provider.go` | ✅ Active (Twelve Data API) | ✅ STANDARD_SWING/TREND_SWING mandatory pillar |
| COT Provider | `marketdata/cot_provider.go` | ✅ Active (FMP API) | ✅ cot_etf_flow pillar |
| News Risk Engine | `pkg/news/risk_engine.go` | ✅ Active (FMP calendar) | ✅ NewsGate hard gate |
| Sentiment Engine | `internal/sentiment/engine.go` | ✅ Configured | ✅ SentimentContribution in scorer |
| Macro Health Monitor | `pkg/macro/dxy_cot_health.go` | ✅ Active | ✅ HealthManager |
| Correlation Detector | `internal/ptb/correlation.go` | ✅ Active | ✅ CorrelationEngine in PTB |
| Confluence Scoring | `internal/strategy/confluence.go` | ✅ Active | ✅ All 4 strategies |
| ML Inference Engine | `internal/mlengine/engine.go` | ✅ Active | ✅ ApplyMLAndSentiment |
| Scoring Protection | `pkg/strategy/scorer.go` | ✅ Active | ✅ Bounded contribution |

## 2. What Already Existed (Reused)

- DXY provider with ICE formula computation (6 currency pairs)
- COT provider with net positioning percentile calculation
- News risk engine with FMP economic calendar
- Sentiment engine with Ollama LLM integration
- Correlation engine with rolling Pearson
- Strategy confluence with weighted evidence pillars
- PTB synthesis with regime classification
- Macro health monitoring

## 3. What Was Missing

- No cross-market confluence scoring layer
- No safe-haven regime detector
- No divergence detector for signal vs macro conflicts
- No bounded score adjustment from macro context
- No anti-double-counting for DXY/EURUSD collinearity
- No freshness-based weight decay per driver
- No data-quality states (CONNECTED/DEGRADED/STALE/MISSING)
- No event-risk integration into score adjustment
- No cross-market API endpoint

## 4. What Was Added

- `crossmarket/types.go` — DriverSnapshot, ConfluenceResult, Config, enums
- `crossmarket/engine.go` — Core confluence engine with bounded adjustment
- `crossmarket/detectors.go` — CorrelationDetector, SafeHavenDetector, DivergenceDetector
- `crossmarket/normalizers.go` — NormalizeDXY, NormalizeEURUSD, NormalizeCOT, NormalizeVIX, NormalizeBTC
- `crossmarket/persistence.go` — TimescaleDB persister for 4 hypertables
- `crossmarket/http.go` — HTTP API handlers for cross-market endpoint
- `crossmarket/engine_test.go` — 22 unit tests

## 5. What Was NOT Recreated

- DXY provider — reused existing `marketdata.DXYProvider`
- COT provider — reused existing `marketdata.COTProvider`
- News risk engine — reused existing `news.RiskEngine`
- Sentiment engine — reused existing `sentiment.Engine`
- Correlation engine — new detector added but existing PTB CorrelationEngine preserved
- Strategy scoring — no changes to existing confluence evaluation
- Hard gates — no changes to any risk gate
- Signal pipeline — no changes to signal generation logic

## 6. Data Sources

| Source | Provider | Freshness TTL | Configured? |
|---|---|---|---|
| DXY | Twelve Data API | 5 minutes | ✅ TWELVEDATA_API_KEY set |
| COT | FMP API | 7 days (weekly) | ✅ FMP_API_KEY set |
| EURUSD | Twelve Data API | 5 minutes | ✅ (via DXY components) |
| VIX | Not configured | N/A | ❌ No VIX feed available |
| Real Yields | Not configured | N/A | ❌ No TIPS feed |
| BTC | Not configured | N/A | ❌ No crypto feed |
| Fed Context | Not configured | N/A | ❌ No Fed expectations feed |

## 7. Architecture Decision

The cross-market engine operates in **shadow mode** by default:
- Calculates confluence scores from available drivers
- Does NOT modify production signal scores
- Scores are persisted for offline analysis
- To go active: set `CROSS_MARKET_MODE=active`

## 8. Collinearity Control

DXY and EURUSD are strongly correlated (both represent USD strength).
The engine implements anti-double-counting:
- DXY is the primary USD driver (weight 25)
- EURUSD is a confirmation driver (weight 10, reduced to 40% when DXY present)
- Effective EURUSD weight: 10 × 0.4 = 4.0 (vs DXY 25)

## 9. Scoring Protection

The engine cannot override hard gates or convert terrible signals to good ones:
- MaxBonus = 10.0 (bounded positive adjustment)
- MaxPenalty = -15.0 (bounded negative adjustment)
- Extreme event risk = 0 adjustment (blocks all macro influence)
- Divergence severity reduces adjustment (EXTREME = 10% of base)
- Shadow mode = 0 adjustment (default)

## 10. Remaining Gaps

| Gap | Priority | Requirement |
|---|---|---|
| VIX feed | Medium | Need market data provider for VIX index |
| Real yields feed | Medium | Need Treasury yield data (TIPS or nominal) |
| Fed expectations | Medium | Need Fed funds futures or policy tracker |
| EURUSD normalization | High | Need to extract EURUSD from DXY components |
| Signal pipeline wiring | High | Need to call engine.Evaluate() in processCandle |
| Walk-forward validation | Medium | Need 30+ days of shadow data |
| Ablation testing | Medium | Need per-driver impact analysis |
| Historical replay | Low | Need deterministic replay path |
| Temporal leakage tests | High | Need to verify no look-ahead bias |
| Integration tests | High | Need end-to-end pipeline tests |
