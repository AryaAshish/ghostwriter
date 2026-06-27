package services

import (
	"fmt"
	"strings"
	"time"

	"github.com/ashisharyan/ghostwriter-prompt-engine/models"
	"gorm.io/gorm"
)

type PromptContext struct {
	SystemPrompt    string
	UserPrompt      string
	PersonaSummary  string
	Scores          models.PersonaScores
	FullPromptText  string
	VoiceMode       string
	VoiceConfidence int
}

type FeedbackResult struct {
	Deltas          map[string]int
	VoiceMode       string
	VoiceConfidence int
	ShiftScore      int
}

type PersonaService interface {
	CreatePersonaFromOnboarding(creatorID uint, profile *models.CreatorProfile, req models.SubmitProfileRequest) (*models.PersonaProfile, error)
	GetPersona(creatorID uint) (*models.PersonaProfile, error)
	UpdatePersona(creatorID uint, req models.UpdatePersonaRequest) (*models.PersonaProfile, error)
	ReanalyzeVoice(creatorID uint, samples []string) (*models.PersonaProfile, error)
	BuildPersonaSummary(profile *models.CreatorProfile, persona *models.PersonaProfile) string
	BuildLexicalRules(lexical models.LexicalProfile) string
	BuildPromptContext(profile *models.CreatorProfile, persona *models.PersonaProfile, topic, variant string) PromptContext
	ApplyFeedback(creatorID uint, req models.ScriptFeedbackRequest) (FeedbackResult, error)
	GetOrDefaultPersona(creatorID uint, profile *models.CreatorProfile) (*models.PersonaProfile, error)
}

type GormPersonaService struct {
	DB                   *gorm.DB
	OnboardingService      OnboardingService
	VoiceAnalysisService VoiceAnalysisService
}

func NewGormPersonaService(db *gorm.DB, onboarding OnboardingService, voice VoiceAnalysisService) PersonaService {
	return &GormPersonaService{DB: db, OnboardingService: onboarding, VoiceAnalysisService: voice}
}

func (s *GormPersonaService) CreatePersonaFromOnboarding(creatorID uint, profile *models.CreatorProfile, req models.SubmitProfileRequest) (*models.PersonaProfile, error) {
	answers := req.StyleAnswers
	baseline := s.OnboardingService.MapAnswersToBaseline(answers)
	baseline = mergeLegacyProfileHints(baseline, profile)
	lexicalHints := s.OnboardingService.ExtractLexicalHints(answers)

	path := req.VoiceInputPath
	if path == "" {
		path = models.VoiceInputPathPasteScripts
	}

	samples := BuildSamplesFromSubmit(req)
	sampleTexts := sampleTexts(samples)
	totalWords := totalSampleWords(samples)

	current := baseline
	voiceSummary := ""
	lexical := lexicalHints

	if len(sampleTexts) > 0 && s.VoiceAnalysisService != nil {
		analysis, err := s.VoiceAnalysisService.AnalyzeVoice(sampleTexts, baseline, profile)
		if err == nil && analysis != nil {
			current = mergeScoresWeighted(baseline, analysis.SuggestedScores, 60, 40)
			lexical = mergeLexicalProfiles(lexicalHints, analysis.LexicalProfile)
			voiceSummary = analysis.VoiceSummary
		}
	}

	fp := models.VoiceFingerprint{}
	if len(sampleTexts) > 0 {
		fp = ExtractFingerprint(sampleTexts)
		lexical = mergeLexicalProfiles(lexical, LexicalHintsFromFingerprint(sampleTexts, fp))
	}

	mode := ResolvePersonaMode(path, totalWords)
	confidence := ComputeVoiceConfidence(totalWords, len(samples), 0, mode)

	legacySamples := make([]string, 0, len(samples))
	for _, sm := range samples {
		legacySamples = append(legacySamples, sm.Text)
	}

	persona := &models.PersonaProfile{
		CreatorID:        creatorID,
		BaselineScores:   baseline,
		CurrentScores:    current,
		LexicalProfile:   lexical,
		WritingSamples:   legacySamples,
		Samples:          samples,
		VoiceSummary:     voiceSummary,
		VoiceMode:        mode,
		VoiceInputPath:   path,
		VoiceConfidence:  confidence,
		VoiceFingerprint: fp,
	}
	if err := s.DB.Create(persona).Error; err != nil {
		return nil, err
	}
	return persona, nil
}

