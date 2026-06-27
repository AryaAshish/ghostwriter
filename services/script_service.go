package services

import "fmt"

// ScriptService generates scripts using OpenAI.
type ScriptService interface {
	GenerateScriptFromPrompt(systemPrompt, userPrompt string) (string, error)
}

type OpenAIScriptService struct {
	Client *OpenAIClient
}

func NewOpenAIScriptService(client *OpenAIClient) ScriptService {
	return &OpenAIScriptService{Client: client}
}

func (s *OpenAIScriptService) GenerateScriptFromPrompt(systemPrompt, userPrompt string) (string, error) {
	if s.Client == nil || s.Client.APIKey == "" {
		return "", fmt.Errorf("OpenAI client not configured")
	}
	if systemPrompt == "" {
		return s.Client.ChatCompletionMessages([]openAIMessage{{Role: "user", Content: userPrompt}}, 800, 0.7)
	}
	return s.Client.ChatCompletion(systemPrompt, userPrompt, 800, 0.7)
}
