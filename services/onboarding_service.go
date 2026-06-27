package services

import (
	"strings"

	"github.com/ashisharyan/ghostwriter-prompt-engine/models"
)

type OnboardingService interface {
	GetQuestions() []models.OnboardingQuestion
	GetQuestionsForPath(voiceInputPath, genre string) []models.OnboardingQuestion
	MapAnswersToBaseline(answers map[string]interface{}) models.PersonaScores
	ExtractLexicalHints(answers map[string]interface{}) models.LexicalProfile
}

type DefaultOnboardingService struct{}

func NewDefaultOnboardingService() OnboardingService {
	return &DefaultOnboardingService{}
}

func (s *DefaultOnboardingService) GetQuestions() []models.OnboardingQuestion {
	return onboardingQuestionCatalog()
}

func (s *DefaultOnboardingService) GetQuestionsForPath(voiceInputPath, genre string) []models.OnboardingQuestion {
	shared := onboardingQuestionCatalog()
	switch voiceInputPath {
	case models.VoiceInputPathGuidedWrite:
		return append(shared, guidedWriteQuestions(genre)...)
	case models.VoiceInputPathImportInstagram:
		return shared
	case models.VoiceInputPathSkipCalibrate:
		return append(shared, skipCalibrateQuestions()...)
	default:
		return shared
	}
}

func guidedWriteQuestions(genre string) []models.OnboardingQuestion {
	topic := genrePlaceholder(genre)
	return []models.OnboardingQuestion{
		{
			ID: "guided_hook", Text: "Write the first 3–5 lines you'd say starting a video about " + topic + ".",
			Type: "guided_write", Required: true, MinWords: guidedWriteMinWords, MaxWords: 150,
			StarterLine: "Okay so suno…",
		},
		{
			ID: "guided_hot_take", Text: "What's one thing everyone in " + topic + " gets wrong? Rant like you're talking to a friend.",
			Type: "guided_write", Required: true, MinWords: guidedWriteMinWords, MaxWords: 150,
		},
		{
			ID: "guided_mini_story", Text: "Tell a 4–6 sentence personal story related to " + topic + ". Talk to camera, not essay mode.",
			Type: "guided_write", Required: true, MinWords: guidedWriteMinWords, MaxWords: 150,
		},
	}
}

func skipCalibrateQuestions() []models.OnboardingQuestion {
	return []models.OnboardingQuestion{
		{
			ID: "comparative_hook", Text: "Which opener sounds more like you?", Type: "comparative_choice", Required: true,
			Options: []models.OnboardingOption{
				{ID: "question", Label: "Ever wondered why…?", ScoreDeltas: models.ScoreDeltas{"brevity": 10, "energy": 5}},
				{ID: "punch", Label: "Okay so suno — this changed everything.", ScoreDeltas: models.ScoreDeltas{"brevity": 20, "energy": 15}},
			},
		},
		{
			ID: "comparative_tone", Text: "Which closing feels more like you?", Type: "comparative_choice", Required: true,
			Options: []models.OnboardingOption{
				{ID: "cta_soft", Label: "Try this and tell me how it goes.", ScoreDeltas: models.ScoreDeltas{"directness": -10}},
				{ID: "cta_bold", Label: "Do this today. No excuses.", ScoreDeltas: models.ScoreDeltas{"directness": 15, "energy": 10}},
			},
		},
		{ID: "anti_voice", Text: "I never sound like… (motivational speaker, news anchor, etc.)", Type: "free_text", Required: true},
		{ID: "inspiration_creators", Text: "Creators whose energy you like (not to copy their words)", Type: "free_text", Required: true},
	}
}

func genrePlaceholder(genre string) string {
	g := strings.TrimSpace(strings.ToLower(genre))
	switch g {
	case "comedy", "entertainment":
		return "comedy content"
	case "finance", "business", "money":
		return "finance"
	case "lifestyle", "fitness", "beauty":
		return "lifestyle"
	default:
		if g == "" {
			return "your niche"
		}
		return g
	}
}

