package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	fmt.Println("Starting RouterClaw Minimal Agent...")

	cfg, err := loadConfig("/root/config.json")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Initialize modules
	telegram := NewTelegramBot(cfg.TelegramBotToken, cfg.TelegramAdminChatID)
	llm := NewLLMClient(cfg.LLMProvider, cfg.LLMAPIKey)
	googleWs := NewGoogleWorkspace(cfg.GmailClientID, cfg.GmailClientSecret, cfg.GmailRefreshToken)
	webBrowser := NewWebBrowser()
	orchestrator := NewOrchestrator("/root/config.json", cfg.WorkerNodes, telegram, llm, googleWs, webBrowser)

	// Link telegram to orchestrator & google workspace (we'll expand this later)
	telegram.Orchestrator = orchestrator

	// Start background processes
	go telegram.StartPolling()
	go StartInternetMonitor(telegram)

	backgroundMonitor := NewBackgroundMonitor(orchestrator)
	backgroundMonitor.Start()

	// Wait for termination signal
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c

	fmt.Println("Shutting down RouterClaw...")
}
