# ALCHEMIST — Institutional Liquidity Strategy (XAUUSD Module)

**Module ID:** `alchemist_xauusd`
**Version:** 1.0.0
**Instrument:** XAUUSD (Gold spot) — module is gold-tuned; do not run FX params on it
**Style:** Intraday, session-based, liquidity/smart-money (ICT-family)
**Analysis TFs:** M → W → D → H4 → H1 → M15 → M5 → M1
**Execution TFs:** M15 / M5 (M1 for surgical refinement)

---

## 1. Philosophy

Align with institutional order flow instead of fighting it.

- Markets are engineered to manipulate liquidity: false breakouts, stop hunts, inducements.
- Retail enters at obvious levels (S/R, trendlines, breakout entries) — exactly where smart money fills the opposite side.
- Edge comes from **top-down bias + liquidity traps + killzone timing**.

> **Core belief:** Price is driven by liquidity grabs. Trade where stops are hunted, not where everyone is entering.

---

## 2. Terminology (data-model vocabulary)

| Term | Definition | Engine representation |
|---|---|---|
| Swing High/Low | 3-candle turning point holding stop liquidity | `swing(idx, price, type)` |
| IRL | Internal Range Liquidity — stops inside the range | `liquidity_pool(kind="IRL")` |
| ERL | External Range Liquidity — stops outside the range; final target | `liquidity_pool(kind="ERL")` |
| Quasimodo (QM) | Liquidity sweep + Break of Structure → marks a *strong* high/low | `structure_event(type="QM")` |
| Order Block (OB) | Candle that initiated the manipulation move; **closing price is the valid level** | `poi(type="OB", level=close)` |
| Breaker Block (BB) | An HTF order block viewed/traded on LTF | `poi(type="BB")` |
| Killzone | The hour surrounding/preceding a major session open (London, NY) | time filter |
| Judas Swing | False move at London open trapping early participants | `manipulation_leg` |
| Range Expansion | Major trend + countertrend align → market leaves consolidation | regime flag |

---

## 3. Top-Down Analysis (the backbone)

1. **Monthly — macro bias**
   - Mark nearest major POIs: OB, FVG, rejection blocks.
   - Monthly levels are magnet zones. Do **not** refine into small zones here.
2. **Weekly — trend confirmation**
   - Use a **line chart** to confirm the most recent BOS.
   - Identify the next *fresh* POI. This sets the week's primary directional bias.
3. **Daily — define the trading range**
   - From the weekly BOS, label daily as bullish or bearish range.
   - Mark strong highs/lows (QM) and internal liquidity. If unclear → drop to H4.
4. **H4 — refinement & scenarios**
   - Build smaller ranges inside the daily bias; these are intraday targets.
   - Asian session typically respects these zones before London manipulation.
5. **H1 → M1 — execution**
   - Asia = accumulation/context. London killzone = Judas swing (manipulation). London/NY = distribution (real move).
   - Trigger = liquidity sweep + BOS near an HTF POI.

> **Golden rule:** Never counter-trade HTF bias. Even a clean LTF setup is skipped if HTF disagrees.

---

## 4. Range behaviour

- Every range begins from a strong high/low (QM).
- Pullbacks = liquidity hunts targeting IRL.
- Expansion = new leg when countertrend aligns with major trend.
- Bullish → support holds, resistance breaks. Bearish → resistance holds, support breaks.
- Ranges give **direction**, not entries. Entries happen inside sessions.

---

## 5. POI validity filter (all five must pass)

1. Fresh / unmitigated (never tapped since formation).
2. Aligned with HTF bias (M, W, D).
3. Defined at the **closing price** of the OB/block — not the wick.
4. Sits **above** liquidity in a bearish trend, **below** liquidity in a bullish trend.
5. Refined to a tight zone (2–5 pips FX equivalent; for gold see §8 tick note).

Never take an unprotected POI: it must be shielded by equal highs/lows, an Asian trap, or engineered stops.

---

## 6. Entry models

### Setup 1 — London Open Trap (primary)
1. Asia consolidates and builds liquidity (equal highs/lows).
2. London killzone prints a Judas swing *against* HTF bias.
3. Sweep taps the HTF POI sitting beyond the Judas extreme.
4. LTF BOS back in the HTF direction confirms.
5. Enter at POI; SL behind the POI/sweep wick.

### Setup 2 — Strong Asian Session
1. Asia itself produces BOS + liquidity grab.
2. Mark the Asian OB/block on H4 or Daily.
3. London killzone retraces into that POI → continuation entry in HTF direction.

### Setup 3 — News Day (NY-driven)
1. On NFP / CPI / FOMC days: Asia consolidates, London builds the trap.
2. **Ignore the London move**; NY session delivers the true expansion.
3. Trade only the NY killzone leg aligned with HTF bias.

