// Package agentlib provides the Windows Agent's hardware fingerprint and telemetry
// collection used to bind licenses to a device and detect misuse. It is best-effort
// and cross-platform (Linux/dev builds fall back to stable host identifiers).
package agentlib

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

// Fingerprint is the device identity sent to the backend for license binding.
type Fingerprint struct {
	DeviceID       string `json:"device_id"`
	Fingerprint    string `json:"fingerprint"`    // stable hash of components
	Components     string `json:"components"`     // json of raw identifiers
	InstallationID string `json:"installation_id"`
	Hostname       string `json:"hostname"`
	OS             string `json:"os"`
}

// Collect gathers a stable device fingerprint. On Windows it enriches with WMI
// (MachineGuid, disk/board serials) when available; otherwise it uses host/MAC.
func Collect() Fingerprint {
	host, _ := os.Hostname()
	osName := runtime.GOOS
	mac := firstMAC()

	comp := map[string]string{
		"hostname": host,
		"os":       osName,
		"arch":     runtime.GOARCH,
		"mac":      mac,
	}
	if runtime.GOOS == "windows" {
		if v := wmic("csproduct", "UUID"); v != "" {
			comp["system_uuid"] = v
		}
		if v := wmic("baseboard", "SerialNumber"); v != "" {
			comp["board_serial"] = v
		}
		if v := wmic("diskdrive", "SerialNumber"); v != "" {
			comp["disk_serial"] = v
		}
		if v := regMachineGuid(); v != "" {
			comp["machine_guid"] = v
		}
	}
	cb, _ := json.Marshal(comp)
	hash := sha256.Sum256(cb)
	fpHex := hex.EncodeToString(hash[:])

	// InstallationID = stable per-host id (host+mac), persisted if possible.
	inst := fpHex[:16]

	return Fingerprint{
		DeviceID:       fpHex,
		Fingerprint:    fpHex,
		Components:     string(cb),
		InstallationID: inst,
		Hostname:       host,
		OS:             osName,
	}
}

func firstMAC() string {
	ifas, err := net.Interfaces()
	if err != nil {
		return ""
	}
	for _, ifa := range ifas {
		if ifa.Flags&net.FlagLoopback != 0 || ifa.Flags&net.FlagUp == 0 {
			continue
		}
		addr := ifa.HardwareAddr.String()
		if addr != "" {
			return addr
		}
	}
	return ""
}

func wmic(class, field string) string {
	out, err := exec.Command("wmic", class, "get", field, "/value").Output()
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, field+"=") {
			return strings.TrimSpace(strings.TrimPrefix(line, field+"="))
		}
	}
	return ""
}

func regMachineGuid() string {
	out, err := exec.Command("reg", "query",
		`HKLM\SOFTWARE\Microsoft\Cryptography`, "/v", "MachineGuid").Output()
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(out), "\n") {
		if strings.Contains(line, "MachineGuid") {
			parts := strings.Fields(line)
			return parts[len(parts)-1]
		}
	}
	return ""
}
