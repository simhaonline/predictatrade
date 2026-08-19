# Predict-A-Trade Website

Predict-A-Trade v1.0.0 is the public marketing and contact website for Simha Online's XAUUSD market-intelligence product. It is a small, dependency-free static site with an optional Node.js server for hosting assets and receiving contact-form submissions by SMTP.

## What is included

- Responsive public landing page with SEO, Open Graph, Twitter Card, and JSON-LD metadata.
- XAUUSD-focused product messaging and visual assets.
- Static sitemap, robots policy, LLM-readable project summary, and Apache rewrite rules.
- Contact form endpoint at `POST /api/contact`.
- Gzip compression for text responses and long-lived caching for fingerprinted assets.
- No database, user authentication, payment processing, trading execution, or live market-data service in this website package.

## Requirements

- Node.js 18 or newer.
- An SMTP mailbox if the contact form is enabled.

The site has no npm dependency installation step; `server.js` uses only Node.js built-ins.

## Run locally

```bash
cd /srv/predictatrade/xauusd/predictatrade.com
PORT=8080 node server.js
```

Open <http://localhost:8080>. The server serves the website from its own directory and returns `404` for paths outside that directory.

## Configuration

Copy `.env.example` to the deployment environment and set the values below. Do not commit real credentials.

| Variable | Default | Purpose |
| --- | --- | --- |
| `PORT` | `8080` | HTTP listening port |
| `SMTP_HOST` | `predictatrade.com` | SMTP server hostname |
| `SMTP_PORT` | `465` | SMTP server port |
| `SMTP_SECURE` | `true` for port 465 | Use implicit TLS; set `false` for STARTTLS |
| `SMTP_USER` | `no-reply@predictatrade.com` | Authenticated sender and recipient mailbox |
| `SMTP_PASS` | unset | SMTP mailbox password; required for contact delivery |

If `SMTP_PASS` is not configured, contact submissions return an error rather than pretending that a message was delivered.

## Contact API

`POST /api/contact` accepts JSON:

```json
{
  "name": "Ada Lovelace",
  "email": "ada@example.com",
  "message": "I would like to learn more."
}
```

The server validates the name, email format, and non-empty message, limits the request body to 20,000 bytes, and sends the message through the configured SMTP account. A successful request returns `{ "ok": true }`.

## Project layout

```text
index.html          Landing-page shell and SEO metadata
assets/             Fingerprinted JavaScript and CSS bundles
media/              Hero, texture, and brand images
contact-form.js     Contact form browser behavior
contact-form.css    Contact form styles
server.js           Static server and SMTP contact endpoint
robots.txt          Crawler policy
sitemap.xml         Canonical site URL list
llms.txt            Concise machine-readable site description
.htaccess           Apache hosting rules
```

## Deployment

Run the server under a process supervisor such as systemd, Docker, or a managed Node.js service. Put TLS and domain routing in the hosting edge or reverse proxy. Keep SMTP credentials in the deployment secret store, restrict access to `server.js`, and expose only the public HTTP port.

For a static-only deployment, serve the files directly with a web server; the contact form requires the Node.js server (or an equivalent backend endpoint) to deliver messages.

## Validation

Before release, verify:

1. `node --check server.js`
2. `node server.js` serves `/`, static assets, `robots.txt`, and `sitemap.xml`.
3. Invalid contact payloads receive HTTP 400.
4. A staging SMTP mailbox receives a valid contact submission.
5. Production secrets are supplied outside Git and HTTPS is enabled at the edge.

## Scope and safety

This repository is the website presentation layer only. It must not be treated as proof of live trading capability, signal accuracy, profitability, broker execution, or financial performance. Those capabilities require separate server-authoritative services and evidence-backed validation.

