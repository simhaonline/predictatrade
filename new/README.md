# Alchemist XAUUSD — Predict-A-Trade module

Drop-in strategy module. Three files:

| File | Purpose |
|---|---|
| `ALCHEMIST_XAUUSD_SPEC.md` | Full human rulebook (philosophy → setups → risk → checklist) |
| `alchemist_xauusd.json` | Machine-readable config: sessions, POI rules, risk, scoring weights |
| `alchemist_xauusd.py` | Framework-agnostic reference logic → returns a `Signal` or `None` |

## Usage

```python
from strategies.alchemist_xauusd.alchemist_xauusd import AlchemistXAUUSD

strat = AlchemistXAUUSD.from_json("strategies/alchemist_xauusd/alchemist_xauusd.json")

signal = strat.evaluate(
    bars={"M5": df_m5, "M15": df_m15, "H1": df_h1, "H4": df_h4,
          "D1": df_d1, "W1": df_w1, "MN1": df_mn1},   # UTC tz-aware index, ohlcv columns
    now=utc_now,
    equity=account_equity,
    spread_points=current_spread,
    high_impact_news_within_min=minutes_to_next_red_news,   # or None
)
if signal:
    publish(signal.to_dict())
```

`Signal` fields: `symbol, direction, setup_id, entry, sl, tp1, tp2, size_lots, confidence, rationale[], generated_at`.

## Gating order (any failure ⇒ `None`)

1. Outside session / past 17:00 UTC hard stop
2. No agreeing W1 + D1 bias (counter-HTF is a hard reject)
3. Not inside the active killzone (London, or NY on news days)
4. No Asian range
5. No fresh, bias-aligned POI
6. No sweep + BOS + POI mitigation confirmation
7. R:R to ERL < 3.0
8. Confidence score < 70

## Tuning knobs

- `find_order_blocks(... )` displacement threshold `1.8 * avg_body` — lower it (1.3–1.5) if you get too few POIs on gold; raise it in high-volatility regimes.
- `poi.refine_zone_pips` widens/narrows the entry zone (5 pips = $0.50 on gold).
- `risk.sl_atr_multiplier` interacts with the 20–25 pip band: the final stop is clamped inside that band.
- `scoring.publish_threshold` controls signal frequency vs quality.

## Before going live

Backtest ≥12 months of M5 gold data, report win rate / avg R / max DD **per setup id**, then forward-test on demo for one month. Setup 3 (news) should be evaluated separately — it has the widest variance.
