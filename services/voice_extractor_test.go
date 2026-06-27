package services

import (
	"strings"
	"testing"

	"github.com/ashisharyan/ghostwriter-prompt-engine/models"
)

func TestExtractFingerprintAndShiftScore(t *testing.T) {
	sample := strings.Repeat("Okay so suno yaar. Real talk — this is basically my style! ", 15)
	fp := ExtractFingerprint([]string{sample})
	if fp.TotalWords < 100 {
		t.Fatalf("expected enough words, got %d", fp.TotalWords)
	}
	if fp.HookPattern == "" || fp.HookPattern == "unknown" {
		t.Fatalf("expected hook pattern, got %s", fp.HookPattern)
	}

	score := ShiftScore(sample, fp)
	if score < 80 {
		t.Fatalf("expected high shift score for same text, got %d", score)
	}
	if ShiftScore("", fp) != 0 {
		t.Fatal("empty output should score 0")
	}
}

func TestResolvePersonaMode(t *testing.T) {
	if ResolvePersonaMode(models.VoiceInputPathSkipCalibrate, 500) != models.PersonaModeDeclared {
		t.Fatal("skip should stay declared")
	}
	if ResolvePersonaMode(models.VoiceInputPathPasteScripts, 350) != models.PersonaModeDerived {
		t.Fatal("paste with enough words should be derived")
	}
	if ResolvePersonaMode(models.VoiceInputPathGuidedWrite, 200) != models.PersonaModeCalibrated {
		t.Fatal("guided with 200 words should be calibrated")
	}
}

func TestComputeVoiceConfidence(t *testing.T) {
	low := ComputeVoiceConfidence(0, 0, 0, models.PersonaModeDeclared)
	high := ComputeVoiceConfidence(400, 3, 2, models.PersonaModeDerived)
	if high <= low {
		t.Fatalf("expected higher confidence: low=%d high=%d", low, high)
	}
}

func TestNormalizeSamplesAndBuildFromSubmit(t *testing.T) {
	legacy := []string{"sample one", "sample two"}
	structured := []models.WritingSample{{Text: "sample one", Source: models.SampleSourcePastScript}}
	merged := NormalizeSamples(legacy, structured)
	if len(merged) != 2 {
		t.Fatalf("expected deduped merge, got %d", len(merged))
	}

	req := models.SubmitProfileRequest{
		VoiceInputPath: models.VoiceInputPathGuidedWrite,
		GuidedWrites: map[string]string{
			"guided_hook":       strings.Repeat("word ", 50),
			"guided_hot_take":   strings.Repeat("word ", 50),
			"guided_mini_story": strings.Repeat("word ", 50),
		},
	}
	samples := BuildSamplesFromSubmit(req)
	if len(samples) != 3 {
		t.Fatalf("expected 3 guided samples, got %d", len(samples))
	}
}

func TestLexicalHintsFromFingerprint(t *testing.T) {
	text := "Okay so suno yaar matlab basically this is fine."
	fp := ExtractFingerprint([]string{text})
	lex := LexicalHintsFromFingerprint([]string{text}, fp)
	if len(lex.FillerWords) == 0 {
		t.Fatal("expected filler words detected")
	}
}

func TestClassifyHookPatterns(t *testing.T) {
	cases := map[string]string{
		"Why does this happen?":     "question",
		"Okay so suno yaar":         "punch",
		"When I was in college":     "story",
		"This is a very long setup that keeps going and going and going and going and going today": "slow_setup",
	}
	for line, want := range cases {
		if got := classifyHook(line); got != want {
			t.Fatalf("hook %q: want %s got %s", line, want, got)
		}
	}
}

func TestPromotePersonaMode(t *testing.T) {
	if PromotePersonaMode(models.PersonaModeDeclared, 350) != models.PersonaModeDerived {
		t.Fatal("expected derived promotion")
	}
}
