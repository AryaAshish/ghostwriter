package services

import (
	"fmt"
	"github.com/ashisharyan/ghostwriter-prompt-engine/models"
)

type PromptABService interface {
	GeneratePromptVariants(profile *models.CreatorProfile, topic string) (map[string]string, string)
	StorePromptVariants(variants map[string]string, profile *models.CreatorProfile, topic string) error
}

type DefaultPromptABService struct {
	PersonaBuilder PromptService
	PromptRepo     PromptRepository
}

func NewDefaultPromptABService(personaBuilder PromptService, repo PromptRepository) PromptABService {
	return &DefaultPromptABService{PersonaBuilder: personaBuilder, PromptRepo: repo}
}

func (s *DefaultPromptABService) GeneratePromptVariants(profile *models.CreatorProfile, topic string) (map[string]string, string) {
	persona := s.PersonaBuilder.GeneratePersonaSummary(profile)
	variants := map[string]string{
		"A": fmt.Sprintf("You are %s\nTopic: %s\nWrite a balanced, engaging script as this creator.", persona, topic),
		"B": fmt.Sprintf("You are %s\nTopic: %s\nWrite a script as this creator, with extra punchlines and witty hooks.", persona, topic),
		"C": fmt.Sprintf("You are %s\nTopic: %s\nWrite a script as this creator, focusing on storytelling and emotional depth.", persona, topic),
	}
	return variants, persona
}

func (s *DefaultPromptABService) StorePromptVariants(variants map[string]string, profile *models.CreatorProfile, topic string) error {
	if s.PromptRepo == nil {
		return nil // No-op if not configured
	}
	for variant, text := range variants {
		prompt := &models.Prompt{
			CreatorID:  fmt.Sprintf("%v", profile.ID),
			Topic:      topic,
			Variant:    variant,
			PromptText: text,
		}
		if err := s.PromptRepo.SavePrompt(prompt); err != nil {
			return err
		}
	}
	return nil
}
