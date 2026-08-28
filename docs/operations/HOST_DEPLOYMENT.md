# Direct Host Deployment Guide
## v1.17.2 — 28 August 2026

Step-by-step guide to deploy Predict-A-Trade directly on a Linux host (no Docker). Use this for bare-metal, VPS, or when Docker is not available.

---

## Prerequisites

| Requirement | Minimum | Install Command |
|-------------|:-------:|-----------------|
| Ubuntu/Debian | 22.04/12 | — |
| Go | 1.26+ | `snap install go --classic` |
| Node.js | 18+ | `curl -fsSL https://deb.nodesource.com/setup_18.x \| sudo -E bash - && sudo apt install -y nodejs` |
| Python | 3.10+ | `sudo apt install -y python3 python3-pip python3-venv` |
| PostgreSQL | 17 | `sudo apt install -y postgresql-17` |
| TimescaleDB | 2.x | `sudo apt install -y timescaledb-2-postgresql-17` |
| Valkey/Redis | 8.0/7.x | `sudo apt install -y valkey` or `sudo apt install -y redis` |
| Nginx | latest | `sudo apt install -y nginx` |
| Git | recent | `sudo apt install -y git` |
| RAM | 4GB+ free | `free -h` |
| Disk | 20GB+ free | `df -h` |

---

## Step 1 — Create Project Directory Structure

```bash
sudo mkdir -p /srv/predictatrade/xauusd
sudo chown $(whoami):$(whoami) /srv/predictatrade/xauusd
cd /srv/predictatrade/xauusd
git clone https://github.com/simhaonline/predictatrade.git .
```

Project root: `/srv/predictatrade/xauusd/`

---

## Step 2 — Install & Configure PostgreSQL + TimescaleDB

```bash
# Install
sudo apt update
sudo apt install -y postgresql-17 timescaledb-2-postgresql-17

# Enable TimescaleDB extension
sudo timescaledb-tune --quiet --yes
sudo systemctl restart postgresql

# Create database and user
sudo -u postgres psql <<EOF
CREATE USER pat_admin WITH PASSWORD 'choose_strong_password_here' CREATEDB;
CREATE DATABASE predictatrade OWNER pat_admin;
\c predictatrade
CREATE EXTENSION IF NOT EXISTS timescaledb;
CREATE EXTENSION IF NOT EXISTS pgcrypto;
CREATE EXTENSION IF NOT EXISTS vector;
EOF

# Apply all migrations
for f in database/migrations/*.sql; do
    PGPASSWORD=choose_strong_password_here psql -U pat_admin -d predictatrade -f "$f"
    echo "Applied: $(basename $f)"
done

# Verify
PGPASSWORD=choose_strong_password_here psql -U pat_admin -d predictatrade \
    -c "SELECT COUNT(*) FROM audit.migration_history;"
# Expected: 30+
```

---

## Step 3 — Install & Configure Valkey (or Redis)

```bash
# Install Valkey
sudo apt install -y valkey
sudo systemctl enable --now valkey

# Or Redis if Valkey not available
# sudo apt install -y redis-server

# Verify
valkey-cli ping     # or redis-cli ping
# Expected: PONG
```

---

## Step 4 — Build the Go Realtime Engine

```bash
export PATH="/root/.gvm/gos/go1.26.6/bin:$PATH"   # Or your Go install path

cd /srv/predictatrade/xauusd/realtime
go build -o bin/realtime-engine ./cmd/realtime-engine

# Verify binary
file bin/realtime-engine
# Expected: ELF 64-bit LSB executable
```

---

## Step 5 — Build the NestJS Control Plane

```bash
cd /srv/predictatrade/xauusd/control
npm install
npm run build

# Verify build
ls dist/main.js
# Expected: dist/main.js exists
```

---

## Step 6 — Build the Next.js Frontend

```bash
cd /srv/predictatrade/xauusd/frontend
npm install
npm run build

# Verify build
ls .next/BUILD_ID
# Expected: BUILD_ID file exists
```

---

## Step 7 — Build the Live Terminal (optional)

```bash
cd /srv/predictatrade/xauusd
# Requires the live-terminal.Dockerfile context
# Build the binary manually:
cd realtime
go build -o bin/live-terminal ./cmd/live-terminal
```

