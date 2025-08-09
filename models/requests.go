package models

// GeneratePromptRequest represents the request body for generating prompts
type GeneratePromptRequest struct {
	CreatorID uint   `json:"creator_id" binding:"required"`
	Topic     string `json:"topic" binding:"required"`
}

// GenerateScriptRequest represents the request body for generating scripts
type GenerateScriptRequest struct {
	CreatorID string `json:"creator_id" binding:"required"`
	Topic     string `json:"topic" binding:"required"`
	Variant   string `json:"variant"`
}
