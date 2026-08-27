package main

import (
	"flag"
	"io"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/predictatrade/windows-agent/internal"
)

func main() {
	// Set up file logging FIRST. In Windows Service mode, os.Stderr is nil.
	// If we can't open a log file, use io.Discard so log writes never panic.
	logDir := filepath.Join(os.Getenv("PROGRAMDATA"), "PredictATrade", "logs")
	os.MkdirAll(logDir, 0755)
	logFile, err := os.OpenFile(filepath.Join(logDir, "master_agent.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		// Can't open file — use discard writer so log.Println never panics
		log.SetOutput(io.Discard)
	} else {
		log.SetOutput(logFile)
		defer logFile.Close()
	}

	defer func() {
		if r := recover(); r != nil {
			log.Printf("[panic] master agent process recovered: %v", r)
		}
	}()

	log.SetFlags(log.LstdFlags | log.Lmicroseconds)
	log.Println("Predict-A-Trade Windows Master Node (data) v" + agent.AgentVersion + " starting...")

	config := agent.LoadConfig()
	// Role is fixed by this binary (Master Node / data-only). The --mode flag is
	// no longer used; it exists only for compatibility and is ignored.
	_ = flag.String("mode", "", "ignored — role is fixed to master (data) by this binary")
	flag.Parse()
	a := agent.NewMasterAgent(config)

	// Always run in interactive mode. NSSM wraps the process as a Windows
	// service — the agent doesn't need to use svc.Run() (which is fragile
	// and crashes if anything goes wrong during initialization).
	// NSSM handles: start, stop, restart, auto-recovery.
	if err := a.Start(); err != nil {
		log.Fatalf("Failed to start agent: %v", err)
	}

	log.Println("Master Node started successfully — waiting for signals")

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Println("Master Node stopping")
	a.Stop()
	log.Println("Master Node stopped.")
}
