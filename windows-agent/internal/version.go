package agent

// AgentVersion is the single source of truth for the agent binary version.
// The installer's version.txt on the server must match this value.
// When pushing a new release: increment this, rebuild pat-agent.exe, update deploy/version.txt.
const AgentVersion = "1.2.14"