func (s *DefaultOnboardingService) MapAnswersToBaseline(answers map[string]interface{}) models.PersonaScores {
	if len(answers) == 0 {
		return defaultScores()
	}

	scores := defaultScores()

	for _, question := range append(onboardingQuestionCatalog(), skipCalibrateQuestions()...) {
		raw, ok := answers[question.ID]
		if !ok || raw == nil {
			continue
		}

		switch question.Type {
		case "scale_1_5":
			value := toInt(raw)
			if value < 1 {
				continue
			}
			if value > 5 {
				value = 5
			}
			scaled := ((value - 1) * 100) / 4
			targetDim := dimFromQuestion(question.ID)
			if targetDim != "" {
				scores = writeDimension(scores, targetDim, scaled)
			}
		case "single_choice", "multi_select", "comparative_choice":
			selected := toStringSlice(raw)
			for _, optionID := range selected {
				option := findOption(question, optionID)
				if option == nil {
					continue
				}
				for dim, delta := range option.ScoreDeltas {
					current := readDimension(scores, dim)
					scores = writeDimension(scores, dim, current+delta)
				}
			}
		}
	}

	return clampScores(scores)
}

func (s *DefaultOnboardingService) ExtractLexicalHints(answers map[string]interface{}) models.LexicalProfile {
	profile := models.LexicalProfile{FillerFrequency: "moderate"}
	if preferred, ok := answers["preferred_words"]; ok {
		profile.PreferredWords = splitWords(toString(preferred))
	}
	if avoid, ok := answers["avoid_words"]; ok {
		profile.AvoidWords = splitWords(toString(avoid))
	}
	return profile
}

func onboardingQuestionCatalog() []models.OnboardingQuestion {
	return []models.OnboardingQuestion{
		{
			ID:       "content_energy",
			Text:     "How energetic is your on-camera delivery?",
			Type:     "scale_1_5",
			Required: true,
		},
		{
			ID:       "humor_style",
			Text:     "How much humor do you use in your content?",
			Type:     "single_choice",
			Required: true,
			Options: []models.OnboardingOption{
				{ID: "serious", Label: "Mostly serious", ScoreDeltas: models.ScoreDeltas{"humor": -25}},
				{ID: "light", Label: "Light humor sometimes", ScoreDeltas: models.ScoreDeltas{"humor": 10}},
				{ID: "funny", Label: "Humor is a core part of my style", ScoreDeltas: models.ScoreDeltas{"humor": 30}},
			},
		},
		{
			ID:       "formality_level",
			Text:     "How formal is your speaking style?",
			Type:     "single_choice",
			Required: true,
			Options: []models.OnboardingOption{
				{ID: "casual", Label: "Very casual and conversational", ScoreDeltas: models.ScoreDeltas{"formality": -30}},
				{ID: "balanced", Label: "Balanced", ScoreDeltas: models.ScoreDeltas{"formality": 0}},
				{ID: "professional", Label: "Polished and professional", ScoreDeltas: models.ScoreDeltas{"formality": 30}},
			},
		},
		{
			ID:       "storytelling_preference",
			Text:     "Do you prefer facts-first or story-driven scripts?",
			Type:     "single_choice",
			Required: true,
			Options: []models.OnboardingOption{
				{ID: "facts", Label: "Facts and tips first", ScoreDeltas: models.ScoreDeltas{"storytelling": -25}},
				{ID: "mixed", Label: "Mix of both", ScoreDeltas: models.ScoreDeltas{"storytelling": 0}},
				{ID: "stories", Label: "Story-driven and personal", ScoreDeltas: models.ScoreDeltas{"storytelling": 30}},
			},
		},
		{
			ID:       "directness_level",
			Text:     "How direct are you when sharing opinions?",
			Type:     "single_choice",
			Required: true,
			Options: []models.OnboardingOption{
				{ID: "soft", Label: "Soft and diplomatic", ScoreDeltas: models.ScoreDeltas{"directness": -25}},
				{ID: "balanced", Label: "Balanced", ScoreDeltas: models.ScoreDeltas{"directness": 0}},
				{ID: "blunt", Label: "Blunt and opinionated", ScoreDeltas: models.ScoreDeltas{"directness": 30}},
			},
		},
		{
			ID:       "hinglish_usage",
			Text:     "How much Hindi-English mix do you use?",
			Type:     "single_choice",
			Required: true,
			Options: []models.OnboardingOption{
				{ID: "english", Label: "Mostly English", ScoreDeltas: models.ScoreDeltas{"hinglish_mix": -30}},
				{ID: "some", Label: "Some Hinglish", ScoreDeltas: models.ScoreDeltas{"hinglish_mix": 10}},
				{ID: "heavy", Label: "Heavy Hinglish/code-mix", ScoreDeltas: models.ScoreDeltas{"hinglish_mix": 35}},
			},
		},
		{
			ID:       "hook_style",
			Text:     "How do you usually open your videos?",
			Type:     "single_choice",
			Required: true,
			Options: []models.OnboardingOption{
				{ID: "slow", Label: "Slow setup and context", ScoreDeltas: models.ScoreDeltas{"brevity": -20, "energy": -10}},
				{ID: "question", Label: "Question or bold statement", ScoreDeltas: models.ScoreDeltas{"brevity": 10, "energy": 10}},
				{ID: "punch", Label: "Instant punch/hook", ScoreDeltas: models.ScoreDeltas{"brevity": 25, "energy": 20}},
			},
		},
		{
			ID:       "emotional_tone",
			Text:     "How personal and emotional is your content?",
			Type:     "single_choice",
			Required: true,
			Options: []models.OnboardingOption{
				{ID: "detached", Label: "Mostly analytical", ScoreDeltas: models.ScoreDeltas{"emotional_warmth": -20}},
				{ID: "warm", Label: "Warm and relatable", ScoreDeltas: models.ScoreDeltas{"emotional_warmth": 15}},
				{ID: "deep", Label: "Deeply personal", ScoreDeltas: models.ScoreDeltas{"emotional_warmth": 30}},
			},
		},
		{
			ID:       "preferred_words",
			Text:     "Words or phrases you use often (comma-separated)",
			Type:     "free_text",
			Required: false,
		},
		{
			ID:       "avoid_words",
			Text:     "Words or phrases you never use (comma-separated)",
			Type:     "free_text",
			Required: false,
		},
	}
}

