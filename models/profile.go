package models

import (
	"time"
)

type CreatorProfile struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
	Name       string    `json:"name" binding:"required"`
	Email      string    `json:"email"`
	Channel    string    `json:"channel"`
	Genre      string    `json:"genre" binding:"required"`
	Language   string    `json:"language" binding:"required"`
	Region     string    `json:"region"`
	Bio        string    `json:"bio"`
	Goal       string    `json:"goal"`
	Audience   string    `json:"audience"`
	Tone       string    `json:"tone"`
	Style      string    `json:"style"`
	Inspiration string   `json:"inspiration"`
	ContentType string   `json:"content_type"`
	Frequency   string   `json:"frequency"`
	Platform    string   `json:"platform"`
	HasTeam     bool     `json:"has_team"`
	Experience  string   `json:"experience"`
	USP         string   `json:"usp"`
	Other       string   `json:"other"`
}
