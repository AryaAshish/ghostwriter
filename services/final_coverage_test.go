package services

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ashisharyan/ghostwriter-prompt-engine/models"
)

func TestMapAnswersEdgeCases(t *testing.T) {
	svc := NewDefaultOnboardingService()
	scores := svc.MapAnswersToBaseline(map[string]interface{}{
		"content_energy": 0,
		"humor_style":    []string{"funny", "light"},
	})
	if scores.Humor == defaultScore {
		t.Fatal("expected humor adjustment from multi select")
	}
	highEnergy := svc.MapAnswersToBaseline(map[string]interface{}{"content_energy": 99})
	if highEnergy.Energy != 100 {
		t.Fatalf("expected clamped energy 100, got %d", highEnergy.Energy)
	}
}

func TestGetOrDefaultPersonaFallback(t *testing.T) {
	db := setupTestDB(t)
	profile := testProfile()
	_ = db.Create(profile)
	svc := NewGormPersonaService(db, NewDefaultOnboardingService(), nil)
	p, err := svc.GetOrDefaultPersona(9999, profile)
	if err != nil || p.CreatorID != 9999 || p.VoiceMode != models.PersonaModeDeclared {
		t.Fatalf("fallback persona failed: %+v err=%v", p, err)
	}
}

func TestUpdatePersonaSaveError(t *testing.T) {
	db := setupTestDB(t)
	profile, _ := seedCreator(t, db)
	closed := setupTestDB(t)
	sqlDB, _ := closed.DB()
	sqlDB.Close()
	svc := NewGormPersonaService(closed, NewDefaultOnboardingService(), nil)
	scores := models.PersonaScores{Humor: 1}
	if _, err := svc.UpdatePersona(profile.ID, models.UpdatePersonaRequest{CurrentScores: &scores}); err == nil {
		t.Fatal("expected update save error")
	}
}

func TestReanalyzeVoiceWithoutOpenAI(t *testing.T) {
	db := setupTestDB(t)
	profile, _ := seedCreator(t, db)
	svc := NewGormPersonaService(db, NewDefaultOnboardingService(), nil)
	updated, err := svc.ReanalyzeVoice(profile.ID, []string{longSample()})
	if err != nil || updated.VoiceFingerprint.TotalWords == 0 {
		t.Fatalf("local reanalyze failed: %+v err=%v", updated, err)
	}
}

func TestApplyFeedbackSaveError(t *testing.T) {
	db := setupTestDB(t)
	profile, _ := seedCreator(t, db)
	closed := setupTestDB(t)
	sqlDB, _ := closed.DB()
	sqlDB.Close()
	svc := NewGormPersonaService(closed, NewDefaultOnboardingService(), nil)
	if _, err := svc.ApplyFeedback(profile.ID, models.ScriptFeedbackRequest{Rating: "no"}); err == nil {
		t.Fatal("expected apply feedback save error")
	}
}

func TestBuildFewShotEmptySamplesDerived(t *testing.T) {
	block := buildFewShotBlock(&models.PersonaProfile{
		VoiceMode: models.PersonaModeDerived,
	})
	if block != "" {
		t.Fatal("expected empty few-shot without samples")
	}
}

func TestResolvePersonaModeCalibratedBand(t *testing.T) {
	if ResolvePersonaMode(models.VoiceInputPathPasteScripts, 200) != models.PersonaModeCalibrated {
		t.Fatal("expected calibrated for 200 words paste")
	}
	if ResolvePersonaMode(models.VoiceInputPathGuidedWrite, 50) != models.PersonaModeDeclared {
		t.Fatal("expected declared for low guided words")
	}
}

func TestShiftScoreCaps(t *testing.T) {
	fp := models.VoiceFingerprint{
		TotalWords: 100, AvgSentenceLength: 10, FillerDensity: 0.01,
		HinglishRatio: 0.01, HookPattern: "punch",
	}
	score := ShiftScore(strings.Repeat("okay so suno yaar basically ", 30), fp)
	if score < 0 || score > 100 {
		t.Fatalf("score out of range: %d", score)
	}
}