func dimFromQuestion(questionID string) string {
	switch questionID {
	case "content_energy":
		return "energy"
	case "humor_style":
		return "humor"
	case "formality_level":
		return "formality"
	case "storytelling_preference":
		return "storytelling"
	case "directness_level":
		return "directness"
	case "hinglish_usage":
		return "hinglish_mix"
	case "hook_style":
		return "brevity"
	case "emotional_tone":
		return "emotional_warmth"
	default:
		return ""
	}
}

func findOption(question models.OnboardingQuestion, optionID string) *models.OnboardingOption {
	for i := range question.Options {
		if question.Options[i].ID == optionID {
			return &question.Options[i]
		}
	}
	return nil
}

func readDimension(scores models.PersonaScores, dim string) int {
	switch dim {
	case "formality":
		return scores.Formality
	case "humor":
		return scores.Humor
	case "energy":
		return scores.Energy
	case "brevity":
		return scores.Brevity
	case "storytelling":
		return scores.Storytelling
	case "directness":
		return scores.Directness
	case "emotional_warmth":
		return scores.EmotionalWarmth
	case "hinglish_mix":
		return scores.HinglishMix
	default:
		return defaultScore
	}
}

func writeDimension(scores models.PersonaScores, dim string, value int) models.PersonaScores {
	value = clampScore(value)
	switch dim {
	case "formality":
		scores.Formality = value
	case "humor":
		scores.Humor = value
	case "energy":
		scores.Energy = value
	case "brevity":
		scores.Brevity = value
	case "storytelling":
		scores.Storytelling = value
	case "directness":
		scores.Directness = value
	case "emotional_warmth":
		scores.EmotionalWarmth = value
	case "hinglish_mix":
		scores.HinglishMix = value
	}
	return scores
}

func toInt(raw interface{}) int {
	switch v := raw.(type) {
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		return int(v)
	default:
		return 0
	}
}

func toString(raw interface{}) string {
	switch v := raw.(type) {
	case string:
		return v
	default:
		return ""
	}
}

func toStringSlice(raw interface{}) []string {
	switch v := raw.(type) {
	case string:
		return []string{v}
	case []interface{}:
		out := make([]string, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok {
				out = append(out, s)
			}
		}
		return out
	case []string:
		return v
	default:
		return nil
	}
}
