# MQL Fix Guide — MT4 & MT5 EA Modifications for Predict-A-Trade

> Companion to `opencode_MASTER_PROMPT.md`. The Go engine computes the signal; the EA is
> responsible for **correct order placement, real partial TP1/2/3, SL/TP sign, magic numbers,
> and reporting the exit back to the engine.** This is where several of your bugs live.

---

## Why the EA side matters (your observed bugs)
From the trade data, the EA (or the signal→order mapping) is responsible for:
- **Wrong-side S/L** (BUY placed with SL *above* entry, SELL with SL *below* entry) — no real protection.
- **TP rarely honored** — only ~9.5% of winners closed at TP; winners manually clipped early.
- **TP1/2/3 opening 3 full positions** instead of scaling ONE position.
- **No `strategy_id`** in the order comment/magic → can't attribute P&L to a plan.

The EA must implement the *execution* half of the fixes. The Go engine supplies the plan
(entry, SL, TP1/2/3, strategy_id, lot, risk$); the EA must place and manage it correctly.

---

## Magic-number / comment convention (enable strategy attribution)
Reserve magic-number ranges per strategy so P&L is attributable:

| Strategy | Magic range (MT4) / magic (MT5) | Comment prefix |
|----------|-------------------------------|----------------|
| STANDARD_SCALPING | 40101–40200 | `PAT-SS:` |
| ULTRA_SCALPING    | 40201–40300 | `PAT-US:` |
| STANDARD_SWING    | 40301–40400 | `PAT-SW:` |
| TREND_SWING       | 40401–40500 | `PAT-TS:` |
| MARNIE_FIB (if used) | 40501–40600 | `PAT-MF:` |

- Include in every order comment: `PAT-<STRATEGY>:<signal_id>` (e.g. `PAT-SW:a1b2c3-...`).
- On close, the EA reports the full magic/comment so the engine's reconciliation maps P&L →
  strategy_id → plan.

---

## CRITICAL — S/L sign enforcement (MT4 & MT5)
Never place a wrong-side stop. Before any `OrderSend`/`Trade.PositionOpen`:

```mql
// direction = ORDER_TYPE_BUY or ORDER_TYPE_SELL, from the signal
bool isBuy   = (direction == ORDER_TYPE_BUY);
double entry = OrderEntry;
double sl    = OrderSL;
double tp    = OrderTP;

// --- Validations (fail-closed) ---
bool slWrong = isBuy ? (sl >= entry) : (sl <= entry);   // BUY: SL must be BELOW; SELL: ABOVE
if (slWrong) {
    Print("REJECTED wrong-side SL: dir=", (string)direction,
          " entry=", DoubleToString(entry,_Digits),
          " sl=", DoubleToString(sl,_Digits));
    return;   // do NOT place; log reason 'wrong_side_sl'
}
```

### MT5 (CTrade) example
```mql5
#include <Trade\Trade.mqh>
CTrade trade;
trade.SetExpertMagicNumber(magic);

if (isBuy)
    ok = trade.Buy(lot, _Symbol, entry, sl, tp, comment);
else
    ok = trade.Sell(lot, _Symbol, entry, sl, tp, comment);
```

### MT4 (OrderSend) example
```mql4
int cmd = isBuy ? OP_BUY : OP_SELL;
int ticket = OrderSend(_Symbol, cmd, lot, price, slippage, sl, tp, comment, magic, 0, clr);
```

---

## Real partial TP1/2/3 — scale ONE position (not 3 tickets)

**Correct model:** open ONE position of total lot. On TP1 close ~1/3 and move SL to breakeven.
On TP2 close another 1/3. On TP3 close the remainder at full target.

### MT5 — use a ticket tracker with three TP levels
```mql5
// Entry:
double entryLot = totalLot / 3;  // e.g. 0.03 -> three 0.01 closes
// Place TP1, TP2, TP3 as separate pending/limit closes, or manage via OnTradeTransaction:
if (PositionGetInteger(POSITION_TYPE) == POSITION_TYPE_BUY) {
    // set TP1
    if (!Trade.PositionModify(ticket, slBreakeven, tp1)) Print("TP1 set fail: ", Trade.ResultRetcodeDescription());
}
```
- **Do NOT open 3 market orders.** If you must use 3 closes, they must each reduce the SAME
  position (partial close), never 3 independent full positions.
- After TP1 fill → modify SL to breakeven (entry ± spread). After TP2 → trail remainder.
- Log each partial fill with its reason (`tp1`, `tp2`, `tp3`) back to the engine.

