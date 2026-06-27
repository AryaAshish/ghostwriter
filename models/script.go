package models

import "time"

type Script struct {
	ID         uint      `gorm:"primaryKey"`
	CreatorID  uint
	PromptID   uint
	ScriptText string
	Source     string    // "GPT-4"
	CreatedAt  time.Time
}
