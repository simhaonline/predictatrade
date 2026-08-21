package agent

import (
	"log"

	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
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
// On non-Windows (dev), generates synthetic values.
// On Windows, reads from registry/WMI. If hardware IDs are unavailable,
// falls back to installation_id + hostname for a stable fingerprint.
func CollectFingerprint(dataDir string) *HardwareFingerprint {
	hostname, _ := os.Hostname()
	fp := &HardwareFingerprint{
		OS:       runtime.GOOS,
		Hostname: hostname,
	}

	if runtime.GOOS == "windows" {
		// Windows: read from registry/WMI
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

	// Fallback: if hardware IDs are empty, use hostname + installation_id
	// This ensures a non-empty fingerprint hash even when WMI/registry fails
	if fp.MachineGUID == "" && fp.SystemUUID == "" && fp.Motherboard == "" && fp.Disk == "" {
		fp.MachineGUID = hashStr("hw-fallback-" + hostname + "-" + fp.InstallationID)
		log.Printf("WARNING: Hardware IDs unavailable, using fallback fingerprint (hostname + installation_id)")
	}

	return fp
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

// --- Windows-specific helpers (stubs on non-Windows) ---

func readRegistry(keyPath, valueName string) string {
	// On Windows: use golang.org/x/sys/windows/registry
	// For now, return empty (agent will use installation_id as fallback)
	return ""
}

func getSystemUUID() string {
	// On Windows: WMI SELECT UUID FROM Win32_ComputerSystemProduct
	return ""
}

func getMotherboardID() string {
	// On Windows: WMI SELECT SerialNumber FROM Win32_BaseBoard
	return ""
}

func getDiskID() string {
	// On Windows: WMI SELECT SerialNumber FROM Win32_DiskDrive
	return ""
}
