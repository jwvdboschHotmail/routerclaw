package main

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

type TelegramBot struct {
	Token        string
	AdminID      int64
	Client       *http.Client
	Orchestrator *Orchestrator
}

func NewTelegramBot(token string, adminID int64) *TelegramBot {
	return &TelegramBot{
		Token:   token,
		AdminID: adminID,
		Client: &http.Client{
			Timeout: 60 * time.Second,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			},
		},
	}
}

// ... SendMessage and StartPolling remain unchanged ...

func (b *TelegramBot) SendMessage(text string) error {
	if b.Token == "" || b.AdminID == 0 {
		return fmt.Errorf("telegram not configured")
	}

	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", b.Token)
	payload := map[string]interface{}{
		"chat_id": b.AdminID,
		"text":    text,
	}
	body, _ := json.Marshal(payload)

	resp, err := b.Client.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("telegram API error: %d", resp.StatusCode)
	}
	return nil
}

func (b *TelegramBot) StartPolling() {
	if b.Token == "" {
		log.Println("Telegram Token is empty, skipping polling.")
		return
	}

	log.Println("Telegram polling started...")
	offset := 0

	for {
		url := fmt.Sprintf("https://api.telegram.org/bot%s/getUpdates?offset=%d&timeout=30", b.Token, offset)
		resp, err := b.Client.Get(url)
		if err != nil {
			log.Printf("Telegram polling error: %v", err)
			time.Sleep(5 * time.Second)
			continue
		}

		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		var result struct {
			Ok     bool `json:"ok"`
			Result []struct {
				UpdateID int `json:"update_id"`
				Message  struct {
					Chat struct {
						ID int64 `json:"id"`
					} `json:"chat"`
					Text string `json:"text"`
				} `json:"message"`
			} `json:"result"`
		}

		if err := json.Unmarshal(body, &result); err != nil {
			log.Printf("Failed to decode telegram updates: %v", err)
			time.Sleep(5 * time.Second)
			continue
		}

		for _, update := range result.Result {
			offset = update.UpdateID + 1
			if update.Message.Chat.ID != b.AdminID {
				continue // Ignore messages from unauthorized users
			}
			
			// Handle commands
			b.handleCommand(update.Message.Text)
		}
	}
}

func (b *TelegramBot) handleCommand(text string) {
	log.Printf("Received command: %s", text)
	
	// Temporary basic routing logic until LLM module is built
	if text == "/ping" {
		b.SendMessage("Pong! NanoClaw Orchestrator is online.")
		return
	}

	if b.Orchestrator != nil {
		go b.Orchestrator.DelegateTask(text)
		return
	}

	b.SendMessage("Orquestador no inicializado.")
}