---

## Step 8 — Configure Environment Variables

### Create database URL file
```bash
echo "postgresql://pat_admin:choose_strong_password_here@127.0.0.1:5432/predictatrade?sslmode=disable" \
    > /srv/predictatrade/xauusd/database_url.txt
chmod 600 /srv/predictatrade/xauusd/database_url.txt
```

### Generate JWT secret
```bash
openssl rand -base64 32 > /srv/predictatrade/xauusd/jwt_secret.txt
chmod 600 /srv/predictatrade/xauusd/jwt_secret.txt
```

### Realtime engine env
```bash
cat > /srv/predictatrade/xauusd/infra/env/realtime.env <<'EOF'
DATABASE_URL=postgresql://pat_admin:***@127.0.0.1:5432/predictatrade?sslmode=disable
VALKEY_ADDR=127.0.0.1:6379
HTTP_HOST=127.0.0.1
HTTP_PORT=13081
WS_PORT=13081
PROVIDER_MODE=agent
SYMBOLS=XAUUSD
TWELVEDATA_API_KEY=your_twelvedata_key
FMP_API_KEY=your_fmp_key
JWT_SECRET=your_jwt_secret_base64
LOG_LEVEL=info
BROKER_TIMEZONE=GMT+3
PTB_ENABLED=true
COT_ENABLED=true
DXY_ENABLED=true
SENTIMENT_ENABLED=false
RL_MODE=disabled
EOF
```

### Control plane env
```bash
cat > /srv/predictatrade/xauusd/infra/env/control.env <<'EOF'
DATABASE_URL=postgresql://pat_admin:choose_strong_password_here@127.0.0.1:5432/predictatrade?sslmode=disable
JWT_SECRET=your_jwt_secret_base64
CONTROL_HOST=127.0.0.1
CONTROL_PORT=13080
CORS_ORIGINS=https://predictatrade.com,https://platform.predictatrade.com
EMAIL_PROVIDER=smtp
SMTP_HOST=mail.predictatrade.com
SMTP_PORT=587
EOF
```

### Frontend env
```bash
cat > /srv/predictatrade/xauusd/infra/env/frontend.env <<'EOF'
NODE_ENV=production
NEXT_PUBLIC_API_URL=https://api.predictatrade.com/api/v1
NEXT_PUBLIC_WS_URL=wss://live.predictatrade.com/ws/v1
NEXT_PUBLIC_PLATFORM_URL=https://platform.predictatrade.com
EOF
```

---

## Step 9 — Create systemd Service Files

Systemd service files are already provided at `infra/systemd/`. Copy them:

```bash
sudo cp infra/systemd/predictatrade-realtime.service /etc/systemd/system/
sudo cp infra/systemd/predictatrade-control.service /etc/systemd/system/
sudo cp infra/systemd/predictatrade-frontend.service /etc/systemd/system/
sudo cp infra/systemd/predictatrade-status.service /etc/systemd/system/
sudo systemctl daemon-reload
```

Or create them manually:

```bash
sudo tee /etc/systemd/system/predictatrade-realtime.service <<'EOF'
[Unit]
Description=Predict-A-Trade Go Real-Time Engine
After=network.target postgresql.service
Wants=postgresql.service

[Service]
Type=simple
User=root
Group=root
WorkingDirectory=/srv/predictatrade/xauusd/realtime
EnvironmentFile=/srv/predictatrade/xauusd/infra/env/realtime.env
ExecStart=/srv/predictatrade/xauusd/realtime/bin/realtime-engine
Restart=always
RestartSec=3
StandardOutput=journal
StandardError=journal
NoNewPrivileges=true
PrivateTmp=true
LimitNOFILE=65536
MemoryMax=1G

[Install]
WantedBy=multi-user.target
EOF

sudo tee /etc/systemd/system/predictatrade-control.service <<'EOF'
[Unit]
Description=Predict-A-Trade NestJS Control Plane
After=network.target postgresql.service
Wants=postgresql.service

[Service]
Type=simple
User=root
Group=root
WorkingDirectory=/srv/predictatrade/xauusd/control
EnvironmentFile=/srv/predictatrade/xauusd/infra/env/control.env
ExecStart=/usr/bin/node dist/main.js
Restart=always
RestartSec=3
StandardOutput=journal
StandardError=journal
NoNewPrivileges=true
PrivateTmp=true
MemoryMax=512M

[Install]
WantedBy=multi-user.target
EOF

sudo tee /etc/systemd/system/predictatrade-frontend.service <<'EOF'
[Unit]
Description=Predict-A-Trade Next.js Frontend
After=network.target

[Service]
Type=simple
User=root
Group=root
WorkingDirectory=/srv/predictatrade/xauusd/frontend
EnvironmentFile=/srv/predictatrade/xauusd/infra/env/frontend.env
ExecStart=/usr/bin/npx next start -p 13082
Restart=always
RestartSec=3
StandardOutput=journal
StandardError=journal
MemoryMax=512M

[Install]
WantedBy=multi-user.target
EOF
```

