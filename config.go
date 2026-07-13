package main

import (
	"encoding/json"
	"os"
)

type WorkerNode struct {
	ID           string   `json:"id"`
	MacAddress   string   `json:"mac_address"`
	IPAddress    string   `json:"ip_address"`
	Port         int      `json:"port"`
	Capabilities []string `json:"capabilities"`
}

type Config struct {
	LLMProvider          string       `json:"llm_provider"`
	LLMAPIKey            string       `json:"llm_api_key"`
	TelegramBotToken     string       `json:"telegram_bot_token"`
	TelegramAdminChatID  int64        `json:"telegram_admin_chat_id"`
	GmailClientID        string       `json:"gmail_client_id"`
	GmailClientSecret    string       `json:"gmail_client_secret"`
	GmailRefreshToken    string       `json:"gmail_refresh_token"`
	WorkerNodes          []WorkerNode `json:"worker_nodes"`
	RSSFeeds             []string     `json:"rss_feeds"`
}

func loadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func saveConfig(path string, cfg *Config) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}
