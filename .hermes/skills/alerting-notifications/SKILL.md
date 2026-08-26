---
name: alerting-notifications
description: "ntfy alerts, Telegram, Discord, email notifications."
---

# alerting-notifications

Use for PAT alerts and notifications.

## ntfy (:8091)
Topics: trading-alerts, system-health, security, billing

## Alert Rules
SL violation (critical), gate failure >1% (high)
Signal latency >500ms p99 (high), DB pool exhaustion (critical)
Agent disconnected >60s (high), disk >85% (warning)

## Channels
Telegram (BotToken+ChatID), Discord (WebhookURL), Email (SendMail), ntfy

## Format
Title: [SEVERITY] Service - Summary
Body: timestamp, metric, threshold, action, runbook link
