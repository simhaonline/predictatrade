package agent

import (
	"log"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

type Config struct {
	LiveWSURL      string
	DataWSURL      string // WebSocket URL for the data-only Master Node agent endpoint
	APIURL         string
	ServerURL      string
	Mode           string // "exec" (default) or "data" — separates data collection from order execution
	LicenseKey     string // User-provided license key for activation
	ActivationCode string
	AgentDataDir   string
	DeviceKeyPath  string
	MT4PipeName    string
	MT5PipeName    string
	UpdateChannel  string
	BrokerName     string
	// Device auth state (populated after activation)
	DeviceID      string
	SessionID     string
	AccessToken   string
	RefreshToken  string
	DeviceSecret  string
	// AgentWSToken is the shared secret the realtime engine requires on the
	// agent WebSocket upgrade (X-Agent-Token header or ?token= query param).
	// It MUST match the engine's AGENT_WS_TOKEN exactly. Legacy/operator infra
	// (e.g. a Master data node) uses this; per-client agents use WSToken instead.
	AgentWSToken string
	// WSToken is a per-device JWT minted by the control plane at device
	// activation. It is bootstrapped from the client's own license key, so no
	// secret needs to be manually distributed to each client. Preferred over
	// AgentWSToken when present.
	WSToken string
}

func LoadConfig() *Config {
	liveWS := resolveWSURL("PAT_LIVE_WS_URL", "wss://api.predictatrade.com/ws/v1/agent")
	dataWS := resolveWSURL("PAT_DATA_WS_URL", "wss://api.predictatrade.com/ws/v1/data")
	apiURL := getEnv("PAT_API_URL", "https://api.predictatrade.com/api/v1")
	mode := getEnv("PAT_AGENT_MODE", "exec")
	dataDir := getEnv("PAT_DATA_DIR", "C:\\ProgramData\\PredictATrade")

	if os.Getenv("PAT_DEV_MODE") == "1" {
		liveWS = resolveWSURL("PAT_LIVE_WS_URL", "ws://127.0.0.1:13081/ws")
		dataWS = resolveWSURL("PAT_DATA_WS_URL", "ws://127.0.0.1:13091/ws/v1/data")
		apiURL = getEnv("PAT_API_URL", "http://127.0.0.1:13080/api/v1")
		dataDir = getEnv("PAT_DATA_DIR", "/tmp/predictatrade")
	}
	if mode != "data" {
		mode = "exec"
	}

	return &Config{
		LiveWSURL:      liveWS,
		DataWSURL:      dataWS,
		APIURL:         apiURL,
		ServerURL:      liveWS,
		Mode:           mode,
		LicenseKey:     getEnv("PAT_LICENSE_KEY", ""),
		ActivationCode: getEnv("PAT_ACTIVATION_CODE", ""),
		AgentDataDir:   dataDir,
		DeviceKeyPath:  getEnv("PAT_DEVICE_KEY_PATH", "C:\\ProgramData\\PredictATrade\\device.key"),
		MT4PipeName:    getEnv("PAT_MT4_PIPE", "\\\\.\\pipe\\PredictATradeMT4"),
		MT5PipeName:    getEnv("PAT_MT5_PIPE", "\\\\.\\pipe\\PredictATradeMT5"),
		UpdateChannel:  getEnv("PAT_UPDATE_CHANNEL", "STABLE"),
		AgentWSToken:   resolveAgentWSToken(),
	}
}

// resolveAgentWSToken locates the shared AGENT_WS_TOKEN the engine requires on
// the agent WebSocket upgrade. Precedence:
//   1. AGENT_WS_TOKEN / PAT_AGENT_WS_TOKEN machine env var
//   2. windows-agent.env in PAT_DATA_DIR (e.g. C:\ProgramData\PredictATrade)
//   3. windows-agent.env next to the running binary
// This lets operators drop a local windows-agent.env instead of setting a
// machine env var, without embedding the secret in the public installer.
func resolveAgentWSToken() string {
	if v := os.Getenv("AGENT_WS_TOKEN"); v != "" {
		return v
	}
	if v := os.Getenv("PAT_AGENT_WS_TOKEN"); v != "" {
		return v
	}
	dataDir := getEnv("PAT_DATA_DIR", `C:\ProgramData\PredictATrade`)
	candidates := []string{
		filepath.Join(dataDir, "windows-agent.env"),
		filepath.Join(exeDir(), "windows-agent.env"),
	}
	for _, p := range candidates {
		if v := readEnvValue(p, "AGENT_WS_TOKEN"); v != "" {
			return v
		}
	}
	return ""
}

// exeDir returns the directory containing the running executable.
func exeDir() string {
	if exe, err := os.Executable(); err == nil {
		return filepath.Dir(exe)
	}
	return "."
}

// readEnvValue parses a minimal KEY=VALUE env file and returns the value for key.
func readEnvValue(path, key string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if i := strings.Index(line, "="); i > 0 {
			if strings.TrimSpace(line[:i]) == key {
				return strings.TrimSpace(line[i+1:])
			}
		}
	}
	return ""
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// resolveWSURL reads a WebSocket URL env var and validates it is usable:
// scheme must be ws:// or wss:// and the host must be non-empty. A broken value
// such as "wss:///ws/v1/data" (empty host) or a typo would otherwise be used
// verbatim and the agent would fail to connect with no clear reason. In that
// case we log a warning and fall back to the safe default.
func resolveWSURL(key, fallback string) string {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	u, err := url.Parse(v)
	if err != nil || (u.Scheme != "wss" && u.Scheme != "ws") || u.Host == "" {
		log.Printf("[config] WARN: %s=%q is not a valid ws/wss URL (missing scheme or host) — using fallback %q", key, v, fallback)
		return fallback
	}
	return v
}
