# Predict-A-Trade v1.0.0 — Codex Project Controls Installation

Copy the **contents** of this package into the repository root containing:

`Predict-A-Trade_FINAL_SCOPE_OF_WORK_v1.0.0.md`

## Required layout

```text
/srv/predict-a-trade/xauusd/
├── Predict-A-Trade_FINAL_SCOPE_OF_WORK_v1.0.0.md
├── AGENTS.md
├── AGENT.md
├── SKILLS.md
├── .mcp.json
├── mcp.env.example
├── .agents/
│   └── skills/
│       ├── repo-audit/SKILL.md
│       ├── architecture-guardrails/SKILL.md
│       ├── xauusd-market-data/SKILL.md
│       ├── xauusd-strategy-spec/SKILL.md
│       ├── xauusd-quant-validation/SKILL.md
│       ├── trading-risk-safety/SKILL.md
│       ├── mt4-mt5-windows/SKILL.md
│       ├── control-plane-saas/SKILL.md
│       ├── frontend-trading-ui/SKILL.md
│       ├── database-migrations/SKILL.md
│       ├── api-contracts/SKILL.md
│       ├── security-supply-chain/SKILL.md
│       ├── observability-sre/SKILL.md
│       ├── release-gates/SKILL.md
│       ├── docs-runbooks/SKILL.md
│       └── broker-execution-qualification/SKILL.md
└── .codex/
    ├── config.toml
    └── agents/
        ├── repo_explorer.toml
        ├── platform_architect.toml
        ├── go_realtime_engineer.toml
        ├── python_quant_researcher.toml
        ├── nestjs_control_engineer.toml
        ├── nextjs_frontend_engineer.toml
        ├── database_engineer.toml
        ├── windows_mql_engineer.toml
        ├── quant_validator.toml
        ├── security_reviewer.toml
        ├── qa_reliability_engineer.toml
        ├── release_manager.toml
        └── broker_execution_validator.toml
```

## Important filename behavior

Codex automatically discovers `AGENTS.md`, not singular `AGENT.md`.

Codex-native skills are the individual `.agents/skills/<name>/SKILL.md` files. `SKILLS.md` is an index for humans/other tooling.

Codex-native MCP configuration is `.codex/config.toml`. `.mcp.json` is included as a compatibility representation for other MCP-aware clients.

## MCP environment

Only export credentials you actually need:

```bash
export GITHUB_PERSONAL_ACCESS_TOKEN='...'
export CONTEXT7_API_KEY='...'   # optional
```

Use a least-privilege GitHub token. Do **not** put production broker credentials, payment/payout credentials, database-superuser credentials, signing private keys or secret-vault exports into MCP configuration.

GitHub MCP is intentionally configured `--read-only` by default. Remove that only in a separately reviewed workflow where GitHub writes are explicitly required.

## Verify Codex project instructions

From the project root:

```bash
codex --ask-for-approval never "Summarize the active project instructions, skills and custom subagents you can see. Do not modify files."
```

Verify MCP servers:

```bash
codex mcp list
```

Interactive Codex commands useful for verification:

```text
/mcp
/agent
/skills
```

Then execute the canonical v1.0.0 SOW using the Codex CLI command already provided in the SOW/previous response.
