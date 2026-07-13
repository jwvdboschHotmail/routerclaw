package main

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type LLMClient struct {
	Provider string
	APIKey   string
	Client   *http.Client
}

func NewLLMClient(provider, apiKey string) *LLMClient {
	return &LLMClient{
		Provider: provider,
		APIKey:   apiKey,
		Client: &http.Client{
			Timeout: 60 * time.Second,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			},
		},
	}
}

// GenerateText sends a prompt to the LLM and returns the text response
func (l *LLMClient) GenerateText(systemPrompt, userPrompt string) (string, error) {
	if l.APIKey == "" {
		return "", fmt.Errorf("llm api key is empty")
	}

	if l.Provider == "anthropic" {
		return l.callAnthropic(systemPrompt, userPrompt)
	}
	
	if l.Provider == "deepseek" {
		return l.callOpenAICompatible("https://api.deepseek.com/chat/completions", "deepseek-chat", systemPrompt, userPrompt)
	}

	// Default to OpenAI
	return l.callOpenAICompatible("https://api.openai.com/v1/chat/completions", "gpt-4o", systemPrompt, userPrompt)
}

func (l *LLMClient) callOpenAICompatible(url, model, systemPrompt, userPrompt string) (string, error) {
	payload := map[string]interface{}{
		"model": model,
		"messages": []map[string]string{
			{"role": "system", "content": systemPrompt},
			{"role": "user", "content": userPrompt},
		},
	}
	
	body, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+l.APIKey)

	resp, err := l.Client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("openai error: %s", string(respBody))
	}

	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}

	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", err
	}
	
	if len(result.Choices) > 0 {
		return result.Choices[0].Message.Content, nil
	}
	return "", fmt.Errorf("empty response from openai")
}

func (l *LLMClient) callAnthropic(systemPrompt, userPrompt string) (string, error) {
	url := "https://api.anthropic.com/v1/messages"
	
	payload := map[string]interface{}{
		"model": "claude-3-5-sonnet-20240620",
		"max_tokens": 1024,
		"system": systemPrompt,
		"messages": []map[string]string{
			{"role": "user", "content": userPrompt},
		},
	}
	
	body, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", l.APIKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := l.Client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("anthropic error: %s", string(respBody))
	}

	var result struct {
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
	}

	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", err
	}
	
	if len(result.Content) > 0 {
		return result.Content[0].Text, nil
	}
	return "", fmt.Errorf("empty response from anthropic")
}
