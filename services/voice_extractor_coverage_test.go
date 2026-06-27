package services

import (
	"strings"
	"testing"

	"github.com/ashisharyan/ghostwriter-prompt-engine/models"
)

func TestValidateSubmitProfileAllBranches(t *testing.T) {
	long := strings.Repeat("word ", 310)
	okPaste := ValidateSubmitProfile(models.SubmitProfileRequest{
		VoiceInputPath: models.VoiceInputPathPasteScripts,
		WritingSamples: []string{long},
	})
	if !okPaste.OK || okPaste.Warning != "" {
		t.Fatal("expected ok paste with 300+ words")
	}

	okGuided := ValidateSubmitProfile(models.SubmitProfileRequest{
		VoiceInputPath: models.VoiceInputPathGuidedWrite,
		GuidedWrites: map[string]string{
			"guided_hook": strings.Repeat("word ", 50),
			"guided_hot_take": strings.Repeat("word ", 50),
			"guided_mini_story": strings.Repeat("word ", 50),
		},
		StyleAnswers: map[string]interface{}{
			"preferred_words": "yaar",
			"avoid_words":     "delve",
		},
	})
	if !okGuided.OK {
		t.Fatalf("expected guided ok: %s", okGuided.Message)
	}

	defaultPath := ValidateSubmitProfile(models.SubmitProfileRequest{
		WritingSamples: []string{long, long},
	})
	if !defaultPath.OK {
		t.Fatal("expected default paste path ok")
	}

	skipMissing := ValidateSubmitProfile(models.SubmitProfileRequest{
		VoiceInputPath: models.VoiceInputPathSkipCalibrate,
		StyleAnswers:   map[string]interface{}{"preferred_words": "yaar"},
	})
	if skipMissing.OK {
		t.Fatal("expected skip missing avoid failure")
	}
}

func TestResolvePersonaModeDefaultPath(t *testing.T) {
	if ResolvePersonaMode("unknown", 400) != models.PersonaModeDerived {
		t.Fatal("unknown path with enough words should derive")
	}
	if ResolvePersonaMode(models.VoiceInputPathPasteScripts, 50) != models.PersonaModeDeclared {
		t.Fatal("low word paste should be declared")
	}
	if PromotePersonaMode(models.PersonaModeCalibrated, 100) != models.PersonaModeCalibrated {
		t.Fatal("calibrated should not downgrade")
	}
}

func TestShiftScoreEdgeCases(t *testing.T) {
	fp := models.VoiceFingerprint{TotalWords: 10, AvgSentenceLength: 5, FillerDensity: 0.1, HinglishRatio: 0.1, HookPattern: "punch"}
	if ShiftScore("totally different unrelated content without overlap", fp) >= 100 {
		t.Fatal("expected lower shift score for mismatched output")
	}
	if ShiftScore("word", models.VoiceFingerprint{}) != 0 {
		t.Fatal("empty fingerprint should score 0")
	}
}

func TestBuildFewShotBlockLimit(t *testing.T) {
	samples := make([]models.WritingSample, 0, 6)
	for i := 0; i < 6; i++ {
		samples = append(samples, models.WritingSample{Text: strings.Repeat("sample ", 20), Source: models.SampleSourcePastScript})
	}
	block := buildFewShotBlock(&models.PersonaProfile{
		VoiceMode: models.PersonaModeDerived,
		Samples:   samples,
	})
	if !strings.Contains(block, "Example 5") || strings.Contains(block, "Example 6") {
		t.Fatal("expected few-shot block capped at 5 examples")
	}
	if buildFewShotBlock(&models.PersonaProfile{VoiceMode: models.PersonaModeDeclared}) != "" {
		t.Fatal("declared mode should not include few-shots")
	}
}

func TestMergeLegacyProfileHintsAll(t *testing.T) {
	profile := &models.CreatorProfile{
		Tone: "witty and fun", Style: "professional humor",
		Language: "Hindi",
	}
	scores := mergeLegacyProfileHints(defaultScores(), profile)
	if scores.Humor <= defaultScore || scores.Formality <= defaultScore || scores.HinglishMix <= defaultScore {
		t.Fatalf("expected positive deltas: %+v", scores)
	}
}

func TestInferDeltasNoWithExisting(t *testing.T) {
	d := inferDeltasFromFeedback("no", "too formal")
	if d["formality"] >= 0 {
		t.Fatal("expected doubled negative formality")
	}
}

func TestVoiceExtractorHelpers(t *testing.T) {
	if functionWordFreq(nil) == nil {
		t.Fatal("expected empty map not nil")
	}
	if mean, std := meanStd(nil); mean != 0 || std != 0 {
		t.Fatal("empty meanStd")
	}
	if firstLine("") != "" {
		t.Fatal("empty first line")
	}
	if firstLineStarter("") != "" {
		t.Fatal("empty starter")
	}
	if classifyHook("") != "unknown" {
		t.Fatal("empty hook unknown")
	}
	if isFiller("yaar") != true || isFiller("zzz") != false {
		t.Fatal("filler check")
	}
	lex := LexicalHintsFromFingerprint([]string{"plain english only"}, models.VoiceFingerprint{FillerDensity: 0.01, HinglishRatio: 0.01})
	if lex.FillerFrequency != "low" {
		t.Fatalf("expected low filler freq, got %s", lex.FillerFrequency)
	}
}

func TestComputeVoiceConfidenceCap(t *testing.T) {
	if c := ComputeVoiceConfidence(500, 5, 10, models.PersonaModeDerived); c != 100 {
		t.Fatalf("expected cap at 100, got %d", c)
	}
}

func TestBuildPersonaSummaryAllOptionalFields(t *testing.T) {
	db := setupTestDB(t)
	svc := NewGormPersonaService(db, NewDefaultOnboardingService(), nil)
	profile := testProfile()
	persona := &models.PersonaProfile{
		VoiceMode: models.PersonaModeDerived, VoiceConfidence: 80,
		CurrentScores: defaultScores(), VoiceSummary: "summary",
	}
	s := svc.BuildPersonaSummary(profile, persona)
	for _, part := range []string{"Bio:", "Audience:", "Goal:", "Inspiration:", "USP:", "Voice summary:"} {
		if !strings.Contains(s, part) {
			t.Fatalf("missing %s in summary", part)
		}
	}
}

func TestStorePromptVariantsError(t *testing.T) {
	db := setupTestDB(t)
	profile, persona := seedCreator(t, db)
	closed := setupTestDB(t)
	sqlDB, _ := closed.DB()
	sqlDB.Close()
	repo := NewGormPromptRepository(closed)
	ab := NewDefaultPromptABService(NewGormPersonaService(db, NewDefaultOnboardingService(), nil), repo)
	variants, _ := ab.GeneratePromptVariants(profile, persona, "topic")
	if err := ab.StorePromptVariants(variants, profile, "topic"); err == nil {
		t.Fatal("expected store error on closed db")
	}
}

func TestSplitSentencesSingleFallback(t *testing.T) {
	s := splitSentences("   ")
	if len(s) != 0 {
		t.Fatal("whitespace only should return nil/empty")
	}
}

func TestBuildSamplesDefaultPath(t *testing.T) {
	samples := BuildSamplesFromSubmit(models.SubmitProfileRequest{
		WritingSamples: []string{"hello world sample text here"},
	})
	if len(samples) != 1 || samples[0].Source != models.SampleSourcePastScript {
		t.Fatal("default path should treat as paste scripts")
	}
}
