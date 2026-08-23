//go:build windows

package agent

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/mgr"
)

// WindowsService implements the svc.Handler interface for Windows Service support.
// SOW Section 21: Native Windows Service support — install, uninstall, start, stop, restart.
type WindowsService struct {
	agent *Agent
}

// IsWindowsService reports whether the process is running as a Windows service.
func IsWindowsService() bool {
	isService, err := svc.IsWindowsService()
	if err != nil {
		return false
	}
	return isService
}

// ServiceExecute runs the agent as a Windows service.
func ServiceExecute(a *Agent) error {
	return svc.Run("pat-agent", &WindowsService{agent: a})
}

func (s *WindowsService) Execute(args []string, r <-chan svc.ChangeRequest, changes chan<- svc.Status) (ssec bool, errno uint32) {
	changes <- svc.Status{State: svc.StartPending}
	
	if err := s.agent.Start(); err != nil {
		log.Printf("Agent start failed: %v", err)
		return false, 1
	}
	
	changes <- svc.Status{State: svc.Running, Accepts: svc.AcceptStop | svc.AcceptShutdown}
	
	for {
		select {
		case c := <-r:
			switch c.Cmd {
			case svc.Interrogate:
				changes <- c.CurrentStatus
			case svc.Stop, svc.Shutdown:
				changes <- svc.Status{State: svc.StopPending}
				s.agent.Stop()
				changes <- svc.Status{State: svc.Stopped}
				return false, 0
			default:
				log.Printf("Unknown service command: %d", c.Cmd)
			}
		case <-time.After(30 * time.Second):
			// Periodic health check — service is still running
		}
	}
}

// InstallService registers the agent as a Windows service.
func InstallService(execPath string) error {
	m, err := mgr.Connect()
	if err != nil {
		return fmt.Errorf("failed to connect to service manager: %w", err)
	}
	defer m.Disconnect()

	s, err := m.OpenService("pat-agent")
	if err == nil {
		s.Close()
		return fmt.Errorf("service already exists")
	}

	s, err = m.CreateService("pat-agent", execPath, mgr.Config{
		DisplayName:    "Predict-A-Trade Agent",
		Description:    "Predict-A-Trade Windows Agent — MT4/MT5 bridge and signal delivery",
		StartType:      mgr.StartAutomatic,
		ErrorControl:   mgr.ErrorNormal,
		ServiceType:    0x00000010, // SERVICE_WIN32_OWN_PROCESS
	})
	if err != nil {
		return fmt.Errorf("failed to create service: %w", err)
	}
	defer s.Close()

	// Configure recovery actions: restart on failure
	err = s.SetRecoveryActions([]mgr.RecoveryAction{
		{Type: 1, Delay: 30 * time.Second},  // SC_ACTION_RESTART
		{Type: 1, Delay: 60 * time.Second},  // SC_ACTION_RESTART
		{Type: 1, Delay: 120 * time.Second}, // SC_ACTION_RESTART
	}, 86400) // Reset period: 24 hours
	if err != nil {
		log.Printf("Warning: failed to set recovery actions: %v", err)
	}

	log.Println("Service 'pat-agent' installed successfully")
	return nil
}

// UninstallService removes the Windows service registration.
func UninstallService() error {
	m, err := mgr.Connect()
	if err != nil {
		return fmt.Errorf("failed to connect to service manager: %w", err)
	}
	defer m.Disconnect()

	s, err := m.OpenService("pat-agent")
	if err != nil {
		return fmt.Errorf("service not found: %w", err)
	}
	defer s.Close()

	err = s.Delete()
	if err != nil {
		return fmt.Errorf("failed to delete service: %w", err)
	}

	log.Println("Service 'pat-agent' uninstalled successfully")
	return nil
}

// StartService starts the registered Windows service.
func StartService() error {
	m, err := mgr.Connect()
	if err != nil {
		return fmt.Errorf("failed to connect to service manager: %w", err)
	}
	defer m.Disconnect()

	s, err := m.OpenService("pat-agent")
	if err != nil {
		return fmt.Errorf("service not found: %w", err)
	}
	defer s.Close()

	err = s.Start()
	if err != nil {
		return fmt.Errorf("failed to start service: %w", err)
	}

	log.Println("Service 'pat-agent' started successfully")
	return nil
}

// StopService stops the running Windows service.
func StopService() error {
	m, err := mgr.Connect()
	if err != nil {
		return fmt.Errorf("failed to connect to service manager: %w", err)
	}
	defer m.Disconnect()

	s, err := m.OpenService("pat-agent")
	if err != nil {
		return fmt.Errorf("service not found: %w", err)
	}
	defer s.Close()

	_, err = s.Control(svc.Stop)
	if err != nil {
		return fmt.Errorf("failed to stop service: %w", err)
	}

	log.Println("Service 'pat-agent' stopped successfully")
	return nil
}

// ServiceDataPath returns the Windows AppData path for the agent.
func ServiceDataPath() string {
	progData := os.Getenv("PROGRAMDATA")
	if progData == "" {
		progData = "C:\\ProgramData"
	}
	return filepath.Join(progData, "PredictATrade")
}

// ServiceLogPath returns the log file path.
func ServiceLogPath() string {
	return filepath.Join(ServiceDataPath(), "logs")
}
