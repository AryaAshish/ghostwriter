package services

import (
	"github.com/ashisharyan/ghostwriter-prompt-engine/models"
)

type PromptABService interface {
	GeneratePromptVariants(profile *models.CreatorProfile, persona *models.PersonaProfile, topic string) (map[string]PromptContext, string)
	StorePromptVariants(contexts map[string]PromptContext, profile *models.CreatorProfile, topic string) error
}

type DefaultPromptABService struct {
	PersonaService PersonaService
	PromptRepo     PromptRepository
}

func NewDefaultPromptABService(personaService PersonaService, repo PromptRepository) PromptABService {
	return &DefaultPromptABService{PersonaService: personaService, PromptRepo: repo}
}

func (s *DefaultPromptABService) GeneratePromptVariants(profile *models.CreatorProfile, persona *models.PersonaProfile, topic string) (map[string]PromptContext, string) {
	personaSummary := s.PersonaService.BuildPersonaSummary(profile, persona)
	variants := map[string]PromptContext{
		"A": s.PersonaService.BuildPromptContext(profile, persona, topic, "A"),
		"B": s.PersonaService.BuildPromptContext(profile, persona, topic, "B"),
		"C": s.PersonaService.BuildPromptContext(profile, persona, topic, "C"),
	}
	return variants, personaSummary
}

func (s *DefaultPromptABService) StorePromptVariants(contexts map[string]PromptContext, profile *models.CreatorProfile, topic string) error {
	if s.PromptRepo == nil {
		return nil
	}
	for variant, ctx := range contexts {
		prompt := &models.Prompt{
			CreatorID:    profile.ID,
			Topic:        topic,
			Variant:      variant,
			PromptText:   ctx.FullPromptText,
			SystemPrompt: ctx.SystemPrompt,
			UserPrompt:   ctx.UserPrompt,
		}
		if err := s.PromptRepo.SavePrompt(prompt); err != nil {
			return err
		}
	}
	return nil
}
