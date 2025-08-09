package services

import (
	"time"
	"github.com/ashisharyan/ghostwriter-prompt-engine/models"
	"gorm.io/gorm"
)

type ScriptWithPrompt struct {
	ScriptText string
	Variant    string
	Source     string
	CreatedAt  time.Time
}

type ScriptRepository interface {
	SaveScript(script *models.Script) error
	GetScriptsByCreatorIDWithPrompt(creatorID string) ([]ScriptWithPrompt, error)
}

type GormScriptRepository struct {
	DB *gorm.DB
}

func NewGormScriptRepository(db *gorm.DB) ScriptRepository {
	return &GormScriptRepository{DB: db}
}

func (r *GormScriptRepository) SaveScript(script *models.Script) error {
	return r.DB.Create(script).Error
}

func (r *GormScriptRepository) GetScriptsByCreatorIDWithPrompt(creatorID string) ([]ScriptWithPrompt, error) {
	var results []ScriptWithPrompt

tx := r.DB.Table("scripts").
	Select("scripts.script_text, prompts.variant, scripts.source, scripts.created_at").
	Joins("JOIN prompts ON scripts.prompt_id = prompts.id").
	Where("scripts.creator_id = ?", creatorID).
	Order("scripts.created_at DESC").
	Scan(&results)
	return results, tx.Error
}

