---
name: go-crypto-security
description: "Audit Go crypto, TLS, secrets, and dependency CVEs."
---

# go-crypto-security

Use when auditing Predict-A-Trade Go code for crypto, TLS, secret leaks, or dependency vulnerabilities.

## Commands
```bash
cd realtime
govulncheck ./...                                      # known CVEs
go vet -vettool=$(which shadow) ./...                  # variable shadowing
gosec -quiet ./...                                     # SAST scan
staticcheck ./...                                      # static analysis
```

## Audit Checklist
1. Secrets: never hardcoded — search for password, secret, key, token, api
2. TLS: MinVersion >= TLS 1.2, no InsecureSkipVerify, cert pinning for brokers
3. Crypto: never math/rand for tokens/sessions (use crypto/rand)
4. WebSocket: origin check, auth tokens validated, message size limits
5. DB: sslmode=require or verify-full, scram-sha-256 preferred
6. Env vars: secrets via os.Getenv only in main.go, config structs everywhere else
7. Error responses: never leak stack traces to clients
8. Input validation: symbol, signal_id, price — bounds and format checks
9. File perms: 0600 for configs, 0644 for logs, 0700 for socket dirs
