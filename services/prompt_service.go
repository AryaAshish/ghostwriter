package services

import (
	"github.com/ashisharyan/ghostwriter-prompt-engine/models"
)

type PromptService interface {
	GeneratePersonaSummary(profile *models.CreatorProfile, persona *models.PersonaProfile) string
	GeneratePrompt(profile *models.CreatorProfile, persona *models.PersonaProfile, topic string) (PromptContext, error)
}

type DefaultPromptService struct {
	PromptRepo     PromptRepository
	PersonaService PersonaService
}

func NewDefaultPromptService(promptRepo PromptRepository, personaService PersonaService) PromptService {
	return &DefaultPromptService{PromptRepo: promptRepo, PersonaService: personaService}
}

func (s *DefaultPromptService) GeneratePersonaSummary(profile *models.CreatorProfile, persona *models.PersonaProfile) string {
	return s.PersonaService.BuildPersonaSummary(profile, persona)
}

func (s *DefaultPromptService) GeneratePrompt(profile *models.CreatorProfile, persona *models.PersonaProfile, topic string) (PromptContext, error) {
	ctx := s.PersonaService.BuildPromptContext(profile, persona, topic, "base")
	if s.PromptRepo != nil {
		dbPrompt := &models.Prompt{
			CreatorID:    profile.ID,
			Topic:        topic,
			Variant:      "base",
			PromptText:   ctx.FullPromptText,
			SystemPrompt: ctx.SystemPrompt,
			UserPrompt:   ctx.UserPrompt,
		}
		if err := s.PromptRepo.SavePrompt(dbPrompt); err != nil {
			return ctx, err
		}
	}
	return ctx, nil
}
