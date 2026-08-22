# Billing Event Model

`billing.subscription_event_v3` records explicit commercial events, amount components, effective eligible revenue, provider payment linkage, and an external idempotency reference. The table is additive to the existing `billing.subscription_events` and `billing.payments` tables. Payment-provider signature verification and activation remain provider-adapter work; the current repository has no verified provider contract.

