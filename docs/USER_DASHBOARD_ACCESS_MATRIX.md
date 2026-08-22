# User Dashboard Access Matrix — 2026-08-23

| Capability | FREE | STANDARD | PRO | ELITE | Server Enforcement |
|-----------|------|----------|-----|-------|-------------------|
| Live dashboard view | ✓ | ✓ | ✓ | ✓ | Public (live.predictatrade.com) |
| Market data (bid/ask/spread) | ✓ | ✓ | ✓ | ✓ | Go engine /api/v1/market |
| Indicator values (ATR/RSI/ADX) | ✓ | ✓ | ✓ | ✓ | Go engine /api/v1/market/state |
| Signal direction (BUY/SELL/NO-TRADE) | Summary only | ✓ | ✓ | ✓ | NestJS entitlement gate |
| Signal entry price | ✗ | ✓ | ✓ | ✓ | NestJS serializer |
| Signal SL/TP1 | ✗ | ✓ | ✓ | ✓ | NestJS serializer |
| Signal TP2/TP3 | ✗ | ✗ | ✓ | ✓ | NestJS serializer |
| Advanced evidence | ✗ | ✗ | ✓ | ✓ | NestJS serializer |
| Scoring metadata | ✗ | Summary | Full | Full | NestJS serializer |
| Strategy pages | Limited | ✓ | ✓ | ✓ | NestJS RBAC |
| Trade reports | ✗ | ✓ | ✓ | ✓ | NestJS RBAC |
| Backtesting | ✗ | ✗ | ✓ | ✓ | NestJS RBAC |
| Referral system | ✓ | ✓ | ✓ | ✓ | NestJS /referrals |
| Commission earnings | ✓ | ✓ | ✓ | ✓ | NestJS /commissions |
| Admin dashboard | ✗ | ✗ | ✗ | Admin only | NestJS RBAC (ADMIN role) |

## Enforcement Points
1. **Go engine**: Entitlement gates in signal engine (GateEntitlement, GateLicense, GateExecutionPermit)
2. **NestJS**: JWT auth guard on all /api/v1/ routes except /auth/* and /health
3. **Frontend**: Route guards redirect unauthorized users (not security — presentation only)
