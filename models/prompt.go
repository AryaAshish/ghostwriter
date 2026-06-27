package models

import (
	"time"
)

type Prompt struct {
	ID           uint      `gorm:"primaryKey"`
	CreatorID    uint
	Topic        string
	Variant      string // A, B, C, or "base"
	PromptText   string
	SystemPrompt string
	UserPrompt   string
	CreatedAt    time.Time
}