func TestValidateSubmitProfileGuidedMissingPreferred(t *testing.T) {
	r := ValidateSubmitProfile(models.SubmitProfileRequest{
		VoiceInputPath: models.VoiceInputPathGuidedWrite,
		GuidedWrites: map[string]string{
			"guided_hook":       strings.Repeat("word ", 50),
			"guided_hot_take":   strings.Repeat("word ", 50),
			"guided_mini_story": strings.Repeat("word ", 50),
		},
		StyleAnswers: map[string]interface{}{"avoid_words": "delve"},
	})
	if r.OK {
		t.Fatal("expected missing preferred words failure")
	}
}

func TestFeedbackServiceSubmitSuccessFields(t *testing.T) {
	db := setupTestDB(t)
	profile, _ := seedCreator(t, db)
	personaSvc := NewGormPersonaService(db, NewDefaultOnboardingService(), nil)
	promptRepo := NewGormPromptRepository(db)
	scriptRepo := NewGormScriptRepository(db)
	feedbackSvc := NewGormFeedbackService(db, personaSvc, scriptRepo)
	prompt := &models.Prompt{CreatorID: profile.ID, Topic: "t", Variant: "base", PromptText: "p"}
	_ = promptRepo.SavePrompt(prompt)
	script := &models.Script{CreatorID: profile.ID, PromptID: prompt.ID, ScriptText: "body", Source: "gpt-4"}
	_ = scriptRepo.SaveScript(script)
	result, err := feedbackSvc.SubmitFeedback(script.ID, models.ScriptFeedbackRequest{
		Rating:          "sounds_like_me",
		GeneratedScript: longSample(),
		EditedScript:    longSample(),
		Toggles:         []string{"too_formal"},
	})
	if err != nil || result.VoiceMode == "" || result.Feedback == nil {
		t.Fatalf("feedback submit failed: %+v err=%v", result, err)
	}
}

func TestFeedbackApplyError(t *testing.T) {
	db := setupTestDB(t)
	profile := testProfile()
	_ = db.Create(profile)
	promptRepo := NewGormPromptRepository(db)
	scriptRepo := NewGormScriptRepository(db)
	personaSvc := NewGormPersonaService(db, NewDefaultOnboardingService(), nil)
	prompt := &models.Prompt{CreatorID: profile.ID, Topic: "t", Variant: "base", PromptText: "p"}
	_ = promptRepo.SavePrompt(prompt)
	script := &models.Script{CreatorID: profile.ID, PromptID: prompt.ID, ScriptText: "body", Source: "gpt-4"}
	_ = scriptRepo.SaveScript(script)
	feedbackSvc := NewGormFeedbackService(db, personaSvc, scriptRepo)
	if _, err := feedbackSvc.SubmitFeedback(script.ID, models.ScriptFeedbackRequest{Rating: "no"}); err == nil {
		t.Fatal("expected apply error without persona")
	}
}

func TestReanalyzeVoiceSaveError(t *testing.T) {
	db := setupTestDB(t)
	profile, persona := seedCreator(t, db)
	_ = persona
	closed := setupTestDB(t)
	sqlDB, _ := closed.DB()
	sqlDB.Close()
	svc := NewGormPersonaService(closed, NewDefaultOnboardingService(), nil)
	if _, err := svc.ReanalyzeVoice(profile.ID, []string{longSample()}); err == nil {
		t.Fatal("expected reanalyze save error")
	}
}

func TestValidateSubmitProfileRemainingBranches(t *testing.T) {
	r := ValidateSubmitProfile(models.SubmitProfileRequest{
		VoiceInputPath: models.VoiceInputPathSkipCalibrate,
		StyleAnswers: map[string]interface{}{
			"preferred_words": "yaar",
			"avoid_words":     "delve",
		},
	})
	if r.OK {
		t.Fatal("expected missing anti_voice")
	}
	r2 := ValidateSubmitProfile(models.SubmitProfileRequest{
		VoiceInputPath: models.VoiceInputPathGuidedWrite,
		GuidedWrites: map[string]string{
			"guided_hook": strings.Repeat("word ", 50),
		},
		StyleAnswers: map[string]interface{}{
			"preferred_words": "yaar",
			"avoid_words":     "delve",
		},
	})
	if r2.OK {
		t.Fatal("expected missing guided exercises")
	}
}

func TestShiftScoreEmptyOutputWords(t *testing.T) {
	fp := models.VoiceFingerprint{TotalWords: 10, AvgSentenceLength: 5}
	if ShiftScore("!!!", fp) != 0 {
		t.Fatal("expected zero score when output has no words")
	}
}

