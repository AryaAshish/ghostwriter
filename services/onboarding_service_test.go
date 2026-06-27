package services

import (
	"testing"

	"github.com/ashisharyan/ghostwriter-prompt-engine/models"
)

func TestMapAnswersToBaseline(t *testing.T) {
	service := NewDefaultOnboardingService()
	answers := map[string]interface{}{
		"content_energy":          5,
		"humor_style":             "funny",
		"formality_level":         "casual",
		"storytelling_preference": "stories",
		"directness_level":        "blunt",
		"hinglish_usage":          "heavy",
		"hook_style":              "punch",
		"emotional_tone":          "deep",
	}

	scores := service.MapAnswersToBaseline(answers)
	if scores.Humor <= 50 {
		t.Fatalf("expected humor above default, got %d", scores.Humor)
	}
	if scores.Formality >= 50 {
		t.Fatalf("expected formality below default, got %d", scores.Formality)
	}
	if scores.HinglishMix <= 50 {
		t.Fatalf("expected hinglish above default, got %d", scores.HinglishMix)
	}
}

func TestExtractLexicalHints(t *testing.T) {
	service := NewDefaultOnboardingService()
	answers := map[string]interface{}{
		"preferred_words": "yaar, matlab",
		"avoid_words":     "delve, folks",
	}
	lexical := service.ExtractLexicalHints(answers)
	if len(lexical.PreferredWords) != 2 {
		t.Fatalf("expected 2 preferred words, got %v", lexical.PreferredWords)
	}
	if len(lexical.AvoidWords) != 2 {
		t.Fatalf("expected 2 avoid words, got %v", lexical.AvoidWords)
	}
}

func TestPersonaScoresUnusedHelpers(t *testing.T) {
	adjusted := scoresFromAdjustments(map[string]int{"humor": 10})
	if adjusted.Humor != 60 {
		t.Fatalf("expected adjusted humor 60, got %d", adjusted.Humor)
	}
	base := defaultScores()
	merged := addDeltasToScores(base, map[string]int{"energy": 5})
	if merged.Energy != 55 {
		t.Fatalf("expected energy 55, got %d", merged.Energy)
	}
}

func TestApplyScoreDeltasAllDimensions(t *testing.T) {
	base := defaultScores()
	updated := applyScoreDeltas(base, map[string]int{
		"formality": 1, "humor": 1, "energy": 1, "brevity": 1,
		"storytelling": 1, "directness": 1, "emotional_warmth": 1, "hinglish_mix": 1,
	})
	if updated.Formality != 51 || updated.HinglishMix != 51 {
		t.Fatalf("expected all dimensions incremented: %+v", updated)
	}
}

func TestUpdatePersonaAllFields(t *testing.T) {
	db := setupTestDB(t)
	profile, _ := seedCreator(t, db)
	svc := NewGormPersonaService(db, NewDefaultOnboardingService(), nil)
	summary := "updated summary"
	lexical := models.LexicalProfile{PreferredWords: []string{"bro"}}
	scores := models.PersonaScores{Humor: 99}
	updated, err := svc.UpdatePersona(profile.ID, models.UpdatePersonaRequest{
		CurrentScores:  &scores,
		LexicalProfile: &lexical,
		VoiceSummary:   &summary,
		WritingSamples: []string{"new sample"},
	})
	if err != nil || updated.VoiceSummary != summary || updated.CurrentScores.Humor != 99 {
		t.Fatalf("update all fields failed: %+v err=%v", updated, err)
	}
}

func TestDimFromQuestionAllCases(t *testing.T) {
	ids := []string{"content_energy", "humor_style", "formality_level", "storytelling_preference", "directness_level", "hinglish_usage", "hook_style", "emotional_tone", "unknown"}
	for _, id := range ids {
		_ = dimFromQuestion(id)
	}
}