---

## Step 10 — Start All Services

```bash
# Enable auto-start on boot
sudo systemctl enable predictatrade-realtime
sudo systemctl enable predictatrade-control
sudo systemctl enable predictatrade-frontend

# Start now
sudo systemctl start predictatrade-realtime
sudo systemctl start predictatrade-control
sudo systemctl start predictatrade-frontend

# Check status
sudo systemctl status predictatrade-realtime
sudo systemctl status predictatrade-control
sudo systemctl status predictatrade-frontend
```

---

## Step 11 — Configure Nginx Reverse Proxy

```bash
# Copy site configs
sudo cp infra/nginx/sites-available/*.conf /etc/nginx/sites-available/
sudo cp infra/nginx/snippets/*.conf /etc/nginx/snippets/
sudo cp infra/nginx/nginx.conf /etc/nginx/nginx.conf

# Enable sites
sudo ln -sf /etc/nginx/sites-available/predictatrade.com.conf /etc/nginx/sites-enabled/
sudo ln -sf /etc/nginx/sites-available/platform.predictatrade.com.conf /etc/nginx/sites-enabled/
sudo ln -sf /etc/nginx/sites-available/api.predictatrade.com.conf /etc/nginx/sites-enabled/
sudo ln -sf /etc/nginx/sites-available/live.predictatrade.com.conf /etc/nginx/sites-enabled/
sudo ln -sf /etc/nginx/sites-available/status.predictatrade.com.conf /etc/nginx/sites-enabled/

# Test and reload
sudo nginx -t
sudo systemctl reload nginx
```

---

## Step 12 — Set Up SSL Certificates

```bash
# Install certbot
sudo apt install -y certbot python3-certbot-nginx

# Get certificates for all domains
sudo certbot --nginx -d predictatrade.com -d www.predictatrade.com
sudo certbot --nginx -d platform.predictatrade.com
sudo certbot --nginx -d api.predictatrade.com
sudo certbot --nginx -d live.predictatrade.com
sudo certbot --nginx -d status.predictatrade.com

# Auto-renewal (certbot installs a systemd timer automatically)
sudo systemctl status certbot.timer
```

---

## Step 13 — Verify All Services

```bash
# Health checks (internal)
curl http://127.0.0.1:13081/health
curl http://127.0.0.1:13080/api/v1/health
curl -I http://127.0.0.1:13082

# Health checks (external via nginx)
curl https://api.predictatrade.com/api/v1/health
curl https://platform.predictatrade.com/
curl https://live.predictatrade.com/health

# Service status
sudo systemctl status predictatrade-realtime predictatrade-control predictatrade-frontend postgresql valkey nginx

# Check logs
sudo journalctl -u predictatrade-realtime -f
sudo journalctl -u predictatrade-control --since "5 minutes ago"
```

---

## Step 14 — Monitoring (Prometheus + Grafana)

Ollama is not required. Install for optional AI sentiment analysis only.

```bash
# Optional: install Ollama for sentiment analysis
# curl -fsSL https://ollama.com/install.sh | sh
# ollama pull llama3.2
```

Prometheus and Grafana are optional. Install for monitoring:

```bash
# Prometheus
sudo apt install -y prometheus
sudo cp infra/prometheus/prometheus.yml /etc/prometheus/
sudo cp infra/prometheus/rules.yml /etc/prometheus/
sudo systemctl enable --now prometheus

# Grafana
sudo apt install -y grafana
sudo systemctl enable --now grafana-server
# Access at http://localhost:3001 (admin/admin)
```

