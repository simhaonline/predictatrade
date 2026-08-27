package agent

import "os"

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
}

func LoadConfig() *Config {
	liveWS := getEnv("PAT_LIVE_WS_URL", "wss://live.predictatrade.com/ws/v1/agent")
	dataWS := getEnv("PAT_DATA_WS_URL", "wss://api.predictatrade.com/ws/v1/data")
	apiURL := getEnv("PAT_API_URL", "https://api.predictatrade.com/api/v1")
	mode := getEnv("PAT_AGENT_MODE", "exec")
	dataDir := getEnv("PAT_DATA_DIR", "C:\\ProgramData\\PredictATrade")

	if os.Getenv("PAT_DEV_MODE") == "1" {
		liveWS = getEnv("PAT_SERVER_URL", "ws://127.0.0.1:13081/ws")
		dataWS = getEnv("PAT_DATA_WS_URL", "ws://127.0.0.1:13091/ws/v1/data")
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
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