func (s *GormPersonaService) GetPersona(creatorID uint) (*models.PersonaProfile, error) {
	var persona models.PersonaProfile
	if err := s.DB.Where("creator_id = ?", creatorID).First(&persona).Error; err != nil {
		return nil, err
	}
	return &persona, nil
}

func (s *GormPersonaService) GetOrDefaultPersona(creatorID uint, profile *models.CreatorProfile) (*models.PersonaProfile, error) {
	persona, err := s.GetPersona(creatorID)
	if err == nil {
		return persona, nil
	}
	return &models.PersonaProfile{
		CreatorID:       creatorID,
		VoiceMode:       models.PersonaModeDeclared,
		VoiceConfidence: 15,
		BaselineScores:  mergeLegacyProfileHints(defaultScores(), profile),
		CurrentScores:   mergeLegacyProfileHints(defaultScores(), profile),
		LexicalProfile:  models.LexicalProfile{FillerFrequency: "moderate"},
	}, nil
}

func (s *GormPersonaService) UpdatePersona(creatorID uint, req models.UpdatePersonaRequest) (*models.PersonaProfile, error) {
	persona, err := s.GetPersona(creatorID)
	if err != nil {
		return nil, err
	}
	if req.CurrentScores != nil {
		persona.CurrentScores = clampScores(*req.CurrentScores)
	}
	if req.LexicalProfile != nil {
		persona.LexicalProfile = *req.LexicalProfile
	}
	if req.VoiceSummary != nil {
		persona.VoiceSummary = *req.VoiceSummary
	}
	if len(req.WritingSamples) > 0 {
		persona.WritingSamples = req.WritingSamples
		now := time.Now()
		structured := make([]models.WritingSample, 0, len(req.WritingSamples))
		for _, text := range req.WritingSamples {
			structured = append(structured, models.WritingSample{
				Text: text, Source: models.SampleSourcePastScript, CreatedAt: now,
			})
		}
		persona.Samples = NormalizeSamples(nil, structured)
	}
	persona.UpdatedAt = time.Now()
	if err := s.DB.Save(persona).Error; err != nil {
		return nil, err
	}
	return persona, nil
}

func (s *GormPersonaService) ReanalyzeVoice(creatorID uint, samples []string) (*models.PersonaProfile, error) {
	persona, err := s.GetPersona(creatorID)
	if err != nil {
		return nil, err
	}
	if len(samples) > 0 {
		persona.WritingSamples = samples
		now := time.Now()
		structured := make([]models.WritingSample, 0, len(samples))
		for _, text := range samples {
			structured = append(structured, models.WritingSample{
				Text: text, Source: models.SampleSourcePastScript, CreatedAt: now,
			})
		}
		persona.Samples = NormalizeSamples(nil, structured)
	}
	allSamples := NormalizeSamples(persona.WritingSamples, persona.Samples)
	texts := sampleTexts(allSamples)
	if len(texts) == 0 {
		return nil, fmt.Errorf("writing samples required")
	}

	var profile models.CreatorProfile
	if err := s.DB.First(&profile, creatorID).Error; err != nil {
		return nil, err
	}

	if s.VoiceAnalysisService != nil {
		analysis, err := s.VoiceAnalysisService.AnalyzeVoice(texts, persona.BaselineScores, &profile)
		if err != nil {
			return nil, err
		}
		persona.CurrentScores = mergeScoresWeighted(persona.BaselineScores, analysis.SuggestedScores, 60, 40)
		persona.LexicalProfile = mergeLexicalProfiles(persona.LexicalProfile, analysis.LexicalProfile)
		persona.VoiceSummary = analysis.VoiceSummary
	}

	fp := ExtractFingerprint(texts)
	persona.VoiceFingerprint = fp
	persona.LexicalProfile = mergeLexicalProfiles(persona.LexicalProfile, LexicalHintsFromFingerprint(texts, fp))
	totalWords := totalSampleWords(allSamples)
	persona.VoiceMode = PromotePersonaMode(persona.VoiceMode, totalWords)
	persona.VoiceConfidence = ComputeVoiceConfidence(totalWords, len(allSamples), persona.FeedbackCount, persona.VoiceMode)
	persona.Samples = allSamples
	persona.WritingSamples = texts
	persona.UpdatedAt = time.Now()
	if err := s.DB.Save(persona).Error; err != nil {
		return nil, err
	}
	return persona, nil
}

