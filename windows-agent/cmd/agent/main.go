package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/predictatrade/windows-agent/internal"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)
	log.Println("Predict-A-Trade Windows Agent v" + agent.AgentVersion + " starting...")

	config := agent.LoadConfig()
	a := agent.NewAgent(config)

	if err := a.Start(); err != nil {
		log.Fatalf("Failed to start agent: %v", err)
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Println("Shutting down agent...")
	a.Stop()
	log.Println("Agent stopped.")
}
