# Predict-A-Trade — Database Traceability Matrix

**Generated:** 2026-08-18  
**PostgreSQL:** 17.11  
**pgvector:** 0.8.6  
**TimescaleDB:** NOT INSTALLED (conditional migrations ready)  
**Total Tables:** 137 across 14 schemas

| Domain | Required Data | Existing Table | Added/Modified | PK | Important FK | Indexes | Retention | Backup | Status |
|--------|--------------|----------------|----------------|-----|-------------|---------|-----------|--------|--------|
| User | users | iam.users | soft-delete cols | id | org_id | idx_users_active | None | ✅ | PASS |
| User | user_sessions | iam.sessions | token_family | id | user_id | idx_sessions_refresh_hash | None | ✅ | PASS |
| User | login_history | iam.login_events | - | id | user_id | idx_login_events_user_time (NEW) | None | ✅ | PASS |
| User | MFA | iam.mfa_methods | - | id | user_id | - | None | ✅ | PASS |
| Admin/RBAC | roles | iam.roles | - | id | - | - | None | ✅ | PASS |
| Admin/RBAC | permissions | iam.permissions | - | id | - | - | None | ✅ | PASS |
| Admin/RBAC | role_permissions | iam.role_permissions | - | id | role_id, perm_id | - | None | ✅ | PASS |
| Admin/RBAC | memberships | iam.memberships | - | id | user_id, org_id, role_id | - | None | ✅ | PASS |
| Audit | audit_events | audit.audit_events | IMMUTABLE triggers | id | - | idx_audit_events_actor_time (NEW) | None | ✅ | PASS |
| Security | security_events | audit.security_events | NEW | id | - | 4 new indexes | None | ✅ | PASS |
| Market Data | ticks | market.ticks | chk constraints | time,symbol,source | - | idx_ticks_symbol_time (NEW) | 90d (TS only) | ✅ | PASS |
| Market Data | candles | market.candles | chk constraints | time,symbol,tf,source | - | - | None | ✅ | PASS |
| Market Data | market_states | market.market_states | - | id | - | - | 365d (TS only) | ✅ | PASS |
| Indicators | indicator_history | trading.indicator_history | NEW | bigserial | - | 2 indexes | 14d compress (TS) | ✅ | PASS |
| Structure | structure_events | trading.structure_events | +6 columns | id | - | - | None | ✅ | PASS |
| Structure | swing/Fibonacci | trading.structure_events | swing_type, fib_anchor | - | - | - | None | ✅ | PASS |
| Regime | regime_history | trading.regime_history | NEW | bigserial | - | 1 index | None (TS hypertable) | ✅ | PASS |
| Strategy | strategy_definitions | trading.strategy_definitions | - | id | - | - | None | ✅ | PASS |
| Strategy | strategy_config_versions | trading.strategy_config_versions | NEW | id | - | 1 index | None | ✅ | PASS |
| Strategy | strategy_evaluations | trading.strategy_evaluations | NEW | id | - | 2 indexes | None (TS hypertable) | ✅ | PASS |
| Candidates | signal_candidates | trading.signal_candidates | NEW | id | signal_id | 4 indexes | None | ✅ | PASS |
| Rejections | signal_rejections | trading.signal_rejections | NEW | id | candidate_id | 3 indexes | None | ✅ | PASS |
| Signals | signals | trading.signals | +candidate_id | id | candidate_id | idx_signals_symbol_strategy (NEW) | None | ✅ | PASS |
| Signals | signal_events | trading.signal_events | - | id | signal_id | - | None | ✅ | PASS |
| Delivery | signal_deliveries | trading.signal_deliveries | - | id | signal_id | - | None | ✅ | PASS |
| Delivery | delivery_receipts | trading.signal_delivery_receipts | - | id | - | - | None | ✅ | PASS |
| Execution | execution_commands | trading.execution_commands | - | id | - | - | None | ✅ | PASS |
| Execution | execution_events | trading.execution_events | - | id | - | - | None | ✅ | PASS |
| Execution | trades | trading.trades | - | id | position_id | - | None | ✅ | PASS |
| Positions | positions | trading.positions | chk lot_positive | id | - | - | None | ✅ | PASS |
| Risk | risk_decisions | trading.risk_decisions | +4 columns | id | signal_id | idx_risk_decisions_gate (NEW) | None | ✅ | PASS |
| Risk | risk_config_versions | trading.risk_config_versions | NEW | id | - | 1 index | None | ✅ | PASS |
| Risk | risk_profiles | trading.risk_profiles | - | id | - | - | None | ✅ | PASS |
| Risk | gate_policy_versions | trading.gate_policy_versions | - | id | - | - | None | ✅ | PASS |
| Brokers | broker_profiles | market.broker_execution_profiles | soft-delete | id | - | - | None | ✅ | PASS |
| MT4/MT5 | mt_accounts | licensing.mt_accounts | - | id | - | - | None | ✅ | PASS |
| MT4/MT5 | mt_connections | licensing.mt_connections | - | id | - | - | None | ✅ | PASS |
| Win Agents | devices | licensing.devices | soft-delete | id | user_id | - | None | ✅ | PASS |
| Win Agents | device_activations | licensing.device_activations | - | id | license, device | - | None | ✅ | PASS |
| License | licenses | licensing.licenses | - | id | user_id | idx_licenses_user_status (NEW) | None | ✅ | PASS |
| License | license_events | licensing.license_events | - | id | license_id | - | None | ✅ | PASS |
| License | activations | licensing.activations | - | id | license_id | - | None | ✅ | PASS |
| Subscription | plans | control.plans | - | id | - | - | None | ✅ | PASS |
| Subscription | plan_versions | control.plan_versions | - | id | plan_id | - | None | ✅ | PASS |
| Subscription | subscriptions | billing.subscriptions | - | id | user_id, plan_id | idx_subscriptions_user (NEW) | None | ✅ | PASS |
| Subscription | subscription_events | billing.subscription_events | - | id | subscription_id | - | None | ✅ | PASS |
| Billing | invoices | billing.invoices | - | id | user_id | - | None | ✅ | PASS |
| Billing | payments | billing.payments | - | id | - | - | None | ✅ | PASS |
| Billing | refunds | billing.refunds | - | id | - | - | None | ✅ | PASS |
| Referral | referral_codes | referral.referral_codes | - | id | user_id | - | None | ✅ | PASS |
| Referral | referral_relationships | referral.referral_relationships | anti-circular chk | id | parent, child | - | None | ✅ | PASS |
| Referral | referral_events | referral.referral_events | - | id | - | - | None | ✅ | PASS |
| Commission | commission_ledger | referral.commission_ledger | IMMUTABLE + chk | id | recipient, source | idx_commission_ledger (NEW) | None | ✅ | PASS |
| Commission | commission_rules | referral.commission_rules | level chk | id | - | - | None | ✅ | PASS |
| Commission | payouts | referral.payouts | - | id | - | - | None | ✅ | PASS |
| Cooldown | cooldown_audit | trading.cooldown_audit | NEW | bigserial | - | 1 index | None | ✅ | PASS |
| Duplicate | duplicate_audit | trading.duplicate_audit | NEW | bigserial | candidate_id | 2 indexes | None | ✅ | PASS |
| AI | models | ai.models | - | id | - | - | None | ✅ | PASS |
| AI | inference_history | ai.inference_history | - | id | - | - | None | ✅ | PASS |
| AI | vector_embeddings | ai.vector_embeddings | NEW | id | - | 3 indexes (HNSW) | None | ✅ | PASS |
| Backtest | backtest_runs | research.backtest_runs | - | id | - | - | None | ✅ | PASS |
| Backtest | backtest_datasets | research.backtest_datasets | NEW | id | - | 1 index | None | ✅ | PASS |
| Backtest | backtest_trades | research.backtest_trades | NEW | id | run_id | 1 index | None | ✅ | PASS |
| System | system_configuration | system.system_configuration | NEW | id | - | - | None | ✅ | PASS |
| System | notifications | system.notifications | NEW | id | - | 2 indexes | None | ✅ | PASS |
| System | backup_metadata | system.backup_metadata | NEW | id | - | 2 indexes | None | ✅ | PASS |