---

## Daily Operations

### View service logs
```bash
sudo journalctl -u predictatrade-realtime -f           # Follow realtime logs
sudo journalctl -u predictatrade-realtime --since today # Today's logs
sudo journalctl -u predictatrade-control -n 100         # Last 100 lines
```

### Restart a service
```bash
sudo systemctl restart predictatrade-realtime
sudo systemctl restart predictatrade-control
sudo systemctl restart predictatrade-frontend
```

### Check service status
```bash
sudo systemctl status predictatrade-realtime
sudo systemctl is-active predictatrade-realtime   # Returns "active" or "inactive"
```

### Database backup
```bash
PGPASSWORD=choose_strong_password_here pg_dump -U pat_admin -h 127.0.0.1 predictatrade \
    > /backups/predictatrade_$(date +%Y%m%d_%H%M%S).sql
gzip /backups/predictatrade_*.sql
```

### Database restore
```bash
gunzip -c /backups/predictatrade_20260826_120000.sql.gz | \
    PGPASSWORD=choose_strong_password_here psql -U pat_admin -h 127.0.0.1 -d predictatrade
```

### Apply new migrations
```bash
cd /srv/predictatrade/xauusd
for f in database/migrations/*.sql; do
    PGPASSWORD=choose_strong_password_here psql -U pat_admin -d predictatrade -f "$f"
done
```

### Deploy code updates
```bash
cd /srv/predictatrade/xauusd
git pull

# Rebuild Go engine
cd realtime && go build -o bin/realtime-engine ./cmd/realtime-engine && cd ..

# Rebuild control plane
cd control && npm install && npm run build && cd ..

# Rebuild frontend
cd frontend && npm install && npm run build && cd ..

# Restart services
sudo systemctl restart predictatrade-realtime predictatrade-control predictatrade-frontend
```

---

## Firewall Configuration

```bash
# Enable UFW
sudo ufw allow 22/tcp           # SSH
sudo ufw allow 80/tcp           # HTTP
sudo ufw allow 443/tcp          # HTTPS
sudo ufw deny 5432              # PostgreSQL — internal only
sudo ufw deny 6379              # Valkey — internal only
sudo ufw allow 13081            # Realtime — or deny, proxy through nginx
sudo ufw allow 13080            # Control — or deny, proxy through nginx
sudo ufw enable
sudo ufw status verbose
```

---

## Troubleshooting

### Service won't start
```bash
sudo systemctl status predictatrade-realtime
sudo journalctl -u predictatrade-realtime -n 50 --no-pager
```

### Database connection refused
```bash
# Check postgres is running
sudo systemctl status postgresql
# Test connection
PGPASSWORD=your_pass psql -U pat_admin -h 127.0.0.1 -d predictatrade -c "SELECT 1"
# Check pg_hba.conf allows connections
sudo grep -v '^#' /etc/postgresql/17/main/pg_hba.conf | grep -v '^$'
```

### Port already in use
```bash
sudo ss -tlnp | grep -E '13081|13080|13082'
sudo lsof -i :13081
```

### Permission denied on socket
```bash
sudo chown -R $(whoami):$(whoami) /srv/predictatrade/xauusd
```

### Nginx 502 Bad Gateway
```bash
# Check upstream service is running
curl http://127.0.0.1:13081/health
# Check nginx error log
sudo tail -50 /var/log/nginx/error.log
```

---

## Production Hardening Checklist

- [ ] Change PostgreSQL password from any default
- [ ] Set strong JWT_SECRET via secret file (32+ random chars)
- [ ] Run services as non-root user (create `pat` user, chown)
- [ ] Configure firewall (only 80/443 public, everything else loopback)
- [ ] Set up SSL certificates with auto-renewal
- [ ] Configure automated database backups (cron job)
- [ ] Set up log rotation for /var/log/nginx/
- [ ] Install and configure fail2ban for SSH
- [ ] Enable automatic security updates (`unattended-upgrades`)
- [ ] Test backup restore procedure
- [ ] Set up monitoring alerts (disk, memory, service down)
- [ ] Review systemd resource limits (MemoryMax, LimitNOFILE)
