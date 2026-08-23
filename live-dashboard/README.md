# Live Dashboard — live.predictatrade.com

## Deployment
The live dashboard is served by nginx from `/var/www/pat-live/` (host bind-mount).
Copy files from this directory to `/var/www/pat-live/` to deploy:

```bash
cp live-dashboard/index.html /var/www/pat-live/index.html
cp live-dashboard/sw.js /var/www/pat-live/sw.js
cp live-dashboard/manifest.json /var/www/pat-live/manifest.json
```

## Fix Applied (2026-08-23)
- **Grid layout fix**: `#app` grid had 3 rows (`30px 1fr 22px`) but 4 children
  (modeBanner, header, main, footer). The modeBanner took the header's 30px row,
  header took main's 1fr row, and main got only 22px — crushing all charts to 0 height.
  Fixed to `auto 30px 1fr 22px` (4 rows for 4 children).
- **Service worker cache bump**: v2 → v3 to force browser refresh of fixed layout.
- **Responsive fix**: Updated mobile media query to match 4-row grid.
