package agent

import (
	"log"
	"os/exec"
	"runtime"
	"strings"

	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
)

// HardwareFingerprint collects privacy-aware composite device identity.
// Individual components are HMAC-hashed with a server pepper (never raw identifiers stored).
type HardwareFingerprint struct {
	MachineGUID     string `json:"machine_guid"`
	SystemUUID      string `json:"system_uuid"`
	Motherboard     string `json:"motherboard"`
	Disk            string `json:"disk"`
	InstallationID  string `json:"installation_id"`
	OS              string `json:"os"`
	Hostname        string `json:"hostname"`
}

// CollectFingerprint gathers hardware identifiers from the OS.
// On non-Windows (dev), generates synthetic values so the agent still runs.
// On Windows, reads real hardware IDs via WMI/registry. If hardware IDs are
// unavailable on Windows (the intended production platform), it returns an
// error so device binding cannot be trivially copied/empty (W10).
func CollectFingerprint(dataDir string) (*HardwareFingerprint, error) {
	hostname, _ := os.Hostname()
	fp := &HardwareFingerprint{
		OS:       runtime.GOOS,
		Hostname: hostname,
	}

	if runtime.GOOS == "windows" {
		// Windows: read real hardware identifiers.
		fp.MachineGUID = readRegistry("SOFTWARE\\Microsoft\\Cryptography", "MachineGuid")
		fp.SystemUUID = getSystemUUID()
		fp.Motherboard = getMotherboardID()
		fp.Disk = getDiskID()
	} else {
		// Dev: generate stable synthetic values based on hostname
		fp.MachineGUID = hashStr("dev-machine-guid-" + hostname)
		fp.SystemUUID = hashStr("dev-uuid-" + hostname)
		fp.Motherboard = hashStr("dev-mb-" + hostname)
		fp.Disk = hashStr("dev-disk-" + hostname)
	}

	// Load or create installation ID (persists across restarts)
	fp.InstallationID = loadOrCreateInstallationID(dataDir)

	// On the production platform, hardware IDs MUST be collectable. If every
	// hardware source is empty, device binding would be trivially copyable —
	// fail hard so the agent does not register a meaningless fingerprint.
	if runtime.GOOS == "windows" &&
		fp.MachineGUID == "" && fp.SystemUUID == "" && fp.Motherboard == "" && fp.Disk == "" {
		return nil, fmt.Errorf("hardware fingerprint collection failed: no BIOS UUID, motherboard serial, disk serial or machine GUID available")
	}

	// Fallback (non-Windows / dev): if hardware IDs are empty, use hostname +
	// installation_id so the fingerprint hash is still stable.
	if fp.MachineGUID == "" && fp.SystemUUID == "" && fp.Motherboard == "" && fp.Disk == "" {
		fp.MachineGUID = hashStr("hw-fallback-" + hostname + "-" + fp.InstallationID)
		log.Printf("WARNING: Hardware IDs unavailable, using fallback fingerprint (hostname + installation_id)")
	}

	return fp, nil
}

// runCmd executes a command and returns trimmed stdout, or "" on any error.
// Used to read Windows hardware identifiers. Safe to compile on non-Windows
// (the callers are runtime-guarded to Windows only).
func runCmd(name string, args ...string) string {
	if runtime.GOOS != "windows" {
		return ""
	}
	cmd := exec.Command(name, args...)
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// ComputeHash returns a SHA256 hash of the fingerprint components.
// This is the stable hardware identity that binds a license to a machine.
func (fp *HardwareFingerprint) ComputeHash() string {
	combined := fp.MachineGUID + "|" + fp.SystemUUID + "|" + fp.Motherboard + "|" + fp.Disk + "|" + fp.InstallationID
	h := sha256.Sum256([]byte(combined))
	return hex.EncodeToString(h[:])
}

// loadOrCreateInstallationID reads or generates a per-installation UUID.
func loadOrCreateInstallationID(dataDir string) string {
	installPath := filepath.Join(dataDir, "installation.id")
	data, err := os.ReadFile(installPath)
	if err == nil && len(data) > 0 {
		return string(data)
	}
	// Generate new
	b := make([]byte, 16)
	rand.Read(b)
	installID := fmt.Sprintf("PAT-%s", hex.EncodeToString(b))
	os.MkdirAll(dataDir, 0755)
	os.WriteFile(installPath, []byte(installID), 0644)
	return installID
}

// hashStr returns a hex SHA256 hash (for dev synthetic values).
func hashStr(s string) string {
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:])
}

// HMACWithPepper computes HMAC-SHA256 with a server pepper.
func HMACWithPepper(pepper, value string) string {
	mac := hmac.New(sha256.New, []byte(pepper))
	mac.Write([]byte(value))
	return hex.EncodeToString(mac.Sum(nil))
}

// --- Windows-specific helpers (real reads; no-ops on non-Windows) ---

func readRegistry(keyPath, valueName string) string {
	if runtime.GOOS != "windows" {
		return ""
	}
	// Query the registry via reg.exe (avoids importing the windows-only
	// golang.org/x/sys/windows/registry package so the agent still builds on Linux).
	return runCmd("reg", "query", keyPath, "/v", valueName)
}

func getSystemUUID() string {
	// WMI: UUID from Win32_ComputerSystemProduct
	return runCmd("wmic", "csproduct", "get", "UUID")
}

func getMotherboardID() string {
	// WMI: SerialNumber from Win32_BaseBoard
	return runCmd("wmic", "baseboard", "get", "SerialNumber")
}

func getDiskID() string {
	// WMI: SerialNumber from Win32_DiskDrive
	return runCmd("wmic", "diskdrive", "get", "SerialNumber")
}
