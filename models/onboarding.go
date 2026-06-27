package models

// ScoreDeltas maps persona dimensions to score adjustments for an option.
type ScoreDeltas map[string]int

// OnboardingOption is a selectable answer for a question.
type OnboardingOption struct {
	ID          string      `json:"id"`
	Label       string      `json:"label"`
	Snippet     string      `json:"snippet,omitempty"`
	ScoreDeltas ScoreDeltas `json:"score_deltas,omitempty"`
}

// OnboardingQuestion is a single onboarding question in the catalog.
type OnboardingQuestion struct {
	ID          string             `json:"id"`
	Text        string             `json:"text"`
	Type        string             `json:"type"`
	Required    bool               `json:"required"`
	MinWords    int                `json:"min_words,omitempty"`
	MaxWords    int                `json:"max_words,omitempty"`
	StarterLine string             `json:"starter_line,omitempty"`
	Options     []OnboardingOption `json:"options,omitempty"`
}
