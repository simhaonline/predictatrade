package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"runtime/debug"
	"syscall"
	"time"

	"github.com/predictatrade/windows-agent/internal"
)

func main() {
	// Set up file logging FIRST. In Windows Service mode, os.Stderr is nil.
	// If we can't open a log file, use io.Discard so log writes never panic.
	// The installer sets PAT_LOG_DIR to the role's monitored logs folder
	// (e.g. C:\PredictATrade\Client\logs) so the agent's log lands exactly
	// where the installer / operator look.
	logDir := os.Getenv("PAT_LOG_DIR")
	if logDir == "" {
		logDir = filepath.Join(os.Getenv("PROGRAMDATA"), "PredictATrade", "logs")
	}
	os.MkdirAll(logDir, 0755)
	logFile, err := os.OpenFile(filepath.Join(logDir, "agent.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		// Can't open file — use discard writer so log.Println never panics
		log.SetOutput(io.Discard)
	} else {
		log.SetOutput(logFile)
		defer logFile.Close()
	}

	defer func() {
		if r := recover(); r != nil {
			log.Printf("[panic] client agent process crashed during startup: %v\n%s", r, debug.Stack())
			os.Exit(1)
		}
	}()

	log.SetFlags(log.LstdFlags | log.Lmicroseconds)
	log.Println("Predict-A-Trade Windows Client Agent v" + agent.AgentVersion + " starting...")

	config := agent.LoadConfig()
	// Role is fixed by this binary (Client / execution). The --mode flag is no
	// longer used; it exists only for compatibility and is ignored.
	_ = flag.String("mode", "", "ignored — role is fixed to client by this binary")
	versionFlag := flag.Bool("version", false, "print version and exit (used by installer warm-up)")
	flag.Parse()
	if *versionFlag {
		fmt.Println(agent.AgentVersion)
		os.Exit(0)
	}
	a := agent.NewClientAgent(config)

	// Always run in interactive mode. NSSM wraps the process as a Windows
	// service — the agent doesn't need to use svc.Run() (which is fragile
	// and crashes if anything goes wrong during initialization).
	// NSSM handles: start, stop, restart, auto-recovery.
	if err := a.Start(); err != nil {
		log.Fatalf("Failed to start agent: %v", err)
	}

	log.Println("Client Agent started successfully — waiting for signals")

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	startTime := time.Now()
	sig := <-sigChan
	// Ignore signals that arrive during the startup grace period. A freshly
	// started service can receive a spurious early stop (e.g. a console Ctrl-C
	// from the launching process tree, or a service-manager control sent before
	// the agent has fully registered). Dying on that signal made the installer's
	// self-healing never see the service come up. After the grace window we stop
	// normally on the next signal.
	if time.Since(startTime) < 15*time.Second {
		log.Printf("WARN: ignoring early signal %v (startup grace <15s); continuing", sig)
		sig = <-sigChan
	}
	log.Printf("Client Agent stopping (signal %v)", sig)
	a.Stop()
	log.Println("Client Agent stopped.")
}