### MT4 partial close
```mql4
// After TP1 hit, close 1/3 then modify SL to breakeven:
if (OrderSelect(ticket, SELECT_BY_TICKET)) {
    double closeLots = OrderLots() / 3;
    OrderClose(ticket, closeLots, closePrice, slippage, clr);
    OrderModify(ticket, OrderOpenPrice(), breakeven, OrderTakeProfit(), 0, clr); // move SL to breakeven
}
```

> Rule: **one position in, partial closes out.** If the broker/EA cannot do partial closes,
> then open ONE position sized at totalLot and rely on trailing SL — never 3 full tickets.

---

## Risk caps on the EA side (belt-and-suspenders to the Go engine)

The EA must refuse an order that violates risk even if the engine sent it (fail-closed):

```mql
double equity   = AccountInfoDouble(ACCOUNT_EQUITY);
double riskPct  = MaxRiskPct();            // from config, e.g. 0.01 (1%)
double stopPts  = MathAbs(entry - sl) / _Point;
double tickVal  = SymbolInfoDouble(_Symbol, SYMBOL_TRADE_TICK_VALUE);
double risk$    = tickVal * stopPts * lot * (SYMBOL_TRADE_TICK_SIZE == 0 ? 1 : ...); // broker-specific

if (risk$ > equity * riskPct) {
    Print("REJECTED risk_oversize: risk$=", risk$, " cap=", equity*riskPct);
    return;
}
```

Also enforce on the EA:
- **Max same-direction positions** (count open tickets with same magic range + direction; reject if ≥ 1 more allowed).
- **Max concurrent positions** (default 2 total).
- **Martingale ban:** reject if `lot > baseLot` for the strategy (config `MaxLotRatioVsBase=1.0`).

---

## Reporting exits back to the engine (Bug 5 — outcome reconciliation)

The EA must record and send the exit reason so the Go engine can write expected-vs-actual:

| Exit | Detect | Report reason |
|------|--------|---------------|
| TP1 | Price touched TP1 | `tp1` |
| TP2 | Price touched TP2 | `tp2` |
| TP3 | Price touched TP3 | `tp3` |
| SL | Price touched SL | `sl` |
| Expiry | TTL exceeded | `expiry` |
| Manual/other | Trader or external close | `manual` |

Send back: `signal_id`, `strategy_id`, `magic`, `exit_reason`, `entry`, `exit`, `lot`,
`realized_pnl`, `expected_pnl`, `sl_correct` (bool). The engine stores this in the
reconciliation table (ADDENDUM 4 / Bug 5).

---

## MT4 vs MT5 API differences to respect

| Concern | MT4 (MQL4) | MT5 (MQL5) |
|---------|------------|------------|
| Order placement | `OrderSend()` (market orders) | `CTrade::Buy/Sell` or `OrderSend` request |
| Order management | `OrderModify()` | `CTrade::PositionModify` |
| Partial close | `OrderClose()` by lots (needs real CLOSE_PARTIAL support) | `CTrade::PositionClosePartial` / by volume |
| Pending SL/TP | Order-level SL/TP | Position-level SL/TP (no per-order SL/TP once open) |
| Ticket identity | Order ticket | Position ticket (`POSITION_TICKET`) |
| History | `OrderSelect(SELECT_BY_POS,MODE_HISTORY)` | `HistoryDealGet` / `PositionGet` |
| Tick value | `MarketInfo(SYMBOL_TRADE_TICK_VALUE)` | `SymbolInfoDouble(SYMBOL_TRADE_TICK_VALUE)` |

> **Key MT5 gotcha:** SL/TP live on the **position**, not per order. When you partial-close,
> the remaining position keeps the position-level SL/TP — modify it after each partial fill.
> **Key MT4 gotcha:** MT4 partial closes may not be supported by all brokers; verify with the
> broker before relying on TP1/2/3 scaling — otherwise fall back to trailing SL on one position.

---

## Checklist (MQL files to modify)
> Status after v1.08 execution-layer rework (`mql/mt4/PredictATrade_MT4.mq4`, `mql/mt5/PredictATrade_MT5.mq5`):

- [x] Order ticket: enforce correct SL side (reject wrong-side) — MT4 & MT5.
      `PAT_ValidateLevels()`: BUY requires SL < entry AND TP > entry; SELL requires
      SL > entry AND TP < entry. Runs BEFORE every send path; on violation it
      Prints a detailed reason and ABORTS — never clamps.
- [x] Order ticket: set SL + TP correctly for the direction.
      Order opens with signal SL and TP3 (final target) as broker-side protection;
      TP1/TP2 are EA-managed partials.
