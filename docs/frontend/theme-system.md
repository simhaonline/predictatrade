# Theme System

## Architecture

The frontend uses a single global theme system with three modes:
- **System Mode**: Follows OS `prefers-color-scheme`
- **Light Mode**: Forces light theme
- **Dark Mode**: Forces dark theme

## Implementation

- `next-themes` provides the theme provider at root layout level
- CSS custom properties (`--pat-*`) define semantic tokens
- Tailwind config maps tokens to utility classes (`bg-pat-bg-surface`, `text-pat-text-primary`, etc.)
- The sidebar uses separate dark tokens (intentionally dark in both modes)

## CSS Tokens

| Token | Light | Dark |
|-------|-------|------|
| --pat-bg-page | 97% light | 3.9% dark |
| --pat-bg-surface | 100% white | 7% dark |
| --pat-card-bg | 100% white | 7% dark |
| --pat-text-primary | dark navy | near-white |
| --pat-border | light gray | dark gray |
| --pat-input-bg | white | dark elevated |

## Brand Assets

Assets from `/srv/predictatrade/xauusd/asset_kit` are copied to `frontend/public/`.
The favicon uses `predict-a-trade_favicon.svg`.
Login uses `predict-a-trade_primary-vertical_white.svg`.

## Theme Preference Control

- Topbar dropdown with System/Light/Dark options
- Settings page has the same options
- Both manipulate the same `next-themes` preference
- Preference persists in localStorage across reloads

## Sidebar

The sidebar is intentionally dark in both Light and Dark modes (Tabler permanent sidebar pattern).
It uses `--pat-bg-sidebar`, `--pat-text-sidebar`, `--pat-border-sidebar` tokens.

## v1.4.0 Color Palette Update (19 August 2026)

The CSS tokens have been updated with the approved Predict-A-Trade color palette:

| Token | Light Hex | Dark Hex | Usage |
|-------|-----------|----------|-------|
| --pat-bg-page | #F8FAFC | #0F172A | Main background |
| --pat-bg-surface | #FFFFFF | #1E293B | Cards, panels |
| --pat-bg-sidebar | #0F172A | #020617 | Sidebar (dark both modes) |
| --pat-text-primary | #0F172A | #F8FAFC | Headings |
| --pat-text-secondary | #334155 | #CBD5E1 | Body text |
| --pat-text-muted | #64748B | #94A3B8 | Captions |
| --pat-border | #E2E8F0 | #334155 | Borders |

### Trading Semantic Tokens (Tailwind)

| Token | Hex | Usage |
|-------|-----|-------|
| pat-success | #10B981 | BUY, TP, BID |
| pat-danger | #EF4444 | SELL, SL, ASK |
| pat-warning | #EAB308 | SESSION |
| pat-info | #3B82F6 | INFO |
| pat-candidate-buy | #F59E0B | BUY_CANDIDATE |
| pat-candidate-sell | #FB923C | SELL_CANDIDATE |

### Critical Fix
HSL CSS variables required `%` signs on saturation/lightness components (e.g., `210 40% 98%` not `210 40 98`). Without `%`, colors were invisible.
