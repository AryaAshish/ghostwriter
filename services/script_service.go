package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"
)

const openAIEndpoint = "https://api.openai.com/v1/chat/completions"

// ScriptService generates scripts using OpenAI
//go:generate mockgen -destination=mock_script_service.go -package=services . ScriptService

type ScriptService interface {
	GenerateScriptFromPrompt(promptText string) (string, error)
}

type OpenAIScriptService struct {
	APIKey string
	Model  string
}

func NewOpenAIScriptService(apiKey, model string) *OpenAIScriptService {
	return &OpenAIScriptService{APIKey: apiKey, Model: model}
}

type openAIRequest struct {
	Model    string        `json:"model"`
	Messages []openAIMessage `json:"messages"`
	MaxTokens int          `json:"max_tokens,omitempty"`
	Temperature float32    `json:"temperature,omitempty"`
}

type openAIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openAIResponse struct {
	Choices []struct {
		Message openAIMessage `json:"message"`
	} `json:"choices"`
}

func (s *OpenAIScriptService) GenerateScriptFromPrompt(promptText string) (string, error) {
	client := &http.Client{Timeout: 45 * time.Second}
	body, _ := json.Marshal(openAIRequest{
		Model: s.Model,
		Messages: []openAIMessage{{Role: "user", Content: promptText}},
		MaxTokens: 800,
		Temperature: 0.7,
	})
	var lastErr error
	for attempt := 1; attempt <= 3; attempt++ {
		req, err := http.NewRequest("POST", openAIEndpoint, bytes.NewBuffer(body))
		if err != nil {
			return "", err
		}
		req.Header.Set("Authorization", "Bearer "+s.APIKey)
		req.Header.Set("Content-Type", "application/json")
		resp, err := client.Do(req)
		if err != nil {
			lastErr = err
			time.Sleep(500 * time.Millisecond)
			continue
		}
		defer resp.Body.Close()
		respBody, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			lastErr = err
			time.Sleep(500 * time.Millisecond)
			continue
		}
		if resp.StatusCode >= 500 && resp.StatusCode < 600 {
			lastErr = fmt.Errorf("OpenAI API server error: %s", string(respBody))
			time.Sleep(500 * time.Millisecond)
			continue
		}
		if resp.StatusCode != 200 {
			return "", fmt.Errorf("OpenAI API error: %s", string(respBody))
		}
		var aiResp openAIResponse
		if err := json.Unmarshal(respBody, &aiResp); err != nil {
			return "", err
		}
		if len(aiResp.Choices) == 0 {
			return "", fmt.Errorf("No choices returned from OpenAI")
		}
		return aiResp.Choices[0].Message.Content, nil
	}
	return "", fmt.Errorf("OpenAI API call failed after 3 attempts: %v", lastErr)
}
