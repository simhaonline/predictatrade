# Predict-A-Trade Website

Predict-A-Trade v1.0.0 is a static public website for XAUUSD market intelligence and probability analytics.

## Project contents

- `index.html` — homepage shell and metadata.
- `assets/` — website JavaScript and CSS bundles.
- `media/` — website imagery.
- `legal/` — legal and trust-center pages.
- `cookie-consent.js` and `cookie-settings.js` — cookie preference controls.
- `manifest.json`, `service-worker.js`, and `media/icon-*.webp` — installable PWA metadata, offline app shell, and app icons.
- `robots.txt`, `sitemap.xml`, and `llms.txt` — site metadata.
- `.htaccess` — optional Apache configuration.

## Legal & Trust Center

The homepage footer includes a `Legal & Trust Center` link to `/legal/`, together with the policy, privacy, complaints, sitemap, and cookie links.

The legal pages describe the product as decision-support and probability analytics. They do not promise profits, guaranteed outcomes, or trading performance.

## Technical scope

This repository contains the public presentation layer only. It does not contain live trading execution, payment processing, private APIs, credentials, or proprietary trading logic.

The website is plain HTML, CSS, and browser JavaScript. It has no Node.js application, database, or required `.env` file.

## Mobile & PWA

The homepage uses a fluid single-column layout below 768px, responsive imagery, and touch-friendly controls. The service worker uses a cache-first strategy for the public app shell and falls back to the homepage when navigating offline.

## Static hero visual

The homepage uses `media/predict-a-trade-static-hero.webp`, a calm editorial visual built around a gold-market signal, muted directional context, and a dark analytical field. It is a static marketing image with no live-price behavior, no ticker, and no client-side market-data request.