func (s *GormPersonaService) BuildPersonaSummary(profile *models.CreatorProfile, persona *models.PersonaProfile) string {
	scores := persona.CurrentScores
	parts := []string{
		fmt.Sprintf("%s is a %s creator from %s making %s content in %s on %s.",
			profile.Name, profile.Genre, profile.Region, profile.ContentType, profile.Language, profile.Platform),
	}
	if profile.Bio != "" {
		parts = append(parts, "Bio: "+profile.Bio)
	}
	if profile.Audience != "" {
		parts = append(parts, "Audience: "+profile.Audience)
	}
	if profile.Goal != "" {
		parts = append(parts, "Goal: "+profile.Goal)
	}
	if profile.Inspiration != "" {
		parts = append(parts, "Inspiration: "+profile.Inspiration)
	}
	if profile.USP != "" {
		parts = append(parts, "USP: "+profile.USP)
	}
	if persona.VoiceSummary != "" {
		parts = append(parts, "Voice summary: "+persona.VoiceSummary)
	}
	parts = append(parts,
		fmt.Sprintf("Voice mode: %s (confidence %d/100)", persona.VoiceMode, persona.VoiceConfidence),
		fmt.Sprintf("Formality: %s", bandLabel(scores.Formality)),
		fmt.Sprintf("Humor: %s", bandLabel(scores.Humor)),
		fmt.Sprintf("Energy: %s", bandLabel(scores.Energy)),
		fmt.Sprintf("Brevity: %s", bandLabel(scores.Brevity)),
		fmt.Sprintf("Storytelling: %s", bandLabel(scores.Storytelling)),
		fmt.Sprintf("Directness: %s", bandLabel(scores.Directness)),
		fmt.Sprintf("Emotional warmth: %s", bandLabel(scores.EmotionalWarmth)),
		fmt.Sprintf("Hinglish mix: %s", bandLabel(scores.HinglishMix)),
	)
	return strings.Join(parts, "\n")
}

func (s *GormPersonaService) BuildLexicalRules(lexical models.LexicalProfile) string {
	lines := []string{"Lexical voice rules:"}
	if len(lexical.SentenceStarters) > 0 {
		lines = append(lines, "- Prefer these sentence openers: "+strings.Join(lexical.SentenceStarters, ", "))
	}
	if len(lexical.SignaturePhrases) > 0 {
		lines = append(lines, "- Use these signature phrases naturally (2-4 times): "+strings.Join(lexical.SignaturePhrases, ", "))
	}
	if len(lexical.PreferredWords) > 0 {
		lines = append(lines, "- Preferred words: "+strings.Join(lexical.PreferredWords, ", "))
	}
	if len(lexical.FillerWords) > 0 {
		freq := lexical.FillerFrequency
		if freq == "" {
			freq = "moderate"
		}
		lines = append(lines, fmt.Sprintf("- Filler words (%s use, max ~1 per 3 lines): %s", freq, strings.Join(lexical.FillerWords, ", ")))
	}
	if len(lexical.AvoidWords) > 0 {
		lines = append(lines, "- Never use these words/phrases: "+strings.Join(lexical.AvoidWords, ", "))
	}
	if lexical.SlangRegister != "" {
		lines = append(lines, "- Slang register: "+lexical.SlangRegister)
	}
	lines = append(lines, "- Avoid generic AI words like delve, tapestry, leverage, folks, game-changer.")
	return strings.Join(lines, "\n")
}

func buildFingerprintRules(fp models.VoiceFingerprint) string {
	if fp.TotalWords == 0 {
		return ""
	}
	return strings.Join([]string{
		"Measured voice fingerprint (match these patterns):",
		fmt.Sprintf("- Target avg sentence length: %.1f words (±%.1f)", fp.AvgSentenceLength, fp.SentenceLengthStd),
		fmt.Sprintf("- Filler density: %.2f per word", fp.FillerDensity),
		fmt.Sprintf("- Hinglish token ratio: %.2f", fp.HinglishRatio),
		fmt.Sprintf("- Preferred hook pattern: %s", fp.HookPattern),
		fmt.Sprintf("- Exclamation rate: %.1f per sentence", fp.ExclamationRate),
	}, "\n")
}

