package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/predictatrade/windows-agent/internal"
)

func main() {
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
	// instead of NSSM. This properly signals RUNNING to the Service Control
	// Manager, eliminating the SERVICE_START_PENDING / SERVICE_STOPPED errors.
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
	log.Println("  Shutdown reason: signal received")
	log.Println("  Exit code: 0")
	a.Stop()
	log.Println("Agent stopped.")
}
