package main

import (
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/predictatrade/windows-agent/internal"
)

func main() {
	// CRITICAL: Set up file logging BEFORE anything else.
	// In Windows Service mode, os.Stdout and os.Stderr are nil — any log
	// write to them causes a panic → process crashes → "Cannot start service".
	logDir := os.Getenv("PROGRAMDATA")
	if logDir == "" {
		logDir = "C:\\ProgramData"
	}
	logDir = filepath.Join(logDir, "PredictATrade", "logs")
	os.MkdirAll(logDir, 0755)
	logFile, err := os.OpenFile(filepath.Join(logDir, "agent.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err == nil {
		log.SetOutput(logFile)
		defer logFile.Close()
	}

	defer func() {
		if r := recover(); r != nil {
			log.Printf("[panic] agent process recovered: %v", r)
		}
	}()

	log.SetFlags(log.LstdFlags | log.Lmicroseconds)
	log.Println("Predict-A-Trade Windows Agent v" + agent.AgentVersion + " starting...")

	config := agent.LoadConfig()
	a := agent.NewAgent(config)

	// Check if running as a Windows Service — if so, use native SCM protocol
	if agent.IsWindowsService() {
		log.Println("Running as Windows Service — using native SCM protocol")
		if err := agent.ServiceExecute(a); err != nil {
			log.Fatalf("Service execution failed: %v", err)
		}
		return
	}

	// Interactive mode (double-click, command line, debug)
	log.Println("Running in interactive mode")
	if err := a.Start(); err != nil {
		log.Fatalf("Failed to start agent: %v", err)
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Println("Agent stopping")
	a.Stop()
	log.Println("Agent stopped.")
}