func buildFewShotBlock(persona *models.PersonaProfile) string {
	if persona.VoiceMode == models.PersonaModeDeclared {
		return ""
	}
	samples := NormalizeSamples(persona.WritingSamples, persona.Samples)
	if len(samples) == 0 {
		return ""
	}
	lines := []string{"Few-shot examples (match voice and rhythm exactly):"}
	limit := len(samples)
	if limit > 5 {
		limit = 5
	}
	for i := 0; i < limit; i++ {
		lines = append(lines, fmt.Sprintf("Example %d:\n%s", i+1, samples[i].Text))
	}
	return strings.Join(lines, "\n\n")
}

func declaredModeFooter() string {
	return "Creator has no writing samples yet. Follow their stated preferences strictly. Avoid generic creator voice and AI-default phrasing."
}

func (s *GormPersonaService) BuildPromptContext(profile *models.CreatorProfile, persona *models.PersonaProfile, topic, variant string) PromptContext {
	summary := s.BuildPersonaSummary(profile, persona)
	scoreRules := buildScoreInstructionBlock(persona.CurrentScores)
	lexicalRules := s.BuildLexicalRules(persona.LexicalProfile)
	fingerprintRules := buildFingerprintRules(persona.VoiceFingerprint)
	nicheRules := NicheVoiceRules(profile.Genre)
	fewShots := buildFewShotBlock(persona)
	structureRule := variantStructureInstruction(variant)

	blocks := []string{
		"You are a voice mimic writing short-form video scripts for a specific creator.",
		"Match their tone, rhythm, vocabulary, and personality exactly.",
		"Do not sound like a generic AI assistant.",
		summary,
		scoreRules,
		lexicalRules,
		nicheRules,
	}
	if fingerprintRules != "" {
		blocks = append(blocks, fingerprintRules)
	}
	if fewShots != "" {
		blocks = append(blocks, fewShots)
	}
	if persona.VoiceMode == models.PersonaModeDeclared {
		blocks = append(blocks, declaredModeFooter())
	}
	blocks = append(blocks, structureRule)

	systemPrompt := strings.Join(blocks, "\n\n")

	userPrompt := strings.Join([]string{
		fmt.Sprintf("Topic: %s", topic),
		"Write a complete short-form video script ready to record.",
	}, "\n")

	fullPrompt := systemPrompt + "\n\n---\n\n" + userPrompt
	return PromptContext{
		SystemPrompt:    systemPrompt,
		UserPrompt:      userPrompt,
		PersonaSummary:  summary,
		Scores:          persona.CurrentScores,
		FullPromptText:  fullPrompt,
		VoiceMode:       persona.VoiceMode,
		VoiceConfidence: persona.VoiceConfidence,
	}
}

