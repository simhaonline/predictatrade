//go:build !windows

package agent

import (
	"fmt"
)

// IsWindowsService reports whether the process is running as a Windows service.
// On non-Windows platforms, always returns false.
func IsWindowsService() bool {
	return false
}

// ServiceExecute runs the agent as a Windows service.
// On non-Windows platforms, returns an error.
func ServiceExecute(a *Agent) error {
	return fmt.Errorf("Windows service mode is not available on this platform")
}

// InstallService registers the agent as a Windows service.
// On non-Windows platforms, returns an error.
func InstallService(execPath string) error {
	return fmt.Errorf("Windows service installation is not available on this platform")
}

// UninstallService removes the Windows service registration.
// On non-Windows platforms, returns an error.
func UninstallService() error {
	return fmt.Errorf("Windows service uninstallation is not available on this platform")
}

// StartService starts the registered Windows service.
func StartService() error {
	return fmt.Errorf("Windows service management is not available on this platform")
}

// StopService stops the running Windows service.
func StopService() error {
	return fmt.Errorf("Windows service management is not available on this platform")
}

// ServiceDataPath returns the Windows AppData path for the agent.
func ServiceDataPath() string {
	return "/tmp/predictatrade"
}

// ServiceLogPath returns the log file path.
func ServiceLogPath() string {
	return "/tmp/predictatrade/logs"
}
