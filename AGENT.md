# AGENT.md — Compatibility Pointer

Codex-native repository instructions are in `AGENTS.md`.

Current Codex automatically discovers `AGENTS.md` / `AGENTS.override.md`, not singular `AGENT.md`.

For Predict-A-Trade v1.0.0:
1. Read `AGENTS.md`.
2. Read `Predict-A-Trade_FINAL_SCOPE_OF_WORK_v1.0.0.md`.
3. Load applicable `.agents/skills/*/SKILL.md`.
4. Use project subagents from `.codex/agents/*.toml`.
5. Use Codex-native MCP configuration from `.codex/config.toml`.

This singular file exists only for compatibility with tooling or human workflows that expect `AGENT.md`.
