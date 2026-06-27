package services

import (
	"github.com/ashisharyan/ghostwriter-prompt-engine/models"
	"gorm.io/gorm"
)

type PromptRepository interface {
	SavePrompt(prompt *models.Prompt) error
	GetPromptsByCreatorID(creatorID uint) ([]models.Prompt, error)
}

type GormPromptRepository struct {
	DB *gorm.DB
}

func NewGormPromptRepository(db *gorm.DB) *GormPromptRepository {
	return &GormPromptRepository{DB: db}
}

func (r *GormPromptRepository) SavePrompt(prompt *models.Prompt) error {
	return r.DB.Create(prompt).Error
}

func (r *GormPromptRepository) GetPromptsByCreatorID(creatorID uint) ([]models.Prompt, error) {
	var prompts []models.Prompt
	err := r.DB.Where("creator_id = ?", creatorID).Order("created_at desc").Find(&prompts).Error
	return prompts, err
}
