package models

// GeneratePromptRequest represents the request body for generating prompts.
type GeneratePromptRequest struct {
	CreatorID uint   `json:"creator_id" binding:"required"`
	Topic     string `json:"topic" binding:"required"`
}

// GenerateScriptRequest represents the request body for generating scripts.
type GenerateScriptRequest struct {
	CreatorID uint   `json:"creator_id" binding:"required"`
	Topic     string `json:"topic" binding:"required"`
	Variant   string `json:"variant"`
}

// SubmitProfileRequest extends creator profile with onboarding answers and samples.
type SubmitProfileRequest struct {
	CreatorProfile
	StyleAnswers   map[string]interface{} `json:"style_answers"`
	WritingSamples []string               `json:"writing_samples"`
	VoiceInputPath string                 `json:"voice_input_path"`
	GuidedWrites    map[string]string      `json:"guided_writes"`
	Instagram       *InstagramProfileSnapshot `json:"instagram,omitempty"`
	InstagramReels  []InstagramReel           `json:"instagram_reels,omitempty"`
}

// InstagramPrepareRequest selects reels to resolve into voice text.
type InstagramPrepareRequest struct {
	SessionID   string   `json:"session_id" binding:"required"`
	ReelIDs     []string `json:"reel_ids" binding:"required"`
	Transcribe  bool     `json:"transcribe"`
}

// UpdatePersonaRequest allows manual persona edits after review.
type UpdatePersonaRequest struct {
	CurrentScores  *PersonaScores  `json:"current_scores"`
	LexicalProfile *LexicalProfile `json:"lexical_profile"`
	VoiceSummary   *string         `json:"voice_summary"`
	WritingSamples []string        `json:"writing_samples"`
}

// ScriptFeedbackRequest is the body for script feedback submission.
type ScriptFeedbackRequest struct {
	Rating           string         `json:"rating" binding:"required"`
	Notes            string         `json:"notes"`
	Adjustments      map[string]int `json:"adjustments"`
	EditedScript     string         `json:"edited_script"`
	Toggles          []string       `json:"toggles"`
	GeneratedScript  string         `json:"generated_script"`
}

// ReanalyzeVoiceRequest optionally supplies new writing samples.
type ReanalyzeVoiceRequest struct {
	WritingSamples []string `json:"writing_samples"`
}