func (s *GormPersonaService) ApplyFeedback(creatorID uint, req models.ScriptFeedbackRequest) (FeedbackResult, error) {
	persona, err := s.GetPersona(creatorID)
	if err != nil {
		return FeedbackResult{}, err
	}

	deltas := map[string]int{}
	if len(req.Adjustments) > 0 {
		for k, v := range req.Adjustments {
			deltas[k] = v
		}
	} else {
		deltas = inferDeltasFromFeedback(req.Rating, req.Notes)
	}
	deltas = applyStructuredToggles(deltas, req.Toggles)

	persona.CurrentScores = applyScoreDeltas(persona.CurrentScores, deltas)
	notesLower := lowerString(req.Notes)
	for _, word := range splitWords(req.Notes) {
		w := lowerString(word)
		if strings.Contains(notesLower, "never say "+w) || strings.Contains(notesLower, "don't use "+w) {
			persona.LexicalProfile.AvoidWords = uniqueStrings(append(persona.LexicalProfile.AvoidWords, word))
		}
	}
	if strings.Contains(notesLower, "yaar") || strings.Contains(notesLower, "more hindi") {
		persona.LexicalProfile.PreferredWords = uniqueStrings(append(persona.LexicalProfile.PreferredWords, "yaar"))
		deltas["hinglish_mix"] = deltas["hinglish_mix"] + 10
		persona.CurrentScores = applyScoreDeltas(persona.CurrentScores, map[string]int{"hinglish_mix": 10})
	}

	shiftScore := 0
	if strings.TrimSpace(req.EditedScript) != "" {
		now := time.Now()
		persona.Samples = append(persona.Samples, models.WritingSample{
			Text:      strings.TrimSpace(req.EditedScript),
			Source:    models.SampleSourceCalibrationEdit,
			CreatedAt: now,
		})
		persona.WritingSamples = append(persona.WritingSamples, strings.TrimSpace(req.EditedScript))
	}

	allSamples := NormalizeSamples(persona.WritingSamples, persona.Samples)
	texts := sampleTexts(allSamples)
	totalWords := totalSampleWords(allSamples)
	if len(texts) > 0 {
		fp := ExtractFingerprint(texts)
		persona.VoiceFingerprint = fp
		persona.LexicalProfile = mergeLexicalProfiles(persona.LexicalProfile, LexicalHintsFromFingerprint(texts, fp))
		persona.VoiceMode = PromotePersonaMode(persona.VoiceMode, totalWords)
	}

	persona.FeedbackCount++
	persona.VoiceConfidence = ComputeVoiceConfidence(totalWords, len(allSamples), persona.FeedbackCount, persona.VoiceMode)
	persona.Samples = allSamples

	if persona.VoiceFingerprint.TotalWords > 0 && strings.TrimSpace(req.GeneratedScript) != "" {
		shiftScore = ShiftScore(req.GeneratedScript, persona.VoiceFingerprint)
	}

	persona.UpdatedAt = time.Now()
	if err := s.DB.Save(persona).Error; err != nil {
		return FeedbackResult{}, err
	}
	return FeedbackResult{
		Deltas:          deltas,
		VoiceMode:       persona.VoiceMode,
		VoiceConfidence: persona.VoiceConfidence,
		ShiftScore:      shiftScore,
	}, nil
}

func applyStructuredToggles(deltas map[string]int, toggles []string) map[string]int {
	if deltas == nil {
		deltas = map[string]int{}
	}
	toggleDeltas := map[string]map[string]int{
		"too_formal":        {"formality": -12},
		"too_casual":        {"formality": 12},
		"wrong_words":       {"formality": -5, "humor": 5},
		"not_enough_hindi":  {"hinglish_mix": 12},
		"too_much_hindi":    {"hinglish_mix": -12},
		"too_long":          {"brevity": 12},
		"too_short":         {"brevity": -12},
		"not_enough_energy": {"energy": 12},
		"too_hyped":         {"energy": -12},
		"not_personal":      {"emotional_warmth": 12},
		"too_emotional":     {"emotional_warmth": -12},
	}
	for _, toggle := range toggles {
		if delta, ok := toggleDeltas[strings.TrimSpace(toggle)]; ok {
			for dim, val := range delta {
				deltas[dim] += val
			}
		}
	}
	return deltas
}

func mergeLegacyProfileHints(scores models.PersonaScores, profile *models.CreatorProfile) models.PersonaScores {
	if profile == nil {
		return scores
	}
	tone := lowerString(profile.Tone)
	style := lowerString(profile.Style)
	deltas := map[string]int{}
	if strings.Contains(tone, "wit") || strings.Contains(tone, "fun") || strings.Contains(style, "humor") {
		deltas["humor"] = 15
	}
	if strings.Contains(tone, "formal") || strings.Contains(style, "professional") {
		deltas["formality"] = 15
	}
	if strings.Contains(tone, "casual") || strings.Contains(style, "relat") {
		deltas["formality"] = -10
	}
	if strings.Contains(profile.Language, "Hindi") {
		deltas["hinglish_mix"] = 10
	}
	return applyScoreDeltas(scores, deltas)
}

func bandLabel(score int) string {
	switch {
	case score <= 33:
		return "low"
	case score <= 66:
		return "medium"
	default:
		return "high"
	}
}