func TestComputeVoiceConfidenceMidBand(t *testing.T) {
	c := ComputeVoiceConfidence(50, 1, 0, models.PersonaModeDeclared)
	if c < 35 || c > 45 {
		t.Fatalf("expected mid confidence ~40, got %d", c)
	}
}

func TestResolvePersonaModeAllPasteBands(t *testing.T) {
	if ResolvePersonaMode(models.VoiceInputPathPasteScripts, 160) != models.PersonaModeCalibrated {
		t.Fatal("expected calibrated at 160 words")
	}
	if ResolvePersonaMode(models.VoiceInputPathPasteScripts, 20) != models.PersonaModeDeclared {
		t.Fatal("expected declared at 20 words")
	}
}

func TestSplitSentencesMultipleDelimiters(t *testing.T) {
	parts := splitSentences("Hi! How are you? Fine.")
	if len(parts) != 3 {
		t.Fatalf("expected 3 parts, got %d", len(parts))
	}
}

func TestResolvePersonaModeGuidedBands(t *testing.T) {
	if ResolvePersonaMode(models.VoiceInputPathGuidedWrite, 350) != models.PersonaModeDerived {
		t.Fatal("expected derived for guided 350 words")
	}
	if ResolvePersonaMode(models.VoiceInputPathGuidedWrite, 200) != models.PersonaModeCalibrated {
		t.Fatal("expected calibrated for guided 200 words")
	}
}

func TestValidateSubmitProfilePasteWarnTwoSamples(t *testing.T) {
	r := ValidateSubmitProfile(models.SubmitProfileRequest{
		VoiceInputPath: models.VoiceInputPathPasteScripts,
		WritingSamples: []string{"one two three four five", "six seven eight nine ten"},
	})
	if !r.OK || r.Warning == "" {
		t.Fatal("expected warning for two short samples")
	}
}

func TestShiftScoreHighCap(t *testing.T) {
	fp := ExtractFingerprint([]string{longSample()})
	score := ShiftScore(longSample(), fp)
	if score < 95 {
		t.Fatalf("expected very high shift score, got %d", score)
	}
}

func TestSplitSentencesFallbackToFullText(t *testing.T) {
	parts := splitSentences("!!!")
	if len(parts) != 1 || parts[0] != "!!!" {
		t.Fatalf("expected fallback single sentence, got %v", parts)
	}
}

func TestFeedbackCreateError(t *testing.T) {
	db := setupTestDB(t)
	profile, _ := seedCreator(t, db)
	personaSvc := NewGormPersonaService(db, NewDefaultOnboardingService(), nil)
	promptRepo := NewGormPromptRepository(db)
	scriptRepo := NewGormScriptRepository(db)
	prompt := &models.Prompt{CreatorID: profile.ID, Topic: "t", Variant: "base", PromptText: "p"}
	_ = promptRepo.SavePrompt(prompt)
	script := &models.Script{CreatorID: profile.ID, PromptID: prompt.ID, ScriptText: "body", Source: "gpt-4"}
	_ = scriptRepo.SaveScript(script)
	if err := db.Migrator().DropTable(&models.ScriptFeedback{}); err != nil {
		t.Fatal(err)
	}
	feedbackSvc := NewGormFeedbackService(db, personaSvc, scriptRepo)
	if _, err := feedbackSvc.SubmitFeedback(script.ID, models.ScriptFeedbackRequest{Rating: "no"}); err == nil {
		t.Fatal("expected feedback create error")
	}
}

func TestReanalyzeVoiceAnalysisError(t *testing.T) {
	db := setupTestDB(t)
	profile, _ := seedCreator(t, db)
	svc := NewGormPersonaService(db, NewDefaultOnboardingService(), &mockVoiceAnalysisService{err: errTestVoice})
	if _, err := svc.ReanalyzeVoice(profile.ID, []string{longSample()}); err == nil {
		t.Fatal("expected voice analysis error")
	}
}

var errTestVoice = fmt.Errorf("voice analysis failed")

func TestPostChatExhaustedRetries(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":"fail"}`))
	}))
	defer server.Close()
	client := &OpenAIClient{APIKey: "k", Model: "m", BaseURL: server.URL, HTTPClient: server.Client()}
	if _, err := client.ChatCompletion("s", "u", 10, 0.1); err == nil {
		t.Fatal("expected error after retries")
	}
	if attempts < 3 {
		t.Fatalf("expected 3 attempts, got %d", attempts)
	}
}