func TestToIntAndToStringSlice(t *testing.T) {
	if toInt(int64(3)) != 3 || toInt(float64(4)) != 4 {
		t.Fatal("toInt failed")
	}
	if toString(123) != "" {
		t.Fatal("toString non-string should be empty")
	}
	if len(toStringSlice([]interface{}{"a", "b"})) != 2 {
		t.Fatal("toStringSlice interface slice failed")
	}
}

func TestCreatePersonaDatabaseError(t *testing.T) {
	db := setupTestDB(t)
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	sqlDB.Close()
	svc := NewGormPersonaService(db, NewDefaultOnboardingService(), nil)
	profile := testProfile()
	if _, err := svc.CreatePersonaFromOnboarding(1, profile, models.SubmitProfileRequest{
		VoiceInputPath: models.VoiceInputPathSkipCalibrate,
		StyleAnswers: map[string]interface{}{
			"preferred_words": "yaar",
			"avoid_words":     "delve",
			"anti_voice":      "news anchor",
		},
	}); err == nil {
		t.Fatal("expected db error")
	}
}

func TestFeedbackServiceDatabaseError(t *testing.T) {
	db := setupTestDB(t)
	profile, _ := seedCreator(t, db)
	promptRepo := NewGormPromptRepository(db)
	scriptRepo := NewGormScriptRepository(db)
	personaSvc := NewGormPersonaService(db, NewDefaultOnboardingService(), nil)
	prompt := &models.Prompt{CreatorID: profile.ID, Topic: "t", Variant: "base", PromptText: "p"}
	_ = promptRepo.SavePrompt(prompt)
	script := &models.Script{CreatorID: profile.ID, PromptID: prompt.ID, ScriptText: "body", Source: "gpt-4"}
	_ = scriptRepo.SaveScript(script)
	feedbackSvc := NewGormFeedbackService(db, personaSvc, scriptRepo)
	sqlDB, _ := db.DB()
	sqlDB.Close()
	if _, err := feedbackSvc.SubmitFeedback(script.ID, models.ScriptFeedbackRequest{Rating: "no"}); err == nil {
		t.Fatal("expected feedback db error")
	}
}

func TestPromptServiceSaveError(t *testing.T) {
	closedDB := setupTestDB(t)
	sqlDB, _ := closedDB.DB()
	sqlDB.Close()
	seedDB := setupTestDB(t)
	profile, persona := seedCreator(t, seedDB)
	promptRepo := NewGormPromptRepository(closedDB)
	personaSvc := NewGormPersonaService(closedDB, NewDefaultOnboardingService(), nil)
	promptSvc := NewDefaultPromptService(promptRepo, personaSvc)
	if _, err := promptSvc.GeneratePrompt(profile, persona, "topic"); err == nil {
		t.Fatal("expected save error on closed db")
	}
}

func TestOpenAIClientDefaults(t *testing.T) {
	client := NewOpenAIClient("key", "model")
	if client.chatURL() != openAIEndpoint {
		t.Fatal("expected default endpoint")
	}
	if client.httpClient() == nil {
		t.Fatal("expected default http client")
	}
}

func TestReanalyzeVoiceProfileMissing(t *testing.T) {
	db := setupTestDB(t)
	profile := testProfile()
	if err := db.Create(profile).Error; err != nil {
		t.Fatal(err)
	}
	svc := NewGormPersonaService(db, NewDefaultOnboardingService(), &mockVoiceAnalysisService{
		result: &VoiceAnalysisResult{SuggestedScores: defaultScores(), VoiceSummary: "x"},
	})
	persona := &models.PersonaProfile{
		CreatorID:      profile.ID,
		BaselineScores: defaultScores(),
		CurrentScores:  defaultScores(),
		WritingSamples: []string{"sample"},
	}
	if err := db.Create(persona).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Delete(&models.CreatorProfile{}, profile.ID).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := svc.ReanalyzeVoice(profile.ID, nil); err == nil {
		t.Fatal("expected missing profile error")
	}
}