---

## 7. Session & killzone clock

| Window | UTC | Dubai (GST, UTC+4) | Role |
|---|---|---|---|
| Asian range build | 00:00–06:00 | 04:00–10:00 | Accumulation, mark high/low |
| London killzone | 06:00–09:00 | 10:00–13:00 | Judas swing + primary entries |
| London distribution | 09:00–11:00 | 13:00–15:00 | Real move / partials |
| NY killzone | 12:00–15:00 | 16:00–19:00 | Setup 3, second leg |
| No-trade | after 17:00 UTC | after 21:00 GST | Thin liquidity |

(DST: London killzone shifts one hour earlier in UTC terms during BST — the config exposes `dst_aware: true`.)

---

## 8. Risk management (gold-specific)

- **Stop loss:** XAUUSD **20–25 pips** (i.e. **$2.00–$2.50** of gold price, since 1 gold "pip" = $0.10). FX pairs max 5 pips — never apply FX stops to gold.
- **Volatility guard:** widen to the greater of 25 pips or 0.35 × ATR(14, M15); skip if spread > 35 points.
- **Position size:** risk % / (SL distance × contract value). Cap 1% per trade, 2% daily, 5% weekly drawdown → flatten and stop.
- **Take profit:**
  1. Partial (50%) at the opposite side of the Asian range.
  2. SL → breakeven after partial.
  3. Final TP at ERL (trading-range high/low).
- Minimum acceptable R:R at entry = 1:3 to ERL.
- Max 2 trades/day, 1 open position at a time.
- **Pay yourself at first target, run the rest with house money.**

---

## 9. Execution flow (algorithmic pseudo-order)

```
1. top_down_analysis(M, W, D, H4)      -> htf_bias, daily_range, poi_list
2. filter_pois(poi_list)               -> valid POIs only (§5)
3. build_asian_range(00:00-06:00 UTC)  -> asia_high, asia_low, eq_highs/lows
4. wait_for_killzone()                 -> London (or NY on news days)
5. detect_judas_swing()                -> sweep of asia extreme against bias
6. confirm_entry()                     -> sweep + BOS + POI mitigation on M5/M15
7. execute(sl_behind_poi, size=risk%)  
8. manage(partial @ asia opposite, BE, final @ ERL)
```

---

## 10. Signal scoring (for Predict-A-Trade confidence output)

Weighted 0–100; publish a signal only at ≥ 70.

| Factor | Weight |
|---|---|
| HTF bias alignment (W + D agree) | 25 |
| POI freshness & quality | 20 |
| Liquidity sweep confirmed | 20 |
| BOS on execution TF after sweep | 15 |
| Inside killzone | 10 |
| R:R ≥ 1:3 to ERL | 10 |
| **Penalties** | |
| High-impact news < 30 min away (non-Setup-3) | −25 |
| Spread > 35 pts / ATR outside band | −20 |
| Counter-HTF | **hard reject** |

---

## 11. Psychology / operator notes

- Patience is the edge — most losses come from entering before manipulation completes.
- Do not rebel against HTF bias; liquidity flow punishes it.
- No setup = no trade. Missing a day costs nothing.

---

## 12. Daily checklist

**Top-down prep**
- [ ] Monthly POI marked (S/R, OB, FVG)
- [ ] Weekly BOS confirmed (line chart)
- [ ] Daily range + bias defined
- [ ] H4 refinement done (zones + liquidity)

**Session prep**
- [ ] Asian range identified
- [ ] Equal highs/lows marked
- [ ] HTF POI near liquidity confirmed
- [ ] Position size calculated

**London execution**
- [ ] Judas swing identified in killzone
- [ ] Entry confirmed (sweep + BOS + POI mitigation)
- [ ] SL set (20–25 pips gold)

**Management**
- [ ] Partial at opposite Asian range
- [ ] SL → breakeven
- [ ] Final TP at ERL

---

## 13. Integration notes for Predict-A-Trade

- Files: `alchemist_xauusd.json` (config), `alchemist_xauusd.py` (reference logic, framework-agnostic).
- Required feed: OHLCV for M1/M5/M15/H1/H4/D1/W1/MN1 + spread + an economic-calendar flag (`high_impact_news_within_minutes`).
- Emitted object: `{symbol, direction, entry, sl, tp1, tp2, confidence, setup_id, rationale[]}` — map this to your existing signal schema.
- Backtest before enabling: minimum 12 months M5 gold data; measure win rate, avg R, max DD, and per-setup breakdown (Setup 1 vs 2 vs 3).

*Not financial advice — validate on demo before live capital.*
