package services

import (
	"fmt"
	"github.com/ashisharyan/ghostwriter-prompt-engine/models"
)

type PromptService interface {
	GeneratePersonaSummary(profile *models.CreatorProfile) string
	GeneratePrompt(profile *models.CreatorProfile, topic string) (string, error)
}

type DefaultPromptService struct {
	PromptRepo PromptRepository
}

func NewDefaultPromptService(promptRepo PromptRepository) PromptService {
	return &DefaultPromptService{PromptRepo: promptRepo}
}

func (s *DefaultPromptService) GeneratePersonaSummary(profile *models.CreatorProfile) string {
	return fmt.Sprintf("%s is a %s creator from %s, making %s content in %s, known for %s. Tone: %s. Style: %s.",
		profile.Name, profile.Genre, profile.Region, profile.ContentType, profile.Language, profile.USP, profile.Tone, profile.Style)
}

func (s *DefaultPromptService) GeneratePrompt(profile *models.CreatorProfile, topic string) (string, error) {
	persona := s.GeneratePersonaSummary(profile)
	prompt := fmt.Sprintf("You are %s\nTopic: %s\nWrite a script as if you are this creator, in their tone and style.", persona, topic)
	if s.PromptRepo != nil {
		dbPrompt := &models.Prompt{
			CreatorID:  fmt.Sprintf("%v", profile.ID),
			Topic:      topic,
			Variant:    "base",
			PromptText: prompt,
		}
		if err := s.PromptRepo.SavePrompt(dbPrompt); err != nil {
			return prompt, err
		}
	}
	return prompt, nil
}

