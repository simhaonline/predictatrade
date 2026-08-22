# Entitlement Architecture

The Go engine generates qualified signals independently. Distribution authorization is a control-plane concern:

```mermaid
flowchart LR
  G[Signal generation] --> S[Persisted signal]
  S --> E[Plan entitlement]
  E --> P[Selected strategy]
  P --> Q[Free durable quota]
  Q --> D[Authorized delivery]
```

`control.plan_entitlements` is the existing capability source. `control.strategy_preferences` stores the user choice. The backend validator in `control/src/modules/subscriptions/entitlement-policy.ts` is authoritative; frontend restrictions are advisory only.

