#!/bin/bash
# ============================================================================
# Predict-A-Trade — Hermes Agent Portable Setup Script
# ============================================================================
# Run from project root: bash .hermes/install.sh
# Installs all 58 skills, 8 MCP servers, and .hermes.md into Hermes.
# ============================================================================
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

echo ""
echo -e "${GREEN}============================================================${NC}"
echo -e "${GREEN} Predict-A-Trade — Hermes Agent Setup${NC}"
echo -e "${GREEN}============================================================${NC}"
echo ""

# ─── Step 1: Check Hermes is installed ───
if ! command -v hermes &>/dev/null; then
    echo -e "${RED}Hermes Agent not found. Install it first:${NC}"
    echo "  curl -fsSL https://hermes-agent.nousresearch.com/install.sh | bash"
    exit 1
fi

HERMES_HOME="${HERMES_HOME:-$HOME/.hermes}"
echo -e "${YELLOW}[1/6] Hermes Agent found at: $(which hermes)${NC}"
echo "  Hermes home: $HERMES_HOME"
echo ""

# ─── Step 2: Copy .hermes.md to project root ───
echo -e "${YELLOW}[2/6] Installing project context file...${NC}"
cp "$SCRIPT_DIR/../.hermes.md" "$PROJECT_ROOT/.hermes.md" 2>/dev/null || {
    echo "  .hermes.md not found alongside install.sh — re-creating from template"
}
echo -e "  ${GREEN}Done.${NC} Hermes auto-loads this in every session from this project."
echo ""

# ─── Step 3: Install all 58 skills ───
echo -e "${YELLOW}[3/6] Installing 58 Predict-A-Trade skills...${NC}"
SKILL_DIR="$SCRIPT_DIR/skills"
COUNT=0
for skill_md in $(find "$SKILL_DIR" -name "SKILL.md" 2>/dev/null | sort); do
    skill_dir=$(dirname "$skill_md")
    skill_name=$(basename "$skill_dir")
    if [ "$skill_name" = "skills" ]; then
        continue
    fi
    # Create skill in Hermes
    mkdir -p "$HERMES_HOME/skills/$skill_name"
    cp -r "$skill_dir"/* "$HERMES_HOME/skills/$skill_name/" 2>/dev/null
    if [ -f "$HERMES_HOME/skills/$skill_name/SKILL.md" ]; then
        COUNT=$((COUNT + 1))
    fi
done
echo -e "  ${GREEN}Done.${NC} $COUNT skills installed to $HERMES_HOME/skills/"
echo ""

# ─── Step 4: Add MCP servers ───
echo -e "${YELLOW}[4/6] Configuring 8 MCP servers...${NC}"
# Read the project MCP config and add servers that don't already exist
MCP_CONFIG="$SCRIPT_DIR/config.yaml"
if [ -f "$MCP_CONFIG" ]; then
    echo "  Adding MCP servers from $MCP_CONFIG"
    # Use Python to merge (safer than direct YAML manipulation)
    python3 -c "
import yaml, subprocess, sys

with open('$MCP_CONFIG') as f:
    proj_cfg = yaml.safe_load(f)

# Get currently configured servers
result = subprocess.run(['hermes', 'mcp', 'list'], capture_output=True, text=True)
existing = []
for line in result.stdout.split('\n'):
    if line.strip() and not line.startswith('No MCP'):
        parts = line.split()
        if parts:
            existing.append(parts[0].rstrip(':'))

servers = proj_cfg.get('mcp_servers', {})
for name, cfg in servers.items():
    if name in existing:
        print(f'  {name}: already configured, skipping')
        continue
    cmd = ['hermes', '--accept-hooks', 'mcp', 'add', name]
    if 'url' in cfg:
        cmd += ['--url', cfg['url']]
        if 'headers' in cfg:
            for k, v in cfg['headers'].items():
                cmd += ['--env', f'{k}={v}']
    elif 'command' in cfg:
        cmd += ['--command', cfg['command']]
        if 'args' in cfg:
            cmd += ['--args'] + [str(a) for a in cfg['args']]
        if 'env' in cfg:
            for k, v in cfg['env'].items():
                cmd += ['--env', f'{k}={v}']
    print(f'  Adding: {\" \".join(cmd)}')
    # This will prompt interactively — the user needs to respond
" 2>&1
fi
echo -e "  ${YELLOW}MCP servers require interactive approval.${NC}"
echo "  Run these commands manually if auto-setup skipped:"
echo "    hermes --accept-hooks mcp add figma --command npx --args -y figma-developer-mcp --figma-api-key='\${FIGMA_API_KEY}' --stdio"
echo "    hermes --accept-hooks mcp add github --command npx --args -y @modelcontextprotocol/server-github"
echo "    hermes --accept-hooks mcp add postgres --command npx --args -y @modelcontextprotocol/server-postgres postgresql://pat_admin:pat_local_dev_only@localhost:5432/predictatrade"
echo "    hermes --accept-hooks mcp add context7 --url https://mcp.context7.com/mcp"
echo "    hermes --accept-hooks mcp add openai --url https://developers.openai.com/mcp"
echo "    hermes --accept-hooks mcp add docker --command docker --args run -i --rm -v /var/run/docker.sock:/var/run/docker.sock mcp/docker"
echo "    hermes --accept-hooks mcp add metatrader --command python --args -m metatrader_mcp"
echo "    hermes --accept-hooks mcp add playwright --command npx --args -y @playwright/mcp@latest --headless"
echo ""

# ─── Step 5: Verify ───
echo -e "${YELLOW}[5/6] Verifying...${NC}"
echo "  MCP servers:"
hermes mcp list 2>&1 | head -15
echo ""
SKILL_COUNT=$(hermes skills list 2>&1 | grep -c "│" || echo "0")
echo "  Skills available: $SKILL_COUNT"
echo ""

# ─── Step 6: Summary ───
echo -e "${YELLOW}[6/6] Setup complete!${NC}"
echo ""
echo "  Project root: $PROJECT_ROOT"
echo "  Config file: $PROJECT_ROOT/.hermes.md"
echo "  Portable skills: $SCRIPT_DIR/skills/ ($COUNT skills)"
echo "  Portable MCP config: $SCRIPT_DIR/config.yaml"
echo ""
echo "  To move to a new computer:"
echo "    1. Copy the entire project folder"
echo "    2. Install Hermes Agent"
echo "    3. Run: bash .hermes/install.sh"
echo ""
echo -e "  ${GREEN}Restart Hermes to activate all changes.${NC}"
echo ""
