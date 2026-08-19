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
