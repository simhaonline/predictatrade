# 24 — User Dashboard Audit

Pages: /dashboard/live (command center), signals, strategies, billing, referrals, trading-reports, backtest, mt4-mt5-client, settings(+accessibility).

## Data integrity findings

| ID | Sev | Finding |
|---|---|---|
| 24-1 | P1 | **Hardcoded Bid/Ask/Spread `2500/.00/.50`** rendered pre-tick with no connecting/stale state (`live-dashboard.tsx:44-58`) — violates §74. |
| 24-2 | P1 | Guest `/preview` renders the FULL live dashboard incl. SignalPipeline+GrowthPanel behind a CSS blur overlay — data already in DOM/network for anonymous visitors (`guest-preview-gate.tsx:63-109`). |
| 24-3 | P1 | Fabricated Hit-Rate/Accuracy/Avg-R performance metrics estimated client-side from conviction, rendered as real (see 02-3). |
| 24-4 | P2 | Client recomputes R:R, trend (EMA50>EMA200), MTF consensus thresholds — presentation-plane authority violations despite footer claiming otherwise. |
| 24-5 | P2 | WS-derived signal rows stub TP2/TP3/RR/Regime as "0"/"" instead of labeled-missing. |
| 24-6 | PASS | NO-TRADE honesty ("Market is quiet — this is correct behavior"), probability "Pending" state, DataTable error/retry states, admin DEGRADED/OFFLINE feed labels. |

## Entitlement UX

Strategy page renders server entitlements read-only ✅. Billing page drives `POST /subscriptions` (which can only create INCOMPLETE). Restricted-content rule violated by 24-2 (blur-after-delivery).

## Schema reconciliation matrix (frontend expectation vs Go payload)

| Field | Go API | Frontend | Status |
|---|---|---|---|
| ID/Direction/RawScore/CalibratedProbability | PascalCase strings | hand-typed PascalCase | MATCH |
| EntryPrice/StopLoss/TP1-3/GrossRR* | present | present (RR also recomputed) | MATCH + violation |
| Regime/Session/NewsRisk/TTL/ExpiresAt | present | present (some variants omit) | PARTIAL — 4 divergent hand-written types exist; no generated types for Go plane |
| Evidence[]/ReasonCodes[] | present | rendered | MATCH |
| sequence/stream envelope fields | sent | **dropped by normalizer** | GAP |

## Theme/responsive/a11y

Light/dark token system complete; reduced-motion is manual-only (no prefers-reduced-motion media query); high-contrast/font-scale toggles present; **no fullscreen/4K mode anywhere; no price chart in user command center** (cards only) — SOW UI gaps (P2).
