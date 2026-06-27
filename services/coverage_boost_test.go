package services

import (
	"strings"
	"testing"

	"github.com/ashisharyan/ghostwriter-prompt-engine/models"
)

func TestGenrePlaceholderBranches(t *testing.T) {
	cases := map[string]string{
		"comedy":    "comedy content",
		"finance":   "finance",
		"lifestyle": "lifestyle",
		"":          "your niche",
		"tech":      "tech",
	}
	for in, wantSub := range cases {
		got := genrePlaceholder(in)
		if !strings.Contains(got, wantSub) {
			t.Fatalf("genrePlaceholder(%q) = %q, want substring %q", in, got, wantSub)
		}
	}
}

func TestApplyFeedbackShiftScore(t *testing.T) {
	db := setupTestDB(t)
	profile := testProfile()
	if err := db.Create(profile).Error; err != nil {
		t.Fatal(err)
	}
	svc := NewGormPersonaService(db, NewDefaultOnboardingService(), nil)
	_, err := svc.CreatePersonaFromOnboarding(profile.ID, profile, models.SubmitProfileRequest{
		VoiceInputPath: models.VoiceInputPathPasteScripts,
		WritingSamples: []string{longSample(), longSample()},
		StyleAnswers:   skipSubmitAnswers(),
	})
	if err != nil {
		t.Fatal(err)
	}
	result, err := svc.ApplyFeedback(profile.ID, models.ScriptFeedbackRequest{
		Rating:          "close",
		GeneratedScript: longSample(),
		Toggles:         []string{"unknown_toggle", "too_casual"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.ShiftScore <= 0 {
		t.Fatal("expected shift score when fingerprint and generated script present")
	}
}

func TestCreateProfileDatabaseError(t *testing.T) {
	db := setupTestDB(t)
	sqlDB, _ := db.DB()
	sqlDB.Close()
	svc := NewGormProfileService(db)
	if _, err := svc.CreateProfile(testProfile()); err == nil {
		t.Fatal("expected create profile error on closed db")
	}
}

func TestBuildSamplesFromSubmitPasteAndSkip(t *testing.T) {
	paste := BuildSamplesFromSubmit(models.SubmitProfileRequest{
		VoiceInputPath: models.VoiceInputPathPasteScripts,
		WritingSamples: []string{"a", "  ", "b"},
	})
	if len(paste) != 2 {
		t.Fatalf("expected 2 paste samples, got %d", len(paste))
	}
	skip := BuildSamplesFromSubmit(models.SubmitProfileRequest{VoiceInputPath: models.VoiceInputPathSkipCalibrate})
	if len(skip) != 0 {
		t.Fatal("skip path should have no samples")
	}
}

func TestApplyStructuredTogglesNil(t *testing.T) {
	d := applyStructuredToggles(nil, []string{"too_formal", "wrong_words"})
	if d["formality"] >= 0 {
		t.Fatal("expected negative formality delta")
	}
}

func TestThreePathFeedbackCalibrationIntegration(t *testing.T) {
	db := setupTestDB(t)
	profile := testProfile()
	if err := db.Create(profile).Error; err != nil {
		t.Fatal(err)
	}
	personaSvc := NewGormPersonaService(db, NewDefaultOnboardingService(), nil)
	persona, err := personaSvc.CreatePersonaFromOnboarding(profile.ID, profile, models.SubmitProfileRequest{
		VoiceInputPath: models.VoiceInputPathSkipCalibrate,
		StyleAnswers:   skipSubmitAnswers(),
	})
	if err != nil || persona.VoiceMode != models.PersonaModeDeclared {
		t.Fatalf("skip onboarding failed: %+v", err)
	}

	promptRepo := NewGormPromptRepository(db)
	scriptRepo := NewGormScriptRepository(db)
	feedbackSvc := NewGormFeedbackService(db, personaSvc, scriptRepo)
	prompt := &models.Prompt{CreatorID: profile.ID, Topic: "t", Variant: "base", PromptText: "p"}
	_ = promptRepo.SavePrompt(prompt)
	script := &models.Script{CreatorID: profile.ID, PromptID: prompt.ID, ScriptText: "ai draft", Source: "manual"}
	_ = scriptRepo.SaveScript(script)

	result, err := feedbackSvc.SubmitFeedback(script.ID, models.ScriptFeedbackRequest{
		Rating:       "close",
		EditedScript: longSample(),
		Toggles:      []string{"not_enough_hindi"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.VoiceMode == models.PersonaModeDeclared && result.VoiceConfidence < 40 {
		t.Fatalf("expected improved confidence after calibration, got mode=%s conf=%d", result.VoiceMode, result.VoiceConfidence)
	}
}
