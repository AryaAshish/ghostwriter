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

type OpenAIClient struct {
	APIKey     string
	Model      string
	BaseURL    string
	HTTPClient *http.Client
}

func NewOpenAIClient(apiKey, model string) *OpenAIClient {
	return &OpenAIClient{APIKey: apiKey, Model: model}
}

type openAIChatRequest struct {
	Model       string          `json:"model"`
	Messages    []openAIMessage `json:"messages"`
	MaxTokens   int             `json:"max_tokens,omitempty"`
	Temperature float32         `json:"temperature,omitempty"`
}

type openAIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openAIChatResponse struct {
	Choices []struct {
		Message openAIMessage `json:"message"`
	} `json:"choices"`
}

func (c *OpenAIClient) chatURL() string {
	if c.BaseURL != "" {
		return c.BaseURL
	}
	return openAIEndpoint
}

func (c *OpenAIClient) httpClient() *http.Client {
	if c.HTTPClient != nil {
		return c.HTTPClient
	}
	return &http.Client{Timeout: 45 * time.Second}
}

func (c *OpenAIClient) ChatCompletion(systemPrompt, userPrompt string, maxTokens int, temperature float32) (string, error) {
	return c.ChatCompletionMessages([]openAIMessage{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: userPrompt},
	}, maxTokens, temperature)
}

func (c *OpenAIClient) ChatCompletionMessages(messages []openAIMessage, maxTokens int, temperature float32) (string, error) {
	body, _ := json.Marshal(openAIChatRequest{
		Model:       c.Model,
		Messages:    messages,
		MaxTokens:   maxTokens,
		Temperature: temperature,
	})
	return c.postChat(body)
}

func (c *OpenAIClient) postChat(body []byte) (string, error) {
	client := c.httpClient()
	var lastErr error
	for attempt := 1; attempt <= 3; attempt++ {
		req, err := http.NewRequest("POST", c.chatURL(), bytes.NewBuffer(body))
		if err != nil {
			return "", err
		}
		req.Header.Set("Authorization", "Bearer "+c.APIKey)
		req.Header.Set("Content-Type", "application/json")

		resp, err := client.Do(req)
		if err != nil {
			lastErr = err
			time.Sleep(50 * time.Millisecond)
			continue
		}

		respBody, readErr := ioutil.ReadAll(resp.Body)
		resp.Body.Close()
		if readErr != nil {
			lastErr = readErr
			time.Sleep(50 * time.Millisecond)
			continue
		}
		if resp.StatusCode >= 500 && resp.StatusCode < 600 {
			lastErr = fmt.Errorf("OpenAI API server error: %s", string(respBody))
			time.Sleep(50 * time.Millisecond)
			continue
		}
		if resp.StatusCode != 200 {
			return "", fmt.Errorf("OpenAI API error: %s", string(respBody))
		}

		var aiResp openAIChatResponse
		if err := json.Unmarshal(respBody, &aiResp); err != nil {
			return "", err
		}
		if len(aiResp.Choices) == 0 {
			return "", fmt.Errorf("no choices returned from OpenAI")
		}
		return aiResp.Choices[0].Message.Content, nil
	}
	return "", fmt.Errorf("OpenAI API call failed after 3 attempts: %v", lastErr)
}
