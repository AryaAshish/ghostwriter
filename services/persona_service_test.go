package services

import (
	"net/http"
	"strings"
	"testing"

	"github.com/ashisharyan/ghostwriter-prompt-engine/models"
)

func skipSubmitAnswers() map[string]interface{} {
	return map[string]interface{}{
		"content_energy":          4,
		"humor_style":             "funny",
		"formality_level":         "casual",
		"preferred_words":         "yaar",
		"avoid_words":             "delve",
		"anti_voice":              "news anchor",
		"storytelling_preference": "mixed",
		"directness_level":        "balanced",
		"hinglish_usage":          "some",
		"hook_style":              "question",
		"emotional_tone":          "warm",
	}
}

func longSample() string {
	return strings.Repeat("Okay so suno yaar this is my natural voice sample sentence. ", 20)
}

func TestGormPersonaServiceCRUD(t *testing.T) {
	db := setupTestDB(t)
	onboarding := NewDefaultOnboardingService()
	svc := NewGormPersonaService(db, onboarding, nil)
	profile := testProfile()
	if err := db.Create(profile).Error; err != nil {
		t.Fatal(err)
	}

	persona, err := svc.CreatePersonaFromOnboarding(profile.ID, profile, models.SubmitProfileRequest{
		VoiceInputPath: models.VoiceInputPathPasteScripts,
		StyleAnswers:   skipSubmitAnswers(),
		WritingSamples: []string{longSample(), longSample()},
	})
	if err != nil {
		t.Fatal(err)
	}
	if persona.CreatorID != profile.ID {
		t.Fatalf("unexpected creator id %d", persona.CreatorID)
	}
	if persona.VoiceMode != models.PersonaModeDerived {
		t.Fatalf("expected derived mode, got %s", persona.VoiceMode)
	}

	got, err := svc.GetPersona(profile.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != persona.ID {
		t.Fatal("persona id mismatch")
	}

	defaultPersona, err := svc.GetOrDefaultPersona(999, profile)
	if err != nil || defaultPersona.CreatorID != 999 {
		t.Fatal("expected default persona for missing record")
	}

	summary := svc.BuildPersonaSummary(profile, got)
	if !strings.Contains(summary, "Amit") || !strings.Contains(summary, "Bio:") {
		t.Fatalf("summary missing fields: %s", summary)
	}

	lexical := svc.BuildLexicalRules(models.LexicalProfile{
		SignaturePhrases: []string{"matlab"},
		FillerWords:      []string{"basically"},
		SentenceStarters: []string{"Dekho"},
		PreferredWords:   []string{"yaar"},
		AvoidWords:       []string{"delve"},
		SlangRegister:    "casual_hinglish",
	})
	if !strings.Contains(lexical, "matlab") || !strings.Contains(lexical, "delve") {
		t.Fatal("lexical rules incomplete")
	}

	ctx := svc.BuildPromptContext(profile, got, "How to go viral", "A")
	if ctx.SystemPrompt == "" || ctx.UserPrompt == "" || !strings.Contains(ctx.FullPromptText, "Topic:") {
		t.Fatal("prompt context incomplete")
	}
	if !strings.Contains(ctx.SystemPrompt, "Few-shot examples") {
		t.Fatal("expected few-shot block for derived mode")
	}
	for _, variant := range []string{"A", "B", "C", "base"} {
		vctx := svc.BuildPromptContext(profile, got, "topic", variant)
		if !strings.Contains(vctx.SystemPrompt, "Structure:") {
			t.Fatalf("missing structure for variant %s", variant)
		}
	}

	declaredPersona := &models.PersonaProfile{VoiceMode: models.PersonaModeDeclared, CurrentScores: got.CurrentScores}
	declaredCtx := svc.BuildPromptContext(profile, declaredPersona, "topic", "base")
	if !strings.Contains(declaredCtx.SystemPrompt, "no writing samples yet") {
		t.Fatal("expected declared footer")
	}

	newScores := models.PersonaScores{Humor: 88}
	updated, err := svc.UpdatePersona(profile.ID, models.UpdatePersonaRequest{CurrentScores: &newScores})
	if err != nil || updated.CurrentScores.Humor != 88 {
		t.Fatalf("update persona failed: %+v", err)
	}

	fb, err := svc.ApplyFeedback(profile.ID, models.ScriptFeedbackRequest{
		Rating: "not_quite",
		Notes:  "too formal, not enough yaar",
		Toggles: []string{"too_formal"},
	})
	if err != nil || fb.Deltas["formality"] >= 0 {
		t.Fatalf("apply feedback failed: %+v err=%v", fb.Deltas, err)
	}

	explicit, err := svc.ApplyFeedback(profile.ID, models.ScriptFeedbackRequest{
		Rating:      "sounds_like_me",
		Adjustments: map[string]int{"energy": 5},
	})
	if err != nil || explicit.Deltas["energy"] != 5 {
		t.Fatalf("explicit adjustments failed: %+v", explicit.Deltas)
	}
}

func TestCreatePersonaThreePaths(t *testing.T) {
	db := setupTestDB(t)
	svc := NewGormPersonaService(db, NewDefaultOnboardingService(), nil)
	profile := testProfile()
	if err := db.Create(profile).Error; err != nil {
		t.Fatal(err)
	}

	paste, err := svc.CreatePersonaFromOnboarding(profile.ID, profile, models.SubmitProfileRequest{
		VoiceInputPath: models.VoiceInputPathPasteScripts,
		WritingSamples: []string{longSample(), longSample()},
		StyleAnswers:   skipSubmitAnswers(),
	})
	if err != nil || paste.VoiceMode != models.PersonaModeDerived {
		t.Fatalf("paste path: mode=%s err=%v", paste.VoiceMode, err)
	}

	profile2 := testProfile()
	profile2.Name = "Riya"
	if err := db.Create(profile2).Error; err != nil {
		t.Fatal(err)
	}
	guided, err := svc.CreatePersonaFromOnboarding(profile2.ID, profile2, models.SubmitProfileRequest{
		VoiceInputPath: models.VoiceInputPathGuidedWrite,
		StyleAnswers:   skipSubmitAnswers(),
		GuidedWrites: map[string]string{
			"guided_hook":       longSample(),
			"guided_hot_take":   longSample(),
			"guided_mini_story": longSample(),
		},
	})
	if err != nil || guided.VoiceMode != models.PersonaModeDerived {
		t.Fatalf("guided path: mode=%s err=%v", guided.VoiceMode, err)
	}

	profile3 := testProfile()
	profile3.Name = "Sam"
	if err := db.Create(profile3).Error; err != nil {
		t.Fatal(err)
	}
	skip, err := svc.CreatePersonaFromOnboarding(profile3.ID, profile3, models.SubmitProfileRequest{
		VoiceInputPath: models.VoiceInputPathSkipCalibrate,
		StyleAnswers:   skipSubmitAnswers(),
	})
	if err != nil || skip.VoiceMode != models.PersonaModeDeclared {
		t.Fatalf("skip path: mode=%s err=%v", skip.VoiceMode, err)
	}
}

func TestGormPersonaServiceReanalyzeVoice(t *testing.T) {
	db := setupTestDB(t)
	profile, _ := seedCreator(t, db)
	mockVoice := &mockVoiceAnalysisService{
		result: &VoiceAnalysisResult{
			SuggestedScores: models.PersonaScores{Humor: 90, Energy: 80},
			LexicalProfile:  models.LexicalProfile{SignaturePhrases: []string{"matlab"}},
			VoiceSummary:    "Playful Hinglish creator",
		},
	}
	svc := NewGormPersonaService(db, NewDefaultOnboardingService(), mockVoice)

	updated, err := svc.ReanalyzeVoice(profile.ID, []string{longSample()})
	if err != nil {
		t.Fatal(err)
	}
	if updated.VoiceSummary != "Playful Hinglish creator" {
		t.Fatalf("unexpected voice summary: %s", updated.VoiceSummary)
	}
}

func TestReanalyzeVoiceMissingSamplesOnFreshPersona(t *testing.T) {
	db := setupTestDB(t)
	profile := testProfile()
	if err := db.Create(profile).Error; err != nil {
		t.Fatal(err)
	}
	svc := NewGormPersonaService(db, NewDefaultOnboardingService(), nil)
	if _, err := svc.CreatePersonaFromOnboarding(profile.ID, profile, models.SubmitProfileRequest{
		VoiceInputPath: models.VoiceInputPathSkipCalibrate,
		StyleAnswers:   skipSubmitAnswers(),
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.ReanalyzeVoice(profile.ID, nil); err == nil {
		t.Fatal("expected error when samples missing")
	}
}

func TestCreatePersonaWithVoiceAnalysis(t *testing.T) {
	db := setupTestDB(t)
	profile := testProfile()
	if err := db.Create(profile).Error; err != nil {
		t.Fatal(err)
	}
	mockVoice := &mockVoiceAnalysisService{
		result: &VoiceAnalysisResult{
			SuggestedScores: models.PersonaScores{Humor: 85},
			LexicalProfile:  models.LexicalProfile{FillerWords: []string{"so"}},
			VoiceSummary:    "Funny",
		},
	}
	svc := NewGormPersonaService(db, NewDefaultOnboardingService(), mockVoice)
	persona, err := svc.CreatePersonaFromOnboarding(profile.ID, profile, models.SubmitProfileRequest{
		VoiceInputPath: models.VoiceInputPathPasteScripts,
		WritingSamples: []string{longSample()},
		StyleAnswers:   skipSubmitAnswers(),
	})
	if err != nil || persona.VoiceSummary != "Funny" {
		t.Fatalf("voice-enriched persona failed: %+v err=%v", persona, err)
	}
}

func TestCreatePersonaWithVoiceAnalysisFailureFallback(t *testing.T) {
	db := setupTestDB(t)
	profile := testProfile()
	if err := db.Create(profile).Error; err != nil {
		t.Fatal(err)
	}
	failVoice := &mockVoiceAnalysisService{err: http.ErrHandlerTimeout}
	svc := NewGormPersonaService(db, NewDefaultOnboardingService(), failVoice)
	persona, err := svc.CreatePersonaFromOnboarding(profile.ID, profile, models.SubmitProfileRequest{
		VoiceInputPath: models.VoiceInputPathPasteScripts,
		StyleAnswers: map[string]interface{}{"humor_style": "funny"},
		WritingSamples: []string{longSample()},
	})
	if err != nil {
		t.Fatal(err)
	}
	if persona.VoiceSummary != "" {
		t.Fatal("expected empty voice summary when analysis fails")
	}
	if persona.VoiceFingerprint.TotalWords == 0 {
		t.Fatal("expected local fingerprint even when OpenAI fails")
	}
}

func TestApplyFeedbackCalibrationEdit(t *testing.T) {
	db := setupTestDB(t)
	profile := testProfile()
	if err := db.Create(profile).Error; err != nil {
		t.Fatal(err)
	}
	svc := NewGormPersonaService(db, NewDefaultOnboardingService(), nil)
	persona, err := svc.CreatePersonaFromOnboarding(profile.ID, profile, models.SubmitProfileRequest{
		VoiceInputPath: models.VoiceInputPathSkipCalibrate,
		StyleAnswers:   skipSubmitAnswers(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if persona.VoiceMode != models.PersonaModeDeclared {
		t.Fatal("expected declared")
	}

	result, err := svc.ApplyFeedback(profile.ID, models.ScriptFeedbackRequest{
		Rating:       "close",
		EditedScript: longSample(),
		Toggles:      []string{"not_enough_hindi"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.VoiceMode == models.PersonaModeDeclared && result.VoiceConfidence < 50 {
		t.Fatalf("expected promotion after edit: mode=%s confidence=%d", result.VoiceMode, result.VoiceConfidence)
	}
}

func TestApplyFeedbackHinglishNotes(t *testing.T) {
	db := setupTestDB(t)
	profile, _ := seedCreator(t, db)
	svc := NewGormPersonaService(db, NewDefaultOnboardingService(), nil)
	_, err := svc.ApplyFeedback(profile.ID, models.ScriptFeedbackRequest{
		Rating: "not_quite",
		Notes:  "not enough yaar, more hindi",
	})
	if err != nil {
		t.Fatal(err)
	}
	updated, _ := svc.GetPersona(profile.ID)
	if len(updated.LexicalProfile.PreferredWords) == 0 {
		t.Fatal("expected preferred words to include yaar")
	}
}

func TestInferDeltasFromFeedbackBranches(t *testing.T) {
	d := inferDeltasFromFeedback("no", "")
	if d["formality"] == 0 && d["humor"] == 0 {
		t.Fatal("expected default deltas for no rating")
	}
	d2 := inferDeltasFromFeedback("not_quite", "")
	if d2["formality"] == 0 {
		t.Fatal("expected formality delta for not_quite")
	}
}

func TestMergeLegacyProfileHintsNil(t *testing.T) {
	got := mergeLegacyProfileHints(defaultScores(), nil)
	if got.Humor != defaultScore {
		t.Fatal("expected unchanged scores for nil profile")
	}
}

func TestBuildFingerprintRulesEmpty(t *testing.T) {
	if buildFingerprintRules(models.VoiceFingerprint{}) != "" {
		t.Fatal("expected empty rules for empty fingerprint")
	}
}
