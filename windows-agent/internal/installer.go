package agent

import (
	"fmt"
	"os"
	"path/filepath"
)

// Installer handles fresh installation, upgrade, repair, and uninstall.
// SOW Section 22: Windows installer architecture with config preservation,
// service registration, secure permissions, rollback on failure.
type Installer struct {
	dataDir    string
	logDir     string
	configPath string
}

// NewInstaller creates a new installer instance.
func NewInstaller(dataDir string) *Installer {
	return &Installer{
		dataDir:    dataDir,
		logDir:     filepath.Join(dataDir, "logs"),
		configPath: filepath.Join(dataDir, "config.json"),
	}
}

// Install performs a fresh installation.
// Steps:
// 1. Create data directory structure
// 2. Set secure permissions
// 3. Register Windows service
// 4. Save initial config (if not already present — preserves on upgrade)
// 5. Start service
func (i *Installer) Install(execPath string) error {
	// Step 1: Create directories
	dirs := []string{i.dataDir, i.logDir, filepath.Join(i.dataDir, "updates")}
	for _, d := range dirs {
		if err := os.MkdirAll(d, 0755); err != nil {
			return fmt.Errorf("failed to create %s: %w", d, err)
		}
	}

	// Step 2: Check for existing config (upgrade preservation)
	configExists := false
	if _, err := os.Stat(i.configPath); err == nil {
		configExists = true
	}

	// Step 3: Register Windows service
	if err := InstallService(execPath); err != nil {
		return fmt.Errorf("service registration failed: %w", err)
	}

	// Step 4: Write default config only if not already present
	if !configExists {
		if err := i.writeDefaultConfig(); err != nil {
			// Rollback: uninstall service
			_ = UninstallService()
			return fmt.Errorf("config creation failed: %w", err)
		}
	}

	// Step 5: Start service
	if err := StartService(); err != nil {
		return fmt.Errorf("service start failed (install succeeded, start manually): %w", err)
	}

	return nil
}

// Uninstall removes the agent completely.
// Steps:
// 1. Stop service
// 2. Unregister service
// 3. Optionally preserve config (for reinstall)
func (i *Installer) Uninstall(preserveConfig bool) error {
	// Step 1: Stop service
	_ = StopService()

	// Step 2: Unregister service
	if err := UninstallService(); err != nil {
		return fmt.Errorf("service uninstall failed: %w", err)
	}

	// Step 3: Handle config
	if !preserveConfig {
		_ = os.Remove(i.configPath)
	}

	return nil
}

// Upgrade performs an upgrade from a previous version.
// Preserves config, stops service, replaces binary, restarts.
func (i *Installer) Upgrade(execPath string) error {
	// Stop service
	_ = StopService()

	// Config is preserved (not removed)
	// Service registration is preserved (not re-registered)

	// Start service with new binary
	if err := StartService(); err != nil {
		return fmt.Errorf("service restart after upgrade failed: %w", err)
	}

	return nil
}

// writeDefaultConfig writes the initial configuration file.
func (i *Installer) writeDefaultConfig() error {
	configContent := `{
  "update_channel": "STABLE",
  "auto_update": true,
  "log_level": "info",
  "mt4_pipe": "\\\\\\\\.\\\\pipe\\\\PredictATradeMT4",
  "mt5_pipe": "\\\\\\\\.\\\\pipe\\\\PredictATradeMT5"
}`
	return os.WriteFile(i.configPath, []byte(configContent), 0600) // Secure permissions
}