func TestValidateSubmitProfileSkipMissingPreferred(t *testing.T) {
	r := ValidateSubmitProfile(models.SubmitProfileRequest{
		VoiceInputPath: models.VoiceInputPathSkipCalibrate,
		StyleAnswers: map[string]interface{}{
			"avoid_words": "delve",
			"anti_voice":  "news anchor",
		},
	})
	if r.OK {
		t.Fatal("expected missing preferred on skip path")
	}
}

func TestResolvePersonaModeDefaultEmptyPath(t *testing.T) {
	if ResolvePersonaMode("", 400) != models.PersonaModeDerived {
		t.Fatal("default empty path high words")
	}
	if ResolvePersonaMode("", 100) != models.PersonaModeDeclared {
		t.Fatal("default empty path low words")
	}
}

func TestShiftScoreNegativeClamp(t *testing.T) {
	fp := models.VoiceFingerprint{
		TotalWords: 200, AvgSentenceLength: 40, FillerDensity: 0.2,
		HinglishRatio: 0.2, HookPattern: "story",
	}
	if ShiftScore("hi", fp) != 0 {
		t.Fatal("expected clamped zero score for large mismatch")
	}
}

type brokenBodyTransport struct{}

func (brokenBodyTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       brokenReader{},
		Header:     make(http.Header),
	}, nil
}

type brokenReader struct{}

func (brokenReader) Read([]byte) (int, error) { return 0, fmt.Errorf("broken body") }
func (brokenReader) Close() error             { return nil }

func TestPostChatReadError(t *testing.T) {
	client := &OpenAIClient{
		APIKey:     "k",
		Model:      "m",
		BaseURL:    "http://example.com",
		HTTPClient: &http.Client{Transport: brokenBodyTransport{}},
	}
	if _, err := client.ChatCompletion("s", "u", 10, 0.1); err == nil {
		t.Fatal("expected read error after retries")
	}
}

func TestValidateSubmitProfilePasteSingleLongSample(t *testing.T) {
	long := strings.Repeat("word ", 310)
	r := ValidateSubmitProfile(models.SubmitProfileRequest{
		VoiceInputPath: models.VoiceInputPathPasteScripts,
		WritingSamples: []string{"", long},
	})
	if !r.OK || r.Warning != "" {
		t.Fatalf("expected clean ok for single long sample, got %+v", r)
	}
}

func TestValidateSubmitProfileGuidedMissingAvoid(t *testing.T) {
	r := ValidateSubmitProfile(models.SubmitProfileRequest{
		VoiceInputPath: models.VoiceInputPathGuidedWrite,
		GuidedWrites: map[string]string{
			"guided_hook":       strings.Repeat("word ", 50),
			"guided_hot_take":   strings.Repeat("word ", 50),
			"guided_mini_story": strings.Repeat("word ", 50),
		},
		StyleAnswers: map[string]interface{}{"preferred_words": "yaar"},
	})
	if r.OK {
		t.Fatal("expected missing avoid words")
	}
}

func TestValidateSubmitProfileUnknownPath(t *testing.T) {
	r := ValidateSubmitProfile(models.SubmitProfileRequest{VoiceInputPath: "legacy"})
	if !r.OK {
		t.Fatal("unknown path should pass validation")
	}
}

func TestResolvePersonaModeDefaultCalibrated(t *testing.T) {
	if ResolvePersonaMode("", 200) != models.PersonaModeCalibrated {
		t.Fatal("expected default calibrated at 200 words")
	}
}

func TestGetOrDefaultPersonaExisting(t *testing.T) {
	db := setupTestDB(t)
	profile, persona := seedCreator(t, db)
	svc := NewGormPersonaService(db, NewDefaultOnboardingService(), nil)
	got, err := svc.GetOrDefaultPersona(profile.ID, profile)
	if err != nil || got.ID != persona.ID {
		t.Fatalf("expected existing persona, got %+v err=%v", got, err)
	}
}

func TestChatCompletionMessagesDirect(t *testing.T) {
	server := newMockOpenAIServer(t, "ok", 0)
	defer server.Close()
	client := newOpenAIClientForTest(t, server)
	out, err := client.ChatCompletionMessages([]openAIMessage{
		{Role: "system", Content: "s"},
		{Role: "user", Content: "u"},
	}, 10, 0.2)
	if err != nil || out != "ok" {
		t.Fatalf("messages call failed: %q err=%v", out, err)
	}
}
