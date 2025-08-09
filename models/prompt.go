package models

import (
	"time"
)

type Prompt struct {
	ID         uint      `gorm:"primaryKey"`
	CreatorID  string
	Topic      string
	Variant    string    // A, B, C, or "base"
	PromptText string
	CreatedAt  time.Time
}

