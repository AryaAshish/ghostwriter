package services

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/ashisharyan/ghostwriter-prompt-engine/models"
)

type VoiceAnalysisResult struct {
	SuggestedScores models.PersonaScores `json:"suggested_scores"`
	LexicalProfile  models.LexicalProfile `json:"lexical_profile"`
	VoiceSummary    string               `json:"voice_summary"`
}

type VoiceAnalysisService interface {
	AnalyzeVoice(samples []string, baseline models.PersonaScores, profile *models.CreatorProfile) (*VoiceAnalysisResult, error)
}

type OpenAIVoiceAnalysisService struct {
	Client *OpenAIClient
}

func NewOpenAIVoiceAnalysisService(client *OpenAIClient) VoiceAnalysisService {
	return &OpenAIVoiceAnalysisService{Client: client}
}

func (s *OpenAIVoiceAnalysisService) AnalyzeVoice(samples []string, baseline models.PersonaScores, profile *models.CreatorProfile) (*VoiceAnalysisResult, error) {
	if s.Client == nil || s.Client.APIKey == "" {
		return nil, fmt.Errorf("OpenAI client not configured")
	}
	if len(samples) == 0 {
		return nil, fmt.Errorf("writing samples required")
	}

	systemPrompt := `You analyze creator writing samples and return JSON only.
Return this exact shape:
{
  "suggested_scores": {
    "formality": 0,
    "humor": 0,
    "energy": 0,
    "brevity": 0,
    "storytelling": 0,
    "directness": 0,
    "emotional_warmth": 0,
    "hinglish_mix": 0
  },
  "lexical_profile": {
    "signature_phrases": [],
    "filler_words": [],
    "sentence_starters": [],
    "preferred_words": [],
    "avoid_words": [],
    "slang_register": "",
    "filler_frequency": "light|moderate|heavy"
  },
  "voice_summary": ""
}
All scores must be integers from 0 to 100.`

	userPrompt := fmt.Sprintf(`Creator context:
Name: %s
Genre: %s
Language: %s
Platform: %s
Tone label: %s
Style label: %s

Baseline scores from onboarding:
%+v

Writing samples:
%s

Extract voice characteristics, slang, filler words, sentence starters, and score suggestions from the samples.`,
		profile.Name, profile.Genre, profile.Language, profile.Platform, profile.Tone, profile.Style,
		baseline,
		strings.Join(samples, "\n---\n"),
	)

	raw, err := s.Client.ChatCompletion(systemPrompt, userPrompt, 1200, 0.2)
	if err != nil {
		return nil, err
	}

	raw = extractJSON(raw)
	var result VoiceAnalysisResult
	if err := json.Unmarshal([]byte(raw), &result); err != nil {
		return nil, fmt.Errorf("failed to parse voice analysis JSON: %w", err)
	}

	result.SuggestedScores = clampScores(result.SuggestedScores)
	if result.LexicalProfile.FillerFrequency == "" {
		result.LexicalProfile.FillerFrequency = "moderate"
	}
	return &result, nil
}

func extractJSON(raw string) string {
	raw = strings.TrimSpace(raw)
	if strings.HasPrefix(raw, "```") {
		raw = strings.TrimPrefix(raw, "```json")
		raw = strings.TrimPrefix(raw, "```")
		raw = strings.TrimSuffix(raw, "```")
		raw = strings.TrimSpace(raw)
	}
	start := strings.Index(raw, "{")
	end := strings.LastIndex(raw, "}")
	if start >= 0 && end > start {
		return raw[start : end+1]
	}
	return raw
}
