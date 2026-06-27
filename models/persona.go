package models

import "time"

// Persona mode constants.
const (
	PersonaModeDeclared   = "declared"
	PersonaModeCalibrated = "calibrated"
	PersonaModeDerived    = "derived"
)

// Voice input path constants.
const (
	VoiceInputPathPasteScripts  = "paste_scripts"
	VoiceInputPathGuidedWrite   = "guided_write"
	VoiceInputPathSkipCalibrate  = "skip_calibrate"
	VoiceInputPathImportInstagram = "import_instagram"
)

// Writing sample source tags.
const (
	SampleSourcePastScript      = "past_script"
	SampleSourceGuidedWrite     = "guided_write"
	SampleSourceCalibrationEdit = "calibration_edit"
	SampleSourceCaption            = "caption"
	SampleSourceInstagramReel      = "instagram_reel"
	SampleSourceInstagramTranscript = "instagram_transcript"
)

// Reel text source tags (caption vs speech-to-text).
const (
	ReelTextSourceCaption    = "caption"
	ReelTextSourceTranscript = "transcript"
)

// PersonaScores holds eight voice dimensions scored 0–100.
type PersonaScores struct {
	Formality       int `json:"formality"`
	Humor           int `json:"humor"`
	Energy          int `json:"energy"`
	Brevity         int `json:"brevity"`
	Storytelling    int `json:"storytelling"`
	Directness      int `json:"directness"`
	EmotionalWarmth int `json:"emotional_warmth"`
	HinglishMix     int `json:"hinglish_mix"`
}

// LexicalProfile captures slang, fillers, and word-level voice habits.
type LexicalProfile struct {
	SignaturePhrases []string `json:"signature_phrases"`
	FillerWords      []string `json:"filler_words"`
	SentenceStarters []string `json:"sentence_starters"`
	PreferredWords   []string `json:"preferred_words"`
	AvoidWords       []string `json:"avoid_words"`
	SlangRegister    string   `json:"slang_register"`
	FillerFrequency  string   `json:"filler_frequency"`
}

// VoiceFingerprint holds stylometric features extracted from writing samples.
type VoiceFingerprint struct {
	FunctionWordFreq   map[string]float64 `json:"function_word_freq"`
	AvgSentenceLength  float64            `json:"avg_sentence_length"`
	SentenceLengthStd  float64            `json:"sentence_length_std"`
	FillerDensity      float64            `json:"filler_density"`
	HinglishRatio      float64            `json:"hinglish_ratio"`
	HookPattern        string             `json:"hook_pattern"`
	ExclamationRate    float64            `json:"exclamation_rate"`
	QuestionRate       float64            `json:"question_rate"`
	TotalWords         int                `json:"total_words"`
}

// WritingSample is a tagged text sample used for voice extraction.
type WritingSample struct {
	Text      string    `json:"text"`
	Source    string    `json:"source"`
	CreatedAt time.Time `json:"created_at"`
}

// PersonaProfile stores baseline and current persona state per creator.
type PersonaProfile struct {
	ID               uint             `gorm:"primaryKey" json:"id"`
	CreatorID        uint             `gorm:"uniqueIndex" json:"creator_id"`
	BaselineScores   PersonaScores    `gorm:"serializer:json" json:"baseline_scores"`
	CurrentScores    PersonaScores    `gorm:"serializer:json" json:"current_scores"`
	LexicalProfile   LexicalProfile   `gorm:"serializer:json" json:"lexical_profile"`
	WritingSamples   []string         `gorm:"serializer:json" json:"writing_samples"`
	Samples          []WritingSample  `gorm:"serializer:json" json:"samples"`
	VoiceSummary     string           `json:"voice_summary"`
	VoiceMode        string           `json:"voice_mode"`
	VoiceInputPath   string           `json:"voice_input_path"`
	VoiceConfidence  int              `json:"voice_confidence"`
	VoiceFingerprint VoiceFingerprint `gorm:"serializer:json" json:"voice_fingerprint"`
	FeedbackCount    int              `json:"feedback_count"`
	CreatedAt        time.Time        `json:"created_at"`
	UpdatedAt        time.Time        `json:"updated_at"`
}
