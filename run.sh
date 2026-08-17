cd /srv/predictatrade/xauusd || exit 1

# ------------------------------------------------------------
# 1. Securely request GitHub PAT
# ------------------------------------------------------------
mkdir -p /root/.config/predict-a-trade

read -rsp "Enter GitHub Personal Access Token: " GITHUB_PERSONAL_ACCESS_TOKEN
echo

if [ -z "$GITHUB_PERSONAL_ACCESS_TOKEN" ]; then
    echo "[ERROR] GitHub token cannot be empty."
    exit 1
fi

cat > /root/.config/predict-a-trade/github-mcp.env <<EOF
export GITHUB_PERSONAL_ACCESS_TOKEN='$GITHUB_PERSONAL_ACCESS_TOKEN'
EOF

chmod 600 /root/.config/predict-a-trade/github-mcp.env

export GITHUB_PERSONAL_ACCESS_TOKEN

# ------------------------------------------------------------
# 2. Backup existing Codex config
# ------------------------------------------------------------
cp -a .codex/config.toml \
    ".codex/config.toml.backup.$(date +%Y%m%d-%H%M%S)"

# ------------------------------------------------------------
# 3. Remove old Docker GitHub MCP config and install
#    official remote GitHub MCP configuration
# ------------------------------------------------------------
python3 <<'PY'
from pathlib import Path

path = Path(".codex/config.toml")

if not path.exists():
    raise SystemExit("ERROR: .codex/config.toml not found")

lines = path.read_text(encoding="utf-8").splitlines()

out = []
skip = False

for line in lines:
    stripped = line.strip()

    if stripped.startswith("[mcp_servers.github"):
        skip = True
        continue

    if skip and stripped.startswith("[") and not stripped.startswith("[mcp_servers.github"):
        skip = False

    if not skip:
        out.append(line)

# Remove excess trailing blank lines
while out and not out[-1].strip():
    out.pop()

out.extend([
    "",
    "# ============================================================",
    "# GitHub MCP — official remote server",
    "# Predict-A-Trade: read-only repository / PR / CI / security",
    "# ============================================================",
    "",
    "[mcp_servers.github]",
    'url = "https://api.githubcopilot.com/mcp/"',
    'bearer_token_env_var = "GITHUB_PERSONAL_ACCESS_TOKEN"',
    'http_headers = { "X-MCP-Toolsets" = "repos,issues,pull_requests,actions,code_security", "X-MCP-Readonly" = "true" }',
    'enabled = true',
    'required = false',
    'startup_timeout_sec = 30',
    'tool_timeout_sec = 120',
    'default_tools_approval_mode = "auto"',
    "",
])

path.write_text("\n".join(out), encoding="utf-8")

print("GitHub MCP configuration updated successfully.")
PY

# ------------------------------------------------------------
# 4. Validate configuration
# ------------------------------------------------------------
echo
echo "========== GITHUB MCP CONFIG =========="
grep -A10 '^\[mcp_servers.github\]' .codex/config.toml

echo
echo "========== CODEX MCP LIST =========="
codex mcp list

echo
echo "============================================================"
echo "GitHub MCP fixed."
echo
echo "Launch Codex with:"
echo
echo "source /root/.config/predict-a-trade/github-mcp.env"
echo "ollama launch codex --model glm-5.2:cloud"
echo "============================================================"
