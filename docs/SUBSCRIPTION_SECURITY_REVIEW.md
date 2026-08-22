# Security Review

The strategy validator uses server-side plan rows and rejects unknown, unauthorized, duplicate-over-limit selections. Free registration is excluded from commission calculation. Durable uniqueness exists for external subscription events, user/signal/channel delivery, and v3 commission idempotency. No production credentials were rotated or exported. Webhook signature verification, provider idempotency integration, and end-to-end user WebSocket/notification authorization remain unverified blockers because their adapters are absent from the audited code.

