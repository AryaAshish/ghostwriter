package models

import "time"

type Script struct {
	ID         uint      `gorm:"primaryKey"`
	CreatorID  string
	PromptID   uint
	ScriptText string
	Source     string    // "GPT-4"
	CreatedAt  time.Time
}