- [x] Implement partial TP1/2/3 scaling on ONE position (not 3 full orders).
      ONE position per signal with total lot; stage 0→1→2 tracked per magic via
      GlobalVariables (`PAT_M<magic>_STAGE`), survives EA reload.
- [x] Move SL to breakeven after TP1; trail after TP2.
      Breakeven = entry ± spread (cost-aware), applied as maintenance from stage 1
      (covers MT4 remainder tickets). Stage ≥ 2 trails by ATR × TrailingATRMult,
      monotonic, respecting stop/freeze levels.
- [x] Magic-number ranges per strategy + `PAT-<STRAT>:<id>` comment.
      Bases 40101/40201/40301/40401/40501 (+ offset within each 100-range);
      comment prefixes `PAT-SS:` / `PAT-US:` / `PAT-SW:` / `PAT-TS:` / `PAT-MF:`
      with signal_id (truncated to fit terminal comment limits).
- [x] Risk-side rejects: risk_oversize, position caps, martingale ban.
      `PAT_PreTradeGate()`: risk$ vs MaxRiskPct×equity (default 1.5%), lot vs
      baseLot×MaxLotRatioVsBase (default 1.0), max 1 same-direction / 2 total PAT
      positions counted by magic range, margin vs MaxMarginUsagePct (default 30%)
      via AccountFreeMarginCheck (MT4) / OrderCalcMargin (MT5).
- [x] Exit-reason reporting (tp1/tp2/tp3/sl/expiry/manual) back to the Go engine.
      `TRADE_RESULT|{...}` messages appended to `PAT_ticks.txt` (the same IPC file
      channel the agent reads): fields type, signal_id, strategy_id, magic, ticket,
      exit_reason, entry, exit, lot, realized_pnl, sl_correct. A compatible
      `CLOSE_ACK|{...}` is emitted alongside because the agent currently parses
      CLOSE_ACK (windows-agent/internal/pipe.go) but has NO case for TRADE_RESULT
      yet — add one to forward TRADE_RESULT to the RT server.
      Detection: deal/order-history polling classified by close-price proximity to
      TP1/TP2/TP3/SL, with explicit forced reasons for EA-initiated closes
      (MAX_HOLD_TIME→expiry, swap/emergency/watchdog/slippage→manual).
      NOTE: the Windows Agent truncation bug was real on MT5 only — `PAT_Append`
      used FILE_WRITE (truncates). FIXED to FILE_READ|FILE_WRITE + FileSeek(SEEK_END).
- [x] Handle MT5 position-level vs MT4 order-level SL/TP correctly.
      MT5: CTrade::PositionClosePartial keeps the same position ticket; state keyed
      by magic. MT4: partial close splits the order into NEW tickets — reporting is
      done exclusively by history-polling closed chunks (one report per chunk),
      never inline, so nothing double-counts.
- [x] Compile clean in MT4 (MQL4) and MT5 (MQL5) with `#property strict`.
      Both files carry `#property strict`; standard well-known API signatures only.
      NOT compiler-verified here (no MetaEditor in this environment) — compile in
      MetaEditor before deploying (pass criteria below).

Additional v1.08 changes:
- Pre-trade spread gate `PAT_CheckSpread()` wired into both Execute paths (was dead code).
- Entry-drift gate (MaxEntryDriftPoints, default 50 pts) and signal-TTL gate
  (server `ExpiresAt` field parsed when present, else MaxSignalAgeSeconds default 300).
- Watchdog OnTimer(15s): missing-SL restore/close (`OnMissingSL`=RESTORE|CLOSE,
  default CLOSE fail-closed; restored SL is side-validated first) and equity floor
  (MinEquityFloorPct default 40% of day-start balance → close ALL PAT positions and
  latch halt via `PAT_EQUITY_HALT` GV until manual ReEnableAfterHalt=true).
- Removed the MT5 pending-LIMIT order path (UsePendingLimit): async fills could not be
  level/state-validated at fill time — market orders with immediate validation are the
  fail-closed choice. Removed superseded inputs CloseAtTP2 / BreakEvenTriggerR /
  TP1ClosePercent(50/30/20 split); new TP1ClosePct=33.33 / TP2ClosePct=33.33 (~1/3 each).

> After these are done, re-run the master prompt's `go test ./...` and a forward-test on the
> demo account. The pass criteria are: no wrong-side SL in any filled order, TP1/2/3 scaling
> works, strategy_id on every order, and reconciliation shows expected vs actual per trade.
