# Subscription API Additions

- `GET /subscriptions/entitlements` — authenticated effective plan, selected strategies, and entitlement map.
- `POST /subscriptions` — authenticated compatibility endpoint; validates plan and strategy selection server-side and creates Free as active or paid plans as incomplete pending validated billing.
- Existing `/plans`, `/billing/invoices`, `/commissions`, and `/referrals` routes remain unchanged.