func buildScoreInstructionBlock(scores models.PersonaScores) string {
	return strings.Join([]string{
		"Style instructions:",
		dimensionInstruction("Formality", scores.Formality, formalityInstructions()),
		dimensionInstruction("Humor", scores.Humor, humorInstructions()),
		dimensionInstruction("Energy", scores.Energy, energyInstructions()),
		dimensionInstruction("Brevity", scores.Brevity, brevityInstructions()),
		dimensionInstruction("Storytelling", scores.Storytelling, storytellingInstructions()),
		dimensionInstruction("Directness", scores.Directness, directnessInstructions()),
		dimensionInstruction("Emotional warmth", scores.EmotionalWarmth, warmthInstructions()),
		dimensionInstruction("Hinglish mix", scores.HinglishMix, hinglishInstructions()),
	}, "\n")
}

func dimensionInstruction(name string, score int, bands [3]string) string {
	idx := 0
	if score > 33 {
		idx = 1
	}
	if score > 66 {
		idx = 2
	}
	return fmt.Sprintf("- %s: %s", name, bands[idx])
}

func formalityInstructions() [3]string {
	return [3]string{"Use casual language, contractions, and conversational phrasing.", "Keep a balanced conversational tone.", "Use polished, professional language with clean structure."}
}

func humorInstructions() [3]string {
	return [3]string{"Stay mostly serious with minimal jokes.", "Include occasional light humor where natural.", "Use humor often with punchlines and playful lines."}
}

func energyInstructions() [3]string {
	return [3]string{"Keep delivery calm and measured.", "Maintain moderate energy throughout.", "Use high energy, urgency, and momentum in delivery."}
}

func brevityInstructions() [3]string {
	return [3]string{"Allow longer setup and explanation.", "Balance concise lines with enough context.", "Use short punchy lines and fast pacing."}
}

func storytellingInstructions() [3]string {
	return [3]string{"Lead with facts, tips, and clear points.", "Mix practical points with brief anecdotes.", "Use narrative flow with personal stories and emotional beats."}
}

func directnessInstructions() [3]string {
	return [3]string{"Use soft language and diplomatic phrasing.", "Share opinions clearly but respectfully.", "Be blunt, direct, and opinionated."}
}

func warmthInstructions() [3]string {
	return [3]string{"Stay analytical and detached.", "Sound relatable and warm.", "Be deeply personal and emotionally open."}
}

func hinglishInstructions() [3]string {
	return [3]string{"Use mostly English with minimal Hindi.", "Mix Hindi and English naturally in casual lines.", "Use heavy Hinglish/code-mixing where it feels authentic."}
}

func variantStructureInstruction(variant string) string {
	switch variant {
	case "A":
		return "Structure: balanced, informative flow with a clear hook, body, and CTA."
	case "B":
		return "Structure: punchy hook-first script with extra witty lines and strong open loops."
	case "C":
		return "Structure: storytelling-first script with emotional depth and a narrative arc."
	default:
		return "Structure: clear hook, concise body, and strong closing CTA."
	}
}

func inferDeltasFromFeedback(rating, notes string) map[string]int {
	deltas := map[string]int{}
	lower := lowerString(notes + " " + rating)

	keywordDeltas := map[string]map[string]int{
		"too formal":      {"formality": -10},
		"more formal":     {"formality": 10},
		"not funny":       {"humor": 10},
		"too funny":       {"humor": -10},
		"too long":        {"brevity": 10},
		"too short":       {"brevity": -10},
		"more energy":     {"energy": 10},
		"too hyped":       {"energy": -10},
		"too soft":        {"directness": 10},
		"too blunt":       {"directness": -10},
		"more personal":   {"emotional_warmth": 10},
		"too emotional":   {"emotional_warmth": -10},
		"more story":      {"storytelling": 10},
		"too narrative":   {"storytelling": -10},
		"more hindi":      {"hinglish_mix": 10},
		"less hindi":      {"hinglish_mix": -10},
		"not enough yaar": {"hinglish_mix": 10},
	}

	for phrase, delta := range keywordDeltas {
		if strings.Contains(lower, phrase) {
			for dim, val := range delta {
				deltas[dim] += val
			}
		}
	}

	switch rating {
	case "no":
		for dim := range deltas {
			deltas[dim] = deltas[dim] * 2
		}
		if len(deltas) == 0 {
			deltas["formality"] = -5
			deltas["humor"] = 5
		}
	case "not_quite":
		if len(deltas) == 0 {
			deltas["formality"] = -5
		}
	}

	return deltas
}
