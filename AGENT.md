# AGENT.md — Compatibility Pointer

Codex-native repository instructions are in `AGENTS.md`.

Current Codex automatically discovers `AGENTS.md` / `AGENTS.override.md`, not singular `AGENT.md`.

For Predict-A-Trade v1.0.0:
1. Read `AGENTS.md`.
2. Read the SOW in `realtime/SCOPE_OF_WORK.md` + version contract in `MANIFEST.md`.
3. Load applicable `.hermes/skills/*/SKILL.md`.
4. Use bounded subagent profiles as described in `AGENTS.md` (Required Subagents).
5. MCP configuration lives in `.hermes/config.yaml` (no `.codex/` or `.mcp.json` in this repo).

This singular file exists only for compatibility with tooling or human workflows that expect `AGENT.md`.
