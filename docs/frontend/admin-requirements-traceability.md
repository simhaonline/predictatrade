# Admin Requirements Traceability

| Requirement | Source Document | Backend Module/API | WS/Event | Admin Route | UI Component | Status |
|-------------|----------------|-------------------|----------|-------------|--------------|--------|
| Platform Overview | SOW §1 | GET /admin/overview | — | /admin/dashboard | StatCard grid | ✅ |
| Master Node Status | SOW §8 | GET /api/v1/agents/status, GET /api/v1/health | agent | /admin/dashboard | MasterNodePanel | ✅ |
| Live XAUUSD Price | SOW §10 | GET /api/v1/market/state, WS /ws | market | /admin/dashboard | MarketPanel | ✅ |
| Feed Staleness | SOW §11 | Timestamp diff calc | market | /admin/dashboard | StatusBadge | ✅ |
| Platform Status Strip | SOW §12 | GET /operations/state, /health, /api/v1/health | — | /admin/dashboard | StatusStrip | ✅ |
| Platform Metrics | SOW §13 | GET /admin/overview | — | /admin/dashboard | StatCards | ✅ |
| Signal Pipeline | SOW §14 | GET /api/v1/signals, WS /ws | signal | /admin/dashboard | LiveSignalFeed | ✅ |
| Admin Signal Panel | SOW §15 | GET /api/v1/signals | signal | /admin/signals | DataTable | ✅ |
| Indicator Panel | SOW §16 | GET /api/v1/market/snapshot | — | /admin/indicators | IndicatorGrid | ✅ |
| Strategy Panel | SOW §18 | GET /operations/state, POST /operations/strategy/:id/* | — | /admin/strategies | StrategyCards | ✅ |
| Scoring Board | SOW §19 | GET /api/v1/market/state | — | /admin/scoring-board | ScoreBoard | ✅ |
| Hard Gates | SOW §20 | GET /api/v1/market/snapshot | — | /admin/scoring-board | GateList | ✅ |
| User Onboarding | SOW §21 | GET /admin/users, PATCH /admin/users/:id/status | — | /admin/users | UserTable+Confirm | ✅ |
| User Detail | SOW §22 | GET /admin/users | — | /admin/users | UserDetailDrawer | ✅ |
| License Management | SOW §23 | GET /admin/licenses | — | /admin/licenses | DataTable | ✅ |
| Activations | SOW §24 | GET /devices/sessions, POST /devices/devices/:id/revoke | — | /admin/activations | DataTable+Revoke | ✅ |
| Device Auth | SOW §25 | GET /admin/devices | — | /admin/device-auth | DataTable | ✅ |
| Subscription Mgmt | SOW §26 | GET /admin/subscriptions | — | /admin/subscriptions | DataTable | ✅ |
| Billing | SOW §28 | GET /billing/invoices, GET /admin/commissions, /admin/payouts | — | /admin/billing | TabbedTable | ✅ |
| Payout Mgmt | SOW §29 | GET /admin/payouts, POST /payouts/:id/approve | — | /admin/billing | ApproveBtn | ✅ |
| Referral Mgmt | SOW §30 | GET /admin/commissions, /admin/commissions/summary | — | /admin/referrals | Summary+Tables | ✅ |
| Trading Reports | SOW §31 | GET /admin/overview, /api/v1/agents/status | — | /admin/trading-reports | Overview | ✅ |
| Backtesting | SOW §33 | GET /api/v1/market/state (framework info) | — | /admin/backtesting | FrameworkInfo | ✅ |
| Logs & Audit | SOW §34 | GET /audit | — | /admin/logs | DataTable | ✅ |
| Platform Operations | SOW §35-39 | GET /operations/*, POST /operations/* | — | /admin/operations | ConfirmDialog+Controls | ✅ |
| System Health | SOW §40-43 | GET /admin/health, /health, /api/v1/health | — | /admin/health | ServiceList | ✅ |
| Admin Settings | SOW §44-47 | GET /users/me, PATCH /users/me, POST /auth/mfa/* | — | /admin/settings | TabbedSettings | ✅ |
| Change Password | SOW §45 | POST /auth/mfa/verify (password change flow) | — | /admin/settings | PasswordForm | ✅ |
| MFA | SOW §44 | POST /auth/mfa/setup, /auth/mfa/verify | — | /admin/settings | MFAPanel | ✅ |
| Theme Preferences | SOW §61 | next-themes | — | /admin/settings | ThemeControls | ✅ |
| Footer | SOW §62 | — | — | All admin pages | Footer | ✅ |
