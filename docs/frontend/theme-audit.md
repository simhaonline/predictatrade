# Theme Audit

## Root Cause

All components used hardcoded Tailwind `neutral-*` color classes (e.g., `bg-neutral-950`, `text-white`, `border-neutral-800`) instead of semantic CSS tokens. These are dark-only colors that don't respond to theme changes, causing the hybrid Light Mode (light page background + dark cards + dark sidebar).

## Fix

Replaced 440 hardcoded color references with semantic tokens:
- `bg-neutral-950` → `bg-pat-bg-surface` / `bg-pat-bg-sidebar` (sidebar)
- `bg-neutral-900` → `bg-pat-bg-surface`
- `bg-neutral-800` → `bg-pat-bg-surface-secondary`
- `text-white` → `text-pat-text-primary`
- `text-neutral-400` → `text-pat-text-secondary`
- `text-neutral-500` → `text-pat-text-muted`
- `border-neutral-800` → `border-pat-border`
- `border-neutral-700` → `border-pat-border-strong`

## Files Fixed

40 component files updated with semantic tokens. Sidebar uses separate dark tokens. Trading colors (green/red/amber) preserved as semantic market colors.
